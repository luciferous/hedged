/*
Package hedged manages hedged requests (sending the same request to multiple
replicas and using the result from the first to respond). Refer to "The Tail at
Scale" [1] for detail.

To illustrate its usefulness, imagine you have a replica set that responds
within 10ms on average, but takes occassionally (p99) 500ms, and even less
frequently (p999) up to 1 second. Multiple factors (queues, GC, etc.) influence
the response time of each server in the replica set, and so, while identical in
operation, the same request may take longer depending on which server receives
it.

The hedged request is a strategy to curb this latency variability: issue the
same request twice and use the first response. This naive method doubles the
load sent to the servers. A better method is to wait between sending the
requests, giving time to the first to complete.

But how long do we wait for? A good starting point is the p95 latency. This
translates into only a %5 increase in load sent to the servers.

Here's an example with sending a GET request to example.com.

	v := hedged.Run(context.Background(), 100 * time.Millisecond, func(ctx context.Context) (interface{}, error) {
		req, err := http.NewRequest("GET", "http://example.com", nil)
		// if err != nil ...
		req = req.WithContext(ctx)
		return http.DefaultClient.Do(req)
	})

	switch v = v.(type) {
	case *http.Response:
		// Do something with the response.
	case error:
		// Uh-oh.
	}

[1]: http://cacm.acm.org/magazines/2013/2/160173-the-tail-at-scale/fulltext
*/
package hedged

import (
	"context"
	"sync"
	"time"
)

// Request defines a cancellable request.
//
// Req may be called multiple times such that several Req may be running in
// parallel. The first Req to finish cancels the others that are in flight.
// Req, therefore, should be idempotent.
//
// Via the supplied context, implementations may respond directly to
// cancellation from the caller,
//
// 	func Req(ctx context.Context) (interface{}, error) {
// 		// ...
// 		select {
// 		case <-ctx.Done():
// 			// Canceled: do something else, clean up, etc...
// 		}
// 	}
//
// or propagate it by passing the context forward, allowing subsequent
// computations to respond instead,
//
// 	func Req(ctx context.Context) (interface{}, error) {
// 		req, err := http.NewRequest("GET", "http://example.com", nil)
// 		// if err != nil ...
// 		req = req.WithContext(ctx)
// 		return http.DefaultClient.Do(req)
// 	}
type Request interface {
	Req(context.Context) (interface{}, error)
}

// TaskFunc is an adapter to allow the use of ordinary functions as Tasks.
type RequestFunc func(context.Context) (interface{}, error)

// Run calls f(ctx).
func (f RequestFunc) Req(ctx context.Context) (interface{}, error) {
	return f(ctx)
}

// Run sends the request.
//
// If the request doesn't complete within the wait time, another request is
// sent as a backup. Whichever request completes first cancels the other.
func Run(ctx context.Context, r Request, wait time.Duration) interface{} {
	return RunN(ctx, r, wait, 1)
}

// RunN is like Run but can send more than one hedge request.
//
// The wait duration is the interval at which requests get sent, until one
// completes, or there are n requests in flight. Whichever request completes
// first cancels the rest.
func RunN(ctx context.Context, r Request, wait time.Duration, n int) interface{} {
	var wg sync.WaitGroup
	var v interface{}

	newCtx, done := context.WithCancel(ctx)
	ch := make(chan interface{}, n)
	sent := 0

	for {
		if sent <= n {
			sent++
			// The scheduler may run goroutines out of the definition order. We
			// increment outside the goroutine to guarantee it happens here,
			// specifically, before the call to wg.Wait further below.
			wg.Add(1)
			go func() {
				res, err := r.Req(newCtx)
				if err != nil {
					ch <- err
				} else {
					ch <- res
				}
				// Calling Done implies that this thread has no further use for the
				// chan (i.e. won't write to it). When every thread signals this, then
				// parent thread may close it safely.
				wg.Done()
			}()
		}

		// Proceed with whichever one is ready first:
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
