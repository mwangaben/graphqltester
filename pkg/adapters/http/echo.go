package http

import (
	"net/http"
	"net/http/httptest"

	"github.com/labstack/echo/v4"
)

/**
 * EchoAdapter provides GraphQL testing support for the Echo web framework.
 *
 * Echo is a high-performance, minimalist Go web framework. This adapter
 * integrates Echo's context system with the GraphQL tester, supporting:
 * - Echo context methods (c.JSON, c.QueryParam, c.Param, etc.)
 * - Echo middleware
 * - Echo's binder and validator
 * - Echo's reverse routing
 *
 * Usage:
 *   config := &Config{
 *       HTTPAdapter: &http.EchoAdapter{},
 *   }
 *
 * When to use:
 * - When your GraphQL server is built with Echo
 * - When resolvers depend on echo.Context
 * - When testing Echo-specific middleware (JWT, CORS, etc.)
 */

// EchoAdapter implements FrameworkAdapter for the Echo framework.
type EchoAdapter struct {
	// echo is the Echo instance used for routing.
	echo *echo.Echo

	// server is the test server instance.
	server *httptest.Server

	// debug enables Echo debug mode.
	debug bool
}

/**
 * NewEchoAdapter creates a new Echo adapter.
 *
 * Parameters:
 *   debug - If true, enables Echo's debug mode for verbose logging
 *
 * Returns:
 *   *EchoAdapter configured for testing
 *
 * Example:
 *   adapter := NewEchoAdapter(false) // Production-like test mode
 *   adapter := NewEchoAdapter(true)  // Debug mode for troubleshooting
 */
func NewEchoAdapter(debug bool) *EchoAdapter {
	e := echo.New()
	e.Debug = debug
	e.HideBanner = true // Hide Echo banner in tests
	e.HidePort = true   // Hide port information

	return &EchoAdapter{
		echo:  e,
		debug: debug,
	}
}

/**
 * Setup creates a test server with Echo as the HTTP framework.
 *
 * This method:
 * 1. Configures Echo routing for the GraphQL endpoint
 * 2. Wraps the GraphQL handler for Echo compatibility
 * 3. Starts an httptest server
 *
 * Parameters:
 *   handler - The GraphQL http.Handler to serve
 *
 * Returns:
 *   *httptest.Server ready to accept requests
 */
func (a *EchoAdapter) Setup(handler http.Handler) *httptest.Server {
	// Configure Echo to handle GraphQL requests
	// Echo's WrapHandler converts http.Handler to echo.HandlerFunc
	a.echo.POST("/graphql", echo.WrapHandler(handler))
	a.echo.GET("/graphql", echo.WrapHandler(handler))

	// Create test server
	a.server = httptest.NewServer(a.echo)

	return a.server
}

/**
 * CreateContext creates an Echo-specific context from an HTTP request.
 *
 * This creates a new echo.Context that wraps the standard http request
 * and response, allowing resolvers to use Echo-specific methods.
 *
 * Parameters:
 *   req - The HTTP request to wrap
 *
 * Returns:
 *   interface{} containing an echo.Context
 *
 * Note:
 *   The returned context is a basic echo.Context. For full Echo
 *   integration, consider using Echo's test utilities directly.
 */
func (a *EchoAdapter) CreateContext(req *http.Request) interface{} {
	// Create a new Echo context
	e := echo.New()
	w := httptest.NewRecorder()
	c := e.NewContext(req, w)

	return c
}

/**
 * AddMiddleware applies Echo-specific middleware to the handler.
 *
 * Echo middleware uses echo.MiddlewareFunc. This method registers
 * middleware with the Echo instance for proper execution order.
 *
 * Parameters:
 *   handler     - The base http.Handler
 *   middlewares - Echo middleware functions to apply
 *
 * Returns:
 *   http.Handler with Echo middleware applied
 *
 * Example:
 *   handler := adapter.AddMiddleware(
 *       myHandler,
 *       echoMiddleware.Logger(),
 *       echoMiddleware.Recover(),
 *       echoMiddleware.CORS(),
 *   )
 */
func (a *EchoAdapter) AddMiddleware(handler http.Handler, middlewares ...interface{}) http.Handler {
	// Register middlewares with Echo
	for _, mw := range middlewares {
		if echoMW, ok := mw.(echo.MiddlewareFunc); ok {
			a.echo.Use(echoMW)
		}
	}

	// Re-setup routes with middleware applied
	a.echo.POST("/graphql", echo.WrapHandler(handler))
	a.echo.GET("/graphql", echo.WrapHandler(handler))

	return a.echo
}

/**
 * Name returns the adapter name for debugging.
 *
 * Returns:
 *   string "echo"
 */
func (a *EchoAdapter) Name() string {
	return "echo"
}

/**
 * Close shuts down the test server and releases resources.
 */
func (a *EchoAdapter) Close() {
	if a.server != nil {
		a.server.Close()
	}
}

/**
 * URL returns the base URL of the test server.
 *
 * Returns:
 *   string URL of the test server
 */
func (a *EchoAdapter) URL() string {
	if a.server != nil {
		return a.server.URL
	}
	return ""
}

/**
 * Echo returns the Echo instance for direct configuration.
 *
 * Use this to add custom routes, middleware, or configuration
 * to the Echo instance before starting the server.
 *
 * Returns:
 *   *echo.Echo instance
 *
 * Example:
 *   adapter.Echo().Use(someMiddleware)
 *   adapter.Echo().Validator = &CustomValidator{}
 */
func (a *EchoAdapter) Echo() *echo.Echo {
	return a.echo
}

/**
 * SetDebug enables or disables Echo debug mode.
 *
 * Parameters:
 *   debug - Whether to enable debug mode
 */
func (a *EchoAdapter) SetDebug(debug bool) {
	a.debug = debug
	a.echo.Debug = debug
}
