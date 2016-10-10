package hedged

import (
	"context"
	"sync"
	"time"
)

// Request is any thing that can be Run with a Context, returning a result or error.
type Request interface {
	Run(context.Context) (interface{}, error)
}

// Run runs the hedged request.
func Run(ctx context.Context, r Request, wait time.Duration) interface{} {
	return RunN(ctx, r, wait, 1)
}

// RunN runs the hedged request with up to n hedges.
func RunN(ctx context.Context, r Request, wait time.Duration, n int) interface{} {
	var wg sync.WaitGroup
	var v interface{}

	newCtx, done := context.WithCancel(ctx)
	ch := make(chan interface{}, n)
	sent := 0

	for {
		if sent <= n {
			sent++
			// The scheduler may not run goroutines in the order which they are
			// defined below. We increment the wait group outside the goroutine to
			// guarantee it happens before the call to wg.Wait further below.
			wg.Add(1)
			go func() {
				res, err := r.Run(newCtx)
				if err != nil {
					ch <- err
				} else {
					ch <- res
				}
				wg.Done()
			}()
		}

		// Select from whatever is ready:
		// 1. One of the requests has finished processing;
		// 2. Caller cancelled the context;
		// 3. Time to issue a hedged request.
		select {
		case v = <-ch:
			goto Done
		case <-ctx.Done():
			v = ctx.Err()
			goto Done
		case <-time.After(wait):
			continue
		}
	}

Done:
	// Cancel the slower requests and wait for threads to acknowledge
	// cancellation before closing the channel.
	done()
	go func() { wg.Wait(); close(ch) }()

	return v
}
