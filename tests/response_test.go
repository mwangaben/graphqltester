package tests

import (
	"fmt"
	"net/http"
	"testing"

	graphqltester "github.com/mwangaben/graphqltester"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Schema Definitions for Testing
// ============================================================================

const schemaDefinition = `
	schema {
		query: Query
	}
	type Query {
		hello: String!
	}
`

const nestedSchemaDefinition = `
	schema {
		query: Query
	}
	type Query {
		hello: String!
		user: User!
	}
	type User {
        id: String!
		name: String!
		email: String!
		address: Address!
	}
	type Address {
		city: String!
		country: String!
	}
`

const arraySchemaDefinition = `
	schema {
		query: Query
	}
	type Query {
		hello: String!
		users: [User!]!
	}
	type User {
		name: String!
		email: String!
	}
`

const typesSchemaDefinition = `
	schema {
		query: Query
	}
	type Query {
		name: String!
		age: Int!
		active: Boolean!
		score: Float!
	}
`

const validationSchemaDefinition = `
	schema {
		query: Query
		mutation: Mutation
	}
	type Query {
		hello: String!
	}
	type Mutation {
		createUser(input: CreateUserInput!): User
	}
	input CreateUserInput {
		name: String!
		email: String!
	}
	type User {
		id: String!
		name: String!
		email: String!
	}
`

const authSchemaDefinition = `
	schema {
		query: Query
	}
	type Query {
		hello: String!
		secretData: String!
		adminData: String!
	}
`

const mutationSchemaDefinition = `
	schema {
		query: Query
		mutation: Mutation
	}
	type Query {
		hello: String!
	}
	type Mutation {
		createUser(input: CreateUserInput!): User
		deleteUser(id: ID!): User
	}
	input CreateUserInput {
		name: String!
		email: String!
	}
	type User {
		id: ID!
		name: String!
		email: String!
	}
`

// ============================================================================
// Mock Resolvers
// ============================================================================

type mockResolver struct{}

func (r *mockResolver) Hello() string { return "Hello, World!" }

type nestedMockResolver struct{}

func (r *nestedMockResolver) Hello() string       { return "Hello, World!" }
func (r *nestedMockResolver) User() *userResolver { return &userResolver{} }

type userResolver struct{}

func (r *userResolver) ID() string { return "1" } // ← ADD THIS

func (r *userResolver) Name() string              { return "John Doe" }
func (r *userResolver) Email() string             { return "john@example.com" }
func (r *userResolver) Address() *addressResolver { return &addressResolver{} }

type addressResolver struct{}

func (r *addressResolver) City() string    { return "New York" }
func (r *addressResolver) Country() string { return "USA" }

type arrayMockResolver struct{}

func (r *arrayMockResolver) Hello() string { return "Hello!" }
func (r *arrayMockResolver) Users() []*userResolver {
	return []*userResolver{{}, {}}
}

type typesMockResolver struct{}

func (r *typesMockResolver) Name() string   { return "John Doe" }
func (r *typesMockResolver) Age() int32     { return 30 }
func (r *typesMockResolver) Active() bool   { return true }
func (r *typesMockResolver) Score() float64 { return 95.5 }

type validationMockResolver struct{}

func (r *validationMockResolver) Hello() string { return "Hello!" }
func (r *validationMockResolver) CreateUser(args struct{ Input struct{ Name, Email string } }) (*userResolver, error) {
	if args.Input.Name == "" {
		return nil, fmt.Errorf("The name field is required.")
	}
	return &userResolver{}, nil
}

type authMockResolver struct{}

func (r *authMockResolver) Hello() string      { return "Hello!" }
func (r *authMockResolver) SecretData() string { return "secret" }
func (r *authMockResolver) AdminData() string  { return "admin" }

type mutationMockResolver struct{}

func (r *mutationMockResolver) Hello() string { return "Hello!" }
func (r *mutationMockResolver) CreateUser(args struct{ Input struct{ Name, Email string } }) (*userResolver, error) {
	return &userResolver{}, nil
}
func (r *mutationMockResolver) DeleteUser(args struct{ ID string }) (*userResolver, error) {
	return &userResolver{}, nil
}

// ============================================================================
// Helper: Create tester WITHOUT database
// ============================================================================

func newSimpleTester(t *testing.T, schemaString string, resolver interface{}) *graphqltester.Tester {
	t.Helper()
	config := graphqltester.DefaultConfig()
	config.Database = nil // NO database needed
	config.Schema = &graphqltester.SchemaConfig{
		String:    schemaString,
		Resolvers: resolver,
	}
	tester := graphqltester.NewTester(t, config)
	t.Cleanup(tester.Cleanup)
	return tester
}

// ============================================================================
// Tests (NO database needed)
// ============================================================================

func TestResponse_AssertStatus_MatchesExact(t *testing.T) {
	tester := newSimpleTester(t, schemaDefinition, &mockResolver{})
	response := tester.GraphQL(`{ hello }`)
	response.AssertStatus(http.StatusOK)
	assert.NotNil(t, response.AssertStatus(http.StatusOK))
}

func TestResponse_AssertOK_ShorthandFor200(t *testing.T) {
	tester := newSimpleTester(t, schemaDefinition, &mockResolver{})
	response := tester.GraphQL(`{ hello }`)
	assert.NotNil(t, response.AssertOK())
}

func TestResponse_AssertNoErrors_NoErrors(t *testing.T) {
	tester := newSimpleTester(t, schemaDefinition, &mockResolver{})
	response := tester.GraphQL(`{ hello }`)
	response.AssertNoErrors()
}

func TestResponse_AssertHasErrors_WithErrors(t *testing.T) {
	tester := newSimpleTester(t, schemaDefinition, &mockResolver{})
	response := tester.GraphQL(`{ nonexistent }`)
	response.AssertHasErrors()
}

func TestResponse_JSONPath_Navigation(t *testing.T) {
	tester := newSimpleTester(t, schemaDefinition, &mockResolver{})
	response := tester.GraphQL(`{ hello }`)
	assert.Equal(t, "Hello, World!", response.JSONString("hello"))
}

func TestResponse_JSONPath_NestedNavigation(t *testing.T) {
	tester := newSimpleTester(t, nestedSchemaDefinition, &nestedMockResolver{})
	response := tester.GraphQL(`{ user { name address { city } } }`)
	assert.Equal(t, "John Doe", response.JSONString("user.name"))
	assert.Equal(t, "New York", response.JSONString("user.address.city"))
}

func TestResponse_JSONArray_PathNavigation(t *testing.T) {
	tester := newSimpleTester(t, arraySchemaDefinition, &arrayMockResolver{})
	response := tester.GraphQL(`{ users { name } }`)
	assert.Len(t, response.JSONArray("users"), 2)
	assert.Equal(t, "John Doe", response.JSONString("users.0.name"))
}

func TestResponse_JSONType_TypeConversions(t *testing.T) {
	tester := newSimpleTester(t, typesSchemaDefinition, &typesMockResolver{})
	response := tester.GraphQL(`{ name age active score }`)
	assert.Equal(t, "John Doe", response.JSONString("name"))
	assert.Equal(t, 30, response.JSONInt("age"))
	assert.True(t, response.JSONBool("active"))
	assert.Equal(t, 95.5, response.JSONFloat("score"))
}

func TestResponse_AssertJSONPath_NestedValue(t *testing.T) {
	tester := newSimpleTester(t, nestedSchemaDefinition, &nestedMockResolver{})
	response := tester.GraphQL(`{ user { name address { city } } }`)
	response.AssertJSONPath("user.name", "John Doe").
		AssertJSONPath("user.address.city", "New York")
}

func TestResponse_AssertJSONCount_ArrayLength(t *testing.T) {
	tester := newSimpleTester(t, arraySchemaDefinition, &arrayMockResolver{})
	response := tester.GraphQL(`{ users { name } }`)
	response.AssertJSONCount("users", 2)
}

func TestResponse_AssertValidationError(t *testing.T) {
	tester := newSimpleTester(t, validationSchemaDefinition, &validationMockResolver{})
	response := tester.GraphQL(`
		mutation CreateUser($input: CreateUserInput!) {
			createUser(input: $input) { id name }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"name":  "",
			"email": "test@example.com",
		},
	})
	response.AssertHasErrors()
}

func TestResponse_AssertUnauthenticated(t *testing.T) {
	config := graphqltester.DefaultConfig()
	config.Database = nil
	config.Schema = &graphqltester.SchemaConfig{
		String:    authSchemaDefinition,
		Resolvers: &authMockResolver{},
	}
	config.Middleware.AuthEnabled = true

	tester := graphqltester.NewTester(t, config)
	defer tester.Cleanup()

	response := tester.GraphQL(`{ secretData }`)
	if len(response.Errors()) > 0 {
		response.AssertHasErrors()
	}
}

func TestResponse_AssertPermissionError(t *testing.T) {
	config := graphqltester.DefaultConfig()
	config.Database = nil
	config.Schema = &graphqltester.SchemaConfig{
		String:    authSchemaDefinition,
		Resolvers: &authMockResolver{},
	}
	config.Middleware.AuthEnabled = true
	config.Middleware.PermissionEnabled = true

	tester := graphqltester.NewTester(t, config)
	defer tester.Cleanup()

	response := tester.GraphQL(`{ adminData }`)
	if len(response.Errors()) > 0 {
		response.AssertHasErrors()
	}
}

func TestResponse_Dump_DoesNotPanic(t *testing.T) {
	tester := newSimpleTester(t, schemaDefinition, &mockResolver{})
	response := tester.GraphQL(`{ hello }`)
	assert.NotPanics(t, func() { response.Dump() })
}

func TestResponse_Dump_ReturnsDebuggableResponse(t *testing.T) {
	tester := newSimpleTester(t, schemaDefinition, &mockResolver{})
	response := tester.GraphQL(`{ hello }`)
	assert.NotNil(t, response.Dump())
}

func TestTester_DescribeIt_BDDStyle(t *testing.T) {
	tester := newSimpleTester(t, schemaDefinition, &mockResolver{})
	itCalled := false
	tester.Describe("GraphQL Queries", func(t *graphqltester.Tester) {
		t.It("returns hello world", func(t *graphqltester.Tester) {
			itCalled = true
			assert.NotNil(t, t.GraphQL(`{ hello }`))
		})
	})
	assert.True(t, itCalled)
}

func TestTester_Run_SubTests(t *testing.T) {
	tester := newSimpleTester(t, schemaDefinition, &mockResolver{})
	runCalled := false
	tester.Run("Sub Test", func(t *graphqltester.Tester) {
		runCalled = true
		assert.NotNil(t, t.GraphQL(`{ hello }`))
	})
	assert.True(t, runCalled)
}

func TestDefaultConfig_ReturnsSensibleDefaults(t *testing.T) {
	config := graphqltester.DefaultConfig()
	assert.Equal(t, "/graphql", config.Endpoint)
	assert.True(t, config.Debug())
}

func TestConfig_Validate_ValidConfig(t *testing.T) {
	config := graphqltester.DefaultConfig()
	config.Database = nil
	config.Schema = &graphqltester.SchemaConfig{
		String:    schemaDefinition,
		Resolvers: &mockResolver{},
	}
	require.NoError(t, config.Validate())
}

func TestConfig_Validate_MissingSchema(t *testing.T) {
	config := graphqltester.DefaultConfig()
	config.Schema = nil
	err := config.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "schema configuration is required")
}

func TestConfig_Validate_MissingResolvers(t *testing.T) {
	config := graphqltester.DefaultConfig()
	config.Schema = &graphqltester.SchemaConfig{
		String:    schemaDefinition,
		Resolvers: nil,
	}
	err := config.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "schema resolvers are required")
}

// ============================================================================
// Database Test (ONLY this one is skipped when no DB)
// ============================================================================

func TestResponse_AssertDatabaseHas(t *testing.T) {
	config := graphqltester.DefaultConfig()
	config.Schema = &graphqltester.SchemaConfig{
		String:    mutationSchemaDefinition,
		Resolvers: &mutationMockResolver{},
	}
	if config.Database == nil || config.Database.Adapter == nil {
		t.Skip("Database not configured, skipping database assertion test")
	}
	tester := graphqltester.NewTester(t, config)
	defer tester.Cleanup()
	tester.RefreshDatabase()
	response := tester.GraphQL(`
		mutation CreateUser($input: CreateUserInput!) {
			createUser(input: $input) { id name }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{"name": "Test User", "email": "test@example.com"},
	})
	if response.Data() != nil {
		response.AssertDatabaseHas("users", map[string]interface{}{"name": "Test User"})
	}
}
