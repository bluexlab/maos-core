package testhelper

import "sync"

// DiscardContinuously drains continuously out of the given channel and discards
// anything that comes out of it. Returns a stop function that should be invoked
// to stop draining. Stop must be invoked before tests finish to stop an
// internal goroutine.
func DiscardContinuously[T any](drainChan <-chan T) func() {
	var (
		stop     = make(chan struct{})
		stopped  = make(chan struct{})
		stopOnce sync.Once
	)

	go func() {
		defer close(stopped)

		for {
			select {
			case <-drainChan:
			case <-stop:
				return
			}
		}
	}()

	return func() {
		stopOnce.Do(func() {
			close(stop)
			<-stopped
		})
	}
}
