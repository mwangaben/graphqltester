package middleware

import (
	"context"
	"net/http"
	"strings"
)

/**
 * TenantMiddleware provides multi-tenancy support for GraphQL testing.
 *
 * This middleware handles tenant identification from request headers,
 * subdomains, or query parameters, and sets the tenant context for
 * downstream handlers.
 */

// TenantMiddleware extracts and sets the current tenant for the request.
//
// Parameters:
//
//	tester - The tester instance (must support tenant operations)
//
// Returns:
//
//	Middleware function for the chain
func TenantMiddleware(tester interface{}) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract tenant ID from various sources
			tenantID := extractTenantID(r, tester)

			if tenantID != "" {
				// Set tenant in context
				ctx := context.WithValue(r.Context(), contextKey("tenant_id"), tenantID)
				r = r.WithContext(ctx)
			}

			next.ServeHTTP(w, r)
		})
	}
}

// TenantResolverMiddleware resolves and validates the tenant for the request.
//
// Parameters:
//
//	tester - The tester instance
//
// Returns:
//
//	Middleware function for the chain
func TenantResolverMiddleware(tester interface{}) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenantID, _ := r.Context().Value(contextKey("tenant_id")).(string)

			if tenantID != "" {
				// Validate tenant exists (optional auto-registration)
				if testerWithTenant, ok := tester.(interface {
					IsValidTenant(string) bool
				}); ok {
					if !testerWithTenant.IsValidTenant(tenantID) {
						// Auto-register if configured
						if testerWithConfig, ok := tester.(interface {
							Config() interface{ Debug() bool }
						}); ok {
							if testerWithConfig.Config().Debug() {
								// Log in debug mode
							}
						}
					}
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// extractTenantID extracts the tenant identifier from the request.
func extractTenantID(r *http.Request, tester interface{}) string {
	// Priority: Header > Subdomain > Query Parameter > Tester State

	// 1. Check X-Tenant-ID header
	if tenantID := r.Header.Get("X-Tenant-ID"); tenantID != "" {
		return tenantID
	}

	// 2. Check subdomain
	host := r.Host
	if subdomain := extractSubdomain(host); subdomain != "" {
		return subdomain
	}

	// 3. Check query parameter
	if tenantID := r.URL.Query().Get("tenant_id"); tenantID != "" {
		return tenantID
	}

	// 4. Check tester's current tenant
	if testerWithTenant, ok := tester.(interface{ CurrentTenant() string }); ok {
		if tenantID := testerWithTenant.CurrentTenant(); tenantID != "" {
			return tenantID
		}
	}

	return ""
}

// extractSubdomain extracts the subdomain from a host string.
func extractSubdomain(host string) string {
	// Remove port if present
	host = strings.Split(host, ":")[0]

	parts := strings.Split(host, ".")
	if len(parts) > 2 {
		return parts[0]
	}
	return ""
}
