# hedged

A helper for hedged requests.

Concept
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
