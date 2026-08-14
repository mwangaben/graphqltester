package graphqltester

import (
	"context"
	"fmt"
	"github.com/graph-gophers/graphql-go/relay"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	httpadapter "github.com/mwangaben/graphqltester/pkg/adapters/http"
	"github.com/mwangaben/graphqltester/pkg/middleware"
	"github.com/mwangaben/graphqltester/types"
)

// Compile-time interface compliance checks
var (
	_ types.TesterInterface = (*Tester)(nil)
)

/**
 * Tester is the main test helper for GraphQL API testing.
 *
 * This struct serves as the central orchestrator for all testing operations.
 * It manages the HTTP server, GraphQL schema, database connections, middleware
 * chain, and test state.
 *
 * Implements types.TesterInterface to work with the assertions package
 * without creating cyclic imports.
 */
type Tester struct {
	t      *testing.T
	config *Config
	mu     sync.RWMutex

	// HTTP Server
	server      *httptest.Server
	httpAdapter httpadapter.FrameworkAdapter
	client      *GraphQLClient

	// GraphQL Schema
	schema *SchemaManager

	// Database
	dbAdapter types.DatabaseAdapter
	txManager *TransactionManager

	// Middleware
	middlewareChain *middleware.MiddlewareChain

	// Context Management
	ctx    context.Context
	cancel context.CancelFunc

	// Authentication State
	currentUser  interface{}
	currentToken string

	// Multi-tenancy
	tenant  *TenantContext
	tenants map[string]*TenantContext

	// Test State
	models map[string]interface{}
	state  map[string]interface{}

	beforeEach func() // ← Add this field

}

// ============================================================================
// types.TesterInterface Implementation
// ============================================================================

// Database returns the database adapter for database assertions.
func (tester *Tester) Database() types.DatabaseAdapter {
	return tester.dbAdapter
}

// Config returns the tester configuration.
func (tester *Tester) Config() types.TesterConfigInterface {
	return tester.config
}

// Context returns the tester's base context.
func (tester *Tester) Context() context.Context {
	return tester.ctx
}

// CurrentUser returns the currently authenticated user.
func (tester *Tester) CurrentUser() interface{} {
	tester.mu.RLock()
	defer tester.mu.RUnlock()
	return tester.currentUser
}

// CurrentToken returns the current authentication token.
func (tester *Tester) CurrentToken() string {
	tester.mu.RLock()
	defer tester.mu.RUnlock()
	return tester.currentToken
}

// Logf logs a formatted message for debugging.
func (tester *Tester) Logf(format string, args ...interface{}) {
	tester.t.Helper()
	tester.t.Logf(format, args...)
}

// Errorf reports a test error with a formatted message.
func (tester *Tester) Errorf(format string, args ...interface{}) {
	tester.t.Helper()
	tester.t.Errorf(format, args...)
}

// Fatalf reports a fatal test error with a formatted message.
func (tester *Tester) Fatalf(format string, args ...interface{}) {
	tester.t.Helper()
	tester.t.Fatalf(format, args...)
}

// Helper marks the calling function as a test helper.
func (tester *Tester) Helper() {
	tester.t.Helper()
}

// Name returns the name of the current test.
func (tester *Tester) Name() string {
	return tester.t.Name()
}

// ============================================================================
// Constructor
// ============================================================================

/**
 * NewTester creates a new GraphQL test instance with the provided configuration.
 */
func NewTester(t *testing.T, config *Config) *Tester {
	// Validate configuration first
	if err := config.Validate(); err != nil {
		t.Fatalf("❌ Invalid test configuration: %v", err)
	}

	// Create context with cancellation for cleanup
	ctx, cancel := context.WithCancel(config.Context)

	tester := &Tester{
		t:       t,
		config:  config,
		ctx:     ctx,
		cancel:  cancel,
		models:  make(map[string]interface{}),
		state:   make(map[string]interface{}),
		tenants: make(map[string]*TenantContext),
	}

	// Initialize HTTP adapter
	if config.HTTPAdapter != nil {
		tester.httpAdapter = config.HTTPAdapter
	} else {
		tester.httpAdapter = &httpadapter.NetHTTPAdapter{}
	}

	// Initialize schema manager
	tester.schema = NewSchemaManager(config.Schema)

	// Load and parse the GraphQL schema
	if err := tester.schema.Load(); err != nil {
		t.Fatalf("❌ Failed to load GraphQL schema: %v", err)
	}

	if config.Debug() {
		t.Logf("✅ GraphQL schema loaded successfully")
	}

	// Initialize database if configured
	if config.Database != nil {
		if err := tester.setupDatabase(); err != nil {
			t.Fatalf("❌ Failed to setup database: %v", err)
		}
	}

	// Initialize middleware chain
	tester.setupMiddleware()

	// Create the GraphQL HTTP handler
	handler := tester.createGraphQLHandler()

	// Apply middleware chain to handler
	if tester.middlewareChain != nil {
		handler = tester.middlewareChain.Build(handler)
	}

	// Create test HTTP server
	if config.HTTPHandler != nil {
		tester.server = httptest.NewServer(config.HTTPHandler)
	} else {
		tester.server = tester.httpAdapter.Setup(handler)
	}

	// Set base URL from server
	config.BaseURL = tester.server.URL

	// Initialize GraphQL client
	tester.client = NewGraphQLClient(tester)

	// Register cleanup with testing.T
	t.Cleanup(tester.Cleanup)

	if config.Debug() {
		t.Logf("🚀 Test server started at %s", tester.server.URL)
		t.Logf("📍 GraphQL endpoint: %s%s", tester.server.URL, config.Endpoint)
	}

	return tester
}

/**
 * Cleanup performs cleanup operations for the tester.
 */
func (tester *Tester) Cleanup() {
	if tester.config.Debug() {
		tester.t.Logf("🧹 Cleaning up tester...")
	}

	// Cancel context first to stop any ongoing operations
	if tester.cancel != nil {
		tester.cancel()
	}

	// Close HTTP server
	if tester.server != nil {
		tester.server.Close()
	}

	// Rollback any active transaction
	if tester.txManager != nil && tester.txManager.IsActive() {
		if err := tester.txManager.Rollback(); err != nil {
			tester.t.Logf("⚠️  Warning: Failed to rollback transaction during cleanup: %v", err)
		}
	}

	// Close database connection
	if tester.dbAdapter != nil {
		if err := tester.dbAdapter.Close(); err != nil {
			tester.t.Logf("⚠️  Warning: Failed to close database during cleanup: %v", err)
		}
	}

	if tester.config.Debug() {
		tester.t.Logf("✅ Cleanup complete")
	}
}

// ============================================================================
// Database Setup
// ============================================================================

/**
 * setupDatabase initializes the database adapter and runs migrations.
 */
func (tester *Tester) setupDatabase() error {
	dbConfig := tester.config.Database

	if tester.config.Debug() {
		tester.t.Logf("🗄️  Connecting to database...")
	}

	// Connect to database
	if err := dbConfig.Adapter.Connect(dbConfig.DSN); err != nil {
		return fmt.Errorf("database connection failed: %w", err)
	}

	tester.dbAdapter = dbConfig.Adapter

	// Configure connection pool
	if dbConfig.MaxOpenConns > 0 {
		tester.dbAdapter.SetMaxOpenConns(dbConfig.MaxOpenConns)
	}
	if dbConfig.MaxIdleConns > 0 {
		tester.dbAdapter.SetMaxIdleConns(dbConfig.MaxIdleConns)
	}

	// Run auto-migrations if enabled
	if dbConfig.AutoMigrate {
		if tester.config.Debug() {
			tester.t.Logf("🔄 Running database migrations...")
		}

		if err := tester.dbAdapter.AutoMigrate(); err != nil {
			return fmt.Errorf("auto-migration failed: %w", err)
		}

		if tester.config.Debug() {
			tester.t.Logf("✅ Database migrations completed successfully")
		}
	}

	// Run seeders in order
	for i, seeder := range dbConfig.Seeders {
		if tester.config.Debug() {
			tester.t.Logf("🌱 Running seeder %d...", i+1)
		}

		if err := seeder(); err != nil {
			return fmt.Errorf("seeder %d failed: %w", i, err)
		}
	}

	if len(dbConfig.Seeders) > 0 {
		if tester.config.Debug() {
			tester.t.Logf("✅ Executed %d seeders successfully", len(dbConfig.Seeders))
		}
	}

	// Initialize transaction manager
	tester.txManager = NewTransactionManager(tester.dbAdapter)

	// Start a transaction for test isolation if enabled
	if dbConfig.UseTransactions {
		if err := tester.txManager.Begin(); err != nil {
			return fmt.Errorf("failed to begin test transaction: %w", err)
		}

		if tester.config.Debug() {
			tester.t.Logf("📌 Started test transaction for isolation")
		}
	}

	return nil
}

// ============================================================================
// Middleware Setup
// ============================================================================

/**
 * setupMiddleware configures the middleware chain based on configuration.
 */
func (tester *Tester) setupMiddleware() {
	chain := middleware.NewMiddlewareChain(tester.config.Debug())

	// Add custom middleware first
	if tester.config.Middleware != nil {
		for i, mw := range tester.config.Middleware.CustomMiddleware {
			chain.Add(fmt.Sprintf("custom-%d", i), mw)
		}
	}

	// Add request ID middleware for traceability
	chain.Add("request-id", middleware.RequestIDMiddleware())

	// Add context propagation middleware
	chain.Add("context", middleware.ContextPropagationMiddleware(tester))

	// Add multi-tenancy middleware if enabled
	if tester.config.Middleware != nil && tester.config.Middleware.TenancyEnabled {
		chain.Add("tenant-resolver", middleware.TenantResolverMiddleware(tester))
		chain.Add("tenant-scope", middleware.TenantMiddleware(tester))
	}

	// Add authentication middleware if enabled
	if tester.config.Middleware != nil && tester.config.Middleware.AuthEnabled {
		chain.Add("auth", middleware.AuthMiddleware(tester))
	}

	// Add permission middleware if enabled (requires auth)
	if tester.config.Middleware != nil && tester.config.Middleware.PermissionEnabled && tester.config.Middleware.AuthEnabled {
		chain.Add("schema-analysis", middleware.SchemaAnalysisMiddleware(tester))
		chain.Add("permission", middleware.PermissionMiddleware(tester))
	}

	// Add validation middleware if enabled
	if tester.config.Middleware != nil && tester.config.Middleware.ValidationEnabled {
		chain.Add("validation", middleware.ValidationMiddleware(tester))
	}

	// Add response capture middleware (always last)
	chain.Add("response-capture", middleware.ResponseContextMiddleware(tester))

	tester.middlewareChain = chain

	if tester.config.Debug() {
		tester.t.Logf("🔗 Middleware chain configured with %d middleware(s)", chain.Count())
		for i, name := range chain.Names() {
			tester.t.Logf("   [%d] %s", i, name)
		}
	}
}

/**
 * createGraphQLHandler creates the HTTP handler for GraphQL requests.
 */
func (tester *Tester) createGraphQLHandler() http.Handler {
	schema := tester.schema.GetSchema()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tester.t.Logf("🌐 Request received: %s %s, Upgrade: %s", r.Method, r.URL.Path, r.Header.Get("Upgrade"))

		// Check if it's a WebSocket upgrade
		if strings.ToLower(r.Header.Get("Upgrade")) == "websocket" {
			tester.t.Logf("🔌 WebSocket upgrade detected, routing to WS handler")
			createWSHandler(schema).ServeHTTP(w, r)
			return
		}

		// Regular HTTP GraphQL request
		tester.t.Logf("📡 Regular HTTP request, routing to relay handler")
		handler := &relay.Handler{Schema: schema}
		handler.ServeHTTP(w, r)
	})
}

/**
 * combinedHandler routes requests to HTTP or WebSocket handler.
 */
type combinedHandler struct {
	httpHandler http.Handler
	wsHandler   http.Handler
}

func (h *combinedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Check if it's a WebSocket upgrade
	if strings.ToLower(r.Header.Get("Upgrade")) == "websocket" {
		h.wsHandler.ServeHTTP(w, r)
		return
	}
	h.httpHandler.ServeHTTP(w, r)
}

// ============================================================================
// GraphQL Operations
// ============================================================================

// GraphQL executes a GraphQL query or mutation and returns a response.
func (tester *Tester) GraphQL(query string, vars ...map[string]interface{}) *Response {
	return tester.client.GraphQL(query, vars...)
}

// Query is semantic sugar for read operations.
func (tester *Tester) Query(query string, vars ...map[string]interface{}) *Response {
	return tester.GraphQL(query, vars...)
}

// Mutation is semantic sugar for write operations.
func (tester *Tester) Mutation(query string, vars ...map[string]interface{}) *Response {
	return tester.GraphQL(query, vars...)
}

// GraphQLFile reads a GraphQL query from a file and executes it.
func (tester *Tester) GraphQLFile(path string, vars ...map[string]interface{}) *Response {
	content, err := os.ReadFile(path)
	if err != nil {
		tester.t.Fatalf("❌ Failed to read GraphQL file %s: %v", path, err)
	}
	return tester.GraphQL(string(content), vars...)
}

// GraphQLWithHeaders sends a GraphQL request with custom HTTP headers.
func (tester *Tester) GraphQLWithHeaders(query string, variables map[string]interface{}, headers map[string]string) *Response {
	return tester.client.GraphQLWithHeaders(query, variables, headers)
}

// ============================================================================
// Database Operations
// ============================================================================

// RefreshDatabase resets the database to a clean state.
func (tester *Tester) RefreshDatabase() *Tester {
	if tester.dbAdapter == nil {
		tester.t.Fatal("❌ Database is not configured. Set Database in Config.")
	}

	if tester.config.Database.UseTransactions && tester.txManager != nil {
		if tester.txManager.IsActive() {
			if err := tester.txManager.Rollback(); err != nil {
				tester.t.Logf("⚠️  Warning: Failed to rollback transaction: %v", err)
			}
		}
		if err := tester.txManager.Begin(); err != nil {
			tester.t.Fatalf("❌ Failed to begin new transaction: %v", err)
		}
		if tester.config.Debug() {
			tester.t.Logf("🔄 Database refreshed via transaction rollback")
		}
	} else {
		if err := tester.dbAdapter.TruncateAll(); err != nil {
			tester.t.Fatalf("❌ Failed to truncate tables: %v", err)
		}
		if tester.config.Debug() {
			tester.t.Logf("🔄 Database refreshed via table truncation")
		}
	}

	return tester
}

// Migrate runs database migrations.
func (tester *Tester) Migrate() *Tester {
	if tester.dbAdapter == nil {
		tester.t.Fatal("❌ Database is not configured")
	}
	if err := tester.dbAdapter.AutoMigrate(); err != nil {
		tester.t.Fatalf("❌ Migration failed: %v", err)
	}
	if tester.config.Debug() {
		tester.t.Logf("✅ Migrations completed")
	}
	return tester
}

// MigrateFresh drops all tables and re-runs migrations.
func (tester *Tester) MigrateFresh() *Tester {
	if tester.dbAdapter == nil {
		tester.t.Fatal("❌ Database is not configured")
	}
	if err := tester.dbAdapter.DropAll(); err != nil {
		tester.t.Fatalf("❌ Failed to drop tables: %v", err)
	}
	if err := tester.dbAdapter.AutoMigrate(); err != nil {
		tester.t.Fatalf("❌ Migration failed: %v", err)
	}
	if tester.config.Debug() {
		tester.t.Logf("✅ Database refreshed (all tables dropped and recreated)")
	}
	return tester
}

// ============================================================================
// Multi-Tenancy Operations
// ============================================================================

// SetTenant sets the current tenant for multi-tenant testing.
func (tester *Tester) SetTenant(tenantID string) *Tester {
	tester.mu.Lock()
	defer tester.mu.Unlock()

	if _, exists := tester.tenants[tenantID]; !exists {
		tester.tenants[tenantID] = &TenantContext{
			ID:   tenantID,
			Data: make(map[string]interface{}),
		}
	}
	tester.tenant = tester.tenants[tenantID]

	if tester.config.Debug() {
		tester.t.Logf("🏢 Set current tenant: %s", tenantID)
	}
	return tester
}

// WithTenantScope scopes a test function to a specific tenant.
func (tester *Tester) WithTenantScope(tenantID string, fn func()) {
	previousTenant := tester.tenant
	tester.SetTenant(tenantID)
	defer func() { tester.tenant = previousTenant }()
	fn()
}

// ============================================================================
// Middleware Management
// ============================================================================

// UseMiddleware configures the tester to use a middleware chain.
func (tester *Tester) UseMiddleware(chain *middleware.MiddlewareChain) *Tester {
	tester.middlewareChain = chain
	handler := tester.createGraphQLHandler()
	handler = chain.Build(handler)

	if tester.server != nil {
		tester.server.Close()
	}
	tester.server = tester.httpAdapter.Setup(handler)
	tester.config.BaseURL = tester.server.URL
	tester.client = NewGraphQLClient(tester)
	return tester
}

// WithoutMiddleware creates a copy of the tester without specific middleware.
func (tester *Tester) WithoutMiddleware(names ...string) *Tester {
	newTester := tester.clone()
	if newTester.middlewareChain != nil {
		for _, name := range names {
			newTester.middlewareChain.Remove(name)
		}
	}
	return newTester
}

// ============================================================================
// Internal Helpers
// ============================================================================

// clone creates a deep copy of the tester for isolated sub-tests.
func (tester *Tester) clone() *Tester {
	tester.mu.RLock()
	defer tester.mu.RUnlock()

	clone := &Tester{
		t:               tester.t,
		config:          tester.config,
		server:          tester.server,
		httpAdapter:     tester.httpAdapter,
		client:          tester.client,
		schema:          tester.schema,
		dbAdapter:       tester.dbAdapter,
		txManager:       tester.txManager,
		middlewareChain: tester.middlewareChain,
		ctx:             tester.ctx,
		cancel:          tester.cancel,
		currentUser:     tester.currentUser,
		currentToken:    tester.currentToken,
		tenant:          tester.tenant,
		tenants:         tester.tenants,
		models:          make(map[string]interface{}),
		state:           make(map[string]interface{}),
	}

	clone.beforeEach = tester.beforeEach

	for k, v := range tester.models {
		clone.models[k] = v
	}
	for k, v := range tester.state {
		clone.state[k] = v
	}

	return clone
}

/**
 * TenantContext holds tenant-specific state for multi-tenant testing.
 */
type TenantContext struct {
	ID       string
	Data     map[string]interface{}
	Database types.DatabaseAdapter
}

// ============================================================================
// Test Organization (BDD Style)
// ============================================================================

/**
 * Run creates a sub-test with isolated state.
 *
 * This follows the testing.T.Run() pattern. The sub-test gets a clone
 * of the tester with its own state.
 *
 * Parameters:
 *   name - Name of the sub-test
 *   fn   - Function receiving the sub-tester
 *
 * Example:
 *   tester.Run("User Tests", func(t *Tester) {
 *       t.GivenAdmin()
 *       response := t.GraphQL(`{ users { name } }`)
 *       assert.NotNil(t, response)
 *   })
 */
func (tester *Tester) Run(name string, fn func(*Tester)) {
	tester.t.Helper()
	tester.t.Run(name, func(t *testing.T) {
		sub := tester.clone()
		sub.t = t
		fn(sub)
	})
}

/**
 * Describe creates a test group for BDD-style testing.
 *
 * Inspired by Jest/Mocha/Pest, this provides a readable way to group
 * related tests together.
 *
 * Parameters:
 *   description - Human-readable description of the test group
 *   fn          - Function containing the test group's tests
 *
 * Example:
 *   tester.Describe("User Authentication", func(t *Tester) {
 *       t.It("requires email and password", func(t *Tester) {
 *           // Test...
 *       })
 *   })
 */

func (tester *Tester) Describe(description string, fn func(*Tester)) {
	tester.t.Helper()
	tester.t.Logf("\n📋 %s", description)
	fn(tester)
}

/**
 * It creates a single test case for BDD-style testing.
 *
 * Each It block represents a single test scenario with a descriptive name
 * that reads like a sentence: "It should require authentication".
 *
 * Parameters:
 *   description - Description of what is being tested
 *   fn          - The test function
 *
 * Example:
 *   tester.It("returns 401 for unauthenticated requests", func(t *Tester) {
 *       tester.GraphQL(`{ users { name } }`).AssertUnauthenticated()
 *   })
 */
func (tester *Tester) It(description string, fn func(*Tester)) {
	tester.t.Helper()
	tester.t.Run(description, func(t *testing.T) {
		sub := tester.clone()
		sub.t = t

		// Run BeforeEach if set
		if sub.beforeEach != nil {
			sub.beforeEach()
		}

		// Clear auth for clean state
		sub.currentUser = nil
		sub.currentToken = ""

		fn(sub)
	})
}

/**
 * BeforeEach sets up state before each test.
 *
 * The provided function runs before each test in the current scope.
 *
 * Parameters:
 *   fn - Setup function to run before each test
 *
 * Example:
 *   tester.BeforeEach(func() {
 *       tester.RefreshDatabase().GivenAdmin()
 *   })
 */
func (tester *Tester) BeforeEach(fn func()) {
	// Store the before function on the tester
	// It will be called by It() before running the test
	tester.beforeEach = fn
}
