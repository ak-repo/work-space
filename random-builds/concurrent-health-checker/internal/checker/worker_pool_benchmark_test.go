package checker

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func BenchmarkCheckerWorkerPool(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Microsecond)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	check := New(server.Client())
	urls := make([]string, 50)
	for i := range urls {
		urls[i] = server.URL
	}

	for _, workers := range []int{1, 5, 10, 25, 50} {
		b.Run(fmt.Sprintf("workers-%d", workers), func(b *testing.B) {
			for range b.N {
				if _, err := RunPool(context.Background(), workers, urls, check.Check); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
