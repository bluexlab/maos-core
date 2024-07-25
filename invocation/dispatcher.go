package invocation

import (
	"fmt"
	"time"

	"github.com/puzpuzpuz/xsync/v3"
)

/*
Dispatcher is a concurrent-safe, ID-based channel manager that creates, manages,
and dispatches payloads across multiple channels identified by unique string IDs.

Key features:
1. ID-based channel management: Manage multiple channels, each with a unique string identifier.
2. Multi-listener support: Allow multiple goroutines to listen on the same ID-based channel.
3. Concurrent-safe operations: Ensure thread-safety across goroutines.
4. Non-blocking dispatch: Send payloads without blocking the dispatcher.
5. Configurable timeout: Set custom waiting timeouts for payload reception.

Main methods:
- NewDispatcher(): Create and initialize a new Dispatcher instance.
- Listen(id string): Ensure a channel exists for the given ID and return it.
- WaitFor(id string, timeout time.Duration): Wait for a payload on an ID's channel with a specified timeout.
- Dispatch(id string, payload interface{}): Send a payload to the channel associated with the given ID.
- Close(): Gracefully shut down the Dispatcher, closing all channels and performing cleanup.

Implementation details:
- Each listening channel has a buffer size of 32 to optimize performance and prevent blocking.
- The Dispatcher uses a xsync.MapOf for efficient concurrent access to channels.
*/
type Dispatcher[T any] struct {
	channels *xsync.MapOf[string, chan *T]
	closed   chan struct{}
}

func NewDispatcher[T any]() *Dispatcher[T] {
	return &Dispatcher[T]{
		channels: xsync.NewMapOf[string, chan *T](),
		closed:   make(chan struct{}),
	}
}

func (d *Dispatcher[T]) IsClosed() bool {
	select {
	case <-d.closed:
		return true
	default:
		return false
	}
}

// For testing purposes
func (d *Dispatcher[T]) Size() int {
	return d.channels.Size()
}

// Listen ensures a channel exists for the given ID.
//
// If a channel for the ID already exists, this function is a no-op.
// Returns an error if the dispatcher is closed.
func (d *Dispatcher[T]) Listen(id string) error {
	_, err := d.getOrCreateCh(id)
	return err
}

// WaitFor waits for a payload on the channel associated with the given ID.
// It returns the payload if received within the timeout, or nil if the timeout
// is reached. Returns an error if the dispatcher is closed. If no channel exists
// for the ID, one is created before waiting.
//
// Parameters:
//   - id: The channel identifier
//   - timeout: Maximum wait duration
//
// Returns (*T, error): The payload (or nil) and any error encountered.
func (d *Dispatcher[T]) WaitFor(id string, timeout time.Duration) (*T, error) {
	ch, err := d.getOrCreateCh(id)
	if err != nil {
		return nil, err
	}

	select {
	case payload, ok := <-ch:
		if !ok {
			return nil, fmt.Errorf("dispatcher is closed")
		}
		return payload, nil
	case <-time.After(timeout):
		return nil, nil
	case <-d.closed:
		return nil, fmt.Errorf("dispatcher is closed")
	}
	// TODO: keep tracking of waiting goroutines and clean up the channel if no one is waiting
}

// Dispatch sends a payload to the channel associated with the given ID.
// If the channel doesn't exist, the payload is discarded without error.
// Returns an error if the dispatcher is closed.
//
// Parameters:
//   - id: The channel identifier
//   - payload: The data to send
//
// Returns an error if the dispatcher is closed, nil otherwise.
func (d *Dispatcher[T]) Dispatch(id string, payload *T) error {
	ch, err := d.getCh(id)
	if err != nil {
		return err
	}

	if ch != nil {
		select {
		case ch <- payload:
		case <-d.closed:
			return fmt.Errorf("dispatcher is closed")
		default:
		}
	}

	return nil
}

func (d *Dispatcher[T]) getCh(id string) (chan *T, error) {
	select {
	case <-d.closed:
		return nil, fmt.Errorf("dispatcher is closed")
	default:
	}

	ch, _ := d.channels.Load(id)
	return ch, nil
}

func (d *Dispatcher[T]) getOrCreateCh(id string) (chan *T, error) {
	select {
	case <-d.closed:
		return nil, fmt.Errorf("dispatcher is closed")
	default:
	}

	ch, _ := d.channels.LoadOrCompute(id, func() chan *T {
		return make(chan *T, 32)
	})

	return ch, nil
}

// Close shuts down the Dispatcher and cleans up any remaining resources
func (d *Dispatcher[T]) Close() {
	select {
	case <-d.closed:
		return
	default:
		close(d.closed)
	}

	d.channels.Range(func(id string, ch chan *T) bool {
		close(ch)
		d.channels.Delete(id)
		return true
	})
}
