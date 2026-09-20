package tests

import (
	"context"
	"testing"

	graphqltester "github.com/mwangaben/graphqltester"
	"github.com/mwangaben/graphqltester/pkg/adapters/database"
)

// ============================================================
// GraphQL Types (resolver wrappers around EntUser)
// ============================================================

// UserResolver is the GraphQL-facing type for User.
//
// It wraps EntUser so that field resolvers can return the exact
// Go types that graph-gophers expects (int32 for GraphQL Int).
type UserResolver struct {
	user *EntUser
}

func (r *UserResolver) ID() int32     { return int32(r.user.ID) }
func (r *UserResolver) Name() string  { return r.user.Name }
func (r *UserResolver) Email() string { return r.user.Email }
func (r *UserResolver) Age() int32    { return int32(r.user.Age) }
func (r *UserResolver) Role() *string {
	if r.user.Role == "" {
		return nil
	}
	return &r.user.Role
}

// ============================================================
// Query Resolver
// ============================================================

type TestResolver struct {
	adapter *database.EntAdapter
}

func (r *TestResolver) User(ctx context.Context, args struct{ ID int32 }) (*UserResolver, error) {
	record, err := r.adapter.GetRecord(ctx, "ent_factory_users", map[string]interface{}{
		"id": args.ID,
	})
	if err != nil {
		return nil, err
	}

	entUser := &EntUser{
		Name:  toString(record["name"]),
		Email: toString(record["email"]),
		Role:  toString(record["role"]),
	}

	if v, ok := record["id"].(int64); ok {
		entUser.ID = int(v)
	}
	if v, ok := record["age"].(int64); ok {
		entUser.Age = int(v)
	}

	return &UserResolver{user: entUser}, nil
}

func toString(v interface{}) string {
	switch s := v.(type) {
	case string:
		return s
	case []byte:
		return string(s)
	default:
		return ""
	}
}

// ============================================================
// Integration Test
// ============================================================

func TestEntAdapter_IntegrationWithTester(t *testing.T) {
	// Setup adapter
	adapter := setupEntDB(t)

	// Setup factory (reusing the existing factory from ent_factory_test.go)
	f := newEntDatabaseFactory()

	// Setup tester
	test := graphqltester.NewTester(t, &graphqltester.Config{
		Schema: &graphqltester.SchemaConfig{
			String: `
				type User {
					id: Int!
					name: String!
					email: String!
					age: Int!
					role: String
				}
				type Query {
					user(id: Int!): User
				}
			`,
			Resolvers: &TestResolver{adapter: adapter},
		},
		Database: &graphqltester.DatabaseConfig{
			Adapter: adapter,
			DSN:     getTestDSNEnt(),
		},
		Packages: &graphqltester.PackageConfig{
			Factory: f,
		},
	})

	// 1. Create a user via factory (in-memory)
	user := test.Factory("User").
		Create(map[string]interface{}{
			"name":  "Integration Test",
			"email": "integration@example.com",
		}).(*EntUser)

	// 2. Persist via adapter
	if err := adapter.Insert(context.Background(), "ent_factory_users", map[string]interface{}{
		"name":  user.Name,
		"email": user.Email,
		"age":   user.Age,
		"role":  user.Role,
	}); err != nil {
		t.Fatal(err)
	}

	// 3. Query via GraphQL
	response := test.GraphQL(`{ user(id: 1) { id name email age } }`)

	response.AssertOK().
		AssertNoErrors().
		AssertJSONPath("user.name", "Integration Test").
		AssertJSONPath("user.email", "integration@example.com")

	// 4. Assert database state
	response.AssertDatabaseHas("ent_factory_users", map[string]interface{}{
		"email": "integration@example.com",
	})

	t.Log("✅ EntAdapter + Tester + Factory + Assertions all working")
}
