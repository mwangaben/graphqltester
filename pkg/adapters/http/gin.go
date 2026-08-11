package http

import (
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
)

/**
 * GinAdapter provides GraphQL testing support for the Gin web framework.
 *
 * Gin is a popular high-performance HTTP framework for Go. This adapter
 * integrates Gin's context system with the GraphQL tester, allowing tests
 * to leverage Gin-specific features like:
 * - Gin context methods (c.JSON, c.Query, c.Param, etc.)
 * - Gin middleware
 * - Gin request binding and validation
 *
 * Usage:
 *   config := &Config{
 *       HTTPAdapter: &http.GinAdapter{},
 *   }
 *
 * When to use:
 * - When your GraphQL server is built with Gin
 * - When your resolvers depend on gin.Context
 * - When you need to test Gin-specific middleware
 */

// GinAdapter implements FrameworkAdapter for the Gin framework.
type GinAdapter struct {
	// engine is the Gin engine instance used for routing.
	engine *gin.Engine

	// server is the test server instance.
	server *httptest.Server

	// mode stores the Gin mode (test, debug, release).
	mode string
}

/**
 * NewGinAdapter creates a new Gin adapter with test mode enabled.
 *
 * Gin's test mode disables debug logging and provides better error
 * messages suitable for testing.
 *
 * Returns:
 *   *GinAdapter configured for testing
 *
 * Example:
 *   adapter := NewGinAdapter()
 *   config.HTTPAdapter = adapter
 */
func NewGinAdapter() *GinAdapter {
	// Set Gin to test mode for better test output
	gin.SetMode(gin.TestMode)

	return &GinAdapter{
		mode: gin.TestMode,
	}
}

/**
 * Setup creates a test server with Gin as the HTTP framework.
 *
 * This method:
 * 1. Creates a new Gin engine in test mode
 * 2. Configures the engine to route all requests to /graphql
 * 3. Wraps the GraphQL handler with Gin's context
 * 4. Starts an httptest server
 *
 * Parameters:
 *   handler - The GraphQL http.Handler to serve
 *
 * Returns:
 *   *httptest.Server ready to accept requests
 *
 * Example:
 *   adapter := NewGinAdapter()
 *   server := adapter.Setup(myGraphQLHandler)
 *   defer server.Close()
 */
func (a *GinAdapter) Setup(handler http.Handler) *httptest.Server {
	// Create a new Gin engine
	a.engine = gin.New()

	// Add recovery middleware for better error messages
	a.engine.Use(gin.Recovery())

	// Route all requests to the GraphQL handler
	// Use Any to handle both GET and POST requests
	a.engine.Any("/graphql", func(c *gin.Context) {
		// Wrap the standard handler to work with Gin context
		gin.WrapH(handler)(c)
	})

	// Also handle the configured endpoint path
	a.engine.POST("/graphql", func(c *gin.Context) {
		gin.WrapH(handler)(c)
	})

	// Create test server
	a.server = httptest.NewServer(a.engine)

	return a.server
}

/**
 * CreateContext creates a Gin-specific context from an HTTP request.
 *
 * This allows resolvers to access gin.Context and use Gin-specific
 * methods like c.Query(), c.Param(), c.GetString(), etc.
 *
 * Parameters:
 *   req - The HTTP request (unused, context is created fresh)
 *
 * Returns:
 *   interface{} containing a *gin.Context
 *
 * Note:
 *   The returned gin.Context is a test context and may not have all
 *   production features. For full Gin integration, use Gin-specific
 *   test patterns.
 */
func (a *GinAdapter) CreateContext(req *http.Request) interface{} {
	// Create a test Gin context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	return c
}

/**
 * AddMiddleware applies Gin-specific middleware to the handler.
 *
 * Gin middleware uses gin.HandlerFunc instead of standard http middleware.
 * This method converts Gin middleware to work with http.Handler.
 *
 * Parameters:
 *   handler     - The base http.Handler
 *   middlewares - Gin middleware functions to apply
 *
 * Returns:
 *   http.Handler with Gin middleware applied
 *
 * Example:
 *   handler := adapter.AddMiddleware(
 *       myHandler,
 *       gin.Logger(),
 *       gin.Recovery(),
 *       customGinMiddleware,
 *   )
 */
func (a *GinAdapter) AddMiddleware(handler http.Handler, middlewares ...interface{}) http.Handler {
	if a.engine == nil {
		a.engine = gin.New()
	}

	// Add middlewares to the Gin engine
	for _, mw := range middlewares {
		if ginMW, ok := mw.(gin.HandlerFunc); ok {
			a.engine.Use(ginMW)
		}
	}

	// Re-setup the handler with the engine that now has middleware
	wrapped := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Create a fresh Gin context for this request
		ginCtx := gin.CreateTestContextOnly(w, a.engine)
		ginCtx.Request = r

		// Execute Gin middleware chain, then the handler
		a.engine.HandleContext(ginCtx)
	})

	return wrapped
}

/**
 * Name returns the adapter name for debugging.
 *
 * Returns:
 *   string "gin"
 */
func (a *GinAdapter) Name() string {
	return "gin"
}

/**
 * Close shuts down the test server and releases resources.
 */
func (a *GinAdapter) Close() {
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
func (a *GinAdapter) URL() string {
	if a.server != nil {
		return a.server.URL
	}
	return ""
}

/**
 * Engine returns the Gin engine for direct configuration.
 *
 * Use this to add custom routes, middleware, or configuration
 * to the Gin engine before starting the server.
 *
 * Returns:
 *   *gin.Engine instance
 */
func (a *GinAdapter) Engine() *gin.Engine {
	return a.engine
}

/**
 * SetMode changes the Gin mode.
 *
 * Available modes:
 * - gin.DebugMode: Verbose logging
 * - gin.ReleaseMode: Production mode
 * - gin.TestMode: Test-optimized mode (default)
 *
 * Parameters:
 *   mode - The Gin mode to set
 */
func (a *GinAdapter) SetMode(mode string) {
	a.mode = mode
	gin.SetMode(mode)
}
