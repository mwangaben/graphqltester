package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

/**
 * Context Propagation Middleware
 *
 * This middleware is responsible for propagating test context values
 * into HTTP request contexts. It ensures that downstream handlers
 * and resolvers have access to:
 * - Request ID for tracing
 * - Authenticated user information
 * - Tenant context for multi-tenancy
 * - Database connections
 * - Active transactions
 * - Test-specific custom values
 *
 * Context Keys:
 *   RequestIDKey        - Unique request identifier
 *   RequestStartTimeKey - Request start timestamp
 *   TestNameKey         - Current test name
 *   UserKey             - Authenticated user object
 *   UserIDKey           - Authenticated user ID
 *   PermissionsListKey  - User permissions
 *   RolesListKey        - User roles
 *   DatabaseKey         - Database adapter instance
 *   TransactionKey      - Active database transaction
 *   TenantIDKey         - Current tenant identifier
 */

// ============================================================================
// Context Keys
// ============================================================================

// contextKey is a private type for context keys to prevent collisions.
type contextKey string

const (
	// RequestIDKey stores the unique request identifier.
	// Value type: string
	RequestIDKey contextKey = "request_id"

	// RequestStartTimeKey stores the request start timestamp.
	// Value type: time.Time
	RequestStartTimeKey contextKey = "request_start_time"

	// TestNameKey stores the name of the currently running test.
	// Value type: string
	TestNameKey contextKey = "test_name"

	// UserKey stores the authenticated user object.
	// Value type: interface{} (user model)
	UserKey contextKey = "user"

	// UserIDKey stores the authenticated user's ID.
	// Value type: string
	UserIDKey contextKey = "user_id"

	// AuthStatusKey stores the authentication status.
	// Value type: AuthStatus
	AuthStatusKey contextKey = "auth_status"

	// PermissionsListKey stores the user's permissions.
	// Value type: []string
	PermissionsListKey contextKey = "permissions_list"

	// RolesListKey stores the user's roles.
	// Value type: []string
	RolesListKey contextKey = "roles_list"

	// DatabaseKey stores the database adapter instance.
	// Value type: DatabaseAdapter
	DatabaseKey contextKey = "database"

	// TransactionKey stores the active database transaction.
	// Value type: interface{} (transaction object)
	TransactionKey contextKey = "transaction"

	// TenantIDKey stores the current tenant identifier.
	// Value type: string
	TenantIDKey contextKey = "tenant_id"

	// ResponseStatusCodeKey stores the response status code.
	// Value type: int
	ResponseStatusCodeKey contextKey = "response_status_code"

	// ResponseHeadersKey stores the response headers.
	// Value type: http.Header
	ResponseHeadersKey contextKey = "response_headers"
)

// ============================================================================
// Middleware Functions
// ============================================================================

/**
 * ContextPropagationMiddleware propagates test context values into HTTP requests.
 *
 * This middleware extracts state from the tester (user, tenant, database, etc.)
 * and injects it into the request context so that GraphQL resolvers can access
 * it during query execution.
 *
 * The tester parameter should provide the following interface:
 *   - currentUser: The authenticated user
 *   - currentToken: The authentication token
 *   - tenant: The current tenant context
 *   - dbAdapter: The database adapter
 *   - txManager: The transaction manager
 *   - context: The base context
 *   - customContext: Map of custom context values
 *
 * Parameters:
 *   tester - The tester instance providing test state
 *
 * Returns:
 *   Middleware function
 */
func ContextPropagationMiddleware(tester interface{}) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Start with a fresh context or the tester's base context
			ctx := r.Context()

			// Add request ID for tracing
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = generateRequestID()
			}
			ctx = context.WithValue(ctx, RequestIDKey, requestID)

			// Add request start time for performance tracking
			ctx = context.WithValue(ctx, RequestStartTimeKey, time.Now())

			// Set response headers
			w.Header().Set("X-Request-ID", requestID)

			// Propagate tester context using reflection or interface assertions
			// This avoids tight coupling to the Tester type
			if ctxTester, ok := tester.(interface {
				GetContext() context.Context
				GetCurrentUser() interface{}
				GetCurrentToken() string
			}); ok {
				// Propagate base context
				if baseCtx := ctxTester.GetContext(); baseCtx != nil {
					// Merge base context values
					ctx = mergeContexts(ctx, baseCtx)
				}

				// Propagate user information
				if user := ctxTester.GetCurrentUser(); user != nil {
					ctx = context.WithValue(ctx, UserKey, user)

					// Try to extract user ID
					if userWithID, ok := user.(interface{ GetID() string }); ok {
						ctx = context.WithValue(ctx, UserIDKey, userWithID.GetID())
					}
				}
			}

			// Propagate test name
			if testerWithName, ok := tester.(interface{ TestName() string }); ok {
				ctx = context.WithValue(ctx, TestNameKey, testerWithName.TestName())
			}

			// Pass the enriched context to the next handler
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

/**
 * ResponseContextMiddleware captures response information into the context.
 *
 * This middleware wraps the response writer to capture:
 * - HTTP status code
 * - Response headers
 *
 * The captured information is stored in the request context and can be
 * used for assertions after the response is written.
 *
 * Parameters:
 *   tester - The tester instance
 *
 * Returns:
 *   Middleware function
 */
func ResponseContextMiddleware(tester interface{}) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Wrap the response writer to capture status and headers
			rw := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK, // Default to 200
			}

			// Process the request
			next.ServeHTTP(rw, r)

			// Store response info in context for post-processing
			ctx := r.Context()
			ctx = context.WithValue(ctx, ResponseStatusCodeKey, rw.statusCode)
			ctx = context.WithValue(ctx, ResponseHeadersKey, rw.Header())

			// Note: The context is updated but the response is already sent
			// This is primarily for logging/metrics, not for modifying the response
		})
	}
}

/**
 * RequestIDMiddleware adds a unique request ID to each request.
 *
 * If the request already has an X-Request-ID header, it is used as-is.
 * Otherwise, a new UUID-like ID is generated.
 *
 * Returns:
 *   Middleware function
 */
func RequestIDMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = generateRequestID()
			}

			// Add to both header and context
			w.Header().Set("X-Request-ID", requestID)
			ctx := context.WithValue(r.Context(), RequestIDKey, requestID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ============================================================================
// Helper Types and Functions
// ============================================================================

/**
 * responseWriter wraps http.ResponseWriter to capture the status code.
 *
 * The status code is captured when WriteHeader is called, which typically
 * happens once per response. The first call to WriteHeader sets the status;
 * subsequent calls are ignored (per HTTP spec).
 */
type responseWriter struct {
	http.ResponseWriter
	statusCode  int
	wroteHeader bool
}

/**
 * WriteHeader captures the status code and delegates to the wrapped writer.
 *
 * Only the first call to WriteHeader is honored, matching the behavior
 * of the standard library's http.ResponseWriter.
 *
 * Parameters:
 *   code - HTTP status code to write
 */
func (rw *responseWriter) WriteHeader(code int) {
	if !rw.wroteHeader {
		rw.statusCode = code
		rw.wroteHeader = true
	}
	rw.ResponseWriter.WriteHeader(code)
}

/**
 * Write captures the default 200 status if WriteHeader hasn't been called.
 *
 * Per HTTP spec, if Write is called before WriteHeader, the status defaults
 * to 200 OK.
 *
 * Parameters:
 *   b - Bytes to write to the response
 *
 * Returns:
 *   int number of bytes written, error if any
 */
func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.statusCode = http.StatusOK
		rw.wroteHeader = true
	}
	return rw.ResponseWriter.Write(b)
}

/**
 * StatusCode returns the captured HTTP status code.
 *
 * Returns:
 *   int HTTP status code
 */
func (rw *responseWriter) StatusCode() int {
	return rw.statusCode
}

/**
 * generateRequestID generates a unique request identifier.
 *
 * The ID format is: timestamp-randomhex
 * Example: "1702123456789-a1b2c3d4"
 *
 * Returns:
 *   string unique request ID
 */
func generateRequestID() string {
	return fmt.Sprintf("%d-%s",
		time.Now().UnixNano(),
		randomHexString(8),
	)
}

/**
 * randomHexString generates a random hexadecimal string.
 *
 * This is a simple implementation for generating unique identifiers.
 * For production use, consider using crypto/rand.
 *
 * Parameters:
 *   length - Desired length of the hex string
 *
 * Returns:
 *   string random hex characters
 */
func randomHexString(length int) string {
	const hexChars = "0123456789abcdef"
	b := make([]byte, length)
	for i := range b {
		b[i] = hexChars[time.Now().UnixNano()%int64(len(hexChars))]
		time.Sleep(1) // Ensure different values for each character
	}
	return string(b)
}

/**
 * mergeContexts merges values from a source context into a destination context.
 *
 * This is a best-effort merge. It copies known context keys from the source
 * to the destination. Unknown keys in the source are ignored.
 *
 * Parameters:
 *   dest - Destination context to merge into
 *   src  - Source context to merge from
 *
 * Returns:
 *   context.Context with merged values
 */
func mergeContexts(dest, src context.Context) context.Context {
	// Copy known context keys
	knownKeys := []contextKey{
		UserKey,
		UserIDKey,
		TenantIDKey,
		DatabaseKey,
		TransactionKey,
	}

	for _, key := range knownKeys {
		if val := src.Value(key); val != nil {
			dest = context.WithValue(dest, key, val)
		}
	}

	return dest
}
