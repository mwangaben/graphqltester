package types

//ContextKey
/**
 * ContextKey is a private type for context keys to prevent collisions.
 */
type ContextKey string

const (
	// RequestIDKey stores the unique request identifier
	RequestIDKey ContextKey = "request_id"

	// UserKey stores the authenticated user object
	UserKey ContextKey = "user"

	// UserIDKey stores the authenticated user's ID
	UserIDKey ContextKey = "user_id"

	// AuthStatusKey stores the authentication status
	AuthStatusKey ContextKey = "auth_status"

	// TenantIDKey stores the current tenant identifier
	TenantIDKey ContextKey = "tenant_id"

	// RequiredPermissionKey stores the required permission
	RequiredPermissionKey ContextKey = "required_permission"

	// PermissionStatusKey stores whether permission was granted
	PermissionStatusKey ContextKey = "permission_status"

	// PermissionErrorKey stores permission error message
	PermissionErrorKey ContextKey = "permission_error"

	// ValidationErrorsKey stores validation errors
	ValidationErrorsKey ContextKey = "validation_errors"

	// ResponseStatusCodeKey stores the response status code
	ResponseStatusCodeKey ContextKey = "response_status_code"

	// ResponseHeadersKey stores the response headers
	ResponseHeadersKey ContextKey = "response_headers"

	// PermissionsListKey stores user permissions
	PermissionsListKey ContextKey = "permissions_list"

	// RolesListKey stores user roles
	RolesListKey ContextKey = "roles_list"

	// DatabaseKey stores the database adapter
	DatabaseKey ContextKey = "database"

	// TransactionKey stores the active transaction
	TransactionKey ContextKey = "transaction"
)

//AuthStatus
/**
 * AuthStatus represents the authentication state.
 */
type AuthStatus int

const (
	// AuthStatusNone indicates no authentication
	AuthStatusNone AuthStatus = iota

	// AuthStatusAuthenticated indicates successful authentication
	AuthStatusAuthenticated

	// AuthStatusInvalid indicates failed authentication
	AuthStatusInvalid
)
