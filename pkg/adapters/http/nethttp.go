package http

import (
	"net/http"
	"net/http/httptest"
)

/**
 * NetHTTPAdapter provides GraphQL testing support for the standard net/http package.
 *
 * This adapter is the default choice when no specific HTTP framework is required.
 * It uses the standard library's httptest.Server for creating test servers and
 * works with any http.Handler implementation.
 *
 * The standard library adapter is:
 * - Zero-dependency: Uses only the Go standard library
 * - Lightweight: Minimal overhead for test execution
 * - Universal: Compatible with any http.Handler-based GraphQL server
 *
 * Usage:
 *   config := &Config{
 *       HTTPAdapter: &http.NetHTTPAdapter{},
 *   }
 *
 * When to use:
 * - Testing GraphQL servers built with standard net/http
 * - When framework-specific features aren't needed
 * - For maximum compatibility across different setups
 */

//FrameworkAdapter
/**
 * FrameworkAdapter defines the interface that all HTTP framework adapters
 * must implement. This allows the tester to work with any HTTP framework
 * through a common interface.
 *
 * Each adapter is responsible for:
 * - Creating a test server from an http.Handler
 * - Adding framework-specific middleware
 * - Creating framework-specific contexts
 * - Providing the framework name for debugging
 */
type FrameworkAdapter interface {
	// Setup creates a test server from an http.Handler.
	// The returned server is already started and ready to accept requests.
	Setup(handler http.Handler) *httptest.Server

	// CreateContext creates a framework-specific context from an HTTP request.
	// This allows resolvers to access framework-specific features.
	CreateContext(req *http.Request) interface{}

	// AddMiddleware wraps the handler with framework-specific middleware.
	// This allows framework-native middleware to be used in tests.
	AddMiddleware(handler http.Handler, middlewares ...interface{}) http.Handler

	// Name returns the framework name for logging and debugging.
	Name() string
}

// NetHTTPAdapter implements FrameworkAdapter for standard net/http.
type NetHTTPAdapter struct {
	// server is the test server instance, stored for cleanup.
	server *httptest.Server
}

//Setup
/**
 * Setup creates a test HTTP server using the standard library.
 *
 * This creates an httptest.Server that listens on a random port and
 * forwards all requests to the provided handler. The server URL can
 * be accessed via server.URL.
 *
 * Parameters:
 *   handler - The http.Handler to serve requests
 *
 * Returns:
 *   *httptest.Server ready to accept requests
 *
 * Example:
 *   adapter := &NetHTTPAdapter{}
 *   server := adapter.Setup(myGraphQLHandler)
 *   defer server.Close()
 *
 *   // Make requests to server.URL
 *   resp, _ := http.Post(server.URL + "/graphql", "application/json", body)
 */
func (a *NetHTTPAdapter) Setup(handler http.Handler) *httptest.Server {
	a.server = httptest.NewServer(handler)
	return a.server
}

//CreateContext
/**
 * CreateContext extracts the standard library context from a request.
 *
 * For net/http, this simply returns the request's context which is
 * a context.Context. Framework-specific adapters may return different
 * context types.
 *
 * Parameters:
 *   req - The HTTP request to extract context from
 *
 * Returns:
 *   interface{} containing the request context
 */
func (a *NetHTTPAdapter) CreateContext(req *http.Request) interface{} {
	return req.Context()
}

//AddMiddleware
/**
 * AddMiddleware applies middleware to the handler.
 *
 * For net/http, middleware is simply a function that wraps an http.Handler.
 * Multiple middlewares are composed in the order they are provided.
 *
 * Parameters:
 *   handler     - The base http.Handler
 *   middlewares - Middleware functions to apply
 *
 * Returns:
 *   http.Handler with all middleware applied
 *
 * Example:
 *   handler := adapter.AddMiddleware(
 *       myHandler,
 *       loggingMiddleware,
 *       authMiddleware,
 *   )
 */
func (a *NetHTTPAdapter) AddMiddleware(handler http.Handler, middlewares ...interface{}) http.Handler {
	wrapped := handler

	// Apply middlewares in order (first middleware is outermost)
	for i := len(middlewares) - 1; i >= 0; i-- {
		if mw, ok := middlewares[i].(func(http.Handler) http.Handler); ok {
			wrapped = mw(wrapped)
		}
	}

	return wrapped
}

//Name
/**
 * Name returns the adapter name for debugging purposes.
 *
 * Returns:
 *   string "net/http"
 */
func (a *NetHTTPAdapter) Name() string {
	return "net/http"
}

//Close
/**
 * Close shuts down the test server if it was created.
 *
 * This should be called during cleanup to free resources.
 */
func (a *NetHTTPAdapter) Close() {
	if a.server != nil {
		a.server.Close()
	}
}

//URL
/**
 * URL returns the base URL of the test server.
 *
 * Returns:
 *   string URL of the test server, or empty string if not started
 */
func (a *NetHTTPAdapter) URL() string {
	if a.server != nil {
		return a.server.URL
	}
	return ""
}
