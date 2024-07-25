package invocation_test

import (
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gitlab.com/navyx/ai/maos/maos-core/invocation"
)

func TestDispatcher(t *testing.T) {
	t.Parallel()

	t.Run("NewDispatcher", func(t *testing.T) {
		d := invocation.NewDispatcher[int]()
		require.NotNil(t, d, "NewDispatcher should not return nil")
		require.False(t, d.IsClosed(), "New dispatcher should not be closed")
		require.Empty(t, d.Payloads(), "New dispatcher should have empty payloads map")
	})

	t.Run("WaitFor", func(t *testing.T) {
		d := invocation.NewDispatcher[string]()

		t.Run("Successful wait", func(t *testing.T) {
			done := make(chan struct{})
			d.Listen("test")
			go func() {
				d.Dispatch("test", ptr("payload"))
				close(done)
			}()

			result, err := d.WaitFor("test", 100*time.Millisecond)
			<-done
			require.NoError(t, err)
			require.Equal(t, "payload", *result)
		})

		t.Run("Timeout", func(t *testing.T) {
			result, err := d.WaitFor("timeout", 10*time.Millisecond)
			require.NoError(t, err)
			require.Nil(t, result)
		})

		t.Run("Closed dispatcher", func(t *testing.T) {
			require.NoError(t, d.Close())
			_, err := d.WaitFor("closed", 10*time.Millisecond)
			require.EqualError(t, err, "dispatcher is closed")
		})
	})

	t.Run("Dispatch", func(t *testing.T) {
		d := invocation.NewDispatcher[int]()

		t.Run("Successful dispatch", func(t *testing.T) {
			done := make(chan struct{})
			d.Listen("test")
			go func() {
				d.Dispatch("test", ptr(42))
				close(done)
			}()

			result, err := d.WaitFor("test", 100*time.Millisecond)
			<-done
			require.NoError(t, err)
			require.Equal(t, 42, *result)
		})

		t.Run("Dispatch to non-existent channel", func(t *testing.T) {
			require.NotPanics(t, func() {
				d.Dispatch("non-existent", ptr(10))
			})
		})

		t.Run("Dispatch to closed dispatcher", func(t *testing.T) {
			require.NoError(t, d.Close())
			require.NotPanics(t, func() {
				d.Dispatch("closed", ptr(5))
			})
		})

		t.Run("Dispatch multiple payloads to same ID", func(t *testing.T) {
			d := invocation.NewDispatcher[int]()
			id := "test_id"
			payloads := []int{1, 2, 3}

			d.Listen(id)
			go func() {
				for _, payload := range payloads {
					d.Dispatch(id, &payload)
				}
			}()

			// Only the first dispatched payload should be received
			result, err := d.WaitFor(id, 100*time.Millisecond)
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Equal(t, payloads[0], *result)

			// Subsequent waits should timeout
			for i := 1; i < len(payloads); i++ {
				result, err = d.WaitFor(id, 10*time.Millisecond)
				require.NoError(t, err)
				require.NotNil(t, result)
			}

			require.NoError(t, d.Close())
		})

	})

	t.Run("Close", func(t *testing.T) {
		d := invocation.NewDispatcher[bool]()

		t.Run("Successful close", func(t *testing.T) {
			err := d.Close()
			require.NoError(t, err)
			require.True(t, d.IsClosed(), "Dispatcher should be marked as closed")
			require.Empty(t, d.Payloads(), "Payloads map should be empty after closing")
		})

		t.Run("Double close", func(t *testing.T) {
			err := d.Close()
			require.EqualError(t, err, "dispatcher is already closed")
		})
	})

	t.Run("Concurrent operations", func(t *testing.T) {
		d := invocation.NewDispatcher[int]()
		const goroutines = 100
		var wg sync.WaitGroup
		wg.Add(goroutines * 2)

		for i := 0; i < goroutines; i++ {
			d.Listen(strconv.Itoa(i))
			go func(id int) {
				defer wg.Done()
				d.Dispatch(strconv.Itoa(id), ptr(id))
			}(i)
		}

		for i := 0; i < goroutines; i++ {
			go func(id int) {
				defer wg.Done()
				result, err := d.WaitFor(strconv.Itoa(id), 100*time.Millisecond)
				require.NoError(t, err)
				if result != nil {
					require.Equal(t, id, *result)
				}
			}(i)
		}

		wg.Wait()
		require.NoError(t, d.Close())
	})

	t.Run("Dispatch to multiple IDs", func(t *testing.T) {
		t.Run("Concurrent dispatch and wait", func(t *testing.T) {
			d := invocation.NewDispatcher[int]()
			const numIDs = 100
			var wg sync.WaitGroup
			wg.Add(numIDs * 2) // For both dispatch and wait operations

			for i := 0; i < numIDs; i++ {
				d.Listen(strconv.Itoa(i))
				go func(id int) {
					defer wg.Done()
					d.Dispatch(strconv.Itoa(id), &id)
				}(i)

				go func(id int) {
					defer wg.Done()
					result, err := d.WaitFor(strconv.Itoa(id), 100*time.Millisecond)
					require.NoError(t, err)
					require.NotNil(t, result)
					require.Equal(t, id, *result)
				}(i)
			}

			wg.Wait()
			require.NoError(t, d.Close())
		})

		t.Run("Dispatch and wait with different goroutines", func(t *testing.T) {
			d := invocation.NewDispatcher[string]()
			const numIDs = 50
			var wg sync.WaitGroup
			wg.Add(numIDs * 2)

			// Wait goroutines
			for i := 0; i < numIDs; i++ {
				go func(id int) {
					defer wg.Done()
					idStr := strconv.Itoa(id)
					result, err := d.WaitFor(idStr, 200*time.Millisecond)
					require.NoError(t, err)
					require.NotNil(t, result)
					require.Equal(t, idStr, *result)
				}(i)
			}

			// Dispatch goroutines
			for i := 0; i < numIDs; i++ {
				go func(id int) {
					defer wg.Done()
					time.Sleep(time.Duration(id+1) * time.Millisecond) // Simulate varying dispatch times
					idStr := strconv.Itoa(id)
					d.Dispatch(idStr, &idStr)
				}(i)
			}

			wg.Wait()
			require.NoError(t, d.Close())
		})
	})
}

func ptr[T any](v T) *T {
	return &v
}
