package middleware

import (
	"context"
	"net/http"
	"strings"
)

/**
 * PermissionMiddleware provides authorization/permission checking middleware.
 *
 * This middleware checks if the authenticated user has the required permissions
 * to execute the requested GraphQL operation. It works in conjunction with
 * the SchemaAnalysisMiddleware to determine what permissions are required.
 */

// PermissionMiddleware checks if the authenticated user has required permissions.
//
// Parameters:
//
//	tester - The tester instance (must support permission checking)
//
// Returns:
//
//	Middleware function for the chain
func PermissionMiddleware(tester interface{}) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only check permissions if user is authenticated
			authStatus, _ := r.Context().Value(contextKey("auth_status")).(string)
			if authStatus != "authenticated" {
				next.ServeHTTP(w, r)
				return
			}

			// Get the required permission from context (set by SchemaAnalysisMiddleware)
			requiredPermission, _ := r.Context().Value(contextKey("required_permission")).(string)

			if requiredPermission != "" {
				// Check if the tester/user has the required permission
				if testerWithPerm, ok := tester.(interface{ HasPermission(string) bool }); ok {
					hasPermission := testerWithPerm.HasPermission(requiredPermission)

					ctx := context.WithValue(r.Context(), contextKey("permission_status"), hasPermission)
					if !hasPermission {
						ctx = context.WithValue(ctx, contextKey("permission_error"), "This action is unauthorized.")
					}
					r = r.WithContext(ctx)
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// SchemaAnalysisMiddleware analyzes the GraphQL query to determine required permissions.
//
// Parameters:
//
//	tester - The tester instance
//
// Returns:
//
//	Middleware function for the chain
func SchemaAnalysisMiddleware(tester interface{}) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Analyze the request to determine required permissions
			// This is a simplified implementation that looks at the operation name
			// In production, you'd parse the GraphQL query to determine permissions

			requiredPerm := analyzeRequiredPermission(r)

			if requiredPerm != "" {
				ctx := context.WithValue(r.Context(), contextKey("required_permission"), requiredPerm)
				r = r.WithContext(ctx)
			}

			next.ServeHTTP(w, r)
		})
	}
}

// analyzeRequiredPermission determines the required permission based on the request.
func analyzeRequiredPermission(r *http.Request) string {
	// This is a simplified implementation
	// In production, you would parse the GraphQL query to determine permissions

	// Check for permission-related headers
	if perm := r.Header.Get("X-Required-Permission"); perm != "" {
		return perm
	}

	// Check URL path for hints
	path := r.URL.Path
	if strings.Contains(path, "create") {
		return extractResourceFromPath(path) + ".create"
	}
	if strings.Contains(path, "update") {
		return extractResourceFromPath(path) + ".update"
	}
	if strings.Contains(path, "delete") {
		return extractResourceFromPath(path) + ".delete"
	}
	if strings.Contains(path, "view") || strings.Contains(path, "list") {
		return extractResourceFromPath(path) + ".view"
	}

	return ""
}

// extractResourceFromPath extracts the resource name from a URL path.
func extractResourceFromPath(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return "unknown"
}
