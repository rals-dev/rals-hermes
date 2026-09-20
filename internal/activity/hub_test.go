package activity

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rals-dev/rals-hermes/internal/hermes"
)

// fakeSource is a scripted Hermes profile whose sessions and messages the
// test mutates between polls.
type fakeSource struct {
	name string
	mu   sync.Mutex
	now  func() time.Time

	sessions []hermes.Session
	messages map[string][]hermes.Message
	fail     error

	sessionCalls atomic.Int32
	messageCalls atomic.Int32
	lastMsgQuery hermes.MessagesQuery
}

func (f *fakeSource) Name() string { return f.name }

func (f *fakeSource) Sessions(_ context.Context, _ hermes.SessionsQuery) (*hermes.SessionList, error) {
	f.sessionCalls.Add(1)
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.fail != nil {
		return nil, f.fail
	}
	out := make([]hermes.Session, len(f.sessions))
	copy(out, f.sessions)
	return &hermes.SessionList{Data: out}, nil
}

func (f *fakeSource) Messages(_ context.Context, id string, q hermes.MessagesQuery) (*hermes.MessageList, error) {
	f.messageCalls.Add(1)
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lastMsgQuery = q
	all := f.messages[id]
	if q.Offset > len(all) {
		q.Offset = len(all)
	}
	end := len(all)
	if q.Limit > 0 && q.Offset+q.Limit < end {
		end = q.Offset + q.Limit
	}
	return &hermes.MessageList{SessionID: id, Data: all[q.Offset:end]}, nil
}

func (f *fakeSource) setSession(s hermes.Session) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i := range f.sessions {
		if f.sessions[i].ID == s.ID {
			f.sessions[i] = s
			return
		}
	}
	f.sessions = append(f.sessions, s)
}

func (f *fakeSource) appendMessages(id string, msgs ...hermes.Message) {
	f.mu.Lock()
	f.messages[id] = append(f.messages[id], msgs...)
	f.mu.Unlock()
	f.mu.Lock()
	for i := range f.sessions {
		if f.sessions[i].ID == id {
			f.sessions[i].MessageCount = len(f.messages[id])
			f.sessions[i].LastActive = hermes.UnixTime(float64(f.now().Unix()))
		}
	}
	f.mu.Unlock()
}

func unix(t time.Time) hermes.UnixTime { return hermes.UnixTime(float64(t.UnixNano()) / 1e9) }

func session(id string, lastActive time.Time, msgCount int) hermes.Session {
	return hermes.Session{ID: id, Source: "telegram", LastActive: unix(lastActive), StartedAt: unix(lastActive), MessageCount: msgCount}
}

func toolCallMsg(id int64, name, args string) hermes.Message {
	m := hermes.Message{ID: id, Role: "assistant", Timestamp: unix(time.Now())}
	tc := hermes.ToolCall{ID: "call-" + name, Type: "function"}
	tc.Function.Name, tc.Function.Arguments = name, args
	m.ToolCalls = []hermes.ToolCall{tc}
	return m
}

func toolResultMsg(id int64, name, content string) hermes.Message {
	return hermes.Message{ID: id, Role: "tool", ToolName: name, ToolCallID: "call-" + name, Content: &content, Timestamp: unix(time.Now())}
}

func newFake(name string) *fakeSource {
	return &fakeSource{name: name, now: time.Now, messages: map[string][]hermes.Message{}}
}

func testConfig() Config {
	return Config{PollInterval: 15 * time.Millisecond, Window: 10 * time.Minute, MaxSessions: 3, IdleStopAfter: 40 * time.Millisecond}
}

func newTestHub(sources ...Source) *Hub {
	return NewHub(testConfig(), sources, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

// collect drains events until pred is satisfied or the deadline passes.
func collect(t *testing.T, ch <-chan Event, deadline time.Duration, pred func(Event) bool) []Event {
	t.Helper()
	var got []Event
	timer := time.After(deadline)
	for {
		select {
		case e := <-ch:
			got = append(got, e)
			if pred != nil && pred(e) {
				return got
			}
		case <-timer:
			return got
		}
	}
}

func hasType(typ string) func(Event) bool { return func(e Event) bool { return e.Type == typ } }

func TestHub_UnknownProfileIsRejected(t *testing.T) {
	h := newTestHub(newFake("default"))
	if _, _, err := h.Subscribe("ghost"); !errors.Is(err, ErrUnknownProfile) {
		t.Fatalf("err = %v, want ErrUnknownProfile", err)
	}
}

func TestHub_IsLazy_NoPollingWithoutSubscribersAndStopsAfterIdle(t *testing.T) {
	src := newFake("default")
	h := newTestHub(src)

	time.Sleep(60 * time.Millisecond)
	if n := src.sessionCalls.Load(); n != 0 {
		t.Fatalf("polled %d times with no subscriber", n)
	}

	_, cancel, err := h.Subscribe("default")
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(60 * time.Millisecond)
	if n := src.sessionCalls.Load(); n == 0 {
		t.Fatal("no polling after subscribe")
	}
	cancel()
	time.Sleep(testConfig().IdleStopAfter + 60*time.Millisecond)
	after := src.sessionCalls.Load()
	time.Sleep(60 * time.Millisecond)
	if src.sessionCalls.Load() != after {
		t.Errorf("still polling %v after the last subscriber left", testConfig().IdleStopAfter)
	}
}

func TestHub_FirstSubscriberGetsSnapshotWithoutMessageReplay(t *testing.T) {
	src := newFake("default")
	src.setSession(session("s1", time.Now(), 3))
	src.appendMessages("s1", toolCallMsg(1, "terminal", `{"cmd":"ls"}`), toolResultMsg(2, "terminal", "ok"), toolCallMsg(3, "read_file", `{}`))
	h := newTestHub(src)

	ch, cancel, _ := h.Subscribe("default")
	defer cancel()
	got := collect(t, ch, 120*time.Millisecond, hasType("session.snapshot"))
	if len(got) == 0 || got[len(got)-1].Type != "session.snapshot" || got[len(got)-1].SessionID != "s1" || got[len(got)-1].Profile != "default" {
		t.Fatalf("events = %+v", got)
	}
	for _, e := range got {
		if e.Type == "tool.started" || e.Type == "tool.completed" {
			t.Errorf("history must not be replayed on subscribe: %+v", e)
		}
	}
	if src.messageCalls.Load() != 0 {
		t.Errorf("messages fetched %d times during baseline, want 0", src.messageCalls.Load())
	}
}

func TestHub_NewMessagesBecomeToolEventsFetchedIncrementally(t *testing.T) {
	src := newFake("default")
	src.setSession(session("s1", time.Now(), 2))
	src.appendMessages("s1", toolCallMsg(1, "terminal", `{"cmd":"ls"}`), toolResultMsg(2, "terminal", "ok"))
	h := newTestHub(src)
	ch, cancel, _ := h.Subscribe("default")
	defer cancel()
	collect(t, ch, 100*time.Millisecond, hasType("session.snapshot"))

	src.appendMessages("s1", toolCallMsg(3, "web_search", `{"q":"golang sse"}`), toolResultMsg(4, "web_search", "3 results"))
	got := collect(t, ch, 300*time.Millisecond, hasType("tool.completed"))

	var started, completed *Event
	for i := range got {
		switch got[i].Type {
		case "tool.started":
			started = &got[i]
		case "tool.completed":
			completed = &got[i]
		}
	}
	if started == nil || completed == nil {
		t.Fatalf("events = %+v", got)
	}
	if started.Tool != "web_search" || started.Preview == "" || started.SessionID != "s1" || started.Profile != "default" {
		t.Errorf("tool.started = %+v", *started)
	}
	if completed.Tool != "web_search" || completed.CallID != "call-web_search" || completed.Preview != "3 results" {
		t.Errorf("tool.completed = %+v", *completed)
	}
	if q := src.lastMsgQuery; q.Offset != 2 || q.Order != "oldest" {
		t.Errorf("incremental fetch used %+v, want Offset=2 Order=oldest", q)
	}
}

func TestHub_ChildSessionsBecomeSubagentEvents(t *testing.T) {
	src := newFake("default")
	src.setSession(session("parent", time.Now(), 0))
	h := newTestHub(src)
	ch, cancel, _ := h.Subscribe("default")
	defer cancel()
	collect(t, ch, 100*time.Millisecond, hasType("session.snapshot"))

	parent := "parent"
	child := session("child", time.Now(), 0)
	child.ParentSessionID = &parent
	src.setSession(child)
	got := collect(t, ch, 300*time.Millisecond, hasType("subagent.start"))
	if len(got) == 0 || got[len(got)-1].SessionID != "child" || got[len(got)-1].ParentSessionID != "parent" {
		t.Fatalf("events = %+v", got)
	}

	reason := "agent_close"
	child.EndReason = &reason
	child.EndedAt = ptr(unix(time.Now()))
	child.OutputTokens = 123
	src.setSession(child)
	got = collect(t, ch, 300*time.Millisecond, hasType("subagent.complete"))
	last := got[len(got)-1]
	if last.Type != "subagent.complete" || last.SessionID != "child" || last.EndReason != "agent_close" || last.Session == nil || last.Session.Usage.OutputTokens != 123 {
		t.Fatalf("events = %+v", got)
	}
}

func ptr[T any](v T) *T { return &v }

func TestHub_WindowAndCapBoundTrackedSessions(t *testing.T) {
	src := newFake("default")
	now := time.Now()
	src.setSession(session("old", now.Add(-time.Hour), 5)) // outside the 10-minute window
	for i := range 5 {
		src.setSession(session("s"+string(rune('a'+i)), now.Add(-time.Duration(i)*time.Second), 1))
	}
	h := newTestHub(src) // MaxSessions = 3
	ch, cancel, _ := h.Subscribe("default")
	defer cancel()
	got := collect(t, ch, 150*time.Millisecond, nil)

	ids := map[string]bool{}
	for _, e := range got {
		if e.Type == "session.snapshot" {
			ids[e.SessionID] = true
		}
	}
	if ids["old"] {
		t.Error("session outside the window was tracked")
	}
	if len(ids) != 3 {
		t.Errorf("tracked %d sessions, want MaxSessions=3: %v", len(ids), ids)
	}
	if !ids["sa"] || !ids["sb"] || !ids["sc"] {
		t.Errorf("expected the three most recent sessions, got %v", ids)
	}
}

func TestHub_UpstreamFailureIsAnErrorEventAndPollingContinues(t *testing.T) {
	src := newFake("default")
	src.mu.Lock()
	src.fail = &hermes.UpstreamError{Profile: "default", Kind: hermes.KindUnreachable}
	src.mu.Unlock()
	h := newTestHub(src)
	ch, cancel, _ := h.Subscribe("default")
	defer cancel()

	got := collect(t, ch, 200*time.Millisecond, hasType("error"))
	if len(got) == 0 || got[len(got)-1].Type != "error" || got[len(got)-1].Error == nil || got[len(got)-1].Error.Code != "upstream_unreachable" {
		t.Fatalf("events = %+v", got)
	}
	src.mu.Lock()
	src.fail = nil
	src.sessions = []hermes.Session{session("s1", time.Now(), 0)}
	src.mu.Unlock()
	got = collect(t, ch, 300*time.Millisecond, hasType("session.snapshot"))
	if len(got) == 0 || got[len(got)-1].Type != "session.snapshot" {
		t.Fatalf("polling did not recover after upstream came back: %+v", got)
	}
}

func TestHub_TwoSubscribersShareOnePoller(t *testing.T) {
	src := newFake("default")
	src.setSession(session("s1", time.Now(), 0))
	h := newTestHub(src)
	ch1, c1, _ := h.Subscribe("default")
	ch2, c2, _ := h.Subscribe("default")
	defer c1()
	defer c2()
	collect(t, ch1, 100*time.Millisecond, hasType("session.snapshot"))
	collect(t, ch2, 100*time.Millisecond, hasType("session.snapshot"))
	calls := src.sessionCalls.Load()
	time.Sleep(100 * time.Millisecond)
	delta := src.sessionCalls.Load() - calls
	// ~100ms / 15ms ≈ 6-7 polls for one poller; two pollers would double it.
	if delta > 9 {
		t.Errorf("%d polls in 100ms — looks like more than one poller", delta)
	}
	if h.Subscribers("default") != 2 {
		t.Errorf("Subscribers = %d, want 2", h.Subscribers("default"))
	}
}

// Regression: a session that was outside the window at baseline and becomes
// active again must not have its whole history replayed (observed live on
// 2026-09-20: 435 messages streamed after one Telegram reply).
func TestHub_SessionReenteringTheWindowDoesNotReplayHistory(t *testing.T) {
	src := newFake("default")
	old := time.Now().Add(-time.Hour)
	// 40 historical messages, all older than the window.
	var history []hermes.Message
	for i := range 20 {
		tc := toolCallMsg(int64(100+2*i), "terminal", `{"cmd":"old"}`)
		tc.Timestamp = unix(old)
		tr := toolResultMsg(int64(101+2*i), "terminal", "old output")
		tr.Timestamp = unix(old)
		history = append(history, tc, tr)
	}
	src.setSession(session("dormant", old, 0))
	src.appendMessages("dormant", history...)
	src.mu.Lock()
	src.sessions[0].LastActive = unix(old) // appendMessages bumped it; keep it dormant
	src.mu.Unlock()

	h := newTestHub(src)
	ch, cancel, _ := h.Subscribe("default")
	defer cancel()
	if got := collect(t, ch, 80*time.Millisecond, hasType("session.snapshot")); len(got) != 0 {
		t.Fatalf("dormant session must not be tracked at baseline, got %+v", got)
	}

	// One new message wakes the session up.
	src.appendMessages("dormant", toolCallMsg(200, "web_search", `{"q":"now"}`))
	got := collect(t, ch, 400*time.Millisecond, func(e Event) bool { return e.Type == "tool.started" && e.Tool == "web_search" })

	var replayed int
	for _, e := range got {
		if e.Tool == "terminal" {
			replayed++
		}
	}
	if replayed != 0 {
		t.Errorf("%d historical tool events replayed, want 0: %+v", replayed, got)
	}
	if len(got) == 0 || got[len(got)-1].Tool != "web_search" {
		t.Errorf("the new message was not delivered: %+v", got)
	}
	if q := src.lastMsgQuery; q.Offset == 0 {
		t.Errorf("tail fetched from offset 0 (full history), want a bounded tail: %+v", q)
	}
	for _, e := range got {
		if e.Type == "session.started" {
			t.Errorf("a re-entering session is not new; got session.started")
		}
	}
}
