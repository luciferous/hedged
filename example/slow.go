package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"
)

func app(w http.ResponseWriter, r *http.Request) {
	x := rand.Float32()
	switch {
	case x >= 0.999:
		time.Sleep(1000 * time.Millisecond)
	case x >= 0.99:
		time.Sleep(800 * time.Millisecond)
	case x >= 0.95:
		time.Sleep(80 * time.Millisecond)
	case x >= 0.5:
		time.Sleep(10 * time.Millisecond)
	default:
		time.Sleep(5 * time.Millisecond)
	}
	r.Body.Close()
	w.WriteHeader(204)
}

func main() {
	fmt.Println("Listening on :8000")
	http.ListenAndServe(":8000", http.HandlerFunc(app))
}
