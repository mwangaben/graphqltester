package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	graphqltester "github.com/mwangaben/graphqltester"
)

// ============================================================================
// Subscription Auth Tests
// ============================================================================

const authSubTestSchema = `
	scalar Int

	type User {
		id: Int!
		name: String!
		email: String!
	}

	type Query {
		me: User
	}

	type Subscription {
		userCreated: User!
	}
`

// AuthTestUser uses exported fields that map directly to GraphQL fields.
type AuthTestUser struct {
	ID    int32
	Name  string
	Email string
}

// ============================================================================
// Resolvers
// ============================================================================

// AuthTestResolver rejects subscriptions without an authenticated user.
type AuthTestResolver struct {
	events chan *AuthTestUser
}

func (r *AuthTestResolver) Me() *AuthTestUser { return nil }

func (r *AuthTestResolver) UserCreated(ctx context.Context) (<-chan *AuthTestUser, error) {
	if ctx.Value("user") == nil {
		return nil, errors.New("Unauthenticated.")
	}
	return r.events, nil
}

// authenticatedResolver always returns events (no auth check).
type authenticatedResolver struct {
	events chan *AuthTestUser
}

func (r *authenticatedResolver) Me() *AuthTestUser { return nil }

func (r *authenticatedResolver) UserCreated(ctx context.Context) (<-chan *AuthTestUser, error) {
	return r.events, nil
}

// ============================================================================
// Tests
// ============================================================================

func TestSubscription_Unauthenticated(t *testing.T) {
	events := make(chan *AuthTestUser, 10)
	defer close(events)

	config := graphqltester.DefaultConfig().
		WithDebug(true).
		LoadSchemaFromString(authSubTestSchema, &AuthTestResolver{events: events})

	config.Database = nil // no database needed

	test := graphqltester.NewTester(t, config)

	client, sub := test.Subscribe(`
		subscription {
			userCreated { id name email }
		}
	`, nil)
	defer client.Disconnect()

	// Expect an Unauthenticated error
	sub.ExpectError("Unauthenticated", 3*time.Second)

	t.Log("✅ Unauthenticated subscription correctly rejected")
}

func TestSubscription_Authenticated(t *testing.T) {
	events := make(chan *AuthTestUser, 10)
	defer close(events)

	config := graphqltester.DefaultConfig().
		WithDebug(true).
		LoadSchemaFromString(authSubTestSchema, &authenticatedResolver{events: events})

	config.Database = nil // no database needed

	test := graphqltester.NewTester(t, config)

	// Set a token (tester will send it over WS)
	test.WithToken("fake-jwt-token-for-testing")

	client, sub := test.Subscribe(`
		subscription {
			userCreated { id name email }
		}
	`, nil)
	defer client.Disconnect()

	// Give the subscription time to start
	time.Sleep(100 * time.Millisecond)

	// Emit an event
	events <- &AuthTestUser{
		ID:    1,
		Name:  "Test User",
		Email: "test@example.com",
	}

	// Expect the event
	sub.ExpectMessageContains(map[string]interface{}{
		"userCreated": map[string]interface{}{
			"name":  "Test User",
			"email": "test@example.com",
		},
	}, 3*time.Second)

	t.Log("✅ Authenticated subscription received the event")
}
