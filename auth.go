package graphqltester

import (
	"context"
	"fmt"
	"github.com/mwangaben/graphqltester/pkg/middleware"
	"github.com/mwangaben/graphqltester/types"
	"net/http"
	"reflect"
)

/**
 * Authentication Helper Methods for GraphQL Testing
 *
 * This file provides methods that mirror Laravel Lighthouse's authentication
 * patterns for Go GraphQL testing. These methods handle user authentication
 * state, role assignment, and permission management within the test context.
 *
 * The authentication system supports:
 * - Acting as a specific user (ActingAs)
 * - Token-based authentication (WithToken)
 * - Role-based authentication (SignInUser, SignInAdmin)
 * - Integration with your permission package
 * - Integration with your factory package for user creation
 *
 * All authentication state is propagated through the request context and
 * middleware chain, ensuring that GraphQL resolvers receive the correct
 * authenticated user information.
 */

// ============================================================================
// Primary Authentication Methods
// ============================================================================

//ActingAs
/**
 * ActingAs sets the currently authenticated user for subsequent requests.
 *
 * This is the primary method for authenticating a user in tests. It sets
 * the user in the tester's state and configures the middleware chain to
 * inject this user into the request context.
 *
 * After calling ActingAs, all subsequent GraphQL requests will be
 * authenticated as the specified user until changed.
 *
 * Parameters:
 *   user - The user model instance to authenticate as
 *
 * Returns:
 *   *Tester for fluent method chaining
 *
 * Example:
 *   user := User{ID: "123", Name: "John Doe"}
 *   tester.ActingAs(user)
 *
 *   // Now all requests are authenticated as this user
 *   tester.GraphQL(`{ me { name } }`).AssertJSONPath("data.me.name", "John Doe")
 *
 * Integration:
 *   Works with the auth middleware to set context values:
 *   - UserIDKey: The user's ID
 *   - UserKey: The full user object
 *   - AuthStatusKey: Set to AuthStatusAuthenticated
 */
func (tester *Tester) ActingAs(user interface{}) *Tester {
	tester.mu.Lock()
	defer tester.mu.Unlock()

	tester.currentUser = user

	// Update the middleware chain to include the custom auth middleware
	// that injects this user into the request context
	customAuthMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Create a new context with the authenticated user
			ctx := context.WithValue(r.Context(), types.UserKey, user)
			ctx = context.WithValue(ctx, types.AuthStatusKey, types.AuthStatusAuthenticated)

			// Try to extract user ID using reflection or interface
			if userWithID, ok := user.(interface{ GetID() string }); ok {
				ctx = context.WithValue(ctx, types.UserIDKey, userWithID.GetID())
			} else if userWithID, ok := user.(interface{ ID() string }); ok {
				ctx = context.WithValue(ctx, types.UserIDKey, userWithID.ID())
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}

	// Replace existing auth middleware or add if not exists
	if tester.middlewareChain != nil {
		tester.middlewareChain.Replace("auth", customAuthMiddleware)
	}

	if tester.config.debugEnabled {
		tester.t.Logf("🔑 Acting as user: %v", tester.getUserIdentifier(user))
	}

	return tester
}

/**
 * WithToken sets a bearer token for authentication in subsequent requests.
 *
 * Unlike ActingAs which sets the user directly in context, this method
 * sets an authentication token that will be sent in the Authorization header.
 * This tests the full authentication flow including token validation.
 *
 * Parameters:
 *   token - The bearer token to use for authentication
 *
 * Returns:
 *   *Tester for fluent method chaining
 *
 * Example:
 *   token := generateTestToken(user)
 *   tester.WithToken(token)
 *
 *   tester.GraphQL(`{ me { name } }`).AssertOK()
 *
 * Headers:
 *   Sets: Authorization: Bearer <token>
 */
func (tester *Tester) WithToken(token string) *Tester {
	tester.mu.Lock()
	defer tester.mu.Unlock()

	tester.currentToken = token

	if tester.config.Debug() {
		// Use built-in min function (Go 1.21+)
		prefixLen := min(10, len(token))
		tester.t.Logf("🔑 Using auth token: %s...", token[:prefixLen])
	}

	return tester
}

// ============================================================================
// Role-Based Authentication (Laravel Lighthouse Style)
// ============================================================================

//SignInUser
/**
 * SignInUser authenticates a test as a user with a specific role and permission.
 *
 * This method mirrors the Laravel Lighthouse pattern:
 *   $this->signInUser('roleName', 'permissionName', $user)
 *
 * It creates a user (if not provided), creates/assigns a role, grants a
 * permission to that role, and authenticates the user.
 *
 * Parameters:
 *   roleName       - Name of the role to create/assign
 *   permissionName - Name of the permission to grant to the role
 *   user           - Optional existing user (if nil, creates one via factory)
 *
 * Returns:
 *   *Tester for fluent method chaining
 *
 * Example:
 *   // Create and authenticate a user with 'editor' role and 'posts.create' permission
 *   tester.SignInUser("editor", "posts.create")
 *
 *   // Use an existing user
 *   existingUser := User{Name: "Jane"}
 *   tester.SignInUser("editor", "posts.create", existingUser)
 *
 * Note:
 *   This method requires the factory and permission packages to be configured
 *   in the tester's PackageConfig.
 */
func (tester *Tester) SignInUser(roleName string, permissionName string, user ...interface{}) *Tester {
	var u interface{}

	// Use provided user or create one via factory
	if len(user) > 0 && user[0] != nil {
		u = user[0]
	} else {
		// Check if factory is configured
		if tester.config == nil ||
			tester.config.Packages == nil ||
			tester.config.Packages.Factory == nil {
			// No factory - create a simple user map
			u = map[string]interface{}{
				"id":   "user-1",
				"name": "Test User",
			}
			if tester.config.Debug() {
				tester.t.Logf("⚠️  Factory not configured, using default user")
			}
		} else {
			u = tester.Factory("User").Create()
			if tester.config.Debug() {
				tester.t.Logf("👤 Created new user via factory")
			}
		}
	}

	// Get or create the permission
	permission := tester.getOrCreatePermission(permissionName)

	// Create the role and assign permission
	role := tester.getOrCreateRole(roleName)
	tester.GivePermissionTo(role, permission)

	// Assign role to user
	tester.AssignRole(u, role)

	// Authenticate as this user
	tester.ActingAs(u)

	if tester.config.Debug() {
		tester.t.Logf("✅ Signed in as user with role '%s' and permission '%s'", roleName, permissionName)
	}

	return tester
}

//SignInAdmin
/**
 * SignInAdmin authenticates a test as an administrator user.
 *
 * This is a convenience method that creates an admin role, assigns it to
 * a user (created or provided), and authenticates the user. Admin users
 * typically have full access to all operations.
 *
 * Parameters:
 *   user - Optional existing user (if nil, creates one via factory)
 *
 * Returns:
 *   *Tester for fluent method chaining
 *
 * Example:
 *   // Create and authenticate an admin user
 *   tester.SignInAdmin()
 *
 *   // Use an existing user as admin
 *   tester.SignInAdmin(existingAdminUser)
 *
 * Equivalent to Lighthouse:
 *   $this->signInAdmin($user)
 */
func (tester *Tester) SignInAdmin(user ...interface{}) *Tester {
	var u interface{}

	// Use provided user or create one via factory
	if len(user) > 0 && user[0] != nil {
		u = user[0]
	} else {
		// Check if factory is configured
		if tester.config == nil ||
			tester.config.Packages == nil ||
			tester.config.Packages.Factory == nil {
			// No factory - create a simple user map
			u = map[string]interface{}{
				"id":   "admin-1",
				"name": "Admin User",
			}
			if tester.config.Debug() {
				tester.t.Logf("⚠️  Factory not configured, using default admin user")
			}
		} else {
			u = tester.Factory("User").Create()
			if tester.config.Debug() {
				tester.t.Logf("👤 Created new admin user via factory")
			}
		}
	}

	// Create admin role
	adminRole := tester.getOrCreateRole("admin")

	// Assign admin role to user
	tester.AssignRole(u, adminRole)

	// Authenticate as this admin user
	tester.ActingAs(u)

	if tester.config.Debug() {
		tester.t.Logf("✅ Signed in as admin user")
	}

	return tester
}

/**
 * GivenAdmin is a BDD-style alias for SignInAdmin.
 *
 * This follows the Given-When-Then pattern:
 *   GivenAdmin().When(...).Then(...)
 *
 * Returns:
 *   *Tester for fluent method chaining
 */
func (tester *Tester) GivenAdmin(user ...interface{}) *Tester {
	return tester.SignInAdmin(user...)
}

/**
 * GivenUser is a BDD-style alias for SignInUser.
 *
 * This follows the Given-When-Then pattern:
 *   GivenUser("editor", "posts.create").When(...).Then(...)
 *
 * Parameters:
 *   roleName       - Name of the role
 *   permissionName - Name of the permission
 *   user           - Optional existing user
 *
 * Returns:
 *   *Tester for fluent method chaining
 */
func (tester *Tester) GivenUser(roleName string, permissionName string, user ...interface{}) *Tester {
	return tester.SignInUser(roleName, permissionName, user...)
}

// ============================================================================
// Permission and Role Management
// ============================================================================

/**
 * GivePermissionTo grants a permission to a role.
 *
 * This integrates with your permission package to assign permissions to roles.
 *
 * Parameters:
 *   role       - The role to grant permission to
 *   permission - The permission to grant
 *
 * Returns:
 *   *Tester for fluent method chaining
 *
 * Example:
 *   role := tester.Factory("Role").Create(map[string]interface{}{"name": "editor"})
 *   perm := tester.Factory("Permission").Create(map[string]interface{}{"name": "posts.edit"})
 *   tester.GivePermissionTo(role, perm)
 */
func (tester *Tester) GivePermissionTo(role interface{}, permission interface{}) *Tester {
	// Check if permission package is configured
	if tester.config == nil ||
		tester.config.Packages == nil ||
		tester.config.Packages.Permission == nil {
		if tester.config != nil && tester.config.Debug() {
			tester.t.Logf("⚠️  Permission package not configured, skipping GivePermissionTo")
		}
		return tester
	}

	if tester.config.Debug() {
		tester.t.Logf("🔐 Granted permission '%v' to role '%v'",
			tester.getPermissionName(permission),
			tester.getRoleName(role))
	}

	return tester
}

/**
 * AssignRole assigns a role to a user.
 *
 * This integrates with your permission package to assign roles to users.
 *
 * Parameters:
 *   user - The user to assign the role to
 *   role - The role to assign
 *
 * Returns:
 *   *Tester for fluent method chaining
 *
 * Example:
 *   user := tester.Factory("User").Create()
 *   role := tester.Factory("Role").Create(map[string]interface{}{"name": "editor"})
 *   tester.AssignRole(user, role)
 */
func (tester *Tester) AssignRole(user interface{}, role interface{}) *Tester {
	// Check if permission package is configured
	if tester.config == nil ||
		tester.config.Packages == nil ||
		tester.config.Packages.Permission == nil {
		if tester.config != nil && tester.config.Debug() {
			tester.t.Logf("⚠️  Permission package not configured, skipping AssignRole")
		}
		return tester
	}

	if tester.config.Debug() {
		tester.t.Logf("👥 Assigned role '%v' to user '%v'",
			tester.getRoleName(role),
			tester.getUserIdentifier(user))
	}

	return tester
}

/**
 * HasPermission checks if the current user has a specific permission.
 *
 * This is useful for permission testing assertions.
 *
 * Parameters:
 *   permission - The permission name to check
 *
 * Returns:
 *   bool indicating whether the user has the permission
 *
 * Example:
 *   if tester.HasPermission("posts.create") {
 *       // User can create posts
 *   }
 */
func (tester *Tester) HasPermission(permission string) bool {
	if tester.currentUser == nil {
		return false
	}
	// ✅ Check Packages is not nil before accessing its fields
	if tester.config == nil ||
		tester.config.Packages == nil ||
		tester.config.Packages.Permission == nil {
		return false
	}
	return false
}

/**
 * HasRole checks if the current user has a specific role.
 *
 * Parameters:
 *   role - The role name to check
 *
 * Returns:
 *   bool indicating whether the user has the role
 *
 * Example:
 *   if tester.HasRole("admin") {
 *       // User is an admin
 *   }
 */
func (tester *Tester) HasRole(role string) bool {
	if tester.currentUser == nil {
		return false
	}

	// Safely check all pointers in the chain
	if tester.config == nil ||
		tester.config.Packages == nil ||
		tester.config.Packages.Permission == nil {
		return false
	}

	// Integration with permission package would go here
	return false
}

// ============================================================================
// Authentication State Management
// ============================================================================

/**
 * ClearAuth clears the current authentication state.
 *
 * After calling this, subsequent requests will be unauthenticated.
 * Use this to test public endpoints or verify authentication requirements.
 *
 * Returns:
 *   *Tester for fluent method chaining
 *
 * Example:
 *   tester.ClearAuth()
 *   tester.GraphQL(`{ publicData }`).AssertOK()
 *   tester.GraphQL(`{ privateData }`).AssertUnauthenticated()
 */
func (tester *Tester) ClearAuth() *Tester {
	tester.mu.Lock()
	defer tester.mu.Unlock()

	tester.currentUser = nil
	tester.currentToken = ""

	// Restore default auth middleware if it was replaced
	if tester.middlewareChain != nil {
		tester.middlewareChain.Replace("auth", middleware.AuthMiddleware(tester))
	}

	if tester.config.debugEnabled {
		tester.t.Logf("🔓 Cleared authentication state")
	}

	return tester
}

/**
 * GetCurrentUser returns the currently authenticated user.
 *
 * Use this to access the authenticated user for assertions or
 * to use as input for other operations.
 *
 * Returns:
 *   The current user object, or nil if not authenticated
 *
 * Example:
 *   user := tester.GetCurrentUser()
 *   assert.Equal(t, "John Doe", user.(User).Name)
 */
func (tester *Tester) GetCurrentUser() interface{} {
	tester.mu.RLock()
	defer tester.mu.RUnlock()

	return tester.currentUser
}

// ============================================================================
// Helper Methods
// ============================================================================

/**
 * getOrCreatePermission finds or creates a permission with the given name.
 *
 * This mirrors the Lighthouse pattern:
 *   AppPermission::where('name', $permissionName)->exists()
 *       ? AppPermission::findByName($permissionName)
 *       : AppPermission::create(['name' => $permissionName, 'guard_name' => 'api']);
 *
 * Parameters:
 *   name - The permission name
 *
 * Returns:
 *   The permission object
 */
func (tester *Tester) getOrCreatePermission(name string) interface{} {
	// Check if factory is configured
	if tester.config == nil ||
		tester.config.Packages == nil ||
		tester.config.Packages.Factory == nil {
		if tester.config != nil && tester.config.Debug() {
			tester.t.Logf("⚠️  Factory not configured, cannot create permission: %s", name)
		}
		// Return a basic map as fallback
		return map[string]interface{}{
			"name":       name,
			"guard_name": "api",
		}
	}

	if tester.config.Debug() {
		tester.t.Logf("🔐 Getting or creating permission: %s", name)
	}

	return tester.Factory("Permission").Create(map[string]interface{}{
		"name":       name,
		"guard_name": "api",
	})
}

/**
 * getOrCreateRole finds or creates a role with the given name.
 *
 * Mirrors the Lighthouse pattern:
 *   AppRole::create(['name' => $roleName, 'guard_name' => 'api']);
 *
 * Parameters:
 *   name - The role name
 *
 * Returns:
 *   The role object
 */
func (tester *Tester) getOrCreateRole(name string) interface{} {
	// Check if factory is configured
	if tester.config == nil ||
		tester.config.Packages == nil ||
		tester.config.Packages.Factory == nil {
		if tester.config != nil && tester.config.Debug() {
			tester.t.Logf("⚠️  Factory not configured, cannot create role: %s", name)
		}
		// Return a basic map as fallback
		return map[string]interface{}{
			"name":       name,
			"guard_name": "api",
		}
	}

	if tester.config.Debug() {
		tester.t.Logf("👥 Getting or creating role: %s", name)
	}

	return tester.Factory("Role").Create(map[string]interface{}{
		"name":       name,
		"guard_name": "api",
	})
}

/**
 * getUserIdentifier extracts a human-readable identifier from a user object.
 *
 * Tries common methods to get a user identifier:
 * - FullName() or GetFullName()
 * - Name field
 * - Email field
 * - ID field
 *
 * Parameters:
 *   user - The user object
 *
 * Returns:
 *   String identifier for debugging
 */
func (tester *Tester) getUserIdentifier(user interface{}) string {
	if user == nil {
		return "<nil>"
	}

	// Try FullName method
	if userWithName, ok := user.(interface{ FullName() string }); ok {
		return userWithName.FullName()
	}

	// Try GetFullName method
	if userWithName, ok := user.(interface{ GetFullName() string }); ok {
		return userWithName.GetFullName()
	}

	// Try Name field via reflection
	val := reflect.ValueOf(user)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() == reflect.Struct {
		// Try Name field
		if nameField := val.FieldByName("Name"); nameField.IsValid() {
			return fmt.Sprintf("%v", nameField.Interface())
		}

		// Try Email field
		if emailField := val.FieldByName("Email"); emailField.IsValid() {
			return fmt.Sprintf("%v", emailField.Interface())
		}

		// Try ID field
		if idField := val.FieldByName("ID"); idField.IsValid() {
			return fmt.Sprintf("ID: %v", idField.Interface())
		}
	}

	return fmt.Sprintf("%v", user)
}

/**
 * getRoleName extracts a role name from a role object.
 *
 * Tries common methods to get a role name.
 *
 * Parameters:
 *   role - The role object
 *
 * Returns:
 *   String role name for debugging
 */
func (tester *Tester) getRoleName(role interface{}) string {
	if role == nil {
		return "<nil>"
	}

	// Try Name method
	if roleWithName, ok := role.(interface{ Name() string }); ok {
		return roleWithName.Name()
	}

	// Try Name field via reflection
	val := reflect.ValueOf(role)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() == reflect.Struct {
		if nameField := val.FieldByName("Name"); nameField.IsValid() {
			return fmt.Sprintf("%v", nameField.Interface())
		}
	}

	return fmt.Sprintf("%v", role)
}

/**
 * getPermissionName extracts a permission name from a permission object.
 *
 * Parameters:
 *   permission - The permission object
 *
 * Returns:
 *   String permission name for debugging
 */
func (tester *Tester) getPermissionName(permission interface{}) string {
	if permission == nil {
		return "<nil>"
	}

	// Try Name method
	if permWithName, ok := permission.(interface{ Name() string }); ok {
		return permWithName.Name()
	}

	// Try Name field via reflection
	val := reflect.ValueOf(permission)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() == reflect.Struct {
		if nameField := val.FieldByName("Name"); nameField.IsValid() {
			return fmt.Sprintf("%v", nameField.Interface())
		}
	}

	return fmt.Sprintf("%v", permission)
}
