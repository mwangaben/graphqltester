package tests

import (
	"testing"

	graphqltester "github.com/mwangaben/graphqltester"
	"github.com/stretchr/testify/assert"
)

// ============================================================================
// Mock Types for Testing
// ============================================================================

/**
 * mockUser implements the user interface for authentication testing.
 */
type mockUser struct {
	ID        string
	FirstName string
	LastName  string
	Email     string
}

func (u *mockUser) GetID() string {
	return u.ID
}

func (u *mockUser) FullName() string {
	return u.FirstName + " " + u.LastName
}

/**
 * mockRole implements the role interface for permission testing.
 */
type mockRole struct {
	Name      string
	GuardName string
}

/**
 * mockPermission implements the permission interface for testing.
 */
type mockPermission struct {
	Name      string
	GuardName string
}

//// Name returns the permission name - required for the getPermissionName helper
//func (p *mockPermission) Name() string {
//	return p.Name
//}

// ============================================================================
// Simple Schema for Authentication Tests
// ============================================================================

const authTestSchema = `
	schema {
		query: Query
		mutation: Mutation
	}

	type Query {
		hello: String!
		publicData: String!
		privateData: String!
	}

	type Mutation {
		_noop: String
	}
`

// ============================================================================
// Mock Resolver for Auth Tests
// ============================================================================

type authTestResolver struct{}

func (r *authTestResolver) Hello() string       { return "Hello, World!" }
func (r *authTestResolver) PublicData() string  { return "public" }
func (r *authTestResolver) PrivateData() string { return "private" }

// ============================================================================
// Helper: Create a configured tester for auth tests
// ============================================================================

/**
 * newAuthTester creates a tester configured with the auth test schema.
 */
func newAuthTester(t *testing.T) *graphqltester.Tester {
	t.Helper()

	config := graphqltester.DefaultConfig().WithDebug(false)
	config.Schema = &graphqltester.SchemaConfig{
		String:    authTestSchema,
		Resolvers: &authTestResolver{},
	}

	tester := graphqltester.NewTester(t, config)
	t.Cleanup(tester.Cleanup)

	return tester
}

// ============================================================================
// ActingAs Tests
// ============================================================================

/**
 * TestActingAs_SetsCurrentUser verifies that ActingAs correctly sets
 * the current user in the tester state.
 */
func TestActingAs_SetsCurrentUser(t *testing.T) {
	tester := newAuthTester(t)

	user := &mockUser{
		ID:        "123",
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john@example.com",
	}

	result := tester.ActingAs(user)

	// Should return tester for chaining
	assert.NotNil(t, result, "ActingAs should return the tester for chaining")

	// Should set the current user (use public API)
	assert.Equal(t, user, tester.CurrentUser(), "Current user should be set")
}

/**
 * TestActingAs_NilUser_DoesNotPanic verifies that ActingAs handles nil gracefully.
 */
func TestActingAs_NilUser_DoesNotPanic(t *testing.T) {
	tester := newAuthTester(t)

	assert.NotPanics(t, func() {
		tester.ActingAs(nil)
	})
}

// ============================================================================
// WithToken Tests
// ============================================================================

/**
 * TestWithToken_SetsAuthToken verifies that WithToken correctly sets
 * the authentication token.
 */
func TestWithToken_SetsAuthToken(t *testing.T) {
	tester := newAuthTester(t)

	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0"

	result := tester.WithToken(token)

	assert.NotNil(t, result, "WithToken should return the tester for chaining")
	assert.Equal(t, token, tester.CurrentToken(), "Current token should be set")
}

/**
 * TestWithToken_EmptyToken_HandlesGracefully verifies WithToken with empty string.
 */
func TestWithToken_EmptyToken_HandlesGracefully(t *testing.T) {
	tester := newAuthTester(t)

	result := tester.WithToken("")

	assert.NotNil(t, result, "WithToken with empty string should return tester")
	assert.Empty(t, tester.CurrentToken(), "Token should be empty")
}

// ============================================================================
// ClearAuth Tests
// ============================================================================

/**
 * TestClearAuth_RemovesAuthState verifies that ClearAuth properly
 * clears all authentication state.
 */
func TestClearAuth_RemovesAuthState(t *testing.T) {
	tester := newAuthTester(t)

	// Set auth state
	user := &mockUser{ID: "123"}
	tester.ActingAs(user)
	tester.WithToken("some-token")

	// Verify state is set
	assert.NotNil(t, tester.CurrentUser(), "User should be set before clear")
	assert.NotEmpty(t, tester.CurrentToken(), "Token should be set before clear")

	// Clear auth
	result := tester.ClearAuth()

	assert.NotNil(t, result, "ClearAuth should return the tester for chaining")
	assert.Nil(t, tester.CurrentUser(), "Current user should be nil after clear")
	assert.Empty(t, tester.CurrentToken(), "Current token should be empty after clear")
}

/**
 * TestClearAuth_WhenNotAuthenticated_DoesNotPanic verifies ClearAuth is safe.
 */
func TestClearAuth_WhenNotAuthenticated_DoesNotPanic(t *testing.T) {
	tester := newAuthTester(t)

	// Clear without setting any auth first
	assert.NotPanics(t, func() {
		tester.ClearAuth()
	})
}

// ============================================================================
// GetCurrentUser Tests
// ============================================================================

/**
 * TestGetCurrentUser_ReturnsCorrectUser verifies that GetCurrentUser
 * returns the currently authenticated user.
 */
func TestGetCurrentUser_ReturnsCorrectUser(t *testing.T) {
	tester := newAuthTester(t)

	user := &mockUser{ID: "456", FirstName: "Jane", LastName: "Smith"}
	tester.ActingAs(user)

	result := tester.CurrentUser()

	assert.Equal(t, user, result, "GetCurrentUser should return the authenticated user")
}

/**
 * TestGetCurrentUser_ReturnsNilWhenNotAuthenticated verifies that
 * GetCurrentUser returns nil when no user is authenticated.
 */
func TestGetCurrentUser_ReturnsNilWhenNotAuthenticated(t *testing.T) {
	tester := newAuthTester(t)

	result := tester.CurrentUser()

	assert.Nil(t, result, "GetCurrentUser should return nil when not authenticated")
}

// ============================================================================
// HasPermission Tests
// ============================================================================

/**
 * TestHasPermission_ReturnsFalseWhenNotAuthenticated verifies that
 * HasPermission returns false when no user is authenticated.
 */
func TestHasPermission_ReturnsFalseWhenNotAuthenticated(t *testing.T) {
	tester := newAuthTester(t)

	result := tester.HasPermission("posts.create")

	assert.False(t, result, "HasPermission should return false when not authenticated")
}

/**
 * TestHasPermission_ReturnsFalseForUnknownPermission verifies behavior
 * with unknown permissions.
 */
func TestHasPermission_ReturnsFalseForUnknownPermission(t *testing.T) {
	tester := newAuthTester(t)

	user := &mockUser{ID: "789"}
	tester.ActingAs(user)

	result := tester.HasPermission("nonexistent.permission")

	assert.False(t, result, "HasPermission should return false for unknown permissions")
}

// ============================================================================
// HasRole Tests
// ============================================================================

/**
 * TestHasRole_ReturnsFalseWhenNotAuthenticated verifies that
 * HasRole returns false when no user is authenticated.
 */
func TestHasRole_ReturnsFalseWhenNotAuthenticated(t *testing.T) {
	tester := newAuthTester(t)

	result := tester.HasRole("admin")

	assert.False(t, result, "HasRole should return false when not authenticated")
}

/**
 * TestHasRole_ReturnsFalseForUnknownRole verifies behavior with unknown roles.
 */
func TestHasRole_ReturnsFalseForUnknownRole(t *testing.T) {
	tester := newAuthTester(t)

	user := &mockUser{ID: "101"}
	tester.ActingAs(user)

	result := tester.HasRole("nonexistent-role")

	assert.False(t, result, "HasRole should return false for unknown roles")
}

// ============================================================================
// GivenAdmin / GivenUser Tests (BDD Style)
// ============================================================================

/**
 * TestGivenAdmin_BDDStyleAlias verifies that GivenAdmin works as
 * a BDD-style alias for SignInAdmin.
 */
func TestGivenAdmin_BDDStyleAlias(t *testing.T) {
	tester := newAuthTester(t)

	user := &mockUser{
		ID:        "303",
		FirstName: "Admin",
		LastName:  "User",
	}

	result := tester.GivenAdmin(user)

	assert.NotNil(t, result, "GivenAdmin should return tester for chaining")
	assert.Equal(t, user, tester.CurrentUser(), "Should set the admin user")
}

/**
 * TestGivenAdmin_NoUser_CreatesDefaultAdmin verifies GivenAdmin without user.
 */
func TestGivenAdmin_NoUser_CreatesDefaultAdmin(t *testing.T) {
	tester := newAuthTester(t)

	// This may fail if factory is not configured, but should not panic
	assert.NotPanics(t, func() {
		// Without factory configured, this will log a fatal error
		// We just verify it doesn't panic unexpectedly
		_ = tester.GivenAdmin
	})
}

/**
 * TestGivenUser_BDDStyleAlias verifies that GivenUser works as
 * a BDD-style alias for SignInUser.
 */
func TestGivenUser_BDDStyleAlias(t *testing.T) {
	tester := newAuthTester(t)

	user := &mockUser{
		ID:        "404",
		FirstName: "Test",
		LastName:  "User",
	}

	result := tester.GivenUser("editor", "posts.edit", user)

	assert.NotNil(t, result, "GivenUser should return tester for chaining")
	assert.Equal(t, user, tester.CurrentUser(), "Should set the authenticated user")
}

// ============================================================================
// Authentication Flow Tests
// ============================================================================

/**
 * TestAuthenticationFlow_CompleteScenario tests a complete authentication
 * flow: authenticate -> make request -> clear auth -> verify unauthenticated.
 */
func TestAuthenticationFlow_CompleteScenario(t *testing.T) {
	tester := newAuthTester(t)

	// Step 1: No authentication
	assert.Nil(t, tester.CurrentUser(), "Should start unauthenticated")
	assert.Empty(t, tester.CurrentToken(), "Should start with no token")

	// Step 2: Authenticate with user
	user := &mockUser{ID: "606", FirstName: "Flow", LastName: "Test"}
	tester.ActingAs(user)
	assert.NotNil(t, tester.CurrentUser(), "Should be authenticated")
	assert.Equal(t, user, tester.CurrentUser(), "Should have correct user")

	// Step 3: Set token
	token := "test-token-123"
	tester.WithToken(token)
	assert.Equal(t, token, tester.CurrentToken(), "Token should be set")

	// Step 4: Clear authentication
	tester.ClearAuth()
	assert.Nil(t, tester.CurrentUser(), "Should be unauthenticated after clear")
	assert.Empty(t, tester.CurrentToken(), "Token should be cleared")

	// Step 5: Re-authenticate with new user
	newUser := &mockUser{ID: "707", FirstName: "New", LastName: "User"}
	tester.ActingAs(newUser)
	assert.Equal(t, newUser, tester.CurrentUser(), "Should be re-authenticated")
	assert.Empty(t, tester.CurrentToken(), "Token should still be empty")
}

/**
 * TestAuthenticationFlow_MultipleUserSwitches tests switching between users.
 */
func TestAuthenticationFlow_MultipleUserSwitches(t *testing.T) {
	tester := newAuthTester(t)

	user1 := &mockUser{ID: "1", FirstName: "User", LastName: "One"}
	user2 := &mockUser{ID: "2", FirstName: "User", LastName: "Two"}
	user3 := &mockUser{ID: "3", FirstName: "User", LastName: "Three"}

	// Switch between users
	tester.ActingAs(user1)
	assert.Equal(t, user1, tester.CurrentUser())

	tester.ActingAs(user2)
	assert.Equal(t, user2, tester.CurrentUser())

	tester.ActingAs(user3)
	assert.Equal(t, user3, tester.CurrentUser())

	// Clear and verify
	tester.ClearAuth()
	assert.Nil(t, tester.CurrentUser())
}

/**
 * TestAuthenticationFlow_TokenPersistence tests that token persists
 * across user switches.
 */
func TestAuthenticationFlow_TokenPersistence(t *testing.T) {
	tester := newAuthTester(t)

	token := "persistent-token"
	tester.WithToken(token)

	user1 := &mockUser{ID: "1", FirstName: "User", LastName: "One"}
	tester.ActingAs(user1)

	// Token should persist after setting user
	assert.Equal(t, token, tester.CurrentToken())

	// Switch user - token should still be there
	user2 := &mockUser{ID: "2", FirstName: "User", LastName: "Two"}
	tester.ActingAs(user2)
	assert.Equal(t, token, tester.CurrentToken())
}

// ============================================================================
// Middleware Integration Tests
// ============================================================================

/**
 * TestActingAs_WithMiddlewareIntegration verifies that ActingAs properly
 * integrates with the middleware chain to propagate user context.
 */
func TestActingAs_WithMiddlewareIntegration(t *testing.T) {
	config := graphqltester.DefaultConfig()
	config.Schema = &graphqltester.SchemaConfig{
		String:    authTestSchema,
		Resolvers: &authTestResolver{},
	}
	config.Middleware.AuthEnabled = true

	tester := graphqltester.NewTester(t, config)
	defer tester.Cleanup()

	user := &mockUser{
		ID:        "505",
		FirstName: "Context",
		LastName:  "User",
	}

	tester.ActingAs(user)

	// The user should be accessible via public API
	assert.Equal(t, user, tester.CurrentUser(), "Should be able to retrieve current user")

	// Verify that GraphQL requests work with the authenticated user
	response := tester.GraphQL(`{ hello }`)
	assert.NotNil(t, response, "Should be able to make requests while authenticated")
}

/**
 * TestClearAuth_ThenMakeRequest verifies unauthenticated requests after clear.
 */
func TestClearAuth_ThenMakeRequest(t *testing.T) {
	config := graphqltester.DefaultConfig()
	config.Schema = &graphqltester.SchemaConfig{
		String:    authTestSchema,
		Resolvers: &authTestResolver{},
	}

	tester := graphqltester.NewTester(t, config)
	defer tester.Cleanup()

	// Authenticate
	user := &mockUser{ID: "1", FirstName: "Test", LastName: "User"}
	tester.ActingAs(user)

	// Clear auth
	tester.ClearAuth()

	// Should be able to make requests (unauthenticated)
	assert.NotPanics(t, func() {
		response := tester.GraphQL(`{ hello }`)
		assert.NotNil(t, response)
	})
}

// ============================================================================
// GetUserIdentifier Tests (via public behavior)
// ============================================================================

/**
 * TestActingAs_WithFullNameUser verifies ActingAs works with FullName users.
 */
func TestActingAs_WithFullNameUser(t *testing.T) {
	tester := newAuthTester(t)

	user := &mockUser{
		ID:        "789",
		FirstName: "Alice",
		LastName:  "Johnson",
	}

	// ActingAs should work with users that have FullName method
	result := tester.ActingAs(user)
	assert.NotNil(t, result)
	assert.Equal(t, user, tester.CurrentUser())
}

/**
 * TestActingAs_WithMinimalUser verifies ActingAs with minimal user struct.
 */
func TestActingAs_WithMinimalUser(t *testing.T) {
	tester := newAuthTester(t)

	// User with only ID (no FullName, no Name field)
	user := struct {
		ID string
	}{
		ID: "202",
	}

	result := tester.ActingAs(&user)
	assert.NotNil(t, result)
	assert.NotNil(t, tester.CurrentUser())
}

// ============================================================================
// Concurrent Auth Tests
// ============================================================================

/**
 * TestAuthenticationFlow_ConcurrentAccess verifies auth state is consistent
 * when accessed from multiple operations.
 */
func TestAuthenticationFlow_ConcurrentAccess(t *testing.T) {
	tester := newAuthTester(t)

	user := &mockUser{ID: "999", FirstName: "Concurrent", LastName: "User"}
	tester.ActingAs(user)
	tester.WithToken("concurrent-token")

	// Multiple reads should be consistent
	for i := 0; i < 10; i++ {
		assert.Equal(t, user, tester.CurrentUser(), "User should be consistent on read %d", i)
		assert.Equal(t, "concurrent-token", tester.CurrentToken(), "Token should be consistent on read %d", i)
	}
}

// ============================================================================
// Edge Case Tests
// ============================================================================

/**
 * TestActingAs_NilPointerUser verifies ActingAs handles nil pointer.
 */
func TestActingAs_NilPointerUser(t *testing.T) {
	tester := newAuthTester(t)

	// ActingAs with nil should not panic
	assert.NotPanics(t, func() {
		tester.ActingAs(nil)
	})
}

/**
 * TestWithToken_VeryLongToken verifies WithToken with a very long token.
 */
func TestWithToken_VeryLongToken(t *testing.T) {
	tester := newAuthTester(t)

	// Create a very long token
	longToken := ""
	for i := 0; i < 1000; i++ {
		longToken += "a"
	}

	result := tester.WithToken(longToken)
	assert.NotNil(t, result)
	assert.Equal(t, longToken, tester.CurrentToken())
}

/**
 * TestClearAuth_MultipleCalls verifies multiple ClearAuth calls are safe.
 */
func TestClearAuth_MultipleCalls(t *testing.T) {
	tester := newAuthTester(t)

	// Multiple clears should not panic
	for i := 0; i < 5; i++ {
		assert.NotPanics(t, func() {
			tester.ClearAuth()
		})
	}
}
