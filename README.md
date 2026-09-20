# GraphQL Tester for Go

[![Go Reference](https://pkg.go.dev/badge/github.com/mwangaben/graphqltester.svg)](https://pkg.go.dev/github.com/mwangaben/graphqltester)
[![Go Report Card](https://goreportcard.com/badge/github.com/mwangaben/graphqltester)](https://goreportcard.com/report/github.com/mwangaben/graphqltester)
[![Tests](https://github.com/mwangaben/graphqltester/workflows/Tests/badge.svg)](https://github.com/mwangaben/graphqltester/actions)

A comprehensive, production-ready GraphQL API testing framework for Go, inspired by [PHP Pest](https://pestphp.com/) and [Laravel Lighthouse](https://lighthouse-php.com/). Test your GraphQL queries, mutations, and subscriptions with elegant, fluent assertions.

## Features

- 🚀 **Fluent API** - Chainable assertions with readable, expressive syntax
- 🔐 **Authentication Testing** - Built-in support for testing auth, roles, and permissions
- 📡 **Subscription Testing** - WebSocket support with **full auth flow** and error handling
- 🗄️ **Database Assertions** - Verify database state after GraphQL operations
- ✅ **Validation Testing** - Test GraphQL validation rules and error messages
- 🏗️ **Multiple HTTP Frameworks** - Supports net/http, Gin, Echo, and Chi
- 💾 **Multiple Database Adapters** - **Ent**, GORM, SQLx, and raw MySQL support
- 🔄 **Transaction Isolation** - Automatic transaction rollback for test isolation
- 🏭 **Laravel-style Factory** - In-memory and database-backed model factories
- 🏢 **Multi-Tenancy** - Test tenant-specific behavior and isolation
- ⚡ **Parallel Testing** - Opt-in concurrent test execution with FailFast support
- 🐛 **Debug Mode** - Verbose logging of requests, responses, and middleware
- 📝 **BDD Style** - Describe/It/Run pattern for readable test organization
- 🔄 **Context Propagation** - Automatic propagation of auth, tenant, and request context

## What's New in v1.2.0

- 🔐 **WebSocket authentication support** — subscription tests can now authenticate
- 📩 **Error message handling** — server-sent errors on subscriptions are captured and assertable
- 🎯 **Per-subscription error state** — accumulate errors during the subscription lifecycle
- 🧪 **Full subscription auth test coverage** — both authenticated and unauthenticated flows

### What's New in v1.1.3

- ✨ **Full Ent ORM support** — first-class `EntAdapter` for Ent-backed GraphQL APIs
- 🎯 **Integration test proving Ent works end-to-end** with the tester's assertions
- 📚 **Ent-specific documentation** — including type-mapping guide for graph-gophers
- 🐛 **Nullable GraphQL fields** — proper support for `*string`, `*int32`, `*bool` returns

## Installation

```bash
go get github.com/mwangaben/graphqltester
```

## Quick Start

```go
package myapp_test

import (
    "testing"
    
    tester "github.com/mwangaben/graphqltester"
    "github.com/mwangaben/graphqltester/pkg/adapters/database"
    "github.com/mwangaben/graphqltester/pkg/factory"
)

func TestZones(t *testing.T) {
    // Create a Laravel-style factory (in-memory, no database needed)
    f := factory.NewFactory()
    f.Define("Zone", func(overrides map[string]interface{}) interface{} {
        zone := map[string]interface{}{
            "name": "Default Zone",
            "code": "ZNE001",
        }
        for k, v := range overrides {
            zone[k] = v
        }
        return zone
    })
    
    // Create tester with configuration
    test := tester.NewTester(t, &tester.Config{
        Schema: &tester.SchemaConfig{
            Path:      "./schema.graphql",
            Resolvers: &MyResolver{},
        },
        Database: &tester.DatabaseConfig{
            Adapter: &database.GORMAdapter{},
            DSN:     "root:password@tcp(localhost:3306)/testdb?parseTime=true",
        },
        Packages: &tester.PackageConfig{
            Factory: f, // Laravel-style factory
        },
    })
    
    // Test a query
    test.GivenAdmin().
        GraphQL(`{ zones { name code } }`).
        AssertOK().
        AssertNoErrors().
        AssertJSONCount("zones", 3)
    
    // Test a mutation with database verification
    test.GraphQL(`
        mutation CreateZone($input: CreateZoneInput!) {
            createZone(input: $input) { slug name code status }
        }
    `, map[string]interface{}{
        "input": map[string]interface{}{
            "name":   "Industrial Zone",
            "code":   "IND001",
            "status": "ACTIVE",
        },
    }).
        AssertNoErrors().
        AssertJSONPath("createZone.name", "Industrial Zone").
        AssertDatabaseHas("zones", map[string]interface{}{"name": "Industrial Zone"})
}
```

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [What's New](#whats-new-in-v120)
- [Configuration](#configuration)
- [Authentication](#-authentication)
- [Assertions](#-assertions)
- [Factory](#-factory)
- [BDD Testing Style](#bdd-testing-style)
- [Parallel Testing](#parallel-testing)
- [Subscription Testing](#subscription-testing)
  - [Basic Subscriptions](#basic-subscriptions)
  - [Authenticated Subscriptions](#authenticated-subscriptions)
  - [Testing Unauthenticated Rejection](#testing-unauthenticated-rejection)
  - [Getting a Token from GraphQL Login](#getting-a-token-from-graphql-login)
- [Framework Adapters](#framework-adapters)
- [Database Adapters](#database-adapters)
  - [Ent](#ent-orm-recommended-for-new-projects)
  - [GORM](#gorm)
  - [SQLx](#sqlx)
  - [Raw MySQL](#raw-mysql)
- [GraphQL Type Mapping](#graphql-type-mapping)
- [Middleware](#middleware)
- [Package Structure](#package-structure)

## Configuration

### Full Configuration Example

```go
config := &tester.Config{
    Endpoint: "/graphql",
    Debug:    true,
    
    // HTTP Framework (default: NetHTTPAdapter)
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
    
    // Database (set to nil for database-free tests)
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
        Factory:    myFactory,           // Laravel-style factory
        Permission: myPermissionManager,
        Validation: myValidator,
    },
}
```

### Database-Free Configuration

For tests that don't need a database:

```go
config := tester.DefaultConfig()
config.Database = nil  // Disable database
config.Schema = &tester.SchemaConfig{
    String:    schemaDefinition,  // Inline schema
    Resolvers: &mockResolver{},
}
```

> **Note on `Debug`:** `Debug` is a **method**, not a field. Use `DefaultConfig().WithDebug(true)` or the fluent `WithDebug(true)` method — you can't set it in a struct literal.

## 🔐 Authentication

### Acting as a User

```go
// Create and authenticate as an admin
test.SignInAdmin()

// Or use an existing user
user := &User{ID: "123", Name: "John"}
test.SignInAdmin(user)

// BDD-style
test.GivenAdmin()

// Sign in with specific role and permissions
test.SignInUser("editor", "posts.create")
test.SignInUser("editor", "posts.edit", existingUser)

// BDD-style
test.GivenUser("editor", "posts.create")

// Set token directly
test.WithToken("eyJhbGciOiJIUzI1NiIs...")

// Set user directly
test.ActingAs(user)

// Clear authentication
test.ClearAuth()
```

### Custom Permission Setup

```go
// Create a user with specific role and permission
func givenUser(tester *graphqltester.Tester, roleName string, permissionName string) *models.User {
    user := tester.Factory("User").Create().(*models.User)
    perm := createPermission(permissionName)
    role := createRole(roleName)
    assignPermissionToRole(role, perm)
    assignRoleToUser(user, role)
    tester.ActingAs(user)
    return user
}

// Usage
givenUser(tester, "editor", "posts.update")
givenViewer(tester) // User with view-only permission
givenAdmin(tester)  // User with all permissions
```

## ✅ Assertions

### HTTP Status Assertions

```go
test.GraphQL(`...`).
    AssertOK().           // 200
    AssertCreated().      // 201
    AssertNoContent().    // 204
    AssertUnauthorized(). // 401
    AssertForbidden().    // 403
    AssertNotFound().     // 404
    AssertUnprocessable().// 422
    AssertServerError()   // 5xx
    AssertStatus(200)     // Custom status code
```

### GraphQL Error Assertions

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
        "user": map[string]interface{}{"name": "John Doe"},
    }).
    AssertJSONSubset(map[string]interface{}{
        "user": map[string]interface{}{"name": "John Doe"},
    }).
    AssertJSONPath("user.name", "John Doe").
    AssertJSONPath("user.address.city", "New York").
    AssertJSONCount("users", 5).
    AssertJSONNotEmpty("users").
    AssertJSONEmpty("users").
    AssertData().       // Data is not nil
    AssertDataNil()     // Data is nil
```

### JSON Path Navigation

```go
response := test.GraphQL(`{ user { name email age active } }`)

// Typed accessors
name := response.JSONString("user.name")    // "John Doe"
age := response.JSONInt("user.age")         // 30
active := response.JSONBool("user.active")  // true

// Generic accessor
val := response.JSON("user.name")           // interface{}

// Array navigation
firstUser := response.JSON("users.0.name")  // First user's name

// Map access
metadata := response.JSONMap("user.metadata")
```

### Validation Assertions

```go
test.GraphQL(`...`).
    AssertValidationError("input.name", "The name field is required.").
    AssertValidationRules(map[string]string{
        "input.name":  "required",
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
    AssertPermissionDenied("zones.delete").
    AssertHasPermission("posts.create").
    AssertHasRole("admin").
    AssertLacksPermission("zones.delete").
    AssertLacksRole("admin")
```

### Database Assertions

```go
test.GraphQL(`...`).
    AssertDatabaseHas("zones", map[string]interface{}{"name": "Test Zone"}).
    AssertDatabaseMissing("zones", map[string]interface{}{"id": 999}).
    AssertSoftDeleted("zones", map[string]interface{}{"id": zoneID}).
    AssertNotSoftDeleted("zones", map[string]interface{}{"id": zoneID}).
    AssertDatabaseCount("zones", 5).
    AssertDatabaseCountWhere("zones", map[string]interface{}{"status": "ACTIVE"}, 3).
    AssertDatabaseValue("zones", map[string]interface{}{"id": zoneID}, "name", "Updated")
```

## 🏭 Factory

The package includes a Laravel-style factory that works both in-memory and with a database.

### In-Memory Factory (No Database Required)

```go
f := factory.NewFactory()

// Define factories
f.Define("User", func(overrides map[string]interface{}) interface{} {
    user := map[string]interface{}{
        "id":    "default-id",
        "name":  "Default User",
        "email": "default@example.com",
    }
    for k, v := range overrides {
        user[k] = v
    }
    return user
})

// Use factories
user := f.Of("User").Create()
adminUser := f.Of("User").Overrides(map[string]interface{}{
    "name": "Admin User",
}).Create()
users := f.Of("User").Times(5).Create()
```

### Database-Backed Factory

```go
f := factory.NewFactory()

f.Define("User", func(overrides map[string]interface{}) interface{} {
    user := User{Name: "Default", Email: "default@example.com"}
    if v, ok := overrides["name"]; ok { user.Name = v.(string) }
    if v, ok := overrides["email"]; ok { user.Email = v.(string) }
    db.Create(&user)  // Persist to database
    return &user
})

// Same API, but now saves to database
user := f.Of("User").Create().(*User)
fmt.Println(user.ID)  // Auto-generated by database
```

### Factory API Reference

| Method                  | Description                                |
|-------------------------|--------------------------------------------|
| Factory(name)           | Get a factory builder for the model        |
| Create()                | Create instance(s) with default attributes |
| Create(overrides)       | Create with attribute overrides            |
| Make()                  | Alias for Create                           |
| Times(n).Create()       | Create multiple instances                  |
| Overrides(map)          | Set attribute overrides                    |
| State(name)             | Apply a named state                        |
| Raw(attrs)              | Create with only provided attributes       |
| CreateMany(n)           | Create multiple instances                  |
| Define(name, fn)        | Register a factory definition              |
| State(model, state, fn) | Register a state transformation            |

## BDD Testing Style

### Pest-Style Individual Tests

```go
// Each test is a top-level function - run individually!
func TestCanCreateUser(t *testing.T) {
    // Test code
}

func TestValidatesCreateUser(t *testing.T) {
    // Test code
}

func TestRequiresAuthForUserQuery(t *testing.T) {
    // Test code
}
```

Run individually:

```bash
go test -run TestCanCreateUser -v
go test -run "TestCan.*" -v
```

### Describe/It Pattern

```go
test.Describe("Zone CRUD Operations", func(t *tester.Tester) {
    
    t.BeforeEach(func() {
        t.RefreshDatabase().GivenAdmin()
    })
    
    t.It("can list zones with pagination", func(t *tester.Tester) {
        t.Factory("Zone").CreateMany(5)
        t.GraphQL(`{ zones(first: 3, page: 1) { data { name } } }`).
            AssertNoErrors().
            AssertJSONCount("zones.data", 3)
    })
    
    t.It("can create a new zone", func(t *tester.Tester) {
        t.GraphQL(`mutation { createZone(input: {name: "New", code: "NEW001"}) { slug } }`).
            AssertNoErrors().
            AssertJSONPath("createZone.name", "New")
    })
    
    t.It("validates required fields", func(t *tester.Tester) {
        t.GraphQL(`mutation { createZone(input: {name: "", code: ""}) { slug } }`).
            AssertValidationError("input.name", "required")
    })
})
```

### Sub-Tests with Run

```go
tester.Run("User CRUD Flow", func(t *graphqltester.Tester) {
    // Create → Read → Update → Delete in sequence
})
```

## Parallel Testing

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
    FailFast:    true,  // Stop all tests on first failure
})
```

## Subscription Testing

The tester supports the full lifecycle of GraphQL subscriptions — from basic event streaming to **authenticated**, **error-asserted** subscriptions over the `graphql-ws` protocol.

### Basic Subscriptions

```go
client, sub := test.Subscribe(`
    subscription ZoneCreated {
        zoneCreated { slug name code }
    }
`, nil)
defer client.Disconnect()

// Trigger the subscription
test.GraphQL(`mutation { createZone(input: {name: "Test", code: "TST001"}) { slug } }`)

// Assert the subscription received the event
sub.ExpectMessageContains(map[string]interface{}{
    "zoneCreated": map[string]interface{}{"name": "Test", "code": "TST001"},
}, 5*time.Second)

// Use assertion helpers
test.AssertSubscription(sub).
    AssertDataPath("zoneCreated.name", "Test").
    AssertNoErrors()
```

### Authenticated Subscriptions

The tester automatically includes the current auth token in WebSocket connections.

```go
// 1. Set the token (from a login mutation or test setup)
test.WithToken("eyJhbGciOiJIUzI1NiIs...")

// 2. Subscribe — the token is sent automatically
client, sub := test.Subscribe(`
    subscription {
        userCreated { id name email }
    }
`, nil)
defer client.Disconnect()

// 3. Assert events
sub.ExpectMessageContains(map[string]interface{}{
    "userCreated": map[string]interface{}{"name": "Test User"},
}, 3*time.Second)
```

**The token is sent in two places:**

1. The HTTP `Authorization` header during the WebSocket handshake
2. The `connection_init` payload (`{"Authorization": "Bearer <token>"}`)

This covers all common server-side auth patterns (`graphql-ws`, custom middleware, `graphql-kit`'s WS authenticator, etc.).

### Testing Unauthenticated Rejection

The tester captures server-sent errors that arrive over WebSocket and exposes them for assertions.

```go
// No token set — the WebSocket connects anonymously
client, sub := test.Subscribe(`
    subscription {
        userCreated { id name }
    }
`, nil)
defer client.Disconnect()

// Assert the server rejected the subscription
sub.ExpectError("Unauthenticated", 3*time.Second)

// Or use fluent assertion helpers
test.AssertSubscription(sub).
    AssertUnauthenticated()      // asserts "Unauthenticated" in the error
    AssertPermissionDenied()     // asserts "permission" in the error
    AssertForbidden()            // asserts "Forbidden" in the error
```

### Getting a Token from GraphQL Login

If your app has a `login` mutation, use the convenience method:

```go
token := test.SignInAndGetToken("user@example.com", "password123")
// tester.currentToken is now set
// Subsequent Subscribe() calls will use it automatically
```

Or manually:

```go
response := test.GraphQL(`
    mutation Login($input: LoginInput!) {
        login(input: $input) { accessToken }
    }
`, map[string]interface{}{
    "input": map[string]interface{}{
        "email":    "user@example.com",
        "password": "password123",
    },
}).AssertNoErrors()

token := response.JSONString("login.accessToken")
test.WithToken(token)
```

### Subscription API Reference

#### Methods on `Tester`

| Method | Description |
|--------|-------------|
| `Subscribe(query, vars...)` | Connect and start a subscription |
| `NewSubscriptionClient()` | Create a client without starting a subscription |
| `AssertSubscription(sub)` | Get assertion helpers for a subscription |
| `SignInAndGetToken(email, password)` | Login via GraphQL and store the token |

#### Methods on `Subscription`

| Method | Description |
|--------|-------------|
| `WaitForMessage(timeout)` | Wait for the next message |
| `WaitForMessages(count, timeout)` | Wait for multiple messages |
| `ExpectMessage(expected, timeout)` | Assert exact match on next message |
| `ExpectMessageContains(expected, timeout)` | Assert subset match on next message |
| `ExpectMessageExact(expected, timeout)` | Alias for `ExpectMessage` |
| `ExpectNoMessage(timeout)` | Assert no message arrives |
| `ExpectError(contains, timeout)` | Assert next message is an error containing the substring |
| `ExpectNoError(timeout)` | Assert no error message arrives |
| `Stop()` | Terminate the subscription |
| `Errors` (field) | Accumulated errors during the subscription |

#### Methods on `SubscriptionAssertions`

| Method | Description |
|--------|-------------|
| `AssertDataPath(path, expected)` | Assert a value at a JSON path |
| `AssertDataPathContains(path, expected)` | Assert a value contains expected data |
| `AssertNoErrors()` | Assert no errors occurred |
| `AssertErrorContains(contains)` | Assert the received error contains a substring |
| `AssertUnauthenticated()` | Assert an "Unauthenticated" error was received |
| `AssertPermissionDenied()` | Assert a permission-denied error was received |
| `AssertForbidden()` | Assert a "Forbidden" error was received |
| `AssertActive()` | Assert the subscription is still active |
| `AssertClosed()` | Assert the subscription is closed |
| `WithTimeout(timeout)` | Set the default timeout |
| `ClearCache()` | Clear the cached message for a new assertion sequence |

## Framework Adapters

```go
// Standard net/http
HTTPAdapter: &http.NetHTTPAdapter{}

// Gin
HTTPAdapter: http.NewGinAdapter()

// Echo
HTTPAdapter: http.NewEchoAdapter(false)

// Chi
HTTPAdapter: http.NewChiAdapter()
```

## Database Adapters

### Ent ORM (Recommended for New Projects)

Full Ent ORM support via raw SQL on the underlying `*sql.DB`. The `EntAdapter` is designed for testing Ent-backed GraphQL APIs.

```go
import (
    "context"
    "time"

    "github.com/mwangaben/graphqltester/pkg/adapters/database"
)

// 1. Create the adapter
adapter := database.NewEntAdapter(&database.EntConfig{
    Debug:              true,
    LogQueries:         false,
    MaxOpenConns:       25,
    MaxIdleConns:       25,
    ConnMaxLifetime:    time.Hour,
    SkipMigration:      false,
    // Optional: provide a migration hook for AutoMigrate
    MigrateFunc: func(ctx context.Context) error {
        return client.Schema.Create(ctx)
    },
})

// 2. Register models for table management
adapter.AddModel(&ent.User{}).AddModel(&ent.Post{})

// 3. Connect (dialect auto-detected from DSN)
if err := adapter.Connect(dsn); err != nil {
    log.Fatal(err)
}
defer adapter.Close()

// 4. Use with graphqltester
test := graphqltester.NewTester(t, &graphqltester.Config{
    Schema: &graphqltester.SchemaConfig{
        String:    resolvers.Schema,
        Resolvers: rootResolver,
    },
    Database: &graphqltester.DatabaseConfig{
        Adapter: adapter,
    },
})
```

**Supported dialects:**

- MySQL / MariaDB
- PostgreSQL
- SQLite

**Dialect detection is automatic** based on the DSN format.

**Operations supported:**

| Operation | Method |
|-----------|--------|
| Insert | `adapter.Insert(ctx, table, data)` |
| Update | `adapter.Update(ctx, table, conditions, data)` |
| Delete | `adapter.Delete(ctx, table, conditions)` |
| Exists | `adapter.HasRecord(ctx, table, conditions)` |
| Get one | `adapter.GetRecord(ctx, table, conditions)` |
| Get many | `adapter.GetRecords(ctx, table, conditions, limit)` |
| Count | `adapter.Count(ctx, table, conditions)` |
| Soft-delete check | `adapter.IsSoftDeleted(ctx, table, conditions)` |
| Drop all | `adapter.DropAll()` |
| Truncate all | `adapter.TruncateAll()` |
| Transaction | `adapter.BeginTx(ctx)` / `Commit(tx)` / `Rollback(tx)` |

**See the "GraphQL Type Mapping" section below for important details about GraphQL resolver types when using Ent.**

### GORM

```go
adapter := database.NewGORMAdapter(&database.GORMConfig{
    PrepareStmt: true,
    SkipDefaultTransaction: true,
})
adapter.AddModel(&User{}).AddModel(&Zone{})
```

### SQLx

```go
adapter := database.NewSQLxAdapter("mysql")
adapter.AddTable("users").AddTable("zones")
```

### Raw MySQL

```go
adapter := database.NewMySQLAdapter()
adapter.AddTable("users", `CREATE TABLE users (...)`)
```

## GraphQL Type Mapping

When wrapping Ent entities in GraphQL resolvers, follow these type rules for `graph-gophers/graphql-go`:

| GraphQL Type | Go Return Type |
|--------------|----------------|
| `Int!` | `int32` |
| `Int` (nullable) | `*int32` |
| `String!` | `string` |
| `String` (nullable) | `*string` |
| `Boolean!` | `bool` |
| `Boolean` (nullable) | `*bool` |
| `Float!` | `float64` |
| `ID!` | `graphql.ID` or `string` |
| `ID` (nullable) | `*graphql.ID` or `*string` |

**Example — wrapping an Ent entity for GraphQL:**

```go
// ent.User has ID int, Age int, Role string
// But the GraphQL schema expects Int for ID and Age

type UserResolver struct {
    user *ent.User
}

// GraphQL Int maps to int32 — NOT int
func (r *UserResolver) ID() int32    { return int32(r.user.ID) }
func (r *UserResolver) Age() int32   { return int32(r.user.Age) }

// Non-nullable String → string
func (r *UserResolver) Name() string { return r.user.Name }
func (r *UserResolver) Email() string { return r.user.Email }

// Nullable String → *string
func (r *UserResolver) Role() *string {
    if r.user.Role == "" {
        return nil
    }
    return &r.user.Role
}
```

**Why this matters:** `graph-gophers` uses `int32` for the built-in `Int` scalar. Returning Go's `int` will fail schema validation with an error like:

```
can not use int as Int
```

**Solutions:**

1. **Return `int32`** from resolver methods (recommended) — matches the schema
2. **Declare a custom `scalar Int`** in your schema — then you can return Go's `int`

The `graphql-kit` package provides a ready-made custom `Int` scalar if you prefer option 2.

## Middleware

The tester includes a configurable middleware chain:

```go
// Enable built-in middleware
config.Middleware.AuthEnabled = true
config.Middleware.PermissionEnabled = true
config.Middleware.ValidationEnabled = true
config.Middleware.TenancyEnabled = true

// Add custom middleware
config.WithCustomMiddleware(myCustomMiddleware)
```

### Middleware Execution Order

1. Custom middleware (user-provided)
2. Request ID middleware
3. Context propagation middleware
4. Multi-tenancy middleware (if enabled)
5. Authentication middleware (if enabled)
6. Permission middleware (if enabled)
7. Validation middleware (if enabled)
8. Response capture middleware

## Package Structure

```text
graphql-tester/
├── types/                         # Shared types and interfaces
│   ├── graphql.go                 # GraphQL request/response types
│   ├── response.go                # Response interface
│   ├── tester.go                  # Tester interface
│   ├── database.go                # Database adapter interface
│   └── context.go                 # Context key constants
├── assertions/                    # Assertion methods
│   ├── response.go                # HTTP/GraphQL response assertions
│   ├── validation.go              # Validation error assertions
│   ├── permission.go              # Permission/authorization assertions
│   └── database.go                # Database state assertions
├── pkg/
│   ├── adapters/
│   │   ├── http/                  # HTTP framework adapters
│   │   │   ├── adapter.go         # FrameworkAdapter interface
│   │   │   ├── nethttp.go         # Standard library adapter
│   │   │   ├── gin.go             # Gin adapter
│   │   │   ├── echo.go            # Echo adapter
│   │   │   └── chi.go             # Chi router adapter
│   │   └── database/              # Database adapters
│   │       ├── adapter.go         # DatabaseAdapter interface
│   │       ├── ent.go             # Ent adapter
│   │       ├── gorm.go            # GORM adapter
│   │       ├── sqlx.go            # SQLx adapter
│   │       └── mysql.go           # Raw MySQL adapter
│   ├── middleware/                 # Middleware implementations
│   │   ├── chain.go               # Middleware chain
│   │   ├── context.go             # Context propagation
│   │   ├── auth.go                # Authentication
│   │   ├── permission.go          # Permission checking
│   │   ├── tenant.go              # Multi-tenancy
│   │   └── validation.go          # Input validation
│   └── factory/                   # Laravel-style factory
│       └── factory.go             # Factory implementation
├── tester.go                      # Main tester struct
├── config.go                      # Configuration management
├── client.go                      # GraphQL HTTP client
├── response.go                    # Response handling
├── auth.go                        # Authentication helpers
├── factory.go                     # Factory integration
├── database.go                    # Database management
├── schema.go                      # Schema management
├── subscription.go                # Subscription testing (auth-aware)
├── websocket_handler.go           # WebSocket message handling
├── concurrent.go                  # Parallel testing
├── go.mod
├── go.sum
├── README.md
└── LICENSE
|-- tests---|
            |--- adapter_test.go
            |--- auth_test.go
            |--- chain_test.go
            |--- concurrent_test.go
            |--- config_test.go
            |--- factory_test.go
            |--- response_test.go
            |--- ent_adapter_test.go
            |--- ent_factory_test.go
            |--- ent_integration_test.go
            |--- subscription_auth_test.go   ← NEW in v1.2.0
```

## Tester API Reference

| Method                                   | Description                       |
|------------------------------------------|-----------------------------------|
| NewTester(t, config)                     | Create a new tester instance      |
| GraphQL(query, vars...)                  | Execute a GraphQL query/mutation  |
| Query(query, vars...)                    | Semantic sugar for queries        |
| Mutation(query, vars...)                 | Semantic sugar for mutations      |
| GraphQLFile(path, vars...)               | Execute query from file           |
| GraphQLWithHeaders(query, vars, headers) | Execute with custom headers       |
| GivenAdmin(user...)                      | Authenticate as admin             |
| GivenUser(role, perm, user...)           | Authenticate with role/permission |
| SignInAdmin(user...)                     | Sign in as admin                  |
| SignInUser(role, perm, user...)          | Sign in with role/permission      |
| SignInAndGetToken(email, password)       | Login via GraphQL, return token   |
| ActingAs(user)                           | Set authenticated user            |
| WithToken(token)                         | Set bearer token                  |
| ClearAuth()                              | Clear authentication state        |
| CurrentUser()                            | Get current user                  |
| CurrentToken()                           | Get current token                 |
| HasPermission(perm)                      | Check permission                  |
| HasRole(role)                            | Check role                        |
| RefreshDatabase()                        | Reset database state              |
| Migrate()                                | Run migrations                    |
| MigrateFresh()                           | Drop and re-run migrations        |
| Seed(fn)                                 | Run seeder function               |
| Factory(name)                            | Get factory builder               |
| SetTenant(id)                            | Set current tenant                |
| WithTenantScope(id, fn)                  | Scope to tenant                   |
| Describe(name, fn)                       | BDD test group                    |
| It(name, fn)                             | BDD test case                     |
| Run(name, fn)                            | Sub-test with isolation           |
| BeforeEach(fn)                           | Setup before each test            |
| RunParallel(tests, config)               | Run tests concurrently            |
| Subscribe(query, vars)                   | Create subscription (auth-aware)  |
| NewSubscriptionClient()                  | Create a subscription client      |
| AssertSubscription(sub)                  | Get subscription assertions       |
| SetShared(key, value)                    | Set shared state                  |
| GetShared(key)                           | Get shared state                  |
| UseMiddleware(chain)                     | Set middleware chain              |
| WithoutMiddleware(names...)              | Remove middleware                 |
| Cleanup()                                | Cleanup resources                 |
| Debug()                                  | Get debug mode status             |
| Helper()                                 | Mark as test helper               |

## Changelog

### v1.2.0 — WebSocket Authentication

- **Added:** WebSocket authentication support — `Connect()` reads `tester.CurrentToken()` and includes it in:
  - HTTP handshake headers (`Authorization: Bearer <token>`)
  - `connection_init` payload (`{"Authorization": "Bearer <token>"}`)
- **Added:** Error message handling — `readPump()` now processes `type: "error"` and `type: "connection_error"` messages
- **Added:** `Subscription.Errors` field — accumulates errors during the subscription lifecycle
- **Added:** `ExpectError(contains, timeout)` — asserts an error is received
- **Added:** `ExpectNoError(timeout)` — asserts no error arrives
- **Added:** `SubscriptionAssertions.AssertErrorContains(contains)` — fluent error assertion
- **Added:** `SubscriptionAssertions.AssertUnauthenticated()` — asserts "Unauthenticated" error
- **Added:** `SubscriptionAssertions.AssertPermissionDenied()` — asserts permission error
- **Added:** `SubscriptionAssertions.AssertForbidden()` — asserts "Forbidden" error
- **Added:** `SignInAndGetToken(email, password)` — convenience method for login
- **Added:** `errorMessages()` helper for extracting message strings from `[]*GraphQLError`
- **Fixed:** Error messages from the server were previously silently ignored during subscriptions- **Test Coverage:** `subscription_auth_test.go` — both authenticated and unauthenticated flows

### v1.1.3 — Ent ORM Support

- **Added:** `EntAdapter` — full Ent ORM support via raw SQL on `*sql.DB`
- **Added:** MySQL, PostgreSQL, and SQLite support for Ent
- **Added:** Automatic dialect detection from DSN
- **Added:** `MigrateFunc` hook for auto-migration with Ent clients
- **Added:** Integration test proving `EntAdapter` works end-to-end with assertions
- **Added:** Comprehensive Ent adapter tests (`ent_adapter_test.go`, `ent_factory_test.go`)
- **Added:** Integration test (`ent_integration_test.go`)
- **Added:** GraphQL type mapping documentation (int32 for Int, *string for nullable String)
- **Fixed:** `Create` operation on Ent models with custom table names

### v1.1.0

- Storage abstraction supporting GORM, SQLx, raw MySQL
- Multi-tenant testing support
- Parallel test execution

### v1.0.0

- Initial release

## License

MIT License — see [LICENSE](LICENSE) for details.