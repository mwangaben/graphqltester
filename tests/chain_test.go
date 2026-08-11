package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	. "github.com/mwangaben/graphqltester/pkg/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/**
 * TestNewMiddlewareChain_CreatesEmptyChain validates that a new chain
 * is created with no middleware and metrics initialized.
 */
func TestNewMiddlewareChain_CreatesEmptyChain(t *testing.T) {
	chain := NewMiddlewareChain(false)

	assert.NotNil(t, chain, "Chain should not be nil")
	assert.Equal(t, 0, chain.Count(), "New chain should be empty")
	assert.NotNil(t, chain.metrics, "Metrics should be initialized")
	assert.False(t, chain.debug, "Debug should be disabled")
}

/**
 * TestNewMiddlewareChain_DebugMode validates that debug mode is set correctly.
 */
func TestNewMiddlewareChain_DebugMode(t *testing.T) {
	chain := NewMiddlewareChain(true)

	assert.True(t, chain.debug, "Debug should be enabled")
}

/**
 * TestAdd_AppendsMiddleware validates that Add appends middleware to the chain.
 */
func TestAdd_AppendsMiddleware(t *testing.T) {
	chain := NewMiddlewareChain(false)

	mw1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-MW1", "true")
			next.ServeHTTP(w, r)
		})
	}

	mw2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-MW2", "true")
			next.ServeHTTP(w, r)
		})
	}

	chain.Add("mw1", mw1)
	chain.Add("mw2", mw2)

	assert.Equal(t, 2, chain.Count(), "Chain should have 2 middleware")
	assert.True(t, chain.Has("mw1"), "Should have mw1")
	assert.True(t, chain.Has("mw2"), "Should have mw2")
	assert.Equal(t, 0, chain.Position("mw1"), "mw1 should be at position 0")
	assert.Equal(t, 1, chain.Position("mw2"), "mw2 should be at position 1")
}

/**
 * TestAdd_DuplicateNamePanics validates that adding a middleware with
 * a duplicate name causes a panic.
 */
func TestAdd_DuplicateNamePanics(t *testing.T) {
	chain := NewMiddlewareChain(false)

	mw := func(next http.Handler) http.Handler { return next }

	chain.Add("test", mw)

	assert.Panics(t, func() {
		chain.Add("test", mw)
	}, "Should panic on duplicate name")
}

/**
 * TestInsertAt_ValidPosition validates inserting middleware at a valid position.
 */
func TestInsertAt_ValidPosition(t *testing.T) {
	chain := NewMiddlewareChain(false)

	mw1 := func(next http.Handler) http.Handler { return next }
	mw2 := func(next http.Handler) http.Handler { return next }
	mw3 := func(next http.Handler) http.Handler { return next }

	chain.Add("mw1", mw1)
	chain.Add("mw3", mw3)

	err := chain.InsertAt(1, "mw2", mw2)

	require.NoError(t, err, "Should insert without error")
	assert.Equal(t, 3, chain.Count(), "Should have 3 middleware")
	assert.Equal(t, 0, chain.Position("mw1"), "mw1 should be first")
	assert.Equal(t, 1, chain.Position("mw2"), "mw2 should be second")
	assert.Equal(t, 2, chain.Position("mw3"), "mw3 should be third")
}

/**
 * TestInsertAt_InvalidPosition validates error on invalid position.
 */
func TestInsertAt_InvalidPosition(t *testing.T) {
	chain := NewMiddlewareChain(false)
	mw := func(next http.Handler) http.Handler { return next }

	err := chain.InsertAt(5, "test", mw)

	assert.Error(t, err, "Should error on invalid position")
	assert.Contains(t, err.Error(), "invalid index")
}

/**
 * TestRemove_ExistingMiddleware validates removing an existing middleware.
 */
func TestRemove_ExistingMiddleware(t *testing.T) {
	chain := NewMiddlewareChain(false)
	mw := func(next http.Handler) http.Handler { return next }

	chain.Add("test", mw)
	assert.True(t, chain.Has("test"), "Should have middleware before removal")

	removed := chain.Remove("test")

	assert.True(t, removed, "Should return true on successful removal")
	assert.False(t, chain.Has("test"), "Should not have middleware after removal")
	assert.Equal(t, 0, chain.Count(), "Chain should be empty")
}

/**
 * TestRemove_NonExistentMiddleware validates removing non-existent middleware.
 */
func TestRemove_NonExistentMiddleware(t *testing.T) {
	chain := NewMiddlewareChain(false)

	removed := chain.Remove("nonexistent")

	assert.False(t, removed, "Should return false for non-existent middleware")
}

/**
 * TestReplace_ExistingMiddleware validates replacing an existing middleware.
 */
func TestReplace_ExistingMiddleware(t *testing.T) {
	chain := NewMiddlewareChain(false)

	originalMW := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Original", "true")
			next.ServeHTTP(w, r)
		})
	}

	replacementMW := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Replacement", "true")
			next.ServeHTTP(w, r)
		})
	}

	chain.Add("test", originalMW)
	replaced := chain.Replace("test", replacementMW)

	assert.True(t, replaced, "Should return true on successful replacement")

	// Test that the replacement middleware is used
	handler := chain.Build(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Empty(t, w.Header().Get("X-Original"), "Original middleware should be replaced")
	assert.Equal(t, "true", w.Header().Get("X-Replacement"), "Replacement middleware should be active")
}

/**
 * TestBuild_ExecutesMiddlewareInOrder validates that middleware executes
 * in the correct order (FIFO for pre-processing).
 */
func TestBuild_ExecutesMiddlewareInOrder(t *testing.T) {
	chain := NewMiddlewareChain(false)
	executionOrder := make([]string, 0)

	// First middleware (outermost)
	chain.Add("first", func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			executionOrder = append(executionOrder, "first-pre")
			next.ServeHTTP(w, r)
			executionOrder = append(executionOrder, "first-post")
		})
	})

	// Second middleware (middle)
	chain.Add("second", func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			executionOrder = append(executionOrder, "second-pre")
			next.ServeHTTP(w, r)
			executionOrder = append(executionOrder, "second-post")
		})
	})

	// Third middleware (innermost)
	chain.Add("third", func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			executionOrder = append(executionOrder, "third-pre")
			next.ServeHTTP(w, r)
			executionOrder = append(executionOrder, "third-post")
		})
	})

	handler := chain.Build(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		executionOrder = append(executionOrder, "handler")
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	expected := []string{
		"first-pre",
		"second-pre",
		"third-pre",
		"handler",
		"third-post",
		"second-post",
		"first-post",
	}

	assert.Equal(t, expected, executionOrder, "Middleware should execute in FIFO order")
}

/**
 * TestBuildFunc_ConvenienceMethod validates that BuildFunc works correctly.
 */
func TestBuildFunc_ConvenienceMethod(t *testing.T) {
	chain := NewMiddlewareChain(false)

	called := false
	handler := chain.BuildFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.True(t, called, "Handler function should be called")
}

/**
 * TestClear_RemovesAllMiddleware validates that Clear removes everything.
 */
func TestClear_RemovesAllMiddleware(t *testing.T) {
	chain := NewMiddlewareChain(false)
	mw := func(next http.Handler) http.Handler { return next }

	chain.Add("mw1", mw)
	chain.Add("mw2", mw)
	chain.Add("mw3", mw)

	assert.Equal(t, 3, chain.Count(), "Should have 3 middleware")

	chain.Clear()

	assert.Equal(t, 0, chain.Count(), "Should have no middleware after clear")
	assert.Empty(t, chain.Names(), "Names should be empty")
}

/**
 * TestGetMetrics_ReturnsAverages validates that metrics are calculated correctly.
 */
func TestGetMetrics_ReturnsAverages(t *testing.T) {
	chain := NewMiddlewareChain(true)

	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}

	chain.Add("test-mw", mw)

	handler := chain.Build(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Make multiple requests
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}

	metrics := chain.GetMetrics()

	assert.Contains(t, metrics, "test-mw", "Should have metrics for test-mw")
	assert.Greater(t, metrics["test-mw"], time.Duration(0), "Average should be positive")
}

/**
 * TestResponseWriter_CapturesStatusCode validates that the response writer
 * correctly captures the status code.
 */
func TestResponseWriter_CapturesStatusCode(t *testing.T) {
	// Test explicit WriteHeader
	rw := &responseWriter{
		ResponseWriter: httptest.NewRecorder(),
		statusCode:     http.StatusOK,
	}

	rw.WriteHeader(http.StatusCreated)
	assert.Equal(t, http.StatusCreated, rw.StatusCode(), "Should capture explicit status")

	// Test implicit 200 via Write
	rw2 := &responseWriter{
		ResponseWriter: httptest.NewRecorder(),
		statusCode:     http.StatusOK,
	}

	rw2.Write([]byte("test"))
	assert.Equal(t, http.StatusOK, rw2.StatusCode(), "Should default to 200 OK")
}

/**
 * TestRequestIDMiddleware_GeneratesID validates that request ID middleware
 * adds a request ID to the response header.
 */
func TestRequestIDMiddleware_GeneratesID(t *testing.T) {
	middleware := RequestIDMiddleware()

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check that request ID is in context
		requestID := r.Context().Value(RequestIDKey)
		assert.NotNil(t, requestID, "Request ID should be in context")

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.NotEmpty(t, w.Header().Get("X-Request-ID"), "Response should have request ID header")
}
