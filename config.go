package graphqltester

import (
	"context"
	"fmt"
	"github.com/graph-gophers/graphql-go"
	"github.com/mwangaben/graphqltester/pkg/factory"
	"net/http"
	"os"
	"time"

	"github.com/mwangaben/graphqltester/pkg/adapters/database"
	appHttp "github.com/mwangaben/graphqltester/pkg/adapters/http"
	_ "github.com/mwangaben/graphqltester/pkg/middleware"
	"github.com/mwangaben/graphqltester/types"
)

// Compile-time interface compliance checks
var (
	_ types.TesterConfigInterface = (*Config)(nil)
)

/**
 * Config holds all configuration options for the GraphQL test environment.
 *
 * This struct provides a centralized way to configure every aspect of the testing
 * environment including HTTP framework selection, database connections, middleware
 * chains, and package integrations.
 *
 * Example Usage:
 *   config := &Config{
 *       Endpoint: "/graphql",
 *       Debug: true,
 *       Schema: &SchemaConfig{
 *           Path: "./schema.graphql",
 *           Resolvers: &MyResolver{},
 *       },
 *   }
 */
type Config struct {
	// Endpoint defines the GraphQL endpoint path.
	// Default: "/graphql"
	Endpoint string

	// HTTPAdapter allows you to specify which HTTP framework to use for testing.
	// Available adapters: NetHTTPAdapter, GinAdapter, EchoAdapter, ChiAdapter
	// If nil, defaults to NetHTTPAdapter.
	HTTPAdapter appHttp.FrameworkAdapter

	// HTTPHandler is a pre-configured HTTP handler for custom server setups.
	// If provided, the HTTPAdapter's Setup method is bypassed.
	HTTPHandler http.Handler

	// BaseURL is the base URL for the test server.
	// Example: "http://localhost:8080"
	BaseURL string

	// Schema holds GraphQL schema configuration.
	Schema *SchemaConfig

	// Database holds database connection and testing configuration.
	Database *DatabaseConfig

	// Middleware configures the middleware chain for test requests.
	Middleware *MiddlewareConfig

	// debugEnabled is the internal debug flag.
	// Use Debug() method to access it.
	debugEnabled bool

	// Parallel enables concurrent test execution.
	// When true, each test runs in its own goroutine with isolated state.
	Parallel bool

	// IsolateState creates completely isolated state for each test.
	// Useful for parallel testing to prevent state leakage.
	IsolateState bool

	// Tenancy holds multi-tenancy configuration.
	Tenancy *TenancyConfig

	// Packages allows integration with external testing packages.
	Packages *PackageConfig

	// Timeout sets the default timeout for operations.
	// Default: 30s
	Timeout time.Duration

	// Context is the parent context for all test operations.
	// If nil, context.Background() is used.
	Context context.Context
}

/**
 * SchemaConfig holds GraphQL schema related configuration.
 *
 * This configuration determines how the GraphQL schema is loaded, parsed,
 * and made available for testing. Supports both file-based and string-based
 * schema loading.
 */
type SchemaConfig struct {
	// Path to the GraphQL schema file (.graphql).
	// Example: "./schema/schema.graphql"
	Path string

	// String contains the GraphQL schema as a string.
	// Alternative to Path when you want to define schema inline.
	String string

	// Resolvers is the resolver implementation for the schema.
	// Must implement the resolver interface expected by your schema.
	Resolvers interface{}

	// Options are additional graphql-go schema options.
	// Example: graphql.UseFieldResolvers(), graphql.MaxDepth(10)
	Options []graphql.SchemaOpt

	// RefreshCache determines if the schema cache should be refreshed
	// between tests. Useful when testing schema changes.
	// Default: true
	RefreshCache bool

	// ValidateOnLoad validates the schema when loading.
	// Catches schema errors early in the test cycle.
	// Default: true
	ValidateOnLoad bool
}

/**
 * DatabaseConfig configures the database connection and test behavior.
 *
 * Supports multiple database adapters (GORM, sqlx, raw MySQL) and provides
 * transaction management for test isolation.
 */
type DatabaseConfig struct {
	// Adapter is the database implementation to use.
	// Available: GORMAdapter, SQLxAdapter, MySQLAdapter
	Adapter database.DatabaseAdapter

	// DSN is the database connection string.
	// Example MySQL: "user:password@tcp(localhost:3306)/testdb?charset=utf8mb4&parseTime=True"
	DSN string

	// AutoMigrate automatically runs database migrations before tests.
	// Ensures your test database schema matches your models.
	// Default: true
	AutoMigrate bool

	// UseTransactions wraps each test in a database transaction.
	// Provides test isolation by rolling back after each test.
	// Default: true
	UseTransactions bool

	// Seeders is a list of seeder functions that populate test data.
	// Run after migrations in the order provided.
	Seeders []func() error

	// MaxOpenConns sets the maximum number of open connections.
	// Default: 25
	MaxOpenConns int

	// MaxIdleConns sets the maximum number of idle connections.
	// Default: 10
	MaxIdleConns int
}

/**
 * MiddlewareConfig configures the middleware pipeline for test requests.
 *
 * Middlewares are executed in order and handle concerns like authentication,
 * authorization, validation, and context propagation before reaching the
 * GraphQL resolver.
 */
type MiddlewareConfig struct {
	// AuthEnabled enables the authentication middleware.
	// When true, requests are processed through the auth middleware.
	AuthEnabled bool

	// PermissionEnabled enables the authorization/permission middleware.
	// Requires AuthEnabled to be true to function properly.
	PermissionEnabled bool

	// ValidationEnabled enables the validation middleware.
	// Integrates with your validation package for input validation.
	ValidationEnabled bool

	// TenancyEnabled enables multi-tenancy middleware.
	// Handles tenant identification and isolation.
	TenancyEnabled bool

	// CustomMiddleware allows adding custom middleware functions.
	// These are prepended to the middleware chain before built-in middleware.
	CustomMiddleware []func(http.Handler) http.Handler
}

/**
 * TenancyConfig configures multi-tenancy support for testing.
 *
 * Enables testing of tenant-specific behavior including data isolation,
 * tenant-specific permissions, and tenant context propagation.
 */
type TenancyConfig struct {
	// Enabled turns on multi-tenancy support.
	// Default: false
	Enabled bool

	// TenantIdentifier specifies how tenants are identified.
	// Supported values: "header", "subdomain", "query"
	// Default: "header"
	TenantIdentifier string

	// HeaderName specifies the HTTP header for tenant identification.
	// Default: "X-Tenant-ID"
	HeaderName string

	// AutoRegister automatically registers new tenants encountered during testing.
	// Default: true
	AutoRegister bool

	// TenantScopingEnabled enables automatic tenant scoping for queries.
	// When true, queries are automatically filtered by current tenant.
	// Default: true
	TenantScopingEnabled bool
}

/**
 * PackageConfig integrates external testing packages with the tester.
 *
 * Seamlessly integrates your existing packages (factory, permission, validation)
 * into the GraphQL testing environment.
 */
type PackageConfig struct {
	// Factory integrates the model factory package for test data generation.
	Factory interface{} // Will be typed to *factory.Factory when available

	// Permission integrates the permission package for authorization testing.
	Permission interface{} // Will be typed to *permission.Manager when available

	// Validation integrates the validation package for input validation testing.
	Validation interface{} // Will be typed to *validation.Validator when available
}

// SimpleFactory is the type for the Laravel-style factory.
// This is used in PackageConfig.Factory.
type SimpleFactory = factory.Factory

// ============================================================================
// types.TesterConfigInterface Implementation
// ============================================================================

/**
 * Debug returns whether debug mode is enabled.
 * Implements types.TesterConfigInterface.
 */
func (c *Config) Debug() bool {
	return c.debugEnabled
}

/**
 * TenancyEnabled returns whether multi-tenancy is enabled.
 * Implements types.TesterConfigInterface.
 */
func (c *Config) TenancyEnabled() bool {
	return c.Tenancy != nil && c.Tenancy.Enabled
}

// ============================================================================
// Constructor
// ============================================================================

/**
 * DefaultConfig returns a sensible default configuration for quick setup.
 *
 * This is the recommended starting point for most applications. Customize
 * individual fields as needed for your specific testing requirements.
 *
 * Default settings:
 * - Endpoint: /graphql
 * - HTTP Adapter: NetHTTP (standard library)
 * - Debug: true (for detailed test output)
 * - AutoMigrate: true
 * - UseTransactions: true (for test isolation)
 * - Schema RefreshCache: true
 * - Timeout: 30 seconds
 *
 * Returns:
 *   *Config with production-safe defaults
 */
func DefaultConfig() *Config {
	return &Config{
		Endpoint:     "/graphql",
		HTTPAdapter:  &appHttp.NetHTTPAdapter{},
		debugEnabled: true,
		Timeout:      30 * time.Second,
		Context:      context.Background(),
		Schema: &SchemaConfig{
			RefreshCache:   true,
			ValidateOnLoad: true,
		},
		Database: &DatabaseConfig{
			UseTransactions: true,
			AutoMigrate:     true,
			MaxOpenConns:    25,
			MaxIdleConns:    10,
		},
		Middleware: &MiddlewareConfig{
			AuthEnabled:       false,
			PermissionEnabled: false,
			ValidationEnabled: false,
			TenancyEnabled:    false,
		},
		Tenancy: &TenancyConfig{
			Enabled:              false,
			TenantIdentifier:     "header",
			HeaderName:           "X-Tenant-ID",
			AutoRegister:         true,
			TenantScopingEnabled: true,
		},
	}
}

/**
 * Validate checks the configuration for common errors and inconsistencies.
 *
 * This method should be called after configuration setup to catch
 * misconfigurations early. It validates:
 * - Schema configuration (either Path or String must be set)
 * - Database configuration consistency
 * - Middleware dependency requirements
 * - Required fields for enabled features
 *
 * Returns:
 *   error if configuration is invalid, nil otherwise
 */
func (c *Config) Validate() error {
	if c.Schema == nil {
		return fmt.Errorf("schema configuration is required")
	}

	if c.Schema.Path == "" && c.Schema.String == "" {
		return fmt.Errorf("either schema path or schema string must be provided")
	}

	if c.Schema.Resolvers == nil {
		return fmt.Errorf("schema resolvers are required")
	}

	if c.Database != nil {
		if c.Database.Adapter == nil {
			return fmt.Errorf("database adapter is required when database config is provided")
		}
		if c.Database.DSN == "" {
			return fmt.Errorf("database DSN is required when database config is provided")
		}
	}

	// Middleware dependency checks
	if c.Middleware != nil {
		if c.Middleware.PermissionEnabled && !c.Middleware.AuthEnabled {
			return fmt.Errorf("permission middleware requires authentication to be enabled")
		}
		if c.Tenancy != nil && c.Tenancy.Enabled && !c.Middleware.TenancyEnabled {
			return fmt.Errorf("tenancy config requires tenancy middleware to be enabled")
		}
	}

	// Set default timeout if not configured
	if c.Timeout == 0 {
		c.Timeout = 30 * time.Second
	}

	// Set default context if not configured
	if c.Context == nil {
		c.Context = context.Background()
	}

	return nil
}

/**
 * LoadSchemaFromFile is a convenience method to load schema configuration from a file.
 *
 * Parameters:
 *   path      - Path to the GraphQL schema file
 *   resolvers - The resolver implementation
 *
 * Returns:
 *   *Config for fluent method chaining
 *
 * Example:
 *   config := DefaultConfig().
 *       LoadSchemaFromFile("./schema.graphql", &MyResolver{})
 */
func (c *Config) LoadSchemaFromFile(path string, resolvers interface{}) *Config {
	// Verify the file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		panic(fmt.Sprintf("schema file not found: %s", path))
	}

	c.Schema = &SchemaConfig{
		Path:           path,
		Resolvers:      resolvers,
		RefreshCache:   true,
		ValidateOnLoad: true,
	}

	return c
}

/**
 * LoadSchemaFromString is a convenience method to load schema from a string.
 *
 * Parameters:
 *   schemaString - The GraphQL schema as a string
 *   resolvers    - The resolver implementation
 *
 * Returns:
 *   *Config for fluent method chaining
 *
 * Example:
 *   config := DefaultConfig().
 *       LoadSchemaFromString(schemaString, &MyResolver{})
 */
func (c *Config) LoadSchemaFromString(schemaString string, resolvers interface{}) *Config {
	c.Schema = &SchemaConfig{
		String:         schemaString,
		Resolvers:      resolvers,
		RefreshCache:   true,
		ValidateOnLoad: true,
	}

	return c
}

/**
 * WithDatabase configures database settings.
 *
 * Parameters:
 *   adapter - Database adapter to use
 *   dsn     - Database connection string
 *
 * Returns:
 *   *Config for fluent method chaining
 *
 * Example:
 *   config := DefaultConfig().
 *       WithDatabase(&database.GORMAdapter{}, "root:pass@tcp(localhost:3306)/testdb?parseTime=true")
 */
func (c *Config) WithDatabase(adapter database.DatabaseAdapter, dsn string) *Config {
	c.Database = &DatabaseConfig{
		Adapter:         adapter,
		DSN:             dsn,
		AutoMigrate:     true,
		UseTransactions: true,
		MaxOpenConns:    25,
		MaxIdleConns:    10,
	}

	return c
}

//noinspection GoCommentFormat
/**
 * WithHTTPAdapter configures the HTTP framework adapter.
 *
 * Parameters:
 *   adapter - The HTTP framework adapter to use
 *
 * Returns:
 *   *Config for fluent method chaining
 *
 * Example:
 *   config := DefaultConfig().
 *       WithHTTPAdapter(http.NewGinAdapter())
 */
func (c *Config) WithHTTPAdapter(adapter appHttp.FrameworkAdapter) *Config {
	c.HTTPAdapter = adapter
	return c
}

/**
 * WithAuth enables authentication middleware.
 *
 * Returns:
 *   *Config for fluent method chaining
 */
func (c *Config) WithAuth() *Config {
	if c.Middleware == nil {
		c.Middleware = &MiddlewareConfig{}
	}
	c.Middleware.AuthEnabled = true
	return c
}

/**
 * WithPermissions enables permission middleware.
 * Requires authentication to be enabled.
 *
 * Returns:
 *   *Config for fluent method chaining
 */
func (c *Config) WithPermissions() *Config {
	if c.Middleware == nil {
		c.Middleware = &MiddlewareConfig{}
	}
	c.Middleware.AuthEnabled = true // Permissions require auth
	c.Middleware.PermissionEnabled = true
	return c
}

/**
 * WithValidation enables validation middleware.
 *
 * Returns:
 *   *Config for fluent method chaining
 */
func (c *Config) WithValidation() *Config {
	if c.Middleware == nil {
		c.Middleware = &MiddlewareConfig{}
	}
	c.Middleware.ValidationEnabled = true
	return c
}

/**
 * WithTenancy enables multi-tenancy middleware.
 *
 * Returns:
 *   *Config for fluent method chaining
 */
func (c *Config) WithTenancy() *Config {
	if c.Middleware == nil {
		c.Middleware = &MiddlewareConfig{}
	}
	c.Middleware.TenancyEnabled = true

	if c.Tenancy == nil {
		c.Tenancy = &TenancyConfig{}
	}
	c.Tenancy.Enabled = true

	return c
}

/**
 * WithDebug enables or disables debug mode.
 *
 * Parameters:
 *   debug - Whether to enable debug mode
 *
 * Returns:
 *   *Config for fluent method chaining
 */
func (c *Config) WithDebug(debug bool) *Config {
	c.debugEnabled = debug
	return c
}

/**
 * WithTimeout sets the default operation timeout.
 *
 * Parameters:
 *   timeout - Duration for operation timeout
 *
 * Returns:
 *   *Config for fluent method chaining
 */
func (c *Config) WithTimeout(timeout time.Duration) *Config {
	c.Timeout = timeout
	return c
}

/**
 * WithParallel enables parallel test execution.
 *
 * Parameters:
 *   isolateState - Whether to isolate state between parallel tests
 *
 * Returns:
 *   *Config for fluent method chaining
 */
func (c *Config) WithParallel(isolateState bool) *Config {
	c.Parallel = true
	c.IsolateState = isolateState
	return c
}

/**
 * WithFactory integrates a model factory package.
 *
 * Parameters:
 *   factory - The factory instance
 *
 * Returns:
 *   *Config for fluent method chaining
 */
func (c *Config) WithFactory(factory interface{}) *Config {
	if c.Packages == nil {
		c.Packages = &PackageConfig{}
	}
	c.Packages.Factory = factory
	return c
}

/**
 * WithPermission integrates a permission package.
 *
 * Parameters:
 *   permission - The permission manager instance
 *
 * Returns:
 *   *Config for fluent method chaining
 */
func (c *Config) WithPermission(permission interface{}) *Config {
	if c.Packages == nil {
		c.Packages = &PackageConfig{}
	}
	c.Packages.Permission = permission
	return c
}

/**
 * WithValidationPackage integrates a validation package.
 *
 * Parameters:
 *   validation - The validator instance
 *
 * Returns:
 *   *Config for fluent method chaining
 */
func (c *Config) WithValidationPackage(validation interface{}) *Config {
	if c.Packages == nil {
		c.Packages = &PackageConfig{}
	}
	c.Packages.Validation = validation
	return c
}

/**
 * WithCustomMiddleware adds custom middleware functions.
 *
 * Parameters:
 *   middlewares - Middleware functions to add
 *
 * Returns:
 *   *Config for fluent method chaining
 */
func (c *Config) WithCustomMiddleware(middlewares ...func(http.Handler) http.Handler) *Config {
	if c.Middleware == nil {
		c.Middleware = &MiddlewareConfig{}
	}
	c.Middleware.CustomMiddleware = append(c.Middleware.CustomMiddleware, middlewares...)
	return c
}

/**
 * WithEndpoint sets a custom GraphQL endpoint path.
 *
 * Parameters:
 *   endpoint - The endpoint path (e.g., "/api/graphql")
 *
 * Returns:
 *   *Config for fluent method chaining
 *
 * Example:
 *   config := DefaultConfig().WithEndpoint("/api/graphql")
 */
func (c *Config) WithEndpoint(endpoint string) *Config {
	c.Endpoint = endpoint
	return c
}
