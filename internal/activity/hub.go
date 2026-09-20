package activity

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/rals-dev/rals-hermes/internal/hermes"
	"github.com/rals-dev/rals-hermes/internal/view"
)

// Source is the slice of hermes.Client the poller needs.
type Source interface {
	Name() string
	Sessions(ctx context.Context, q hermes.SessionsQuery) (*hermes.SessionList, error)
	Messages(ctx context.Context, id string, q hermes.MessagesQuery) (*hermes.MessageList, error)
}

// Config bounds the poller (ADR-018).
type Config struct {
	PollInterval  time.Duration
	Window        time.Duration
	MaxSessions   int
	IdleStopAfter time.Duration
}

// SubscriberGauge is implemented by observ.Metrics; optional.
type SubscriberGauge interface {
	SetActivitySubscribers(profile string, n int)
}

// ErrUnknownProfile is returned by Subscribe for a profile not in the hub.
var ErrUnknownProfile = errors.New("activity: unknown profile")

// reentryTail is how many trailing messages are fetched when a session that
// was not tracked (dormant, then active again) enters the window. Only
// messages newer than the window cutoff are emitted from that tail, so an
// old session waking up costs one small fetch instead of a full replay.
const reentryTail = 8

// subscriberBuffer is the per-subscriber channel depth. A subscriber that
// falls this far behind loses events rather than stalling the poller; the
// next snapshot resynchronises it.
const subscriberBuffer = 256

// Hub owns one lazy poller per profile.
type Hub struct {
	cfg     Config
	log     *slog.Logger
	gauge   SubscriberGauge
	pollers map[string]*poller
}

// NewHub builds a hub for the given sources. Nothing runs until Subscribe.
func NewHub(cfg Config, sources []Source, log *slog.Logger) *Hub {
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 5 * time.Second
	}
	if cfg.Window <= 0 {
		cfg.Window = 10 * time.Minute
	}
	if cfg.MaxSessions <= 0 {
		cfg.MaxSessions = 10
	}
	if cfg.IdleStopAfter <= 0 {
		cfg.IdleStopAfter = 30 * time.Second
	}
	h := &Hub{cfg: cfg, log: log, pollers: make(map[string]*poller, len(sources))}
	for _, s := range sources {
		h.pollers[s.Name()] = newPoller(h, s)
	}
	return h
}

// SetGauge wires the subscriber gauge (optional).
func (h *Hub) SetGauge(g SubscriberGauge) { h.gauge = g }

// Subscribe attaches to a profile's feed. The poller starts on the first
// subscriber and stops IdleStopAfter after the last one cancels.
func (h *Hub) Subscribe(profile string) (<-chan Event, func(), error) {
	p, ok := h.pollers[profile]
	if !ok {
		return nil, nil, ErrUnknownProfile
	}
	return p.subscribe()
}

// Subscribers reports the open subscriptions for a profile.
func (h *Hub) Subscribers(profile string) int {
	p, ok := h.pollers[profile]
	if !ok {
		return 0
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.subs)
}

// poller tails one profile.
type poller struct {
	hub *Hub
	src Source
	log *slog.Logger

	mu       sync.Mutex
	subs     map[*subscriber]struct{}
	running  bool
	stop     chan struct{} // closed to stop the current run loop
	idleStop *time.Timer

	// tracked sessions: id → last seen message_count (the message cursor)
	// and the last state, for change detection.
	seen map[string]trackedSession
	// baselined flips after the first successful tick; until then every
	// tick is a baseline so a failed first poll never replays history.
	baselined bool
}

type trackedSession struct {
	cursor int
	ended  bool
}

type subscriber struct {
	ch chan Event
}

func newPoller(h *Hub, src Source) *poller {
	return &poller{hub: h, src: src, log: h.log.With("profile", src.Name()), subs: map[*subscriber]struct{}{}}
}

func (p *poller) subscribe() (<-chan Event, func(), error) {
	s := &subscriber{ch: make(chan Event, subscriberBuffer)}
	p.mu.Lock()
	p.subs[s] = struct{}{}
	if p.idleStop != nil {
		p.idleStop.Stop()
		p.idleStop = nil
	}
	if !p.running {
		p.running = true
		p.stop = make(chan struct{})
		p.seen = map[string]trackedSession{}
		p.baselined = false
		go p.run(p.stop)
	}
	n := len(p.subs)
	p.mu.Unlock()
	p.reportGauge(n)

	var once sync.Once
	cancel := func() {
		once.Do(func() {
			p.mu.Lock()
			delete(p.subs, s)
			n := len(p.subs)
			if n == 0 && p.running {
				stop := p.stop
				p.idleStop = time.AfterFunc(p.hub.cfg.IdleStopAfter, func() {
					p.mu.Lock()
					defer p.mu.Unlock()
					if len(p.subs) == 0 && p.running && p.stop == stop {
						close(stop)
						p.running = false
					}
				})
			}
			p.mu.Unlock()
			p.reportGauge(n)
		})
	}
	return s.ch, cancel, nil
}

func (p *poller) reportGauge(n int) {
	if p.hub.gauge != nil {
		p.hub.gauge.SetActivitySubscribers(p.src.Name(), n)
	}
}

// run is the poll loop. The first tick establishes a baseline (snapshots,
// cursors at the current message counts); later ticks emit deltas.
func (p *poller) run(stop <-chan struct{}) {
	p.log.Debug("activity poller started")
	defer p.log.Debug("activity poller stopped")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { <-stop; cancel() }()

	p.tick(ctx)
	t := time.NewTicker(p.hub.cfg.PollInterval)
	defer t.Stop()
	for {
		select {
		case <-stop:
			return
		case <-t.C:
			p.tick(ctx)
		}
	}
}

func (p *poller) tick(ctx context.Context) {
	now := time.Now().UTC()
	baseline := !p.baselined
	list, err := p.src.Sessions(ctx, hermes.SessionsQuery{Limit: p.hub.cfg.MaxSessions * 2, IncludeChildren: true})
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		p.log.Warn("activity poll failed", "err", err)
		p.emit(Event{Type: TypeError, Profile: p.src.Name(), At: now, Error: errorBodyFor(err)})
		return
	}

	// Sessions arrive newest-first; keep those inside the window, capped.
	cutoff := now.Add(-p.hub.cfg.Window)
	current := make(map[string]hermes.Session, p.hub.cfg.MaxSessions)
	for _, s := range list.Data {
		if s.LastActive.Time().Before(cutoff) {
			continue
		}
		if len(current) >= p.hub.cfg.MaxSessions {
			break
		}
		current[s.ID] = s
	}

	p.baselined = true

	for id, s := range current {
		if baseline {
			// Adopt the current cursor and describe the session; never replay
			// what happened before the subscriber arrived.
			p.seen[id] = trackedSession{cursor: s.MessageCount, ended: s.EndReason != nil}
			p.emit(p.sessionEvent(TypeSnapshot, s, now))
			continue
		}
		prev, known := p.seen[id]
		if !known {
			// First sight after baseline. Either the session is genuinely new
			// (started inside the window) or it was dormant and woke up; in
			// both cases only the tail is fetched and anything older than the
			// window is dropped, so history is never replayed.
			prev = trackedSession{cursor: max(0, s.MessageCount-reentryTail)}
			switch {
			case s.StartedAt.Time().Before(cutoff):
				p.emit(p.sessionEvent(TypeSnapshot, s, now))
			case s.ParentSessionID != nil:
				p.emit(p.sessionEvent(TypeSubagentStart, s, now))
			default:
				p.emit(p.sessionEvent(TypeSessionStarted, s, now))
			}
		}
		if s.MessageCount > prev.cursor {
			p.tail(ctx, s, prev.cursor, cutoff, now)
			prev.cursor = s.MessageCount
			p.emit(p.sessionEvent(TypeSnapshot, s, now))
		}
		if s.EndReason != nil && !prev.ended {
			prev.ended = true
			typ := TypeSessionEnded
			if s.ParentSessionID != nil {
				typ = TypeSubagentComplete
			}
			p.emit(p.sessionEvent(typ, s, now))
		}
		p.seen[id] = prev
	}

	// Forget sessions that left the window so cursors do not accumulate.
	for id := range p.seen {
		if _, still := current[id]; !still {
			delete(p.seen, id)
		}
	}
}

// tail fetches messages after cursor and emits one event per message that
// is not older than cutoff.
func (p *poller) tail(ctx context.Context, s hermes.Session, cursor int, cutoff, now time.Time) {
	delta := s.MessageCount - cursor
	msgs, err := p.src.Messages(ctx, s.ID, hermes.MessagesQuery{Offset: cursor, Limit: delta, Order: "oldest"})
	if err != nil {
		if ctx.Err() == nil {
			p.log.Warn("activity tail failed", "session", s.ID, "err", err)
			p.emit(Event{Type: TypeError, Profile: p.src.Name(), SessionID: s.ID, At: now, Error: errorBodyFor(err)})
		}
		return
	}
	for _, m := range msgs.Data {
		if m.Timestamp != 0 && m.Timestamp.Time().Before(cutoff) {
			continue
		}
		for _, e := range messageEvents(p.src.Name(), s.ID, m) {
			p.emit(e)
		}
	}
}

// messageEvents converts one Hermes message into feed events.
func messageEvents(profile, sessionID string, m hermes.Message) []Event {
	at := m.Timestamp.Time()
	base := Event{Profile: profile, SessionID: sessionID, At: at, MessageID: m.ID, Role: m.Role}
	switch m.Role {
	case "assistant":
		if len(m.ToolCalls) == 0 {
			e := base
			e.Type = TypeMessage
			if m.Content != nil {
				e.Preview = preview(*m.Content)
			}
			return []Event{e}
		}
		out := make([]Event, 0, len(m.ToolCalls))
		for _, tc := range m.ToolCalls {
			e := base
			e.Type, e.Tool, e.CallID, e.Preview = TypeToolStarted, tc.Function.Name, tc.ID, preview(tc.Function.Arguments)
			out = append(out, e)
		}
		return out
	case "tool":
		e := base
		e.Type, e.Tool, e.CallID = TypeToolCompleted, m.ToolName, m.ToolCallID
		if m.Content != nil {
			e.Preview = preview(*m.Content)
		}
		return []Event{e}
	case "user":
		e := base
		e.Type = TypeMessage
		if m.Content != nil {
			e.Preview = preview(*m.Content)
		}
		return []Event{e}
	default:
		return nil // session_meta and friends carry nothing for the feed
	}
}

func (p *poller) sessionEvent(typ string, s hermes.Session, now time.Time) Event {
	sc := view.FromSession(s)
	e := Event{Type: typ, Profile: p.src.Name(), SessionID: s.ID, At: now, Session: &sc}
	if s.ParentSessionID != nil {
		e.ParentSessionID = *s.ParentSessionID
	}
	if s.EndReason != nil {
		e.EndReason = *s.EndReason
	}
	return e
}

// emit fans an event out to every subscriber without blocking the poller.
func (p *poller) emit(e Event) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for s := range p.subs {
		select {
		case s.ch <- e:
		default:
			// Subscriber is not keeping up; drop. The next snapshot resyncs.
		}
	}
}

func errorBodyFor(err error) *ErrorBody {
	var ue *hermes.UpstreamError
	if errors.As(err, &ue) {
		switch ue.Kind {
		case hermes.KindUnauthorized:
			return &ErrorBody{Code: "upstream_unauthorized", Message: "the profile key was rejected by Hermes"}
		case hermes.KindBusy:
			return &ErrorBody{Code: "upstream_busy", Message: "Hermes is at its concurrency limit"}
		case hermes.KindUnreachable:
			return &ErrorBody{Code: "upstream_unreachable", Message: "Hermes did not answer in time"}
		default:
			return &ErrorBody{Code: "upstream_error", Message: "Hermes returned an unexpected response"}
		}
	}
	return &ErrorBody{Code: "internal_error", Message: "internal error"}
}
