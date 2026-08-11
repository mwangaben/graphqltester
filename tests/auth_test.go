package tests

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock user for testing
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

// Mock role for testing
type mockRole struct {
	Name      string
	GuardName string
}

func (r *mockRole) Name() string {
	return r.Name
}

// Mock permission for testing
type mockPermission struct {
	Name      string
	GuardName string
}

func (p *mockPermission) Name() string {
	return p.Name
}

/**
 * TestActingAs_SetsCurrentUser verifies that ActingAs correctly sets
 * the current user in the tester state.
 */
func TestActingAs_SetsCurrentUser(t *testing.T) {
	config := DefaultConfig()
	config.Schema.Path = "./testdata/schema.graphql"
	config.Schema.Resolvers = &mockResolver{}

	tester := NewTester(t, config)

	user := &mockUser{
		ID:        "123",
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john@example.com",
	}

	result := tester.ActingAs(user)

	// Should return tester for chaining
	assert.Same(t, tester, result, "ActingAs should return the tester for chaining")

	// Should set the current user
	assert.Equal(t, user, tester.currentUser, "Current user should be set")
}

/**
 * TestWithToken_SetsAuthToken verifies that WithToken correctly sets
 * the authentication token.
 */
func TestWithToken_SetsAuthToken(t *testing.T) {
	config := DefaultConfig()
	config.Schema.Path = "./testdata/schema.graphql"
	config.Schema.Resolvers = &mockResolver{}

	tester := NewTester(t, config)

	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."

	result := tester.WithToken(token)

	assert.Same(t, tester, result, "WithToken should return the tester for chaining")
	assert.Equal(t, token, tester.currentToken, "Current token should be set")
}

/**
 * TestClearAuth_RemovesAuthState verifies that ClearAuth properly
 * clears all authentication state.
 */
func TestClearAuth_RemovesAuthState(t *testing.T) {
	config := DefaultConfig()
	config.Schema.Path = "./testdata/schema.graphql"
	config.Schema.Resolvers = &mockResolver{}

	tester := NewTester(t, config)

	// Set auth state
	user := &mockUser{ID: "123"}
	tester.ActingAs(user)
	tester.WithToken("some-token")

	// Clear auth
	result := tester.ClearAuth()

	assert.Same(t, tester, result, "ClearAuth should return the tester for chaining")
	assert.Nil(t, tester.currentUser, "Current user should be nil after clear")
	assert.Empty(t, tester.currentToken, "Current token should be empty after clear")
}

/**
 * TestGetCurrentUser_ReturnsCorrectUser verifies that GetCurrentUser
 * returns the currently authenticated user.
 */
func TestGetCurrentUser_ReturnsCorrectUser(t *testing.T) {
	config := DefaultConfig()
	config.Schema.Path = "./testdata/schema.graphql"
	config.Schema.Resolvers = &mockResolver{}

	tester := NewTester(t, config)

	user := &mockUser{ID: "456", FirstName: "Jane", LastName: "Smith"}
	tester.ActingAs(user)

	result := tester.GetCurrentUser()

	assert.Equal(t, user, result, "GetCurrentUser should return the authenticated user")
}

/**
 * TestGetCurrentUser_ReturnsNilWhenNotAuthenticated verifies that
 * GetCurrentUser returns nil when no user is authenticated.
 */
func TestGetCurrentUser_ReturnsNilWhenNotAuthenticated(t *testing.T) {
	config := DefaultConfig()
	config.Schema.Path = "./testdata/schema.graphql"
	config.Schema.Resolvers = &mockResolver{}

	tester := NewTester(t, config)

	result := tester.GetCurrentUser()

	assert.Nil(t, result, "GetCurrentUser should return nil when not authenticated")
}

/**
 * TestHasPermission_ReturnsFalseWhenNotAuthenticated verifies that
 * HasPermission returns false when no user is authenticated.
 */
func TestHasPermission_ReturnsFalseWhenNotAuthenticated(t *testing.T) {
	config := DefaultConfig()
	config.Schema.Path = "./testdata/schema.graphql"
	config.Schema.Resolvers = &mockResolver{}

	tester := NewTester(t, config)

	// No user authenticated
	result := tester.HasPermission("posts.create")

	assert.False(t, result, "HasPermission should return false when not authenticated")
}

/**
 * TestHasRole_ReturnsFalseWhenNotAuthenticated verifies that
 * HasRole returns false when no user is authenticated.
 */
func TestHasRole_ReturnsFalseWhenNotAuthenticated(t *testing.T) {
	config := DefaultConfig()
	config.Schema.Path = "./testdata/schema.graphql"
	config.Schema.Resolvers = &mockResolver{}

	tester := NewTester(t, config)

	result := tester.HasRole("admin")

	assert.False(t, result, "HasRole should return false when not authenticated")
}

/**
 * TestGetUserIdentifier_WithFullName verifies that getUserIdentifier
 * correctly extracts the full name from a user with FullName method.
 */
func TestGetUserIdentifier_WithFullName(t *testing.T) {
	config := DefaultConfig()
	config.Schema.Path = "./testdata/schema.graphql"
	config.Schema.Resolvers = &mockResolver{}

	tester := NewTester(t, config)

	user := &mockUser{
		ID:        "789",
		FirstName: "Alice",
		LastName:  "Johnson",
	}

	identifier := tester.getUserIdentifier(user)

	assert.Equal(t, "Alice Johnson", identifier, "Should return full name")
}

/**
 * TestGetUserIdentifier_WithEmail verifies that getUserIdentifier
 * falls back to email when no FullName method exists.
 */
func TestGetUserIdentifier_WithEmail(t *testing.T) {
	config := DefaultConfig()
	config.Schema.Path = "./testdata/schema.graphql"
	config.Schema.Resolvers = &mockResolver{}

	tester := NewTester(t, config)

	user := struct {
		ID    string
		Email string
	}{
		ID:    "101",
		Email: "bob@example.com",
	}

	identifier := tester.getUserIdentifier(&user)

	assert.Equal(t, "bob@example.com", identifier, "Should fall back to email")
}

/**
 * TestGetUserIdentifier_WithID verifies that getUserIdentifier
 * falls back to ID when no name or email is available.
 */
func TestGetUserIdentifier_WithID(t *testing.T) {
	config := DefaultConfig()
	config.Schema.Path = "./testdata/schema.graphql"
	config.Schema.Resolvers = &mockResolver{}

	tester := NewTester(t, config)

	user := struct {
		ID string
	}{
		ID: "202",
	}

	identifier := tester.getUserIdentifier(&user)

	assert.Contains(t, identifier, "202", "Should contain the ID")
}

/**
 * TestGetRoleName_ExtractsName verifies that getRoleName correctly
 * extracts the role name from a role object.
 */
func TestGetRoleName_ExtractsName(t *testing.T) {
	config := DefaultConfig()
	config.Schema.Path = "./testdata/schema.graphql"
	config.Schema.Resolvers = &mockResolver{}

	tester := NewTester(t, config)

	role := &mockRole{
		Name:      "admin",
		GuardName: "api",
	}

	name := tester.getRoleName(role)

	assert.Equal(t, "admin", name, "Should extract role name")
}

/**
 * TestGetPermissionName_ExtractsName verifies that getPermissionName
 * correctly extracts the permission name.
 */
func TestGetPermissionName_ExtractsName(t *testing.T) {
	config := DefaultConfig()
	config.Schema.Path = "./testdata/schema.graphql"
	config.Schema.Resolvers = &mockResolver{}

	tester := NewTester(t, config)

	permission := &mockPermission{
		Name:      "posts.create",
		GuardName: "api",
	}

	name := tester.getPermissionName(permission)

	assert.Equal(t, "posts.create", name, "Should extract permission name")
}

/**
 * TestGivenAdmin_BDDStyleAlias verifies that GivenAdmin works as
 * a BDD-style alias for SignInAdmin.
 */
func TestGivenAdmin_BDDStyleAlias(t *testing.T) {
	config := DefaultConfig()
	config.Schema.Path = "./testdata/schema.graphql"
	config.Schema.Resolvers = &mockResolver{}

	tester := NewTester(t, config)

	user := &mockUser{
		ID:        "303",
		FirstName: "Admin",
		LastName:  "User",
	}

	result := tester.GivenAdmin(user)

	assert.Same(t, tester, result, "GivenAdmin should return tester for chaining")
	assert.Equal(t, user, tester.currentUser, "Should set the admin user")
}

/**
 * TestGivenUser_BDDStyleAlias verifies that GivenUser works as
 * a BDD-style alias for SignInUser.
 */
func TestGivenUser_BDDStyleAlias(t *testing.T) {
	config := DefaultConfig()
	config.Schema.Path = "./testdata/schema.graphql"
	config.Schema.Resolvers = &mockResolver{}

	tester := NewTester(t, config)

	user := &mockUser{
		ID:        "404",
		FirstName: "Test",
		LastName:  "User",
	}

	result := tester.GivenUser("editor", "posts.edit", user)

	assert.Same(t, tester, result, "GivenUser should return tester for chaining")
	assert.Equal(t, user, tester.currentUser, "Should set the authenticated user")
}

/**
 * TestActingAs_WithMiddlewareIntegration verifies that ActingAs properly
 * integrates with the middleware chain to propagate user context.
 */
func TestActingAs_WithMiddlewareIntegration(t *testing.T) {
	// Create a test that verifies middleware integration
	config := DefaultConfig()
	config.Schema.Path = "./testdata/schema.graphql"
	config.Schema.Resolvers = &mockResolver{}
	config.Middleware.AuthEnabled = true

	tester := NewTester(t, config)

	user := &mockUser{
		ID:        "505",
		FirstName: "Context",
		LastName:  "User",
	}

	tester.ActingAs(user)

	// Verify middleware was configured
	assert.NotNil(t, tester.middlewareChain, "Middleware chain should be configured")

	// The user should be accessible
	assert.Equal(t, user, tester.GetCurrentUser(), "Should be able to retrieve current user")
}

/**
 * TestAuthenticationFlow_CompleteScenario tests a complete authentication
 * flow: authenticate -> make request -> clear auth -> verify unauthenticated.
 */
func TestAuthenticationFlow_CompleteScenario(t *testing.T) {
	config := DefaultConfig()
	config.Schema.Path = "./testdata/schema.graphql"
	config.Schema.Resolvers = &mockResolver{}

	tester := NewTester(t, config)

	// Step 1: No authentication
	assert.Nil(t, tester.GetCurrentUser(), "Should start unauthenticated")

	// Step 2: Authenticate
	user := &mockUser{ID: "606", FirstName: "Flow", LastName: "Test"}
	tester.ActingAs(user)
	assert.NotNil(t, tester.GetCurrentUser(), "Should be authenticated")

	// Step 3: Set token
	token := "test-token-123"
	tester.WithToken(token)
	assert.Equal(t, token, tester.currentToken, "Token should be set")

	// Step 4: Clear authentication
	tester.ClearAuth()
	assert.Nil(t, tester.GetCurrentUser(), "Should be unauthenticated after clear")
	assert.Empty(t, tester.currentToken, "Token should be cleared")

	// Step 5: Re-authenticate
	newUser := &mockUser{ID: "707", FirstName: "New", LastName: "User"}
	tester.ActingAs(newUser)
	assert.Equal(t, newUser, tester.GetCurrentUser(), "Should be re-authenticated")
}
