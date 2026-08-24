package tests

import (
	"testing"
)

// ============================================================================
// Schema Definitions for JSON Assertion Testing
// ============================================================================

const jsonAssertionSchema = `
	schema {
		query: Query
	}
	type Query {
		hello: String!
		user: User!
		users: [User!]!
		searchUsers(filters: UserFilters): [User!]!
	}
	type User {
		id: String!
		name: String!
		email: String!
		age: Int!
		active: Boolean!
		address: Address!
		metadata: Metadata
	}
	type Address {
		street: String!
		city: String!
		country: String!
		zipCode: String!
	}
	type Metadata {
		createdAt: String!
		updatedAt: String!
		tags: [String!]!
	}
	input UserFilters {
		status: String   # ✅ Nullable
		search: String   # ✅ Nullable
	}
`

// ============================================================================
// Mock Resolvers for JSON Assertion Testing
// ============================================================================

type jsonAssertionResolver struct{}

func (r *jsonAssertionResolver) Hello() string {
	return "Hello, World!"
}

func (r *jsonAssertionResolver) User() *jsonUserResolver {
	return &jsonUserResolver{
		id:      "user-123",
		name:    "John Doe",
		email:   "john@example.com",
		age:     30,
		active:  true,
		street:  "123 Main St",
		city:    "New York",
		country: "USA",
		zipCode: "10001",
		created: "2024-01-15T10:00:00Z",
		updated: "2024-01-20T15:30:00Z",
		tags:    []string{"premium", "verified", "active"},
	}
}

func (r *jsonAssertionResolver) Users() []*jsonUserResolver {
	return []*jsonUserResolver{
		{
			id:      "user-1",
			name:    "Alice Johnson",
			email:   "alice@example.com",
			age:     25,
			active:  true,
			street:  "456 Oak Ave",
			city:    "Los Angeles",
			country: "USA",
			zipCode: "90001",
			created: "2024-01-10T08:00:00Z",
			updated: "2024-01-18T12:00:00Z",
			tags:    []string{"new", "active"},
		},
		{
			id:      "user-2",
			name:    "Bob Smith",
			email:   "bob@example.com",
			age:     35,
			active:  false,
			street:  "789 Pine St",
			city:    "Chicago",
			country: "USA",
			zipCode: "60601",
			created: "2023-12-01T09:00:00Z",
			updated: "2024-01-05T14:00:00Z",
			tags:    []string{"inactive", "legacy"},
		},
		{
			id:      "user-3",
			name:    "Carol White",
			email:   "carol@example.com",
			age:     28,
			active:  true,
			street:  "321 Elm Blvd",
			city:    "Miami",
			country: "USA",
			zipCode: "33101",
			created: "2024-01-12T11:00:00Z",
			updated: "2024-01-19T16:00:00Z",
			tags:    []string{"premium", "active"},
		},
	}
}

// ✅ Fix: Use pointer types for nullable fields
func (r *jsonAssertionResolver) SearchUsers(args struct {
	Filters *struct {
		Status *string `json:"status"`
		Search *string `json:"search"`
	} `json:"filters"`
}) []*jsonUserResolver {
	users := r.Users()

	// If no filters, return all
	if args.Filters == nil {
		return users
	}

	// Filter the results
	filtered := []*jsonUserResolver{}
	for _, user := range users {
		match := true

		// Check status filter
		if args.Filters.Status != nil {
			// Simple mock filtering - in real app, would check actual status
			// For test purposes, we'll just include all
		}

		// Check search filter
		if args.Filters.Search != nil && *args.Filters.Search != "" {
			// Simple mock filtering - in real app, would search by name/email
			// For test purposes, we'll just include all
		}

		if match {
			filtered = append(filtered, user)
		}
	}

	return filtered
}

type jsonUserResolver struct {
	id      string
	name    string
	email   string
	age     int
	active  bool
	street  string
	city    string
	country string
	zipCode string
	created string
	updated string
	tags    []string
}

func (r *jsonUserResolver) ID() string    { return r.id }
func (r *jsonUserResolver) Name() string  { return r.name }
func (r *jsonUserResolver) Email() string { return r.email }
func (r *jsonUserResolver) Age() int32    { return int32(r.age) }
func (r *jsonUserResolver) Active() bool  { return r.active }
func (r *jsonUserResolver) Address() *jsonAddressResolver {
	return &jsonAddressResolver{
		street:  r.street,
		city:    r.city,
		country: r.country,
		zipCode: r.zipCode,
	}
}
func (r *jsonUserResolver) Metadata() *jsonMetadataResolver {
	return &jsonMetadataResolver{
		created: r.created,
		updated: r.updated,
		tags:    r.tags,
	}
}

type jsonAddressResolver struct {
	street  string
	city    string
	country string
	zipCode string
}

func (r *jsonAddressResolver) Street() string  { return r.street }
func (r *jsonAddressResolver) City() string    { return r.city }
func (r *jsonAddressResolver) Country() string { return r.country }
func (r *jsonAddressResolver) ZipCode() string { return r.zipCode }

type jsonMetadataResolver struct {
	created string
	updated string
	tags    []string
}

func (r *jsonMetadataResolver) CreatedAt() string { return r.created }
func (r *jsonMetadataResolver) UpdatedAt() string { return r.updated }
func (r *jsonMetadataResolver) Tags() []string    { return r.tags }

// ============================================================================
// Tests for Improved AssertJSON and AssertJSONSubset
// ============================================================================

func TestAssertJSON_ExactMatch_Success(t *testing.T) {
	tester := newSimpleTester(t, jsonAssertionSchema, &jsonAssertionResolver{})
	response := tester.GraphQL(`{ user { id name email age active } }`)

	response.AssertJSON(map[string]interface{}{
		"user": map[string]interface{}{
			"id":     "user-123",
			"name":   "John Doe",
			"email":  "john@example.com",
			"age":    float64(30),
			"active": true,
		},
	})
}

func TestAssertJSON_ExactMatch_NestedSuccess(t *testing.T) {
	tester := newSimpleTester(t, jsonAssertionSchema, &jsonAssertionResolver{})
	response := tester.GraphQL(`{ user { id name address { street city country } } }`)

	response.AssertJSON(map[string]interface{}{
		"user": map[string]interface{}{
			"id":   "user-123",
			"name": "John Doe",
			"address": map[string]interface{}{
				"street":  "123 Main St",
				"city":    "New York",
				"country": "USA",
			},
		},
	})
}

func TestAssertJSON_ExactMatch_Failure(t *testing.T) {
	tester := newSimpleTester(t, jsonAssertionSchema, &jsonAssertionResolver{})
	response := tester.GraphQL(`{ user { id name email age active } }`)

	// This should fail and show detailed diff
	response.AssertJSON(map[string]interface{}{
		"user": map[string]interface{}{
			"id":     "user-123",
			"name":   "John Doe",
			"email":  "wrong@example.com", // Wrong email
			"age":    float64(30),
			"active": true,
		},
	})
}

func TestAssertJSON_ExactMatch_ArraySuccess(t *testing.T) {
	tester := newSimpleTester(t, jsonAssertionSchema, &jsonAssertionResolver{})
	response := tester.GraphQL(`{ users { id name age active } }`)

	response.AssertJSON(map[string]interface{}{
		"users": []interface{}{
			map[string]interface{}{
				"id":     "user-1",
				"name":   "Alice Johnson",
				"age":    float64(25),
				"active": true,
			},
			map[string]interface{}{
				"id":     "user-2",
				"name":   "Bob Smith",
				"age":    float64(35),
				"active": false,
			},
			map[string]interface{}{
				"id":     "user-3",
				"name":   "Carol White",
				"age":    float64(28),
				"active": true,
			},
		},
	})
}

func TestAssertJSON_ExactMatch_ArrayFailure(t *testing.T) {
	tester := newSimpleTester(t, jsonAssertionSchema, &jsonAssertionResolver{})
	response := tester.GraphQL(`{ users { id name age active } }`)

	// This should fail - wrong count and wrong data
	response.AssertJSON(map[string]interface{}{
		"users": []interface{}{
			map[string]interface{}{
				"id":     "user-1",
				"name":   "Alice Johnson",
				"age":    float64(25),
				"active": true,
			},
			map[string]interface{}{
				"id":     "user-2",
				"name":   "Wrong Name", // Wrong name
				"age":    float64(35),
				"active": false,
			},
		},
	})
}

func TestAssertJSONSubset_Success(t *testing.T) {
	tester := newSimpleTester(t, jsonAssertionSchema, &jsonAssertionResolver{})
	response := tester.GraphQL(`{ user { id name email age active address { city country } } }`)

	// Only check subset of fields - should pass
	response.AssertJSONSubset(map[string]interface{}{
		"user": map[string]interface{}{
			"id":   "user-123",
			"name": "John Doe",
			"address": map[string]interface{}{
				"city": "New York",
			},
		},
	})
}

func TestAssertJSONSubset_NestedSuccess(t *testing.T) {
	tester := newSimpleTester(t, jsonAssertionSchema, &jsonAssertionResolver{})
	response := tester.GraphQL(`{ user { id name metadata { createdAt tags } } }`)

	response.AssertJSONSubset(map[string]interface{}{
		"user": map[string]interface{}{
			"name": "John Doe",
			"metadata": map[string]interface{}{
				"createdAt": "2024-01-15T10:00:00Z",
				"tags": []interface{}{
					"premium",
					"verified",
					"active",
				},
			},
		},
	})
}

func TestAssertJSONSubset_ArraySuccess(t *testing.T) {
	tester := newSimpleTester(t, jsonAssertionSchema, &jsonAssertionResolver{})
	response := tester.GraphQL(`{ users { id name email } }`)

	// Check subset of array elements
	response.AssertJSONSubset(map[string]interface{}{
		"users": []interface{}{
			map[string]interface{}{
				"id":   "user-1",
				"name": "Alice Johnson",
			},
			map[string]interface{}{
				"id":   "user-2",
				"name": "Bob Smith",
			},
		},
	})
}

func TestAssertJSONSubset_Failure_MissingKey(t *testing.T) {
	tester := newSimpleTester(t, jsonAssertionSchema, &jsonAssertionResolver{})
	response := tester.GraphQL(`{ user { id name email } }`)

	// This should fail - 'age' key doesn't exist in response
	response.AssertJSONSubset(map[string]interface{}{
		"user": map[string]interface{}{
			"id":   "user-123",
			"name": "John Doe",
			"age":  float64(30), // This key doesn't exist in the query
		},
	})
}

func TestAssertJSONSubset_Failure_WrongValue(t *testing.T) {
	tester := newSimpleTester(t, jsonAssertionSchema, &jsonAssertionResolver{})
	response := tester.GraphQL(`{ user { id name email } }`)

	// This should fail - wrong email value
	response.AssertJSONSubset(map[string]interface{}{
		"user": map[string]interface{}{
			"id":    "user-123",
			"name":  "John Doe",
			"email": "wrong@example.com",
		},
	})
}

func TestAssertJSONSubset_Failure_ArrayMismatch(t *testing.T) {
	tester := newSimpleTester(t, jsonAssertionSchema, &jsonAssertionResolver{})
	response := tester.GraphQL(`{ users { id name } }`)

	// This should fail - expecting 2 users but got 3
	response.AssertJSONSubset(map[string]interface{}{
		"users": []interface{}{
			map[string]interface{}{
				"id":   "user-1",
				"name": "Alice Johnson",
			},
			map[string]interface{}{
				"id":   "user-2",
				"name": "Bob Smith",
			},
			map[string]interface{}{
				"id":   "user-3",
				"name": "Wrong Name", // Wrong name and extra element
			},
		},
	})
}

func TestAssertJSONSubset_Failure_WrongPath(t *testing.T) {
	tester := newSimpleTester(t, jsonAssertionSchema, &jsonAssertionResolver{})
	response := tester.GraphQL(`{ user { id name } }`)

	// This should fail - 'address' key doesn't exist
	response.AssertJSONSubset(map[string]interface{}{
		"user": map[string]interface{}{
			"id":   "user-123",
			"name": "John Doe",
			"address": map[string]interface{}{
				"city": "New York",
			},
		},
	})
}

func TestAssertJSONSubset_WithFilters_Success(t *testing.T) {
	tester := newSimpleTester(t, jsonAssertionSchema, &jsonAssertionResolver{})
	response := tester.GraphQL(`
		query($filters: UserFilters) {
			searchUsers(filters: $filters) {
				id
				name
				email
			}
		}
	`, map[string]interface{}{
		"filters": map[string]interface{}{
			"status": "active",
		},
	})

	response.AssertJSONSubset(map[string]interface{}{
		"searchUsers": []interface{}{
			map[string]interface{}{
				"id":   "user-1",
				"name": "Alice Johnson",
			},
		},
	})
}

func TestAssertJSONSubset_ComplexNested_Failure(t *testing.T) {
	tester := newSimpleTester(t, jsonAssertionSchema, &jsonAssertionResolver{})
	response := tester.GraphQL(`{ user { id name address { street city country zipCode } metadata { createdAt tags } } }`)

	// This should fail - multiple mismatches
	response.AssertJSONSubset(map[string]interface{}{
		"user": map[string]interface{}{
			"id":   "user-123",
			"name": "John Doe",
			"address": map[string]interface{}{
				"street":  "Wrong Street",
				"city":    "Wrong City",
				"country": "USA",
			},
			"metadata": map[string]interface{}{
				"createdAt": "2024-01-15T10:00:00Z",
				"tags": []interface{}{
					"premium",
					"wrong-tag", // Wrong tag
				},
			},
		},
	})
}

func TestAssertJSONSubset_WithNullValues(t *testing.T) {
	tester := newSimpleTester(t, jsonAssertionSchema, &jsonAssertionResolver{})
	response := tester.GraphQL(`{ user { id name metadata { tags } } }`)

	// Test with null values - should pass if field is null
	response.AssertJSONSubset(map[string]interface{}{
		"user": map[string]interface{}{
			"id":   "user-123",
			"name": "John Doe",
			// metadata.tags is not null, but we're not checking it
		},
	})
}

func TestAssertJSONSubset_WithEmptyArray(t *testing.T) {
	tester := newSimpleTester(t, jsonAssertionSchema, &jsonAssertionResolver{})
	response := tester.GraphQL(`{ users { id name } }`)

	// Checking for empty array when it's not empty - should fail
	response.AssertJSONSubset(map[string]interface{}{
		"users": []interface{}{},
	})
}

func TestAssertJSON_MultipleAssertions_Chaining(t *testing.T) {
	tester := newSimpleTester(t, jsonAssertionSchema, &jsonAssertionResolver{})
	response := tester.GraphQL(`{ user { id name email address { city } } }`)

	// Chain multiple assertions
	response.
		AssertJSONSubset(map[string]interface{}{
			"user": map[string]interface{}{
				"id":   "user-123",
				"name": "John Doe",
			},
		}).
		AssertJSONSubset(map[string]interface{}{
			"user": map[string]interface{}{
				"address": map[string]interface{}{
					"city": "New York",
				},
			},
		})
}

func TestAssertJSON_WithDifferentTypes(t *testing.T) {
	tester := newSimpleTester(t, jsonAssertionSchema, &jsonAssertionResolver{})
	response := tester.GraphQL(`{ user { id name age active } }`)

	// Test with different data types
	response.AssertJSONSubset(map[string]interface{}{
		"user": map[string]interface{}{
			"id":     "user-123",
			"name":   "John Doe",
			"age":    float64(30),
			"active": true,
		},
	})
}

func TestAssertJSONSubset_WithMultipleObjects(t *testing.T) {
	tester := newSimpleTester(t, jsonAssertionSchema, &jsonAssertionResolver{})
	response := tester.GraphQL(`{ users { id name email } }`)

	// Check multiple objects in array with subset of fields
	response.AssertJSONSubset(map[string]interface{}{
		"users": []interface{}{
			map[string]interface{}{
				"name": "Alice Johnson",
			},
			map[string]interface{}{
				"name": "Bob Smith",
			},
			map[string]interface{}{
				"name": "Carol White",
			},
		},
	})
}
