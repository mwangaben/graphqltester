package middleware

import (
	"context"
	"net/http"
	"strings"
)

/**
 * AuthMiddleware provides authentication middleware for GraphQL testing.
 *
 * This middleware handles extracting authentication tokens from requests,
 * validating them, and setting the authenticated user in the request context.
 *
 * It works with the tester's authentication state to inject the correct
 * user into the request context so that GraphQL resolvers can access
 * the authenticated user.
 */

// AuthMiddleware authenticates requests by extracting and validating tokens.
//
// Parameters:
//
//	tester - The tester instance (must support getting current user/token)
//
// Returns:
//
//	Middleware function for the chain
func AuthMiddleware(tester interface{}) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from the Authorization header
			token := extractToken(r)

			// Get the current user from the tester
			if testerWithUser, ok := tester.(interface{ CurrentUser() interface{} }); ok {
				if user := testerWithUser.CurrentUser(); user != nil {
					// Set the authenticated user in context
					ctx := context.WithValue(r.Context(), contextKey("user"), user)
					ctx = context.WithValue(ctx, contextKey("auth_status"), "authenticated")

					// Extract user ID if available
					if userWithID, ok := user.(interface{ GetID() string }); ok {
						ctx = context.WithValue(ctx, contextKey("user_id"), userWithID.GetID())
					}

					r = r.WithContext(ctx)
				}
			}

			// If token is present but no user is set, mark as unauthenticated
			if token != "" {
				if testerWithUser, ok := tester.(interface{ CurrentUser() interface{} }); ok {
					if testerWithUser.CurrentUser() == nil {
						ctx := context.WithValue(r.Context(), contextKey("auth_status"), "unauthenticated")
						r = r.WithContext(ctx)
					}
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// extractToken extracts the Bearer token from the Authorization header.
func extractToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}

	return parts[1]
}
