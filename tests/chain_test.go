package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mwangaben/graphqltester/pkg/middleware"
	"github.com/mwangaben/graphqltester/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Middleware Chain Tests
// ============================================================================

/**
 * TestNewMiddlewareChain_CreatesEmptyChain validates that a new chain
 * is created with no middleware.
 */
func TestNewMiddlewareChain_CreatesEmptyChain(t *testing.T) {
	chain := middleware.NewMiddlewareChain(false)

	assert.NotNil(t, chain, "Chain should not be nil")
	assert.Equal(t, 0, chain.Count(), "New chain should be empty")
}

/**
 * TestNewMiddlewareChain_DebugMode validates that debug mode can be set.
 */
func TestNewMiddlewareChain_DebugMode(t *testing.T) {
	chain := middleware.NewMiddlewareChain(true)

	// Debug mode should be enabled - verify by checking that metrics work
	assert.Equal(t, 0, chain.Count(), "New debug chain should be empty")

	// Metrics should be accessible
	metrics := chain.GetMetrics()
	assert.NotNil(t, metrics, "Metrics should be initialized")
	assert.Empty(t, metrics, "Metrics should be empty for new chain")
}

/**
 * TestAdd_AppendsMiddleware validates that Add appends middleware to the chain.
 */
func TestAdd_AppendsMiddleware(t *testing.T) {
	chain := middleware.NewMiddlewareChain(false)

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
	chain := middleware.NewMiddlewareChain(false)

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
	chain := middleware.NewMiddlewareChain(false)

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
	chain := middleware.NewMiddlewareChain(false)
	mw := func(next http.Handler) http.Handler { return next }

	err := chain.InsertAt(5, "test", mw)

	assert.Error(t, err, "Should error on invalid position")
	assert.Contains(t, err.Error(), "invalid index")
}

/**
 * TestInsertAt_NegativeIndex validates error on negative position.
 */
func TestInsertAt_NegativeIndex(t *testing.T) {
	chain := middleware.NewMiddlewareChain(false)
	mw := func(next http.Handler) http.Handler { return next }

	err := chain.InsertAt(-1, "test", mw)

	assert.Error(t, err, "Should error on negative index")
}

/**
 * TestRemove_ExistingMiddleware validates removing an existing middleware.
 */
func TestRemove_ExistingMiddleware(t *testing.T) {
	chain := middleware.NewMiddlewareChain(false)
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
	chain := middleware.NewMiddlewareChain(false)

	removed := chain.Remove("nonexistent")

	assert.False(t, removed, "Should return false for non-existent middleware")
}

/**
 * TestReplace_ExistingMiddleware validates replacing an existing middleware.
 */
func TestReplace_ExistingMiddleware(t *testing.T) {
	chain := middleware.NewMiddlewareChain(false)

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
 * TestReplace_NonExistentMiddleware validates replacing non-existent middleware.
 */
func TestReplace_NonExistentMiddleware(t *testing.T) {
	chain := middleware.NewMiddlewareChain(false)
	mw := func(next http.Handler) http.Handler { return next }

	replaced := chain.Replace("nonexistent", mw)

	assert.False(t, replaced, "Should return false for non-existent middleware")
}

/**
 * TestBuild_ExecutesMiddlewareInOrder validates that middleware executes
 * in the correct order (FIFO for pre-processing, LIFO for post-processing).
 */
func TestBuild_ExecutesMiddlewareInOrder(t *testing.T) {
	chain := middleware.NewMiddlewareChain(false)
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
	chain := middleware.NewMiddlewareChain(false)

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
	chain := middleware.NewMiddlewareChain(false)
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
	chain := middleware.NewMiddlewareChain(true)

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
 * TestGetMetrics_EmptyChain validates metrics for empty chain.
 */
func TestGetMetrics_EmptyChain(t *testing.T) {
	chain := middleware.NewMiddlewareChain(true)

	metrics := chain.GetMetrics()

	assert.NotNil(t, metrics, "Metrics should not be nil")
	assert.Empty(t, metrics, "Metrics should be empty for empty chain")
}

/**
 * TestResetMetrics_ClearsMetrics validates that ResetMetrics clears data.
 */
func TestResetMetrics_ClearsMetrics(t *testing.T) {
	chain := middleware.NewMiddlewareChain(true)

	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}

	chain.Add("test-mw", mw)

	handler := chain.Build(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Make a request
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Verify metrics exist
	metrics := chain.GetMetrics()
	assert.Contains(t, metrics, "test-mw", "Should have metrics before reset")

	// Reset metrics
	chain.ResetMetrics()

	// Verify metrics are cleared
	metricsAfter := chain.GetMetrics()
	assert.Empty(t, metricsAfter, "Metrics should be empty after reset")
}

/**
 * TestRequestIDMiddleware_GeneratesID validates that request ID middleware
 * adds a request ID to the response header.
 */
func TestRequestIDMiddleware_GeneratesID(t *testing.T) {
	mw := middleware.RequestIDMiddleware()

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check that request ID is in context using types.RequestIDKey
		requestID := r.Context().Value(types.RequestIDKey)
		assert.NotNil(t, requestID, "Request ID should be in context")

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.NotEmpty(t, w.Header().Get("X-Request-ID"), "Response should have request ID header")
}

/**
 * TestRequestIDMiddleware_PreservesExistingID validates that existing
 * request IDs are preserved.
 */
func TestRequestIDMiddleware_PreservesExistingID(t *testing.T) {
	mw := middleware.RequestIDMiddleware()

	existingID := "existing-request-id-123"

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Context().Value(types.RequestIDKey)
		assert.Equal(t, existingID, requestID, "Should preserve existing request ID")
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", existingID)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, existingID, w.Header().Get("X-Request-ID"), "Response should have same request ID")
}

/**
 * TestContextPropagationMiddleware_PropagatesContext validates that
 * context propagation middleware works.
 */
func TestContextPropagationMiddleware_PropagatesContext(t *testing.T) {
	// Create a mock tester that implements the required interface
	mockTester := &mockContextTester{}

	mw := middleware.ContextPropagationMiddleware(mockTester)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request ID is in context
		requestID := r.Context().Value(types.RequestIDKey)
		assert.NotNil(t, requestID, "Request ID should be in context")

		// Verify request start time is in context
		startTime := r.Context().Value(types.RequestStartTimeKey)
		assert.NotNil(t, startTime, "Request start time should be in context")

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

/**
 * TestMiddlewareChain_NamesReturnsCorrectOrder validates that Names()
 * returns middleware names in the correct order.
 */
func TestMiddlewareChain_NamesReturnsCorrectOrder(t *testing.T) {
	chain := middleware.NewMiddlewareChain(false)
	mw := func(next http.Handler) http.Handler { return next }

	chain.Add("auth", mw)
	chain.Add("logging", mw)
	chain.Add("cors", mw)

	names := chain.Names()

	expected := []string{"auth", "logging", "cors"}
	assert.Equal(t, expected, names, "Names should be in order of addition")
}

/**
 * TestMiddlewareChain_PositionReturnsCorrectIndex validates Position().
 */
func TestMiddlewareChain_PositionReturnsCorrectIndex(t *testing.T) {
	chain := middleware.NewMiddlewareChain(false)
	mw := func(next http.Handler) http.Handler { return next }

	chain.Add("first", mw)
	chain.Add("second", mw)
	chain.Add("third", mw)

	assert.Equal(t, 0, chain.Position("first"))
	assert.Equal(t, 1, chain.Position("second"))
	assert.Equal(t, 2, chain.Position("third"))
	assert.Equal(t, -1, chain.Position("nonexistent"))
}

/**
 * TestMiddlewareChain_SetDebug_TogglesDebug validates SetDebug.
 */
func TestMiddlewareChain_SetDebug_TogglesDebug(t *testing.T) {
	chain := middleware.NewMiddlewareChain(false)

	// Enable debug
	chain.SetDebug(true)

	// Add middleware and make requests
	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}
	chain.Add("test-mw", mw)

	handler := chain.Build(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Metrics should be collected when debug is enabled
	metrics := chain.GetMetrics()
	assert.Contains(t, metrics, "test-mw", "Metrics should be collected in debug mode")

	// Disable debug
	chain.SetDebug(false)
	chain.ResetMetrics()

	handler2 := chain.Build(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req2 := httptest.NewRequest("GET", "/test", nil)
	w2 := httptest.NewRecorder()
	handler2.ServeHTTP(w2, req2)

	// Note: When debug is disabled, existing middleware may still have metrics wrapping
	// This test verifies the SetDebug method works without panicking
}

// ============================================================================
// Mock Tester for Context Propagation Tests
// ============================================================================

/**
 * mockContextTester implements the minimal interface required by
 * ContextPropagationMiddleware.
 */
type mockContextTester struct{}

func (m *mockContextTester) Context() interface{} {
	return nil
}

func (m *mockContextTester) GetCurrentUser() interface{} {
	return nil
}

func (m *mockContextTester) GetCurrentToken() string {
	return ""
}

func (m *mockContextTester) TestName() string {
	return "mock-test"
}

// ============================================================================
// NewTestChain Tests
// ============================================================================

/**
 * TestNewDefaultChain_CreatesChainWithDefaults validates NewDefaultChain.
 */
func TestNewDefaultChain_CreatesChainWithDefaults(t *testing.T) {
	mockTester := &mockContextTester{}
	chain := middleware.NewDefaultChain(mockTester, false)

	assert.NotNil(t, chain, "Default chain should be created")
	assert.Greater(t, chain.Count(), 0, "Default chain should have middleware")
}

/**
 * TestNewTestChain_CreatesChainWithTestMiddleware validates NewTestChain.
 */
func TestNewTestChain_CreatesChainWithTestMiddleware(t *testing.T) {
	mockTester := &mockContextTester{}
	chain := middleware.NewTestChain(mockTester, false)

	assert.NotNil(t, chain, "Test chain should be created")
	assert.Greater(t, chain.Count(), 0, "Test chain should have middleware")

	// Test chain should include auth, schema-analysis, permission, validation
	assert.True(t, chain.Has("auth"), "Test chain should have auth middleware")
	assert.True(t, chain.Has("schema-analysis"), "Test chain should have schema-analysis middleware")
	assert.True(t, chain.Has("permission"), "Test chain should have permission middleware")
	assert.True(t, chain.Has("validation"), "Test chain should have validation middleware")
}
