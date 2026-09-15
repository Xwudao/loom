// Package loom is a compile-time dependency injection toolkit for Go.
//
// Dependency graphs are declared with [Graph] and ordinary constructor
// functions; the `loom` command reads those declarations and generates plain
// Go initialization code. Nothing about graph resolution happens at runtime:
// the generated code calls constructors directly.
//
// The only runtime primitive shipped by this package is [Lifecycle], an ordered
// list of start/stop hooks plus constructor cleanups. It exists so that
// generated applications can start and stop long-lived services in the right
// order. It is not a container and performs no dependency lookup.
package loom

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// Cleanup releases a resource acquired while a value was constructed.
//
// A provider may return a Cleanup alongside its value:
//
//	func NewDB(cfg *Config) (*DB, loom.Cleanup, error)
//
// Generated code registers each cleanup with the [Lifecycle] as soon as the
// constructor returns. Cleanups run in reverse registration order: during
// [Lifecycle.Rollback] when construction fails, or as part of
// [Lifecycle.Stop] on normal shutdown.
type Cleanup func(context.Context) error

// Hook couples an optional start callback with an optional stop callback.
//
// Hooks registered with [Lifecycle.Append] run in registration order on
// [Lifecycle.Start] and in reverse registration order on [Lifecycle.Stop].
type Hook struct {
	// OnStart runs when the lifecycle starts. A nil OnStart is a no-op.
	OnStart func(context.Context) error
	// OnStop runs when the lifecycle stops. A nil OnStop is a no-op.
	OnStop func(context.Context) error
}

// Sentinel errors returned by [Lifecycle.Start].
var (
	// ErrAlreadyStarted is returned by Start when the lifecycle is already
	// running.
	ErrAlreadyStarted = errors.New("loom: lifecycle is already started")
	// ErrAlreadyStopped is returned by Start when the lifecycle has been
	// stopped. Lifecycles are single-use and cannot be restarted.
	ErrAlreadyStopped = errors.New("loom: lifecycle is already stopped")
	// ErrStartFailed is returned by Start when a previous Start attempt failed.
	ErrStartFailed = errors.New("loom: lifecycle failed to start")
)

type lifecycleState uint8

const (
	stateInit lifecycleState = iota
	stateStarting
	stateStarted
	stateStopping
	stateStopped
	stateFailed
)

func (s lifecycleState) String() string {
	switch s {
	case stateInit:
		return "initialized"
	case stateStarting:
		return "starting"
	case stateStarted:
		return "started"
	case stateStopping:
		return "stopping"
	case stateStopped:
		return "stopped"
	case stateFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// entry is a single element of the ordered start/stop chain. It is either a
// user Hook or a constructor Cleanup, never both.
type entry struct {
	hook    Hook
	cleanup Cleanup
	// done records that a Cleanup has already run, so rollback followed by
	// stop cannot release the same resource twice.
	done bool
}

// Lifecycle is the runtime primitive used by generated code to start and stop
// an application in dependency order.
//
// The zero value is not ready for use; construct one with [NewLifecycle].
// Lifecycle is safe for concurrent use by multiple goroutines.
type Lifecycle struct {
	mu      sync.Mutex
	entries []entry
	state   lifecycleState
	// started is the number of leading entries whose OnStart callback
	// succeeded (cleanups count as already started).
	started int
}

// NewLifecycle returns an empty, initialized lifecycle.
func NewLifecycle() *Lifecycle { return &Lifecycle{} }

// Append registers a hook. Hooks are started in registration order and
// stopped in reverse registration order.
//
// Append must be called before the lifecycle is started. Calling it
// afterwards panics, because hooks registered after Start cannot be given
// consistent start/stop semantics.
func (l *Lifecycle) Append(h Hook) {
	if h.OnStart == nil && h.OnStop == nil {
		return
	}
	l.append(entry{hook: h}, "Append")
}

// AddCleanup registers a constructor cleanup. Cleanups run in reverse
// registration order, after any stop hooks that were started.
//
// AddCleanup must be called before the lifecycle is started. Calling it
// afterwards panics. A nil cleanup is ignored, which lets generated code
// register a constructor's cleanup unconditionally.
func (l *Lifecycle) AddCleanup(c Cleanup) {
	if c == nil {
		return
	}
	l.append(entry{cleanup: c}, "AddCleanup")
}

func (l *Lifecycle) append(e entry, what string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	switch l.state {
	case stateInit, stateStarting:
		l.entries = append(l.entries, e)
	default:
		panic("loom: Lifecycle." + what + " called after the lifecycle was started")
	}
}

// State reports the current lifecycle state. It is intended for diagnostics
// and tests.
func (l *Lifecycle) State() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.state.String()
}

// Start runs every registered OnStart hook in registration order.
//
// If a hook fails or ctx is canceled, Start rolls back: it runs Stop hooks and
// cleanups for everything started so far, in reverse registration order, and
// returns the start error joined with any rollback errors. A lifecycle whose
// start failed cannot be started again.
func (l *Lifecycle) Start(ctx context.Context) error {
	if ctx == nil {
		return errors.New("loom: Lifecycle.Start called with a nil context")
	}

	l.mu.Lock()
	switch l.state {
	case stateInit:
		l.state = stateStarting
	case stateStarting:
		l.mu.Unlock()
		return errors.New("loom: Lifecycle.Start called while the lifecycle is starting")
	case stateStarted:
		l.mu.Unlock()
		return ErrAlreadyStarted
	case stateStopping:
		l.mu.Unlock()
		return errors.New("loom: Lifecycle.Start called while the lifecycle is stopping")
	case stateStopped:
		l.mu.Unlock()
		return ErrAlreadyStopped
	case stateFailed:
		l.mu.Unlock()
		return ErrStartFailed
	}
	l.mu.Unlock()

	if err := l.start(ctx); err != nil {
		// Releasing resources must not be abandoned just because the failure
		// canceled ctx, so rollback deliberately ignores cancellation.
		rollbackErr := l.stop(context.WithoutCancel(ctx))
		l.mu.Lock()
		l.state = stateFailed
		l.mu.Unlock()
		return errors.Join(err, rollbackErr)
	}

	l.mu.Lock()
	l.state = stateStarted
	l.mu.Unlock()
	return nil
}

func (l *Lifecycle) start(ctx context.Context) error {
	for i := 0; ; i++ {
		l.mu.Lock()
		if i >= len(l.entries) {
			l.mu.Unlock()
			return nil
		}
		e := l.entries[i]
		l.started = i + 1
		l.mu.Unlock()

		if err := ctx.Err(); err != nil {
			l.rewindStarted(i)
			return err
		}
		if e.cleanup != nil || e.hook.OnStart == nil {
			continue
		}
		if err := callHook("OnStart", ctx, e.hook.OnStart); err != nil {
			l.rewindStarted(i)
			return err
		}
	}
}

func (l *Lifecycle) rewindStarted(n int) {
	l.mu.Lock()
	l.started = n
	l.mu.Unlock()
}

// Stop runs registered OnStop hooks and cleanups in reverse registration
// order. Hooks are stopped only if their OnStart ran successfully; cleanups
// always run.
//
// Errors from individual hooks are aggregated with errors.Join so that one
// failing hook does not prevent the rest of the shutdown. If ctx is canceled
// mid-shutdown, Stop records ctx.Err() and stops running further entries.
//
// Stop is idempotent: calling it again after the lifecycle has stopped returns
// nil.
func (l *Lifecycle) Stop(ctx context.Context) error {
	if ctx == nil {
		return errors.New("loom: Lifecycle.Stop called with a nil context")
	}

	l.mu.Lock()
	switch l.state {
	case stateInit, stateStarted:
		l.state = stateStopping
	case stateStopped, stateFailed, stateStopping:
		// Repeated or concurrent stops are no-ops.
		l.mu.Unlock()
		return nil
	default: // stateStarting
		l.mu.Unlock()
		return errors.New("loom: Lifecycle.Stop called while the lifecycle is starting")
	}
	l.mu.Unlock()

	err := l.stop(ctx)

	l.mu.Lock()
	l.state = stateStopped
	l.mu.Unlock()
	return err
}

// Rollback releases resources acquired during a failed construction. It runs
// only registered cleanups, in reverse registration order, and never runs
// OnStop hooks (nothing was started). Generated code calls Rollback when a
// constructor returns an error.
//
// Rollback ignores context cancellation so that a failed shutdown does not
// abandon still-open resources. It is a no-op unless the lifecycle is still
// initialized, so a later Stop cannot release the same resources twice.
func (l *Lifecycle) Rollback(ctx context.Context) error {
	if ctx == nil {
		return errors.New("loom: Lifecycle.Rollback called with a nil context")
	}

	l.mu.Lock()
	switch l.state {
	case stateInit:
		l.state = stateFailed
	default:
		l.mu.Unlock()
		return nil
	}
	l.mu.Unlock()

	return l.stop(context.WithoutCancel(ctx))
}

// stop walks the entry chain backwards. It must not be called while another
// stop is in progress.
func (l *Lifecycle) stop(ctx context.Context) error {
	l.mu.Lock()
	n := len(l.entries)
	l.mu.Unlock()

	var errs []error
	for i := n - 1; i >= 0; i-- {
		if err := ctx.Err(); err != nil {
			errs = append(errs, err)
			break
		}

		l.mu.Lock()
		e := l.entries[i]
		started := i < l.started
		if e.cleanup != nil {
			if e.done {
				l.mu.Unlock()
				continue
			}
			l.entries[i].done = true
			l.mu.Unlock()
			if err := callHook("Cleanup", ctx, e.cleanup); err != nil {
				errs = append(errs, err)
			}
			continue
		}
		l.mu.Unlock()
		if started && e.hook.OnStop != nil {
			if err := callHook("OnStop", ctx, e.hook.OnStop); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errors.Join(errs...)
}

// callHook invokes fn and converts a panic into an error so that a single
// misbehaving hook cannot abort the rest of the shutdown sequence.
func callHook(kind string, ctx context.Context, fn func(context.Context) error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("loom: panic in %s: %v", kind, r)
		}
	}()
	return fn(ctx)
}
