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
- 🏭 **Laravel-style Factory** - In-memory and database-backed model factories
- 🏢 **Multi-Tenancy** - Test tenant-specific behavior and isolation
- ⚡ **Parallel Testing** - Opt-in concurrent test execution with FailFast support
- 🐛 **Debug Mode** - Verbose logging of requests, responses, and middleware
- 📝 **BDD Style** - Describe/It/Run pattern for readable test organization
- 🔄 **Context Propagation** - Automatic propagation of auth, tenant, and request context

## Installation

```bash
go get github.com/mwangaben/graphqltester
````


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


- Installation

- Quick Start

- Configuration

- Authentication

- Assertions

   - HTTP Status

  - GraphQL Errors

  - Data Assertions

  - JSON Path Navigation

  - Validation Assertions

  - Permission Assertions

  - Database Assertions

- Factory

- BDD Testing Style

- Parallel Testing

- Subscription Testing

- Framework Adapters

- Database Adapters

- Middleware

- Package Structure

## Configuration

#### Full Configuration Example

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

#### Database-Free Configuration

For tests that don't need a database:



```go
config := tester.DefaultConfig()
config.Database = nil  // Disable database
config.Schema = &tester.SchemaConfig{
    String:    schemaDefinition,  // Inline schema
    Resolvers: &mockResolver{},
}
```

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


#### In-Memory Factory (No Database Required)
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


#### Database-Backed Factory

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


### BDD Testing Style

#### Pest-Style Individual Tests

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

##### Run individually:

```go
go test -run TestCanCreateUser -v
go test -run "TestCan.*" -v
```


#### Describe/It Pattern

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

// Sub-tests with isolation
test.Run("User Tests", func(t *tester.Tester) {
    t.GivenAdmin()
    t.GraphQL(`...`).AssertNoErrors()
})
```

#### Sub-Tests with Run

```go
tester.Run("User CRUD Flow", func(t *graphqltester.Tester) {
    // Create → Read → Update → Delete in sequence
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
    FailFast:    true,  // Stop all tests on first failure
})
```

### Subscription Testing

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

### Framework Adapters

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

### Database Adapter Configuration

```go
// GORM
adapter := database.NewGORMAdapter(&database.GORMConfig{
    PrepareStmt: true,
    SkipDefaultTransaction: true,
})
adapter.AddModel(&User{}).AddModel(&Zone{})

// SQLx
adapter := database.NewSQLxAdapter("mysql")
adapter.AddTable("users").AddTable("zones")

// Raw MySQL
adapter := database.NewMySQLAdapter()
adapter.AddTable("users", `CREATE TABLE users (...)`)
```


### Middleware

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

- Custom middleware (user-provided)
1. Custom middleware (user-provided)
2. Request ID middleware
3. Context propagation middleware
4. Multi-tenancy middleware (if enabled)
5. Authentication middleware (if enabled)
6. Permission middleware (if enabled)
7. Validation middleware (if enabled)
8. Response capture middleware


### Package Structure

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
├── subscription.go                # Subscription testing
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
```


```go
config := &tester.Config{
    Packages: &tester.PackageConfig{
        Factory:    myFactory,
        Permission: myPermissionManager,
        Validation: myValidator,
    },
}
```

### Tester API Reference

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
| Subscribe(query, vars)                   | Create subscription               |
| SetShared(key, value)                    | Set shared state                  |
| GetShared(key)                           | Get shared state                  |
| UseMiddleware(chain)                     | Set middleware chain              |
| WithoutMiddleware(names...)              | Remove middleware                 |
| Cleanup()                                | Cleanup resources                 |
| Debug()                                  | Get debug mode status             |
| Helper()                                 | Mark as test helper               |


### License


```text

The README now includes:
- ✅ Updated factory section with both in-memory and database examples
- ✅ Database-free configuration option
- ✅ Complete factory API reference
- ✅ JSON path navigation examples
- ✅ Parallel testing with FailFast
- ✅ Package structure diagram
- ✅ Full tester API reference table
- ✅ Middleware execution order
- ✅ BDD testing with Describe/It/Run
```




















