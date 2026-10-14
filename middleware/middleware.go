package middleware

import (
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/harshit3011/Custom-HTTP-Server-Go/parser"
	"github.com/harshit3011/Custom-HTTP-Server-Go/response"
	"github.com/harshit3011/Custom-HTTP-Server-Go/router"
)

var requestSequence atomic.Uint64

// It logs the method, target, status code, and duration of each request.
func Logging(next router.Handler) router.Handler {
	return func(w *response.ResponseWriter, req *parser.Request) {
		start := time.Now()

		defer func() {
			log.Printf(
				"request_id=%s method=%s target=%s status=%d duration=%s",
				w.Header().Get("X-Request-ID"),
				req.Method,
				req.Target,
				w.StatusCode(),
				time.Since(start),
			)
		}()

		next(w, req)
	}
}

// RequestID assigns a unique ID to each request.
func RequestID(next router.Handler) router.Handler {
	return func(w *response.ResponseWriter, req *parser.Request) {
		id := fmt.Sprintf("req-%d", requestSequence.Add(1))

		_ = w.Header().Set("X-Request-ID", id)

		next(w, req)
	}
}

// Recover prevents handler panics from crashing the server.
func Recover(next router.Handler) router.Handler {
	return func(w *response.ResponseWriter, req *parser.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf("panic recovered: %v", recovered)

				w.Reset(
					500,
					"text/plain; charset=utf-8",
					"Internal Server Error\n",
				)
			}
		}()

		next(w, req)
	}
}

// Metrics tracks request counts, server errors, and cumulative duration.
type Metrics struct {
	requests      atomic.Uint64
	serverErrors  atomic.Uint64
	totalDuration atomic.Uint64
}

// Snapshot contains a consistent-enough, concurrency-safe view of the metrics.
// Individual counters are atomic, so values may reflect slightly different moments.
type Snapshot struct {
	Requests      uint64
	ServerErrors  uint64
	TotalDuration time.Duration
}

// Middleware records metrics for each request that reaches it.
func (m *Metrics) Middleware(next router.Handler) router.Handler {
	return func(w *response.ResponseWriter, req *parser.Request) {
		start := time.Now()

		defer func() {
			duration := time.Since(start)

			m.requests.Add(1)

			if w.StatusCode() >= 500 {
				m.serverErrors.Add(1)
			}

			m.totalDuration.Add(uint64(duration))
		}()

		next(w, req)
	}
}

// Snapshot returns the current metrics.
func (m *Metrics) Snapshot() Snapshot {
	return Snapshot{
		Requests:      m.requests.Load(),
		ServerErrors:  m.serverErrors.Load(),
		TotalDuration: time.Duration(m.totalDuration.Load()),
	}
}
