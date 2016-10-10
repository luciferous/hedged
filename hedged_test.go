package hedged

import (
	"context"
	"testing"
	"time"
)

type str struct {
	val string
}

func (s *str) Run(ctx context.Context) (interface{}, error) {
	return s.val, nil
}

func BenchmarkRun(b *testing.B) {
	ctx := context.TODO()
	s := &str{"howdy"}
	for i := 0; i < b.N; i++ {
		switch v := Run(ctx, s, 1*time.Second).(type) {
		case string:
			if v != "howdy" {
				b.Errorf("Expected howdy, got %s", v)
			}
		}
	}
}

type slowOdds struct {
	i    int
	wait time.Duration
}

func (s *slowOdds) Run(ctx context.Context) (interface{}, error) {
	s.i++
	if Odd(s.i) {
		time.Sleep(s.wait)
	}
	return s.i, nil
}

func Odd(i int) bool {
	return i%2 != 0
}

func BenchmarkHedge(b *testing.B) {
	ctx := context.TODO()
	s := &slowOdds{0, 1 * time.Second}
	d := s.wait / 10
	for i := 0; i < b.N; i++ {
		switch v := Run(ctx, s, d).(type) {
		case int:
			if Odd(v) {
				b.Errorf("Expected even number, got %d", v)
			}
		}
	}
}

type hungOdds struct {
	i    int
	done chan<- struct{}
}

func (h *hungOdds) Run(ctx context.Context) (interface{}, error) {
	h.i++
	if !Odd(h.i) {
		return h.i, nil
	}

	select {
	case <-ctx.Done():
		close(h.done)
		return nil, ctx.Err()
	}
}

func TestCancel(t *testing.T) {
	ctx := context.TODO()
	done := make(chan struct{})
	h := &hungOdds{0, done}
	switch v := Run(ctx, h, 10*time.Millisecond).(type) {
	case int:
		if Odd(v) {
			t.Errorf("Expected even number, got %d", v)
		}
	}
	select {
	case <-done:
		break
	case <-time.After(1 * time.Millisecond):
		t.Error("Hung request not cancelled")
	}
}

const ctxKey = 321

type c struct{}

func (p c) Run(ctx context.Context) (interface{}, error) {
	return ctx.Value(ctxKey), nil
}

func TestContext(t *testing.T) {
	ctx := context.WithValue(context.TODO(), ctxKey, "howdy")
	switch v := Run(ctx, c{}, 10*time.Second).(type) {
	case string:
		if v != "howdy" {
			t.Errorf("Expected howdy, got %s", v)
		}
	}
}
