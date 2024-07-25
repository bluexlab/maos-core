package invocation

import (
	"fmt"
	"sync"
	"time"
)

/*
The Dispatcher is a flexible, ID-based channel manager that creates, manages,
and dispatches payloads to multiple channels, each identified by a unique string ID.

Key features:
1. ID-based channel management: Manage multiple channels, each with a unique string ID.
2. Multi-listener support: Multiple goroutines can listen on the same ID-based channel.
3. Dynamic channel creation: Channels are created on-demand when first accessed.
4. Concurrent-safe: Thread-safe operations across goroutines.
5. Non-blocking dispatch: Payload sending doesn't block the dispatcher.
6. Timeout support: Set waiting timeouts for payloads.
7. Graceful shutdown: Close all managed channels and clean up resources.

Main components:
- NewDispatcher(): Create a new Dispatcher instance.
- Listen(id string): Ensure a channel exists for the given ID.
- WaitFor(id string, timeout time.Duration): Wait for a payload on an ID's channel with timeout.
- Dispatch(id string, payload): Send a payload to an ID's channel.
- Close(): Shut down the Dispatcher, closing all channels and cleaning up.

The Dispatcher uses a done channel for lifecycle management, enabling
efficient closure detection and preventing operations when closed.

Ideal for scenarios requiring:
1. Coordinated communication between goroutines using specific identifiers.
2. Flexible, ID-based publish-subscribe patterns.
3. Management of multiple asynchronous operations with ID-specific results.
4. Dynamic, ID-based event dispatching systems.
*/
type Dispatcher[T any] struct {
	payloads map[string]chan *T
	mutex    sync.Mutex
	closed   chan struct{}
}

func NewDispatcher[T any]() *Dispatcher[T] {
	return &Dispatcher[T]{
		payloads: make(map[string]chan *T),
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
func (d *Dispatcher[T]) Payloads() map[string]chan *T {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	return d.payloads
}

// Listen ensures a channel exists for the given ID.
//
// If a channel for the ID already exists, this function is a no-op.
// Returns an error if the dispatcher is closed.
func (d *Dispatcher[T]) Listen(id string) error {
	_, err := d.getCh(id, true)
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
	ch, err := d.getCh(id, true)
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
	ch, err := d.getCh(id, false)
	if err != nil {
		return err
	}

	if ch != nil {
		select {
		case ch <- payload:
		case <-d.closed:
			return fmt.Errorf("dispatcher is closed")
		}
	}

	return nil
}

func (d *Dispatcher[T]) getCh(id string, createOnNotExist bool) (chan *T, error) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	select {
	case <-d.closed:
		return nil, fmt.Errorf("dispatcher is closed")
	default:
	}

	ch, exists := d.payloads[id]
	if !exists && createOnNotExist {
		ch = make(chan *T, 1)
		d.payloads[id] = ch
	}
	return ch, nil
}

// Close shuts down the Dispatcher and cleans up any remaining resources
func (d *Dispatcher[T]) Close() error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	select {
	case <-d.closed:
		return fmt.Errorf("dispatcher is already closed")
	default:
		close(d.closed)
	}

	for id, ch := range d.payloads {
		close(ch)
		delete(d.payloads, id)
	}

	return nil
}
