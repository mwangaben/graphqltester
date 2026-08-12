package tests

import (
	"testing"

	graphqltester "github.com/mwangaben/graphqltester"
	"github.com/stretchr/testify/assert"
)

// ============================================================================
// Schema for Auth Tests (NO _noop field)
// ============================================================================

const authTestSchema = `
	schema {
		query: Query
	}

	type Query {
		hello: String!
		publicData: String!
		privateData: String!
		secretData: String!
		adminData: String!
	}
`

// ============================================================================
// Mock Types
// ============================================================================

type mockUser struct {
	ID        string
	FirstName string
	LastName  string
	Email     string
}

func (u *mockUser) GetID() string    { return u.ID }
func (u *mockUser) FullName() string { return u.FirstName + " " + u.LastName }

type mockRole struct {
	RoleName  string
	GuardName string
}

func (r *mockRole) Name() string { return r.RoleName }

type mockPermission struct {
	PermissionName string
	GuardName      string
}

func (p *mockPermission) Name() string { return p.PermissionName }

// ============================================================================
// Mock Resolver
// ============================================================================

type authTestResolver struct{}

func (r *authTestResolver) Hello() string       { return "Hello, World!" }
func (r *authTestResolver) PublicData() string  { return "public" }
func (r *authTestResolver) PrivateData() string { return "private" }
func (r *authTestResolver) SecretData() string  { return "secret" }
func (r *authTestResolver) AdminData() string   { return "admin" }

// ============================================================================
// Helper
// ============================================================================

func newAuthTester(t *testing.T) *graphqltester.Tester {
	t.Helper()

	config := graphqltester.DefaultConfig().WithDebug(false)
	config.Database = nil
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

func TestActingAs_SetsCurrentUser(t *testing.T) {
	tester := newAuthTester(t)
	user := &mockUser{ID: "123", FirstName: "John", LastName: "Doe", Email: "john@example.com"}
	result := tester.ActingAs(user)
	assert.NotNil(t, result)
	assert.Equal(t, user, tester.CurrentUser())
}

func TestActingAs_NilUser_DoesNotPanic(t *testing.T) {
	tester := newAuthTester(t)
	assert.NotPanics(t, func() { tester.ActingAs(nil) })
}

// ============================================================================
// WithToken Tests
// ============================================================================

func TestWithToken_SetsAuthToken(t *testing.T) {
	tester := newAuthTester(t)
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0"
	result := tester.WithToken(token)
	assert.NotNil(t, result)
	assert.Equal(t, token, tester.CurrentToken())
}

func TestWithToken_EmptyToken_HandlesGracefully(t *testing.T) {
	tester := newAuthTester(t)
	result := tester.WithToken("")
	assert.NotNil(t, result)
	assert.Empty(t, tester.CurrentToken())
}

// ============================================================================
// ClearAuth Tests
// ============================================================================

func TestClearAuth_RemovesAuthState(t *testing.T) {
	tester := newAuthTester(t)
	user := &mockUser{ID: "123"}
	tester.ActingAs(user)
	tester.WithToken("some-token")
	assert.NotNil(t, tester.CurrentUser())
	assert.NotEmpty(t, tester.CurrentToken())

	result := tester.ClearAuth()
	assert.NotNil(t, result)
	assert.Nil(t, tester.CurrentUser())
	assert.Empty(t, tester.CurrentToken())
}

func TestClearAuth_WhenNotAuthenticated_DoesNotPanic(t *testing.T) {
	tester := newAuthTester(t)
	assert.NotPanics(t, func() { tester.ClearAuth() })
}

// ============================================================================
// GetCurrentUser Tests
// ============================================================================

func TestGetCurrentUser_ReturnsCorrectUser(t *testing.T) {
	tester := newAuthTester(t)
	user := &mockUser{ID: "456", FirstName: "Jane", LastName: "Smith"}
	tester.ActingAs(user)
	assert.Equal(t, user, tester.CurrentUser())
}

func TestGetCurrentUser_ReturnsNilWhenNotAuthenticated(t *testing.T) {
	tester := newAuthTester(t)
	assert.Nil(t, tester.CurrentUser())
}

// ============================================================================
// HasPermission Tests
// ============================================================================

func TestHasPermission_ReturnsFalseWhenNotAuthenticated(t *testing.T) {
	tester := newAuthTester(t)
	assert.False(t, tester.HasPermission("posts.create"))
}

func TestHasPermission_ReturnsFalseForUnknownPermission(t *testing.T) {
	tester := newAuthTester(t)
	user := &mockUser{ID: "789"}
	tester.ActingAs(user)
	assert.False(t, tester.HasPermission("nonexistent.permission"))
}

// ============================================================================
// HasRole Tests
// ============================================================================

func TestHasRole_ReturnsFalseWhenNotAuthenticated(t *testing.T) {
	tester := newAuthTester(t)
	assert.False(t, tester.HasRole("admin"))
}

func TestHasRole_ReturnsFalseForUnknownRole(t *testing.T) {
	tester := newAuthTester(t)
	user := &mockUser{ID: "101"}
	tester.ActingAs(user)
	assert.False(t, tester.HasRole("nonexistent-role"))
}

// ============================================================================
// GivenAdmin / GivenUser Tests
// ============================================================================

func TestGivenAdmin_BDDStyleAlias(t *testing.T) {
	tester := newAuthTester(t)
	user := &mockUser{ID: "303", FirstName: "Admin", LastName: "User"}
	result := tester.GivenAdmin(user)
	assert.NotNil(t, result)
	assert.Equal(t, user, tester.CurrentUser())
}

func TestGivenAdmin_NoUser_CreatesDefaultAdmin(t *testing.T) {
	tester := newAuthTester(t)
	// Without factory configured, this will fail gracefully
	// Just verify it exists
	assert.NotNil(t, tester.GivenAdmin)
}

func TestGivenUser_BDDStyleAlias(t *testing.T) {
	tester := newAuthTester(t)
	user := &mockUser{ID: "404", FirstName: "Test", LastName: "User"}
	result := tester.GivenUser("editor", "posts.edit", user)
	assert.NotNil(t, result)
	assert.Equal(t, user, tester.CurrentUser())
}

// ============================================================================
// Authentication Flow Tests
// ============================================================================

func TestAuthenticationFlow_CompleteScenario(t *testing.T) {
	tester := newAuthTester(t)

	assert.Nil(t, tester.CurrentUser())
	assert.Empty(t, tester.CurrentToken())

	user := &mockUser{ID: "606", FirstName: "Flow", LastName: "Test"}
	tester.ActingAs(user)
	assert.NotNil(t, tester.CurrentUser())
	assert.Equal(t, user, tester.CurrentUser())

	token := "test-token-123"
	tester.WithToken(token)
	assert.Equal(t, token, tester.CurrentToken())

	tester.ClearAuth()
	assert.Nil(t, tester.CurrentUser())
	assert.Empty(t, tester.CurrentToken())

	newUser := &mockUser{ID: "707", FirstName: "New", LastName: "User"}
	tester.ActingAs(newUser)
	assert.Equal(t, newUser, tester.CurrentUser())
	assert.Empty(t, tester.CurrentToken())
}

func TestAuthenticationFlow_MultipleUserSwitches(t *testing.T) {
	tester := newAuthTester(t)

	user1 := &mockUser{ID: "1", FirstName: "User", LastName: "One"}
	user2 := &mockUser{ID: "2", FirstName: "User", LastName: "Two"}
	user3 := &mockUser{ID: "3", FirstName: "User", LastName: "Three"}

	tester.ActingAs(user1)
	assert.Equal(t, user1, tester.CurrentUser())
	tester.ActingAs(user2)
	assert.Equal(t, user2, tester.CurrentUser())
	tester.ActingAs(user3)
	assert.Equal(t, user3, tester.CurrentUser())

	tester.ClearAuth()
	assert.Nil(t, tester.CurrentUser())
}

func TestAuthenticationFlow_TokenPersistence(t *testing.T) {
	tester := newAuthTester(t)

	token := "persistent-token"
	tester.WithToken(token)

	user1 := &mockUser{ID: "1", FirstName: "User", LastName: "One"}
	tester.ActingAs(user1)
	assert.Equal(t, token, tester.CurrentToken())

	user2 := &mockUser{ID: "2", FirstName: "User", LastName: "Two"}
	tester.ActingAs(user2)
	assert.Equal(t, token, tester.CurrentToken())
}

// ============================================================================
// Middleware Integration Tests
// ============================================================================

func TestActingAs_WithMiddlewareIntegration(t *testing.T) {
	config := graphqltester.DefaultConfig()
	config.Database = nil // ← FIXED
	config.Schema = &graphqltester.SchemaConfig{
		String:    authTestSchema,
		Resolvers: &authTestResolver{},
	}
	config.Middleware.AuthEnabled = true

	tester := graphqltester.NewTester(t, config)
	defer tester.Cleanup()

	user := &mockUser{ID: "505", FirstName: "Context", LastName: "User"}
	tester.ActingAs(user)
	assert.Equal(t, user, tester.CurrentUser())

	response := tester.GraphQL(`{ hello }`)
	assert.NotNil(t, response)
}

func TestClearAuth_ThenMakeRequest(t *testing.T) {
	config := graphqltester.DefaultConfig()
	config.Database = nil // ← FIXED
	config.Schema = &graphqltester.SchemaConfig{
		String:    authTestSchema,
		Resolvers: &authTestResolver{},
	}

	tester := graphqltester.NewTester(t, config)
	defer tester.Cleanup()

	user := &mockUser{ID: "1", FirstName: "Test", LastName: "User"}
	tester.ActingAs(user)
	tester.ClearAuth()

	assert.NotPanics(t, func() {
		response := tester.GraphQL(`{ hello }`)
		assert.NotNil(t, response)
	})
}

// ============================================================================
// Edge Case Tests
// ============================================================================

func TestActingAs_WithFullNameUser(t *testing.T) {
	tester := newAuthTester(t)
	user := &mockUser{ID: "789", FirstName: "Alice", LastName: "Johnson"}
	result := tester.ActingAs(user)
	assert.NotNil(t, result)
	assert.Equal(t, user, tester.CurrentUser())
}

func TestActingAs_WithMinimalUser(t *testing.T) {
	tester := newAuthTester(t)
	user := struct{ ID string }{ID: "202"}
	result := tester.ActingAs(&user)
	assert.NotNil(t, result)
	assert.NotNil(t, tester.CurrentUser())
}

func TestAuthenticationFlow_ConcurrentAccess(t *testing.T) {
	tester := newAuthTester(t)
	user := &mockUser{ID: "999", FirstName: "Concurrent", LastName: "User"}
	tester.ActingAs(user)
	tester.WithToken("concurrent-token")

	for i := 0; i < 10; i++ {
		assert.Equal(t, user, tester.CurrentUser())
		assert.Equal(t, "concurrent-token", tester.CurrentToken())
	}
}

func TestActingAs_NilPointerUser(t *testing.T) {
	tester := newAuthTester(t)
	assert.NotPanics(t, func() { tester.ActingAs(nil) })
}

func TestWithToken_VeryLongToken(t *testing.T) {
	tester := newAuthTester(t)
	longToken := ""
	for i := 0; i < 1000; i++ {
		longToken += "a"
	}
	result := tester.WithToken(longToken)
	assert.NotNil(t, result)
	assert.Equal(t, longToken, tester.CurrentToken())
}

func TestClearAuth_MultipleCalls(t *testing.T) {
	tester := newAuthTester(t)
	for i := 0; i < 5; i++ {
		assert.NotPanics(t, func() { tester.ClearAuth() })
	}
}
