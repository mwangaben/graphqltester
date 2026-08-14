package middleware

import (
	"bufio"
	"context"
	"fmt"
	"github.com/mwangaben/graphqltester/types"
	"net"
	"net/http"
	"strings"
	"time"
)

/**
 * Context Propagation Middleware
 *
 * This middleware propagates test context values into HTTP request contexts.
 * It ensures downstream handlers and resolvers have access to:
 * - Request ID for tracing
 * - Request start time for performance tracking
 * - Authenticated user information
 * - Test-specific values
 */

// contextKey is a private type for context keys to prevent collisions.
type contextKey string

// ContextPropagationMiddleware propagates test context into HTTP requests.
//
// Parameters:
//
//	tester - The tester instance providing test state
//
// Returns:
//
//	Middleware function
func ContextPropagationMiddleware(tester interface{}) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip WebSocket requests
			if strings.ToLower(r.Header.Get("Upgrade")) == "websocket" {
				next.ServeHTTP(w, r)
				return
			}

			ctx := r.Context()

			// Add request ID if not present
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = generateRequestID()
			}
			ctx = context.WithValue(ctx, types.RequestIDKey, requestID)
			w.Header().Set("X-Request-ID", requestID)

			// Add request start time
			ctx = context.WithValue(ctx, types.RequestStartTimeKey, time.Now())

			// Propagate tester context values
			if testerWithCtx, ok := tester.(interface{ Context() context.Context }); ok {
				testerCtx := testerWithCtx.Context()
				if testerCtx != nil {
					for _, key := range []types.ContextKey{
						types.UserKey,
						types.UserIDKey,
						types.TenantIDKey,
						types.TestNameKey,
					} {
						if val := testerCtx.Value(key); val != nil {
							ctx = context.WithValue(ctx, key, val)
						}
					}
				}
			}

			// ✅ IMPORTANT: Pass the modified context via r.WithContext()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ResponseContextMiddleware captures response information.
//
// Parameters:
//
//	tester - The tester instance
//
// Returns:
//
//	Middleware function
func ResponseContextMiddleware(tester interface{}) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip wrapping for WebSocket requests (they need http.Hijacker)
			if strings.ToLower(r.Header.Get("Upgrade")) == "websocket" {
				next.ServeHTTP(w, r)
				return
			}

			rw := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}
			next.ServeHTTP(rw, r)
		})
	}
}

// RequestIDMiddleware adds a unique request ID to each request.
//
// Returns:
//
//	Middleware function
func RequestIDMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip WebSocket requests
			if strings.ToLower(r.Header.Get("Upgrade")) == "websocket" {
				next.ServeHTTP(w, r)
				return
			}

			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = generateRequestID()
			}
			w.Header().Set("X-Request-ID", requestID)
			ctx := context.WithValue(r.Context(), types.RequestIDKey, requestID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode  int
	wroteHeader bool
}

// WriteHeader captures the status code and delegates to the wrapped writer.
func (rw *responseWriter) WriteHeader(code int) {
	if !rw.wroteHeader {
		rw.statusCode = code
		rw.wroteHeader = true
	}
	rw.ResponseWriter.WriteHeader(code)
}

// Write captures the default 200 status if WriteHeader hasn't been called.
func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.statusCode = http.StatusOK
		rw.wroteHeader = true
	}
	return rw.ResponseWriter.Write(b)
}

func (rw *responseWriter) StatusCode() int {
	return rw.statusCode
}

// Hijack implements http.Hijacker interface for WebSocket support
func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	// Check if the underlying writer supports Hijack
	if hijacker, ok := rw.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, fmt.Errorf("underlying ResponseWriter does not support http.Hijacker")
}

// Flush implements http.Flusher interface (needed for WebSocket)
func (rw *responseWriter) Flush() {
	if flusher, ok := rw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// CloseNotify implements http.CloseNotifier interface (deprecated but needed)
func (rw *responseWriter) CloseNotify() <-chan bool {
	if notifier, ok := rw.ResponseWriter.(http.CloseNotifier); ok {
		return notifier.CloseNotify()
	}
	return nil
}

// Push implements http.Pusher interface (for HTTP/2)
func (rw *responseWriter) Push(target string, opts *http.PushOptions) error {
	if pusher, ok := rw.ResponseWriter.(http.Pusher); ok {
		return pusher.Push(target, opts)
	}
	return http.ErrNotSupported
}

// generateRequestID generates a unique request identifier.
func generateRequestID() string {
	return fmt.Sprintf("%d-%s", time.Now().UnixNano(), randomHexString(8))
}

// randomHexString generates a random hexadecimal string.
func randomHexString(length int) string {
	const hexChars = "0123456789abcdef"
	b := make([]byte, length)
	for i := range b {
		b[i] = hexChars[time.Now().UnixNano()%int64(len(hexChars))]
		time.Sleep(1 * time.Nanosecond) // Ensure different values
	}
	return string(b)
}
