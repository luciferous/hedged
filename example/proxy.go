package main

import (
	"context"
	"fmt"
	"github.com/luciferous/hedged"
	"io"
	"net/http"
	"time"
)

type req struct{}

func (r req) Req(ctx context.Context) (interface{}, error) {
	req, err := http.NewRequest("GET", "http://localhost:8000", nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	return http.DefaultClient.Do(req)
}

func hedgedApp(w http.ResponseWriter, r *http.Request) {
	switch v := hedged.Run(r.Context(), 100*time.Millisecond, req{}).(type) {
	case error:
		http.Error(w, v.Error(), 503)
	case *http.Response:
		r.Body.Close()
		w.WriteHeader(v.StatusCode)
		io.Copy(w, v.Body)
	}
}

func main() {
	fmt.Println("Listening on :8001")
	http.ListenAndServe(":8001", http.HandlerFunc(hedgedApp))
}
