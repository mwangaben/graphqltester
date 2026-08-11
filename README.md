# GraphQL Tester for Go

[![Go Reference](https://pkg.go.dev/badge/github.com/mwangaben/graphql-tester.svg)](https://pkg.go.dev/github.com/mwangaben/graphql-tester)
[![Go Report Card](https://goreportcard.com/badge/github.com/mwangaben/graphql-tester)](https://goreportcard.com/report/github.com/mwangaben/graphql-tester)
[![Tests](https://github.com/mwangaben/graphql-tester/workflows/Tests/badge.svg)](https://github.com/mwangaben/graphql-tester/actions)

A comprehensive, production-ready GraphQL API testing framework for Go, inspired by [PHP Pest](https://pestphp.com/) and [Laravel Lighthouse](https://lighthouse-php.com/). Test your GraphQL queries, mutations, and subscriptions with elegant, fluent assertions.

## Features

- 🚀 **Fluent API** - Chainable assertions with readable, expressive syntax
- 🔐 **Authentication Testing** - Built-in support for testing auth, roles, and permissions
- 📡 **Subscription Testing** - WebSocket support for real-time GraphQL subscriptions
- 🗄️ **Database Assertions** - Verify database state after GraphQL operations
- ✅ **Validation Testing** - Test GraphQL validation rules and error messages
- 🏗️ **Multiple HTTP Frameworks** - Supports net/http, Gin, Echo, and Chi
- 💾 **Multiple Database Adapters** - GORM, SQLx, and raw MySQL support
- 🔄 **Transaction Isolation** - Automatic transaction rollback for test isolation
- 🏭 **Factory Integration** - Seamless integration with model factories
- 🏢 **Multi-Tenancy** - Test tenant-specific behavior and isolation
- ⚡ **Parallel Testing** - Opt-in concurrent test execution
- 🐛 **Debug Mode** - Verbose logging of requests, responses, and middleware
- 📝 **BDD Style** - Describe/It pattern for readable test organization

## Installation

```bash
go get github.com/mwangaben/graphql-tester
```

## Quick Start

```go
package myapp_test

import (
    "testing"
    
    tester "github.com/mwangaben/graphql-tester"
)

func TestZones(t *testing.T) {
    // Create tester with default configuration
    test := tester.NewTester(t, &tester.Config{
        Schema: &tester.SchemaConfig{
            Path: "./schema.graphql",
            Resolvers: &MyResolver{},
        },
        Database: &tester.DatabaseConfig{
            Adapter: &database.GORMAdapter{},
            DSN:     "root:password@tcp(localhost:3306)/testdb?parseTime=true",
        },
    })
    
    // Test a query
    test.GivenAdmin().
        GraphQL(`{ zones { name code } }`).
        AssertStatus(200).
        AssertNoErrors().
        AssertJSONCount("data.zones", 3)
    
    // Test a mutation
    test.GraphQL(`
        mutation CreateZone($input: CreateZoneInput!) {
            createZone(input: $input) {
                slug
                name
                code
                status
            }
        }
    `, map[string]interface{}{
        "input": map[string]interface{}{
            "name": "Industrial Zone",
            "code": "IND001",
            "status": "ACTIVE",
        },
    }).
        AssertNoErrors().
        AssertJSONPath("data.createZone.name", "Industrial Zone").
        AssertDatabaseHas("zones", map[string]interface{}{
            "name": "Industrial Zone",
        })
    
    // Test validation errors
    test.GraphQL(`
        mutation CreateZone($input: CreateZoneInput!) {
            createZone(input: $input) { slug }
        }
    `, map[string]interface{}{
        "input": map[string]interface{}{
            "name": "",
            "code": "",
        },
    }).
        AssertValidationError("input.name", "The zone name field is required.").
        AssertValidationError("input.code", "The zone code field is required.")
    
    // Test permissions
    test.GivenUser("user", "zone.view").  // Has view, but not create
        GraphQL(`
            mutation CreateZone($input: CreateZoneInput!) {
                createZone(input: $input) { slug }
            }
        `, map[string]interface{}{
            "input": map[string]interface{}{
                "name": "Test Zone",
                "code": "TST001",
            },
        }).
        AssertPermissionError()
}
```

### Authentication Patterns

### Acting as a User

```go
// Create and authenticate as an admin
test.SignInAdmin()

// Or use an existing user
user := User{ID: "123", Name: "John"}
test.SignInAdmin(user)

// BDD-style
test.GivenAdmin()

// Sign in with specific role and permissions
test.SignInUser("editor", "posts.create")

// With an existing user
test.SignInUser("editor", "posts.edit", existingUser)

// BDD-style
test.GivenUser("editor", "posts.create")
```

#### Token-Based Authentication

```go
token := "eyJhbGciOiJIUzI1NiIs..."
test.WithToken(token)
```

### Assertions

#### HTTP Status Assertions

```go
test.GraphQL(`...`).
    AssertOK().           // 200
    AssertCreated().      // 201
    AssertUnauthorized(). // 401
    AssertForbidden().    // 403
    AssertNotFound().     // 404
    AssertServerError()   // 5xx
```

#### GraphQL Error Assertions

```go
test.GraphQL(`...`).
    AssertNoErrors().
    AssertHasErrors().
    AssertErrorCount(3).
    AssertErrorMessage("Unauthenticated.").
    AssertErrorContains("already exists").
    AssertErrorCategory("validation")
```

### Data Assertions

```go
test.GraphQL(`...`).
    AssertJSON(map[string]interface{}{
        "data": map[string]interface{}{
            "user": map[string]interface{}{
                "name": "John Doe",
            },
        },
    }).
    AssertJSONSubset(map[string]interface{}{
        "data": map[string]interface{}{
            "user": map[string]interface{}{
                "name": "John Doe",
            },
        },
    }).
    AssertJSONPath("data.user.name", "John Doe").
    AssertJSONPath("data.user.address.city", "New York").
    AssertJSONCount("data.users", 5).
    AssertJSONNotEmpty("data.users").
    AssertJSONEmpty("data.users")
```

### Validation Assertions

```go
test.GraphQL(`...`).
    AssertValidationError("input.name", "The name field is required.").
    AssertValidationRules(map[string]string{
        "input.name": "required",
        "input.email": "invalid email",
    }).
    AssertValidationErrors(2).
    AssertValidationFields("input.name", "input.email").
    AssertNoValidationErrors().
    AssertValidationErrorCode("input.name", "REQUIRED")
```

### Permission Assertions

```go
test.GraphQL(`...`).
    AssertUnauthenticated().
    AssertForbidden().
    AssertPermissionError().
    AssertPermissionDenied("zones.delete")
```


### Database Assertions

```go
test.GraphQL(`...`).
    AssertDatabaseHas("zones", map[string]interface{}{
        "name": "Test Zone",
    }).
    AssertDatabaseMissing("zones", map[string]interface{}{
        "id": 999,
    }).
    AssertSoftDeleted("zones", map[string]interface{}{
        "id": zoneID,
    }).
    AssertNotSoftDeleted("zones", map[string]interface{}{
        "id": zoneID,
    }).
    AssertDatabaseCount("zones", 5).
    AssertDatabaseCountWhere("zones", map[string]interface{}{
        "status": "ACTIVE",
    }, 3).
    AssertDatabaseValue("zones",
        map[string]interface{}{"id": zoneID},
        "name",
        "Updated Zone",
    )
```

### Subscription Testing

```go
// Subscribe to events
client, sub := test.Subscribe(`
    subscription ZoneCreated {
        zoneCreated {
            slug
            name
            code
        }
    }
`, nil)
defer client.Disconnect()

// Trigger the subscription
test.GraphQL(`
    mutation CreateZone($input: CreateZoneInput!) {
        createZone(input: $input) {
            slug
        }
    }
`, map[string]interface{}{
    "input": map[string]interface{}{
        "name": "Subscription Zone",
        "code": "SUB001",
    },
})

// Assert the subscription received the event
sub.ExpectMessage(map[string]interface{}{
    "zoneCreated": map[string]interface{}{
        "name": "Subscription Zone",
        "code": "SUB001",
    },
}, 5*time.Second)

// Or use assertion helpers
test.AssertSubscription(sub).
    AssertDataPath("zoneCreated.name", "Subscription Zone").
    AssertDataPath("zoneCreated.code", "SUB001").
    AssertNoErrors()
```

### BDD Testing Style

```go
test.Describe("Zone CRUD Operations", func(t *tester.Tester) {
    
    t.BeforeEach(func() {
        t.RefreshDatabase().GivenAdmin()
    })
    
    t.It("can list zones with pagination", func(t *tester.Tester) {
        t.Factory("Zone").CreateMany(5)
        
        t.GraphQL(`{ zones(first: 3, page: 1) { data { name } } }`).
            AssertNoErrors().
            AssertJSONCount("data.zones.data", 3)
    })
    
    t.It("can create a new zone", func(t *tester.Tester) {
        t.GraphQL(`
            mutation CreateZone($input: CreateZoneInput!) {
                createZone(input: $input) { slug name }
            }
        `, map[string]interface{}{
            "input": map[string]interface{}{
                "name": "New Zone",
                "code": "NEW001",
            },
        }).
            AssertNoErrors().
            AssertJSONPath("data.createZone.name", "New Zone")
    })
    
    t.It("validates required fields", func(t *tester.Tester) {
        t.GraphQL(`
            mutation CreateZone($input: CreateZoneInput!) {
                createZone(input: $input) { slug }
            }
        `, map[string]interface{}{
            "input": map[string]interface{}{
                "name": "",
                "code": "",
            },
        }).
            AssertValidationError("input.name", "required").
            AssertValidationError("input.code", "required")
    })
})
```

### Parallel Testing

```go
test.RunParallel([]func(*tester.IsolatedTester){
    func(it *tester.IsolatedTester) {
        it.RefreshDatabase().GivenAdmin()
        it.GraphQL(`...`).AssertNoErrors()
    },
    func(it *tester.IsolatedTester) {
        it.RefreshDatabase().GivenUser("editor", "posts.create")
        it.GraphQL(`...`).AssertPermissionError()
    },
}, &tester.ConcurrentConfig{
    MaxParallel: 4,
    Timeout:     30 * time.Second,
})
```


### Configuration

#### Full Configuration Example

```go
config := &tester.Config{
    Endpoint: "/graphql",
    Debug:    true,
    
    // HTTP Framework
    HTTPAdapter: &http.GinAdapter{},
    
    // Schema
    Schema: &tester.SchemaConfig{
        Path:      "./schema.graphql",
        Resolvers: &MyResolver{},
        Options: []graphql.SchemaOpt{
            graphql.MaxDepth(15),
            graphql.MaxParallelism(20),
        },
    },
    
    // Database
    Database: &tester.DatabaseConfig{
        Adapter:         &database.GORMAdapter{},
        DSN:             "root:password@tcp(localhost:3306)/testdb?parseTime=true",
        AutoMigrate:     true,
        UseTransactions: true,
    },
    
    // Middleware
    Middleware: &tester.MiddlewareConfig{
        AuthEnabled:       true,
        PermissionEnabled: true,
        ValidationEnabled: true,
    },
    
    // Packages Integration
    Packages: &tester.PackageConfig{
        Factory:    myFactory,
        Permission: myPermissionManager,
        Validation: myValidator,
    },
}
```

## Framework Adapters

### Standard net/http

```go
HTTPAdapter: &http.NetHTTPAdapter{}
```

### Gin

```go
HTTPAdapter: http.NewGinAdapter()
```

### Echo
```go
HTTPAdapter: http.NewEchoAdapter(false)
```

### Chi
```go
HTTPAdapter: http.NewChiAdapter()
```

### Database Adapters


#### GORM

```go
adapter := database.NewGORMAdapter(&database.GORMConfig{
    PrepareStmt: true,
})
adapter.AddModel(&User{}).AddModel(&Zone{})
```

#### SQLx
```go
adapter := database.NewSQLxAdapter("mysql")
adapter.AddTable("users").AddTable("zones")
```

### Raw MySQL

```go
adapter := database.NewMySQLAdapter()
adapter.AddTable("users", `CREATE TABLE users (...)`)
```

### Integration with Existing Packages

This package is designed to work seamlessly with your existing packages:

- [mwangaben/factory](https://github.com/mwangaben/factory) - Model factories for test data
- [mwangaben/permission](https://github.com/mwangaben/permission) - Role and permission management
- [mwangaben/validation](https://github.com/mwangaben/validation) - Input validation

```go
config := &tester.Config{
    Packages: &tester.PackageConfig{
        Factory:    myFactory,
        Permission: myPermissionManager,
        Validation: myValidator,
    },
}
```

### License
#### MIT License.

### Package structure 

```text

This completes the major components of the GraphQL Tester package. The package now includes:

1. ✅ **Configuration Management** (`config.go`)
2. ✅ **Main Tester** (`tester.go`)
3. ✅ **HTTP Client** (`client.go`)
4. ✅ **Response Handling** (`response.go`)
5. ✅ **Authentication** (`auth.go`)
6. ✅ **Factory Integration** (`factory.go`)
7. ✅ **Database Management** (`database.go`)
8. ✅ **Validation Assertions** (`assertions/validation.go`)
9. ✅ **Permission Assertions** (`assertions/permission.go`)
10. ✅ **Database Assertions** (`assertions/database.go`)
11. ✅ **Middleware Chain** (`pkg/middleware/chain.go`)
12. ✅ **Context Propagation** (`pkg/middleware/context.go`)
13. ✅ **HTTP Adapters** (`pkg/adapters/http/`)
14. ✅ **Database Adapters** (`pkg/adapters/database/`)
15. ✅ **Schema Management** (`schema.go`)
16. ✅ **Subscription Testing** (`subscription.go`)
17. ✅ **Concurrent Testing** (`concurrent.go`)
18. ✅ **Comprehensive Tests** (all `*_test.go` files)
19. ✅ **Complete README Documentation**

The package is now production-ready with:
- Comprehensive test coverage
- Detailed GoDoc comments on every function
- Fluent API matching Laravel Lighthouse patterns
- Multiple framework and database adapters
- Full subscription testing support
- Concurrent test execution
- BDD-style test organization
```

































