package cache

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func (f *fakeClock) Now() time.Time { f.mu.Lock(); defer f.mu.Unlock(); return f.t }
func (f *fakeClock) Advance(d time.Duration) {
	f.mu.Lock()
	f.t = f.t.Add(d)
	f.mu.Unlock()
}

func newTestCache(ttl time.Duration) (*Cache[int], *fakeClock) {
	clk := &fakeClock{t: time.Unix(1_000_000, 0)}
	c := New[int](ttl)
	c.now = clk.Now
	return c, clk
}

func TestGet_HitWithinTTLDoesNotCallLoader(t *testing.T) {
	c, clk := newTestCache(3 * time.Second)
	var calls atomic.Int32
	load := func(context.Context) (int, error) { calls.Add(1); return 42, nil }

	v, hit, err := c.Get(context.Background(), "k", load)
	if err != nil || v != 42 || hit {
		t.Fatalf("first Get = %d hit=%v err=%v; want 42 miss", v, hit, err)
	}
	clk.Advance(2999 * time.Millisecond)
	v, hit, err = c.Get(context.Background(), "k", load)
	if err != nil || v != 42 || !hit {
		t.Fatalf("second Get = %d hit=%v err=%v; want 42 hit", v, hit, err)
	}
	if calls.Load() != 1 {
		t.Errorf("loader called %d times, want 1", calls.Load())
	}
}

func TestGet_MissAfterTTLReloads(t *testing.T) {
	c, clk := newTestCache(3 * time.Second)
	n := 0
	load := func(context.Context) (int, error) { n++; return n, nil }

	_, _, _ = c.Get(context.Background(), "k", load)
	clk.Advance(3 * time.Second)
	v, hit, _ := c.Get(context.Background(), "k", load)
	if hit || v != 2 {
		t.Errorf("after TTL: v=%d hit=%v, want 2 miss", v, hit)
	}
}

func TestGet_ErrorsAreNotCached(t *testing.T) {
	c, _ := newTestCache(time.Minute)
	var calls atomic.Int32
	boom := errors.New("boom")
	failing := func(context.Context) (int, error) { calls.Add(1); return 0, boom }

	if _, _, err := c.Get(context.Background(), "k", failing); !errors.Is(err, boom) {
		t.Fatalf("err = %v, want boom", err)
	}
	if _, _, err := c.Get(context.Background(), "k", failing); !errors.Is(err, boom) {
		t.Fatalf("err = %v, want boom again (not cached)", err)
	}
	if calls.Load() != 2 {
		t.Errorf("loader called %d times, want 2", calls.Load())
	}
}

func TestGet_KeysAreIsolated(t *testing.T) {
	c, _ := newTestCache(time.Minute)
	a, _, _ := c.Get(context.Background(), "a", func(context.Context) (int, error) { return 1, nil })
	b, _, _ := c.Get(context.Background(), "b", func(context.Context) (int, error) { return 2, nil })
	if a != 1 || b != 2 {
		t.Errorf("a=%d b=%d", a, b)
	}
}

func TestGet_ConcurrentMissesCoalesceIntoOneLoad(t *testing.T) {
	c, _ := newTestCache(time.Minute)
	var calls atomic.Int32
	started := make(chan struct{})
	release := make(chan struct{})
	load := func(context.Context) (int, error) {
		if calls.Add(1) == 1 {
			close(started)
		}
		<-release
		return 7, nil
	}

	const n = 20
	var wg sync.WaitGroup
	results := make([]int, n)
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i], _, _ = c.Get(context.Background(), "k", load)
		}()
	}
	<-started
	time.Sleep(20 * time.Millisecond) // let the other goroutines queue behind the first
	close(release)
	wg.Wait()

	if calls.Load() != 1 {
		t.Errorf("loader called %d times under concurrency, want 1", calls.Load())
	}
	for i, r := range results {
		if r != 7 {
			t.Errorf("results[%d] = %d, want 7", i, r)
		}
	}
}

func TestStats_CountHitsAndMisses(t *testing.T) {
	c, _ := newTestCache(time.Minute)
	load := func(context.Context) (int, error) { return 1, nil }
	_, _, _ = c.Get(context.Background(), "k", load)
	_, _, _ = c.Get(context.Background(), "k", load)
	_, _, _ = c.Get(context.Background(), "k", load)
	hits, misses := c.Stats()
	if hits != 2 || misses != 1 {
		t.Errorf("hits=%d misses=%d, want 2/1", hits, misses)
	}
}

func TestExpiredEntriesAreSweptOnStore(t *testing.T) {
	c, clk := newTestCache(time.Second)
	load := func(context.Context) (int, error) { return 1, nil }
	for i := range sweepThreshold + 10 {
		_, _, _ = c.Get(context.Background(), "k"+string(rune('a'+i%26))+string(rune('a'+i/26)), load)
	}
	clk.Advance(2 * time.Second) // everything above is now expired
	_, _, _ = c.Get(context.Background(), "fresh", load)

	if n := c.Len(); n != 1 {
		t.Errorf("Len() = %d after sweep, want 1 (only the fresh key)", n)
	}
}
