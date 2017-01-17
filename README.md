# hedged

A helper for hedged requests.

Concept
-------

Package hedged manages hedged requests - sending the same request to multiple replicas and using the result from the first to respond. Refer to [The Tail at Scale][1] for detail.

To illustrate: imagine you have a set of identical servers responding to various requests. Most of the time the servers respond quickly, but sometimes a response can be up to 100x slower than average. Multiple factors (e.g. queues, garbage collection) can account for the variability in the response time of each server.

Hedged requests are a strategy to curb this latency variability: issue the same request twice and use the first response. The method employed here issues the second request only after passing a duration threshold supplied as a parameter.

The idea is that if a server can respond fast enough, we can avoid sending a second request, duplicating work for little gain. Issuing a hedge request for only the slowest 5%, ensures the latency reduction is impactful, costing only a 5% increase in duplicated work.

Here's an example with sending a GET request to example.com.

```go
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
```

[1]: http://cacm.acm.org/magazines/2013/2/160173-the-tail-at-scale/fulltext

Usage
-------

Start the slow server.

```
$ go run example/slow.go
Listening on :8000
```

Start the proxy.
```
$ go run example/proxy.go
Listening on :8001
```

Run a load test against the slow server directly.
```
$ echo "GET http://localhost:8000" | vegeta attack -rate 1000 -duration 10s| vegeta report
Latencies     [mean, 50, 95, 99, max]  19.502382ms, 6.35981ms, 11.391445ms, 800.422033ms, 1.001342266s
```

Run a load test against the proxy.
```
$ echo "GET http://localhost:8001" | vegeta attack -rate 1000 -duration 10s| vegeta report
Latencies     [mean, 50, 95, 99, max]  12.531694ms, 7.384947ms, 80.573057ms, 106.057402ms, 181.430068ms
```
