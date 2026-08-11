package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

/**
 * Middleware Chain for GraphQL Test Requests
 *
 * The middleware chain provides a flexible, ordered pipeline for processing
 * HTTP requests before they reach the GraphQL handler. Each middleware in the
 * chain can:
 * - Modify the request before passing it to the next handler
 * - Modify the response after receiving it from the next handler
 * - Short-circuit the chain by not calling the next handler
 * - Add values to the request context for downstream handlers
 *
 * Middleware Execution Order:
 *   Request → MW1 → MW2 → MW3 → GraphQL Handler → MW3 → MW2 → MW1 → Response
 *
 * The chain supports:
 * - Ordered middleware execution (FIFO for pre-processing, LIFO for post-processing)
 * - Named middleware for identification and replacement
 * - Insertion and removal at specific positions
 * - Performance metrics collection
 * - Debug logging of middleware execution
 *
 * Example:
 *   chain := NewMiddlewareChain(true) // Enable debug mode
 *   chain.Add("auth", AuthMiddleware(tester))
 *   chain.Add("logging", LoggingMiddleware())
 *
 *   handler := chain.Build(graphqlHandler)
 *   // handler now processes requests through auth → logging → graphql
 */

// ============================================================================
// Core Types
// ============================================================================

//Middleware
/**
 * Middleware is a function that wraps an http.Handler to provide additional
 * functionality. It follows the standard Go middleware pattern.
 *
 * Each middleware receives the next handler in the chain and returns a new
 * handler that wraps it. This creates a chain of responsibility where each
 * middleware can process the request before and after the next handler.
 *
 * Standard signature:
 *   func(next http.Handler) http.Handler
 */
type Middleware func(http.Handler) http.Handler

/**
 * MiddlewareChain manages an ordered collection of middleware functions.
 *
 * The chain maintains middleware in order of registration and provides
 * methods for building the final handler, adding/removing middleware,
 * and collecting performance metrics.
 *
 * Thread-safety: All methods are safe for concurrent use.
 */
type MiddlewareChain struct {
	// middlewares holds the middleware functions in order of execution
	middlewares []Middleware

	// names holds the names of each middleware for identification
	names []string

	// mu protects concurrent access to the chain
	mu sync.RWMutex

	// debug enables detailed logging of middleware execution
	debug bool

	// metrics collects performance data when enabled
	metrics *MiddlewareMetrics
}

//MiddlewareMetrics
/**
 * MiddlewareMetrics collects timing data for middleware execution.
 *
 * When debug mode is enabled, each middleware's execution time is
 * recorded and can be retrieved for performance analysis.
 */
type MiddlewareMetrics struct {
	// ExecutionTimes maps middleware names to their execution durations
	ExecutionTimes map[string][]time.Duration

	// mu protects concurrent access to metrics
	mu sync.RWMutex
}

// ============================================================================
// Constructor
// ============================================================================

//NewMiddlewareChain
/**
 * NewMiddlewareChain creates a new middleware chain with optional debug mode.
 *
 * Parameters:
 *   debug - If true, enables execution time tracking and debug logging
 *
 * Returns:
 *   *MiddlewareChain ready for middleware registration
 *
 * Example:
 *   // Production chain (no debug)
 *   chain := NewMiddlewareChain(false)
 *
 *   // Development chain (with debug)
 *   chain := NewMiddlewareChain(true)
 */
func NewMiddlewareChain(debug bool) *MiddlewareChain {
	return &MiddlewareChain{
		middlewares: make([]Middleware, 0),
		names:       make([]string, 0),
		debug:       debug,
		metrics: &MiddlewareMetrics{
			ExecutionTimes: make(map[string][]time.Duration),
		},
	}
}

// ============================================================================
// Middleware Management
// ============================================================================

//Add
/**
 * Add appends a middleware to the end of the chain.
 *
 * Middleware added last executes closest to the handler (innermost).
 * The first middleware added is the outermost wrapper.
 *
 * Parameters:
 *   name       - Unique name for this middleware (used for identification and removal)
 *   middleware  - The middleware function to add
 *
 * Returns:
 *   *MiddlewareChain for fluent method chaining
 *
 * Example:
 *   chain.Add("auth", AuthMiddleware(tester)).
 *         Add("logging", LoggingMiddleware()).
 *         Add("cors", CORSMiddleware())
 *
 * Execution order for the above:
 *   Request → cors → logging → auth → Handler → auth → logging → cors → Response
 */
func (mc *MiddlewareChain) Add(name string, middleware Middleware) *MiddlewareChain {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	// Validate name uniqueness
	for _, n := range mc.names {
		if n == name {
			panic(fmt.Sprintf("middleware with name '%s' already exists in the chain", name))
		}
	}

	mc.names = append(mc.names, name)

	// Wrap with metrics if debug is enabled
	if mc.debug {
		middleware = mc.wrapWithMetrics(name, middleware)
	}

	mc.middlewares = append(mc.middlewares, middleware)

	if mc.debug {
		fmt.Printf("🔗 Added middleware '%s' at position %d\n", name, len(mc.middlewares)-1)
	}

	return mc
}

//InsertAt
/**
 * InsertAt inserts a middleware at a specific position in the chain.
 *
 * Position 0 is the outermost middleware (first to receive the request).
 * Higher positions are closer to the handler.
 *
 * Parameters:
 *   index      - Position to insert at (0-based)
 *   name       - Unique name for the middleware
 *   middleware  - The middleware function
 *
 * Returns:
 *   error if the index is out of bounds
 *
 * Example:
 *   // Insert auth middleware at the beginning of the chain
 *   err := chain.InsertAt(0, "auth", AuthMiddleware(tester))
 */
func (mc *MiddlewareChain) InsertAt(index int, name string, middleware Middleware) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if index < 0 || index > len(mc.middlewares) {
		return fmt.Errorf(
			"invalid index %d for middleware chain of length %d",
			index,
			len(mc.middlewares),
		)
	}

	// Validate name uniqueness
	for _, n := range mc.names {
		if n == name {
			return fmt.Errorf("middleware with name '%s' already exists", name)
		}
	}

	// Wrap with metrics if debug is enabled
	if mc.debug {
		middleware = mc.wrapWithMetrics(name, middleware)
	}

	// Insert the middleware
	mc.middlewares = append(
		mc.middlewares[:index],
		append([]Middleware{middleware}, mc.middlewares[index:]...)...,
	)
	mc.names = append(
		mc.names[:index],
		append([]string{name}, mc.names[index:]...)...,
	)

	if mc.debug {
		fmt.Printf("🔗 Inserted middleware '%s' at position %d\n", name, index)
	}

	return nil
}

/**
 * Remove removes a middleware from the chain by name.
 *
 * Parameters:
 *   name - Name of the middleware to remove
 *
 * Returns:
 *   bool indicating whether the middleware was found and removed
 *
 * Example:
 *   if chain.Remove("cors") {
 *       fmt.Println("CORS middleware removed")
 *   }
 */
func (mc *MiddlewareChain) Remove(name string) bool {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	for i, n := range mc.names {
		if n == name {
			mc.middlewares = append(mc.middlewares[:i], mc.middlewares[i+1:]...)
			mc.names = append(mc.names[:i], mc.names[i+1:]...)

			if mc.debug {
				fmt.Printf("🔗 Removed middleware '%s'\n", name)
			}

			return true
		}
	}

	return false
}

/**
 * Replace replaces an existing middleware with a new one.
 *
 * This is useful when you need to swap out a middleware for a different
 * implementation while keeping the same position in the chain.
 *
 * Parameters:
 *   name          - Name of the middleware to replace
 *   newMiddleware  - The replacement middleware function
 *
 * Returns:
 *   bool indicating whether the middleware was found and replaced
 *
 * Example:
 *   // Replace default auth middleware with a custom one
 *   chain.Replace("auth", CustomAuthMiddleware(testUser))
 */
func (mc *MiddlewareChain) Replace(name string, newMiddleware Middleware) bool {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	for i, n := range mc.names {
		if n == name {
			if mc.debug {
				newMiddleware = mc.wrapWithMetrics(name, newMiddleware)
			}
			mc.middlewares[i] = newMiddleware

			if mc.debug {
				fmt.Printf("🔗 Replaced middleware '%s'\n", name)
			}

			return true
		}
	}

	return false
}

/**
 * Has checks if a middleware with the given name exists in the chain.
 *
 * Parameters:
 *   name - Name of the middleware to check
 *
 * Returns:
 *   bool indicating whether the middleware exists
 */
func (mc *MiddlewareChain) Has(name string) bool {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	for _, n := range mc.names {
		if n == name {
			return true
		}
	}
	return false
}

/**
 * Position returns the index of a middleware by name.
 *
 * Parameters:
 *   name - Name of the middleware
 *
 * Returns:
 *   int index (0-based) or -1 if not found
 */
func (mc *MiddlewareChain) Position(name string) int {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	for i, n := range mc.names {
		if n == name {
			return i
		}
	}
	return -1
}

/**
 * Names returns the names of all middleware in the chain.
 *
 * Returns:
 *   []string slice of middleware names in order
 */
func (mc *MiddlewareChain) Names() []string {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	names := make([]string, len(mc.names))
	copy(names, mc.names)
	return names
}

/**
 * Count returns the number of middleware in the chain.
 *
 * Returns:
 *   int count of middleware
 */
func (mc *MiddlewareChain) Count() int {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	return len(mc.middlewares)
}

/**
 * Clear removes all middleware from the chain.
 *
 * Returns:
 *   *MiddlewareChain for fluent method chaining
 */
func (mc *MiddlewareChain) Clear() *MiddlewareChain {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.middlewares = make([]Middleware, 0)
	mc.names = make([]string, 0)
	mc.metrics = &MiddlewareMetrics{
		ExecutionTimes: make(map[string][]time.Duration),
	}

	if mc.debug {
		fmt.Println("🔗 Cleared all middleware from chain")
	}

	return mc
}

// ============================================================================
// Chain Building
// ============================================================================

/**
 * Build constructs the final http.Handler by composing all middleware.
 *
 * This method applies middleware in order, wrapping the final handler
 * with each middleware from innermost to outermost. The first middleware
 * added to the chain is the outermost wrapper.
 *
 * Parameters:
 *   finalHandler - The innermost handler (typically the GraphQL handler)
 *
 * Returns:
 *   http.Handler with all middleware applied
 *
 * Example:
 *   graphqlHandler := createGraphQLHandler()
 *   finalHandler := chain.Build(graphqlHandler)
 *
 *   http.ListenAndServe(":8080", finalHandler)
 *
 * How it works:
 *   Given chain: [MW1, MW2, MW3] and finalHandler H
 *   Result: MW1(MW2(MW3(H)))
 *
 *   Request flow: Request → MW1 → MW2 → MW3 → H → MW3 → MW2 → MW1 → Response
 */
func (mc *MiddlewareChain) Build(finalHandler http.Handler) http.Handler {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	handler := finalHandler

	// Apply middleware in reverse order
	// The last middleware added wraps closest to the handler
	for i := len(mc.middlewares) - 1; i >= 0; i-- {
		handler = mc.middlewares[i](handler)
	}

	if mc.debug {
		fmt.Printf("🔗 Built middleware chain with %d middleware(s)\n", len(mc.middlewares))
		for i, name := range mc.names {
			fmt.Printf("   [%d] %s\n", i, name)
		}
	}

	return handler
}

/**
 * BuildFunc is a convenience method that accepts a handler function instead
 * of an http.Handler interface.
 *
 * Parameters:
 *   handlerFunc - The final handler function
 *
 * Returns:
 *   http.Handler with all middleware applied
 *
 * Example:
 *   handler := chain.BuildFunc(func(w http.ResponseWriter, r *http.Request) {
 *       w.Write([]byte("Hello, World!"))
 *   })
 */
func (mc *MiddlewareChain) BuildFunc(handlerFunc http.HandlerFunc) http.Handler {
	return mc.Build(handlerFunc)
}

// ============================================================================
// Metrics and Debugging
// ============================================================================

/**
 * wrapWithMetrics wraps a middleware with execution time tracking.
 *
 * When debug mode is enabled, this wrapper records how long each middleware
 * takes to execute, which can be useful for performance profiling.
 *
 * Parameters:
 *   name       - Name of the middleware for metrics identification
 *   middleware  - The middleware to wrap
 *
 * Returns:
 *   Middleware with metrics collection
 */
func (mc *MiddlewareChain) wrapWithMetrics(name string, middleware Middleware) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Execute the middleware
			middleware(next).ServeHTTP(w, r)

			// Record execution time
			duration := time.Since(start)
			mc.metrics.mu.Lock()
			mc.metrics.ExecutionTimes[name] = append(
				mc.metrics.ExecutionTimes[name],
				duration,
			)
			mc.metrics.mu.Unlock()

			if mc.debug {
				fmt.Printf("⏱️  Middleware '%s' took %v\n", name, duration)
			}
		})
	}
}

/**
 * GetMetrics returns average execution times for each middleware.
 *
 * This provides performance insights for the middleware chain.
 *
 * Returns:
 *   map[string]time.Duration with average execution time per middleware
 *
 * Example:
 *   metrics := chain.GetMetrics()
 *   for name, avgTime := range metrics {
 *       fmt.Printf("%s: %v\n", name, avgTime)
 *   }
 */
func (mc *MiddlewareChain) GetMetrics() map[string]time.Duration {
	mc.metrics.mu.RLock()
	defer mc.metrics.mu.RUnlock()

	avgTimes := make(map[string]time.Duration)

	for name, times := range mc.metrics.ExecutionTimes {
		if len(times) == 0 {
			continue
		}

		var total time.Duration
		for _, t := range times {
			total += t
		}
		avgTimes[name] = total / time.Duration(len(times))
	}

	return avgTimes
}

/**
 * ResetMetrics clears all collected metrics.
 *
 * Returns:
 *   *MiddlewareChain for fluent method chaining
 */
func (mc *MiddlewareChain) ResetMetrics() *MiddlewareChain {
	mc.metrics.mu.Lock()
	defer mc.metrics.mu.Unlock()

	mc.metrics.ExecutionTimes = make(map[string][]time.Duration)

	return mc
}

/**
 * SetDebug enables or disables debug mode.
 *
 * When debug mode is enabled:
 * - Middleware execution times are tracked
 * - Add/Remove operations are logged
 * - Build operation outputs the chain structure
 *
 * Parameters:
 *   debug - Whether to enable debug mode
 *
 * Returns:
 *   *MiddlewareChain for fluent method chaining
 */
func (mc *MiddlewareChain) SetDebug(debug bool) *MiddlewareChain {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.debug = debug

	// If enabling debug, re-wrap existing middleware with metrics
	if debug {
		for i, mw := range mc.middlewares {
			mc.middlewares[i] = mc.wrapWithMetrics(mc.names[i], func(next http.Handler) http.Handler {
				return mw(next) // Preserve the original middleware
			})
		}
	}

	return mc
}

// ============================================================================
// Pre-configured Chains
// ============================================================================

/**
 * NewDefaultChain creates a middleware chain with common defaults.
 *
 * This provides a sensible starting point for most test scenarios:
 * 1. Request ID middleware for traceability
 * 2. Context propagation middleware
 * 3. Response capture middleware
 *
 * Parameters:
 *   tester - The test tester instance for context propagation
 *   debug  - Whether to enable debug mode
 *
 * Returns:
 *   *MiddlewareChain with default middleware pre-configured
 *
 * Example:
 *   chain := NewDefaultChain(tester, true)
 *   handler := chain.Build(graphqlHandler)
 */
func NewDefaultChain(tester interface{}, debug bool) *MiddlewareChain {
	chain := NewMiddlewareChain(debug)

	chain.Add("request-id", RequestIDMiddleware())
	chain.Add("context", ContextPropagationMiddleware(tester))
	chain.Add("response-capture", ResponseContextMiddleware(tester))

	return chain
}

/**
 * NewTestChain creates a middleware chain configured for testing.
 *
 * This chain includes authentication, permission checking, and
 * validation middleware for comprehensive test coverage.
 *
 * Parameters:
 *   tester - The test tester instance
 *   debug  - Whether to enable debug mode
 *
 * Returns:
 *   *MiddlewareChain with test middleware pre-configured
 *
 * Example:
 *   chain := NewTestChain(tester, true)
 *   chain.Build(graphqlHandler)
 */
func NewTestChain(tester interface{}, debug bool) *MiddlewareChain {
	chain := NewDefaultChain(tester, debug)

	// Add test-specific middleware
	chain.Add("auth", AuthMiddleware(tester))
	chain.Add("schema-analysis", SchemaAnalysisMiddleware(tester))
	chain.Add("permission", PermissionMiddleware(tester))
	chain.Add("validation", ValidationMiddleware(tester))

	return chain
}
