package http

import (
	"net/http"
	"net/http/httptest"

	"github.com/go-chi/chi/v5"
)

/**
 * ChiAdapter provides GraphQL testing support for the Chi router.
 *
 * Chi is a lightweight, idiomatic Go router that builds on net/http.
 * This adapter integrates Chi's routing and middleware system with
 * the GraphQL tester, supporting:
 * - Chi's route parameters and context
 * - Chi middleware (compatible with standard http.Handler)
 * - Chi's middleware chaining (Use, With, Route, Group)
 * - URL parameter extraction via chi.URLParam
 *
 * Usage:
 *   config := &Config{
 *       HTTPAdapter: &http.ChiAdapter{},
 *   }
 *
 * When to use:
 * - When your GraphQL server uses Chi for routing
 * - When you need Chi-specific middleware features
 * - When your resolvers use chi.URLParam or chi.RouteContext
 */

// ChiAdapter implements FrameworkAdapter for the Chi router.
type ChiAdapter struct {
	// router is the Chi router instance.
	router chi.Router

	// server is the test server instance.
	server *httptest.Server

	// middlewares stores middleware to be applied.
	middlewares []func(http.Handler) http.Handler
}

/*******
 * NewChiAdapter creates a new Chi adapter.
 *
 * Returns:
 **ChiAdapter with a new Chi router
 *
 * Example:
 *   adapter := NewChiAdapter()
 *   config.HTTPAdapter = adapter
 */
func NewChiAdapter() *ChiAdapter {
	return &ChiAdapter{
		router:      chi.NewRouter(),
		middlewares: make([]func(http.Handler) http.Handler, 0),
	}
}

//Setup
/**
 * Setup creates a test server with Chi as the router.
 *
 * This method:
 * 1. Configures Chi routing for the GraphQL endpoint
 * 2. Applies any registered middleware
 * 3. Creates an httptest server
 *
 * Parameters:
 *   handler - The GraphQL http.Handler to serve
 *
 * Returns:
 *   *httptest.Server ready to accept requests
 */
func (a *ChiAdapter) Setup(handler http.Handler) *httptest.Server {
	// Apply stored middleware to the router
	for _, mw := range a.middlewares {
		a.router.Use(mw)
	}

	// Route GraphQL requests
	a.router.Handle("/graphql", handler)
	a.router.Post("/graphql", handler.ServeHTTP)
	a.router.Get("/graphql", handler.ServeHTTP)

	// Create test server
	a.server = httptest.NewServer(a.router)

	return a.server
}

/**
 * CreateContext creates a Chi-specific context from an HTTP request.
 *
 * Chi uses standard context.Context with Chi-specific values stored
 * using context.WithValue. This method returns the request's context
 * which already contains Chi's routing information.
 *
 * Parameters:
 *   req - The HTTP request containing Chi context
 *
 * Returns:
 *   interface{} containing the request context
 */
func (a *ChiAdapter) CreateContext(req *http.Request) interface{} {
	return req.Context()
}

/**
 * AddMiddleware applies Chi-compatible middleware to the handler.
 *
 * Chi middleware is standard http.Handler middleware, making it
 * compatible with the standard library pattern. This method stores
 * middleware for application during Setup.
 *
 * Parameters:
 *   handler     - The base http.Handler
 *   middlewares - Middleware functions to apply
 *
 * Returns:
 *   http.Handler with middleware applied
 *
 * Example:
 *   handler := adapter.AddMiddleware(
 *       myHandler,
 *       chiMiddleware.Logger,
 *       chiMiddleware.Recoverer,
 *       chiMiddleware.RequestID,
 *   )
 */
func (a *ChiAdapter) AddMiddleware(handler http.Handler, middlewares ...interface{}) http.Handler {
	for _, mw := range middlewares {
		if chiMW, ok := mw.(func(http.Handler) http.Handler); ok {
			a.middlewares = append(a.middlewares, chiMW)
		}
	}

	// Apply middleware to the handler directly
	wrapped := handler
	for i := len(a.middlewares) - 1; i >= 0; i-- {
		wrapped = a.middlewares[i](wrapped)
	}

	return wrapped
}

/**
 * Name returns the adapter name for debugging.
 *
 * Returns:
 *   string "chi"
 */
func (a *ChiAdapter) Name() string {
	return "chi"
}

/**
 * Close shuts down the test server and releases resources.
 */
func (a *ChiAdapter) Close() {
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
func (a *ChiAdapter) URL() string {
	if a.server != nil {
		return a.server.URL
	}
	return ""
}

/**
 * Router returns the Chi router for direct configuration.
 *
 * Use this to add custom routes, middleware, or sub-routers
 * to the Chi instance before starting the server.
 *
 * Returns:
 *   chi.Router instance
 *
 * Example:
 *   adapter.Router().Use(middleware.Logger)
 *   adapter.Router().Route("/api", func(r chi.Router) {
 *       r.Post("/graphql", handler)
 *   })
 */
func (a *ChiAdapter) Router() chi.Router {
	return a.router
}

/**
 * Group creates a new Chi route group with shared middleware.
 *
 * This follows Chi's pattern for creating route groups with
 * common middleware applied to all routes in the group.
 *
 * Parameters:
 *   fn - Function to configure the route group
 *
 * Returns:
 *   *ChiAdapter for fluent method chaining
 *
 * Example:
 *   adapter.Group(func(r chi.Router) {
 *       r.Use(authMiddleware)
 *       r.Post("/graphql", handler)
 *   })
 */
func (a *ChiAdapter) Group(fn func(chi.Router)) *ChiAdapter {
	a.router.Group(fn)
	return a
}

/**
 * Route creates a new Chi route with specific middleware.
 *
 * Parameters:
 *   pattern     - URL pattern for the route
 *   fn          - Function to configure sub-routes
 *
 * Returns:
 *   *ChiAdapter for fluent method chaining
 *
 * Example:
 *   adapter.Route("/graphql", func(r chi.Router) {
 *       r.Use(graphqlMiddleware)
 *       r.Post("/", handler)
 *   })
 */
func (a *ChiAdapter) Route(pattern string, fn func(chi.Router)) *ChiAdapter {
	a.router.Route(pattern, fn)
	return a
}

// Ensure adapters implement FrameworkAdapter interface
var (
	_ FrameworkAdapter = (*NetHTTPAdapter)(nil)
	_ FrameworkAdapter = (*GinAdapter)(nil)
	_ FrameworkAdapter = (*EchoAdapter)(nil)
	_ FrameworkAdapter = (*ChiAdapter)(nil)
)
