package tests

import (
	"github.com/go-chi/chi/v5"
	"net/http"
	"net/http/httptest"
	"testing"

	appHttp "github.com/mwangaben/graphqltester/pkg/adapters/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/**
 * TestNetHTTPAdapter_Setup_CreatesServer validates that the standard
 * library adapter creates a working test server.
 */
func TestNetHTTPAdapter_Setup_CreatesServer(t *testing.T) {
	adapter := &appHttp.NetHTTPAdapter{}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": {"message": "Hello, World!"}}`))
	})

	server := adapter.Setup(handler)
	defer server.Close()

	// Make a request to the server
	resp, err := http.Get(server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NotEmpty(t, adapter.URL())
	assert.Equal(t, "net/http", adapter.Name())
}

/**
 * TestNetHTTPAdapter_AddMiddleware_AppliesMiddleware validates that
 * middleware is correctly applied to the handler.
 */
func TestNetHTTPAdapter_AddMiddleware_AppliesMiddleware(t *testing.T) {
	adapter := &appHttp.NetHTTPAdapter{}

	baseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Base", "true")
		w.WriteHeader(http.StatusOK)
	})

	middleware1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-MW1", "true")
			next.ServeHTTP(w, r)
		})
	}

	middleware2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-MW2", "true")
			next.ServeHTTP(w, r)
		})
	}

	wrapped := adapter.AddMiddleware(baseHandler, middleware1, middleware2)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	wrapped.ServeHTTP(w, req)

	assert.Equal(t, "true", w.Header().Get("X-MW1"), "MW1 header should be set")
	assert.Equal(t, "true", w.Header().Get("X-MW2"), "MW2 header should be set")
	assert.Equal(t, "true", w.Header().Get("X-Base"), "Base header should be set")
}

/**
 * TestGinAdapter_Setup_CreatesGinServer validates that the Gin adapter
 * creates a working test server with Gin framework.
 */
func TestGinAdapter_Setup_CreatesGinServer(t *testing.T) {
	adapter := appHttp.NewGinAdapter()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	})

	server := adapter.Setup(handler)
	defer server.Close()

	// Make a POST request (Gin routes require specific methods)
	resp, err := http.Post(server.URL+"/graphql", "application/json", nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "gin", adapter.Name())
}

/**
 * TestEchoAdapter_Setup_CreatesEchoServer validates that the Echo adapter
 * creates a working test server with Echo framework.
 */
func TestEchoAdapter_Setup_CreatesEchoServer(t *testing.T) {
	adapter := appHttp.NewEchoAdapter(false)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	})

	server := adapter.Setup(handler)
	defer server.Close()

	resp, err := http.Post(server.URL+"/graphql", "application/json", nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "echo", adapter.Name())
}

/**
 * TestChiAdapter_Setup_CreatesChiServer validates that the Chi adapter
 * creates a working test server with Chi router.
 */
func TestChiAdapter_Setup_CreatesChiServer(t *testing.T) {
	adapter := appHttp.NewChiAdapter()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	})

	server := adapter.Setup(handler)
	defer server.Close()

	resp, err := http.Get(server.URL + "/graphql")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "chi", adapter.Name())
}

/**
 * TestChiAdapter_Group_CreatesRouteGroup validates that Chi route groups
 * work correctly for organizing routes.
 */
func TestChiAdapter_Group_CreatesRouteGroup(t *testing.T) {
	adapter := appHttp.NewChiAdapter()

	groupMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Group", "true")
			next.ServeHTTP(w, r)
		})
	}

	adapter.Group(func(r chi.Router) {
		r.Use(groupMiddleware)
		r.Get("/api/test", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}).ServeHTTP)
	})

	server := adapter.Setup(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/test")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, "true", resp.Header.Get("X-Group"), "Group middleware should be applied")
}

/**
 * TestFrameworkAdapter_InterfaceImplementation validates that all adapters
 * implement the FrameworkAdapter interface.
 */
func TestFrameworkAdapter_InterfaceImplementation(t *testing.T) {
	// These will fail at compile time if the interface isn't implemented
	var _ appHttp.FrameworkAdapter = &appHttp.NetHTTPAdapter{}
	var _ appHttp.FrameworkAdapter = &appHttp.GinAdapter{}
	var _ appHttp.FrameworkAdapter = &appHttp.EchoAdapter{}
	var _ appHttp.FrameworkAdapter = &appHttp.ChiAdapter{}

	// Test passes if we get here (compile-time check)
	assert.True(t, true)
}
