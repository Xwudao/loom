package loom_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/Xwudao/loom"
)

func TestLifecycleStartStopOrder(t *testing.T) {
	var mu sync.Mutex
	var order []string
	record := func(s string) func(context.Context) error {
		return func(context.Context) error {
			mu.Lock()
			defer mu.Unlock()
			order = append(order, s)
			return nil
		}
	}

	lc := loom.NewLifecycle()
	lc.Append(loom.Hook{OnStart: record("start-a"), OnStop: record("stop-a")})
	lc.AddCleanup(record("cleanup-b"))
	lc.Append(loom.Hook{OnStart: record("start-c"), OnStop: record("stop-c")})
	lc.AddCleanup(record("cleanup-d"))

	if err := lc.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := lc.Stop(context.Background()); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	want := []string{"start-a", "start-c", "cleanup-d", "stop-c", "cleanup-b", "stop-a"}
	if strings.Join(order, ",") != strings.Join(want, ",") {
		t.Fatalf("order = %v, want %v", order, want)
	}
}

func TestLifecycleStartFailureRollsBack(t *testing.T) {
	boom := errors.New("boom")
	var mu sync.Mutex
	var events []string
	record := func(s string) func(context.Context) error {
		return func(context.Context) error {
			mu.Lock()
			defer mu.Unlock()
			events = append(events, s)
			return nil
		}
	}

	lc := loom.NewLifecycle()
	lc.AddCleanup(record("cleanup-a"))
	lc.Append(loom.Hook{OnStart: record("start-b"), OnStop: record("stop-b")})
	lc.Append(loom.Hook{OnStart: func(context.Context) error { return boom }, OnStop: record("stop-c")})
	lc.Append(loom.Hook{OnStart: record("start-d"), OnStop: record("stop-d")})

	err := lc.Start(context.Background())
	if !errors.Is(err, boom) {
		t.Fatalf("Start error = %v, want %v", err, boom)
	}
	want := []string{"start-b", "stop-b", "cleanup-a"}
	if strings.Join(events, ",") != strings.Join(want, ",") {
		t.Fatalf("events = %v, want %v", events, want)
	}
	if got := lc.State(); got != "failed" {
		t.Fatalf("state = %q, want failed", got)
	}
	if err := lc.Start(context.Background()); !errors.Is(err, loom.ErrStartFailed) {
		t.Fatalf("second Start = %v, want ErrStartFailed", err)
	}
	// A failed lifecycle can still be stopped (idempotently) and must not
	// release resources twice.
	if err := lc.Stop(context.Background()); err != nil {
		t.Fatalf("Stop after failure: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("cleanups ran twice: %v", events)
	}
}

func TestLifecycleRollbackOnConstructionFailure(t *testing.T) {
	var mu sync.Mutex
	var cleaned []string
	mk := func(s string) loom.Cleanup {
		return func(context.Context) error {
			mu.Lock()
			defer mu.Unlock()
			cleaned = append(cleaned, s)
			return nil
		}
	}

	lc := loom.NewLifecycle()
	lc.AddCleanup(mk("db"))
	lc.AddCleanup(mk("redis"))

	if err := lc.Rollback(context.Background()); err != nil {
		t.Fatalf("Rollback: %v", err)
	}
	want := []string{"redis", "db"}
	if strings.Join(cleaned, ",") != strings.Join(want, ",") {
		t.Fatalf("rollback order = %v, want %v", cleaned, want)
	}
	// Rollback must not run hooks and must not double-run cleanups.
	if err := lc.Stop(context.Background()); err != nil {
		t.Fatalf("Stop after Rollback: %v", err)
	}
	if len(cleaned) != 2 {
		t.Fatalf("cleanups ran twice: %v", cleaned)
	}
}

func TestLifecycleStopAggregatesErrors(t *testing.T) {
	errA := errors.New("a")
	errB := errors.New("b")
	lc := loom.NewLifecycle()
	lc.Append(loom.Hook{OnStart: func(context.Context) error { return nil }, OnStop: func(context.Context) error { return errA }})
	lc.Append(loom.Hook{OnStart: func(context.Context) error { return nil }, OnStop: func(context.Context) error { return errB }})

	if err := lc.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	err := lc.Stop(context.Background())
	if !errors.Is(err, errA) || !errors.Is(err, errB) {
		t.Fatalf("Stop error = %v, want both a and b", err)
	}
}

func TestLifecycleStopWithoutStartRunsCleanupsOnly(t *testing.T) {
	var cleanups, stops int
	lc := loom.NewLifecycle()
	lc.AddCleanup(func(context.Context) error { cleanups++; return nil })
	lc.Append(loom.Hook{
		OnStart: func(context.Context) error { return nil },
		OnStop:  func(context.Context) error { stops++; return nil },
	})

	if err := lc.Stop(context.Background()); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if cleanups != 1 {
		t.Fatalf("cleanups = %d, want 1", cleanups)
	}
	if stops != 0 {
		t.Fatalf("stops = %d, want 0 (hook never started)", stops)
	}
}

func TestLifecycleRepeatedStopIsIdempotent(t *testing.T) {
	var cleanups int
	lc := loom.NewLifecycle()
	lc.AddCleanup(func(context.Context) error { cleanups++; return nil })
	lc.Append(loom.Hook{OnStart: func(context.Context) error { return nil }})

	if err := lc.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	for i := 0; i < 3; i++ {
		if err := lc.Stop(context.Background()); err != nil {
			t.Fatalf("Stop #%d: %v", i, err)
		}
	}
	if cleanups != 1 {
		t.Fatalf("cleanups = %d, want 1", cleanups)
	}
}

func TestLifecycleRepeatedStartFails(t *testing.T) {
	lc := loom.NewLifecycle()
	lc.Append(loom.Hook{OnStart: func(context.Context) error { return nil }})
	if err := lc.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := lc.Start(context.Background()); !errors.Is(err, loom.ErrAlreadyStarted) {
		t.Fatalf("second Start = %v, want ErrAlreadyStarted", err)
	}
}

func TestLifecycleContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var stops []string
	lc := loom.NewLifecycle()
	lc.Append(loom.Hook{
		OnStart: func(context.Context) error { cancel(); return nil },
		OnStop:  func(context.Context) error { stops = append(stops, "a"); return nil },
	})
	lc.Append(loom.Hook{
		OnStart: func(context.Context) error { return nil },
		OnStop:  func(context.Context) error { stops = append(stops, "b"); return nil },
	})

	if err := lc.Start(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Start = %v, want context.Canceled", err)
	}
	if len(stops) != 1 || stops[0] != "a" {
		t.Fatalf("stops = %v, want [a]", stops)
	}
}

func TestLifecycleCanceledStopStillRunsCleanups(t *testing.T) {
	ctx, cancel := context.WithCancel(context.WithValue(context.Background(), "key", "value"))
	var cleanups, stops int
	lc := loom.NewLifecycle()
	lc.AddCleanup(func(ctx context.Context) error {
		cleanups++
		if ctx.Err() != nil || ctx.Value("key") != "value" {
			t.Errorf("cleanup context: err=%v, value=%v", ctx.Err(), ctx.Value("key"))
		}
		return nil
	})
	lc.Append(loom.Hook{OnStop: func(context.Context) error { stops++; return nil }})
	lc.AddCleanup(func(context.Context) error { cancel(); cleanups++; return nil })
	if err := lc.Start(ctx); err != nil {
		t.Fatal(err)
	}
	if err := lc.Stop(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Stop = %v, want context.Canceled", err)
	}
	if cleanups != 2 || stops != 0 {
		t.Fatalf("cleanups=%d stops=%d, want 2 and 0", cleanups, stops)
	}
	if err := lc.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if cleanups != 2 {
		t.Fatalf("cleanup ran twice: %d", cleanups)
	}
}

func TestLifecyclePanicBecomesError(t *testing.T) {
	lc := loom.NewLifecycle()
	lc.Append(loom.Hook{OnStart: func(context.Context) error { panic("kaboom") }})

	err := lc.Start(context.Background())
	if err == nil || !strings.Contains(err.Error(), "kaboom") {
		t.Fatalf("Start = %v, want panic error", err)
	}
	if got := lc.State(); got != "failed" {
		t.Fatalf("state = %q, want failed", got)
	}
}

func TestLifecycleAppendOnceStartBeginsPanics(t *testing.T) {
	lc := loom.NewLifecycle()
	entered := make(chan struct{})
	release := make(chan struct{})
	lc.Append(loom.Hook{OnStart: func(context.Context) error {
		close(entered)
		<-release
		return nil
	}})
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = lc.Start(context.Background())
	}()
	<-entered
	panicked := func() (panicked bool) {
		defer func() { panicked = recover() != nil }()
		lc.Append(loom.Hook{OnStart: func(context.Context) error { return nil }})
		return false
	}()
	close(release)
	<-done
	if !panicked {
		t.Fatal("Append after Start began did not panic")
	}
}

func TestLifecycleNilHookIsIgnored(t *testing.T) {
	lc := loom.NewLifecycle()
	lc.Append(loom.Hook{})
	lc.AddCleanup(nil)
	if err := lc.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := lc.Stop(context.Background()); err != nil {
		t.Fatalf("Stop: %v", err)
	}
}

func TestLifecycleConcurrentStopRunsCleanupsOnce(t *testing.T) {
	var cleanups int32
	lc := loom.NewLifecycle()
	lc.AddCleanup(func(context.Context) error {
		atomic.AddInt32(&cleanups, 1)
		return nil
	})
	lc.Append(loom.Hook{OnStart: func(context.Context) error { return nil }})
	if err := lc.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := lc.Stop(context.Background()); err != nil {
				t.Errorf("concurrent Stop: %v", err)
			}
		}()
	}
	wg.Wait()
	if got := atomic.LoadInt32(&cleanups); got != 1 {
		t.Fatalf("cleanups ran %d times, want 1", got)
	}
}
