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
 * integrates Gin's context system with the GraphQL tester.
 *
 * Usage:
 *   config := &Config{
 *       HTTPAdapter: http.NewGinAdapter(),
 *   }
 */
type GinAdapter struct {
	engine *gin.Engine
	server *httptest.Server
	mode   string
}

/**
 * NewGinAdapter creates a new Gin adapter with test mode enabled.
 *
 * Gin's test mode disables debug logging and provides better error
 * messages suitable for testing.
 *
 * Returns:
 *   *GinAdapter configured for testing
 */
func NewGinAdapter() *GinAdapter {
	gin.SetMode(gin.TestMode)
	return &GinAdapter{
		mode: gin.TestMode,
	}
}

/**
 * Setup creates a test server with Gin as the HTTP framework.
 *
 * Parameters:
 *   handler - The GraphQL http.Handler to serve
 *
 * Returns:
 *   *httptest.Server ready to accept requests
 */
func (a *GinAdapter) Setup(handler http.Handler) *httptest.Server {
	// Create a new Gin engine
	a.engine = gin.New()

	// Add recovery middleware for better error messages
	a.engine.Use(gin.Recovery())

	// Register the GraphQL endpoint - use Any() which handles GET, POST, etc.
	// Do NOT register POST separately as Any() already covers it
	a.engine.Any("/graphql", func(c *gin.Context) {
		gin.WrapH(handler)(c)
	})

	// Create test server
	a.server = httptest.NewServer(a.engine)

	return a.server
}

/**
 * CreateContext creates a Gin-specific context from an HTTP request.
 *
 * Parameters:
 *   req - The HTTP request
 *
 * Returns:
 *   interface{} containing a *gin.Context
 */
func (a *GinAdapter) CreateContext(req *http.Request) interface{} {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	return c
}

/**
 * AddMiddleware applies Gin-specific middleware to the handler.
 *
 * Parameters:
 *   handler     - The base http.Handler
 *   middlewares - Gin middleware functions to apply
 *
 * Returns:
 *   http.Handler with Gin middleware applied
 */
func (a *GinAdapter) AddMiddleware(handler http.Handler, middlewares ...interface{}) http.Handler {
	if a.engine == nil {
		a.engine = gin.New()
	}

	for _, mw := range middlewares {
		if ginMW, ok := mw.(gin.HandlerFunc); ok {
			a.engine.Use(ginMW)
		}
	}

	wrapped := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ginCtx := gin.CreateTestContextOnly(w, a.engine)
		ginCtx.Request = r
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
 * Returns:
 *   *gin.Engine instance
 */
func (a *GinAdapter) Engine() *gin.Engine {
	return a.engine
}

/**
 * SetMode changes the Gin mode.
 *
 * Parameters:
 *   mode - The Gin mode to set
 */
func (a *GinAdapter) SetMode(mode string) {
	a.mode = mode
	gin.SetMode(mode)
}
