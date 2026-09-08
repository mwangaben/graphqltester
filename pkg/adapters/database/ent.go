// pkg/adapters/database/ent.go
package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"entgo.io/ent/entc"
	"entgo.io/ent/schema/field"
)

// EntAdapter implements DatabaseAdapter for Ent ORM.
//
// Ent is a type-safe ORM for Go that uses code generation to provide
// a strongly-typed query builder. This adapter wraps Ent's functionality
// to implement the DatabaseAdapter interface.
//
// Features:
// - Type-safe queries with Ent's generated client
// - Auto-migration with Ent's schema migration
// - Transaction support with Ent's Tx
// - Support for MySQL, PostgreSQL, and SQLite
// - Debug logging of generated SQL
//
// Usage:
//
//	adapter := database.NewEntAdapter(&database.EntConfig{
//	    Debug: true,
//	})
//	adapter.AddModel(&ent.User{}).AddModel(&ent.Zone{})
//
//	config := &Config{
//	    Database: &DatabaseConfig{
//	        Adapter: adapter,
//	        DSN: "root:password@tcp(localhost:3306)/testdb?parseTime=true",
//	    },
//	}
type EntAdapter struct {
	// client is the Ent client interface.
	client interface {
		Close() error
		Schema() interface {
			Create(ctx context.Context, opts ...interface{}) error
			Drop(ctx context.Context, opts ...interface{}) error
		}
	}

	// db is the underlying sql.DB connection.
	db *sql.DB

	// driver is the Ent SQL driver.
	driver *entsql.Driver

	// config holds Ent-specific configuration.
	config *EntConfig

	// models holds the models for auto-migration.
	models []interface{}

	// driverName is the database driver name.
	driverName string

	// schemaName is the name of the schema.
	schemaName string
}

// EntConfig holds Ent-specific configuration options.
type EntConfig struct {
	// Debug enables debug logging of SQL queries.
	Debug bool

	// LogQueries logs all SQL queries to stdout (implies Debug).
	LogQueries bool

	// SlowQueryThreshold logs queries that take longer than this duration.
	SlowQueryThreshold time.Duration

	// SkipMigration skips auto-migration on connect.
	SkipMigration bool

	// MaxOpenConns sets the maximum number of open connections.
	MaxOpenConns int

	// MaxIdleConns sets the maximum number of idle connections.
	MaxIdleConns int

	// ConnMaxLifetime sets the maximum lifetime of a connection.
	ConnMaxLifetime time.Duration

	// TablePrefix adds a prefix to all table names.
	TablePrefix string
}

// RecordNotFoundError is returned when a record is not found.
type RecordNotFoundError struct {
	Table      string
	Conditions map[string]interface{}
}

func (e *RecordNotFoundError) Error() string {
	return fmt.Sprintf("record not found in table %s with conditions %v", e.Table, e.Conditions)
}

// NewEntAdapter creates a new Ent adapter with the given configuration.
//
// Parameters:
//
//	config - Ent-specific configuration (can be nil for defaults)
//
// Returns:
//
//	*EntAdapter ready for connection
//
// Example:
//
//	adapter := NewEntAdapter(&EntConfig{
//	    Debug: true,
//	    SlowQueryThreshold: 100 * time.Millisecond,
//	})
func NewEntAdapter(config *EntConfig) *EntAdapter {
	if config == nil {
		config = &EntConfig{
			Debug:              false,
			LogQueries:         false,
			SlowQueryThreshold: 100 * time.Millisecond,
			SkipMigration:      false,
		}
	}

	return &EntAdapter{
		config: config,
		models: make([]interface{}, 0),
	}
}

// AddModel registers a model for auto-migration.
//
// Models must be registered before calling Connect or AutoMigrate.
//
// Parameters:
//
//	model - The model struct to register
//
// Returns:
//
//	*EntAdapter for fluent method chaining
//
// Example:
//
//	adapter.AddModel(&User{}).AddModel(&Zone{}).AddModel(&Role{})
func (a *EntAdapter) AddModel(model interface{}) *EntAdapter {
	a.models = append(a.models, model)
	return a
}

// Connect establishes an Ent database connection.
//
// The driver is determined from the DSN format:
// - "user:pass@tcp(...)" -> MySQL
// - "host=... port=..." -> PostgreSQL
// - "file:..." -> SQLite
//
// Parameters:
//
//	dsn - Data Source Name (connection string)
//
// Returns:
//
//	error if connection fails
func (a *EntAdapter) Connect(dsn string) error {
	var dialectName string
	var drv *entsql.Driver
	var err error

	// Determine dialect from DSN
	switch {
	case isMySQLDSN(dsn):
		dialectName = dialect.MySQL
		drv, err = a.connectMySQL(dsn)
	case isPostgresDSN(dsn):
		dialectName = dialect.PostgreSQL
		drv, err = a.connectPostgres(dsn)
	case isSQLiteDSN(dsn):
		dialectName = dialect.SQLite
		drv, err = a.connectSQLite(dsn)
	default:
		// Default to MySQL
		dialectName = dialect.MySQL
		drv, err = a.connectMySQL(dsn)
	}

	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	a.driver = drv
	a.driverName = dialectName

	// Get underlying sql.DB for connection management
	sqlDB := drv.DB()
	a.db = sqlDB

	// Configure connection pool
	if a.config.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(a.config.MaxOpenConns)
	}
	if a.config.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(a.config.MaxIdleConns)
	}
	if a.config.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(a.config.ConnMaxLifetime)
	}

	// Create Ent client using the generated client
	// Note: You need to generate the Ent client for your project
	// and pass it here, or use a factory pattern
	client, err := a.createClient()
	if err != nil {
		return fmt.Errorf("failed to create Ent client: %w", err)
	}
	a.client = client

	// Run auto-migration
	if !a.config.SkipMigration {
		ctx := context.Background()
		if err := a.AutoMigrate(); err != nil {
			return fmt.Errorf("auto-migration failed: %w", err)
		}
	}

	return nil
}

// createClient creates an Ent client.
// This is a placeholder - you should use your generated Ent client.
func (a *EntAdapter) createClient() (interface {
	Close() error
	Schema() interface {
		Create(ctx context.Context, opts ...interface{}) error
		Drop(ctx context.Context, opts ...interface{}) error
	}
}, error) {
	// This is a placeholder implementation.
	// In practice, you would import your generated Ent client:
	// import "your-project/ent"
	// client := ent.NewClient(ent.Driver(a.driver))
	// return client, nil

	return nil, fmt.Errorf("createClient must be implemented with your generated Ent client")
}

// connectMySQL establishes a MySQL connection.
func (a *EntAdapter) connectMySQL(dsn string) (*entsql.Driver, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	drv := entsql.OpenDB(dialect.MySQL, db)
	return drv, nil
}

// connectPostgres establishes a PostgreSQL connection.
func (a *EntAdapter) connectPostgres(dsn string) (*entsql.Driver, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	drv := entsql.OpenDB(dialect.PostgreSQL, db)
	return drv, nil
}

// connectSQLite establishes a SQLite connection.
func (a *EntAdapter) connectSQLite(dsn string) (*entsql.Driver, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	drv := entsql.OpenDB(dialect.SQLite, db)
	return drv, nil
}

// Close closes the Ent database connection.
//
// Returns:
//
//	error if closing fails
func (a *EntAdapter) Close() error {
	var errs []string

	if a.client != nil {
		if err := a.client.Close(); err != nil {
			errs = append(errs, fmt.Sprintf("client close: %v", err))
		}
	}

	if a.db != nil {
		if err := a.db.Close(); err != nil {
			errs = append(errs, fmt.Sprintf("db close: %v", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing database: %s", strings.Join(errs, "; "))
	}

	return nil
}

// SetMaxOpenConns sets the maximum number of open connections.
//
// Parameters:
//
//	n - Maximum open connections
func (a *EntAdapter) SetMaxOpenConns(n int) {
	if a.db != nil {
		a.db.SetMaxOpenConns(n)
	}
}

// SetMaxIdleConns sets the maximum number of idle connections.
//
// Parameters:
//
//	n - Maximum idle connections
func (a *EntAdapter) SetMaxIdleConns(n int) {
	if a.db != nil {
		a.db.SetMaxIdleConns(n)
	}
}

// BeginTx starts a new Ent transaction.
//
// Parameters:
//
//	ctx - Context for the transaction
//
// Returns:
//
//	interface{} containing *ent.Tx
//	error if transaction start fails
func (a *EntAdapter) BeginTx(ctx context.Context) (interface{}, error) {
	// This should return a transaction from your generated Ent client
	return nil, fmt.Errorf("BeginTx must be implemented with your generated Ent client")
}

// Commit commits an Ent transaction.
//
// Parameters:
//
//	tx - The transaction object (*ent.Tx)
//
// Returns:
//
//	error if commit fails
func (a *EntAdapter) Commit(tx interface{}) error {
	// This should commit the transaction from your generated Ent client
	return fmt.Errorf("Commit must be implemented with your generated Ent client")
}

// Rollback rolls back an Ent transaction.
//
// Parameters:
//
//	tx - The transaction object (*ent.Tx)
//
// Returns:
//
//	error if rollback fails
func (a *EntAdapter) Rollback(tx interface{}) error {
	// This should rollback the transaction from your generated Ent client
	return fmt.Errorf("Rollback must be implemented with your generated Ent client")
}

// Exec executes a raw SQL statement.
//
// Parameters:
//
//	query - SQL statement
//
// Returns:
//
//	error if execution fails
func (a *EntAdapter) Exec(query string) error {
	if a.driver == nil {
		return fmt.Errorf("driver not initialized. Call Connect first")
	}
	return a.driver.Exec(query)
}

// Insert inserts a new record using Ent.
//
// Parameters:
//
//	ctx   - Context for the operation
//	table - Table name
//	data  - Column-value pairs
//
// Returns:
//
//	error if insertion fails
func (a *EntAdapter) Insert(ctx context.Context, table string, data map[string]interface{}) error {
	if a.driver == nil {
		return fmt.Errorf("driver not initialized. Call Connect first")
	}

	// Build insert query
	insert := a.driver.Insert(table)
	for key, value := range data {
		insert.Set(key, value)
	}

	return insert.Exec(ctx)
}

// Update updates records matching conditions using Ent.
//
// Parameters:
//
//	ctx        - Context for the operation
//	table      - Table name
//	conditions - WHERE conditions
//	data       - Column-value pairs to update
//
// Returns:
//
//	error if update fails
func (a *EntAdapter) Update(ctx context.Context, table string, conditions map[string]interface{}, data map[string]interface{}) error {
	if a.driver == nil {
		return fmt.Errorf("driver not initialized. Call Connect first")
	}

	// Build update query
	update := a.driver.Update(table)
	for key, value := range data {
		update.Set(key, value)
	}

	// Add conditions
	for key, value := range conditions {
		update.Where(entsql.EQ(key, value))
	}

	return update.Exec(ctx)
}

// Delete removes records matching conditions using Ent.
//
// Note: If the model uses soft delete, records will be soft deleted
// instead of permanently removed.
//
// Parameters:
//
//	ctx        - Context for the operation
//	table      - Table name
//	conditions - WHERE conditions
//
// Returns:
//
//	error if deletion fails
func (a *EntAdapter) Delete(ctx context.Context, table string, conditions map[string]interface{}) error {
	if a.driver == nil {
		return fmt.Errorf("driver not initialized. Call Connect first")
	}

	// Build delete query
	delete := a.driver.Delete(table)

	// Add conditions
	for key, value := range conditions {
		delete.Where(entsql.EQ(key, value))
	}

	return delete.Exec(ctx)
}

// HasRecord checks if a record exists using Ent.
//
// Parameters:
//
//	ctx        - Context for the operation
//	table      - Table name
//	conditions - WHERE conditions
//
// Returns:
//
//	bool - true if record exists
//	error if query fails
func (a *EntAdapter) HasRecord(ctx context.Context, table string, conditions map[string]interface{}) (bool, error) {
	if a.driver == nil {
		return false, fmt.Errorf("driver not initialized. Call Connect first")
	}

	// Build select query for count
	query := a.driver.Select().From(entsql.Table(table))

	// Add conditions
	for key, value := range conditions {
		query.Where(entsql.EQ(key, value))
	}

	var count int64
	if err := query.Count(ctx, &count); err != nil {
		return false, err
	}

	return count > 0, nil
}

// GetRecord retrieves a single record using Ent.
//
// Parameters:
//
//	ctx        - Context for the operation
//	table      - Table name
//	conditions - WHERE conditions
//
// Returns:
//
//	map[string]interface{} - The record
//	error if query fails or record not found
func (a *EntAdapter) GetRecord(ctx context.Context, table string, conditions map[string]interface{}) (map[string]interface{}, error) {
	if a.driver == nil {
		return nil, fmt.Errorf("driver not initialized. Call Connect first")
	}

	// Build select query
	query := a.driver.Select().From(entsql.Table(table))

	// Add conditions
	for key, value := range conditions {
		query.Where(entsql.EQ(key, value))
	}

	// Limit to one record
	query.Limit(1)

	var result map[string]interface{}
	if err := query.Scan(ctx, &result); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &RecordNotFoundError{Table: table, Conditions: conditions}
		}
		return nil, err
	}

	if result == nil {
		return nil, &RecordNotFoundError{Table: table, Conditions: conditions}
	}

	return result, nil
}

// GetRecords retrieves multiple records using Ent.
//
// Parameters:
//
//	ctx        - Context for the operation
//	table      - Table name
//	conditions - WHERE conditions
//	limit      - Maximum records to return
//
// Returns:
//
//	[]map[string]interface{} - Slice of records
//	error if query fails
func (a *EntAdapter) GetRecords(ctx context.Context, table string, conditions map[string]interface{}, limit int) ([]map[string]interface{}, error) {
	if a.driver == nil {
		return nil, fmt.Errorf("driver not initialized. Call Connect first")
	}

	// Build select query
	query := a.driver.Select().From(entsql.Table(table))

	// Add conditions
	for key, value := range conditions {
		query.Where(entsql.EQ(key, value))
	}

	if limit > 0 {
		query.Limit(limit)
	}

	var results []map[string]interface{}
	if err := query.Scan(ctx, &results); err != nil {
		return nil, err
	}

	return results, nil
}

// Count returns the number of matching records using Ent.
//
// Parameters:
//
//	ctx        - Context for the operation
//	table      - Table name
//	conditions - WHERE conditions
//
// Returns:
//
//	int - Number of matching records
//	error if query fails
func (a *EntAdapter) Count(ctx context.Context, table string, conditions map[string]interface{}) (int, error) {
	if a.driver == nil {
		return 0, fmt.Errorf("driver not initialized. Call Connect first")
	}

	// Build select query for count
	query := a.driver.Select().From(entsql.Table(table))

	// Add conditions
	for key, value := range conditions {
		query.Where(entsql.EQ(key, value))
	}

	var count int64
	if err := query.Count(ctx, &count); err != nil {
		return 0, err
	}

	return int(count), nil
}

// IsSoftDeleted checks if a record is soft deleted using Ent.
//
// Ent soft deletes set the delete_time column. This method checks
// if delete_time is non-null.
//
// Parameters:
//
//	ctx        - Context for the operation
//	table      - Table name
//	conditions - WHERE conditions
//
// Returns:
//
//	bool - true if record is soft deleted
//	error if query fails
func (a *EntAdapter) IsSoftDeleted(ctx context.Context, table string, conditions map[string]interface{}) (bool, error) {
	// Get the record
	record, err := a.GetRecord(ctx, table, conditions)
	if err != nil {
		return false, err
	}

	// Check if delete_time exists and is non-null
	deleteTime, exists := record["delete_time"]
	if !exists {
		// Also check for GORM-style soft delete for compatibility
		deleteTime, exists = record["deleted_at"]
		if !exists {
			return false, nil
		}
	}

	// Return true if delete_time is non-nil
	return deleteTime != nil, nil
}

// AutoMigrate runs Ent's auto-migration for registered models.
//
// This creates or updates database tables based on the model structs
// that were registered via AddModel().
//
// Returns:
//
//	error if migration fails
func (a *EntAdapter) AutoMigrate() error {
	if a.client == nil {
		return fmt.Errorf("client not initialized. Call Connect first")
	}

	if len(a.models) == 0 {
		return fmt.Errorf("no models registered for auto-migration. Use AddModel() to register models")
	}

	ctx := context.Background()

	// Use Ent's schema migration
	// Note: This uses the generated client's Schema
	schema := a.client.Schema()
	if err := schema.Create(ctx); err != nil {
		return fmt.Errorf("auto-migration failed: %w", err)
	}

	return nil
}

// DropAll drops all tables in the database.
//
// Returns:
//
//	error if dropping fails
func (a *EntAdapter) DropAll() error {
	if a.client == nil {
		return fmt.Errorf("client not initialized. Call Connect first")
	}

	ctx := context.Background()

	// Use Ent's schema drop
	schema := a.client.Schema()
	if err := schema.Drop(ctx); err != nil {
		return fmt.Errorf("failed to drop tables: %w", err)
	}

	return nil
}

// TruncateAll truncates all tables (removes data, keeps structure).
//
// Uses raw SQL for truncation as Ent doesn't have built-in truncate.
//
// Returns:
//
//	error if truncation fails
func (a *EntAdapter) TruncateAll() error {
	if a.db == nil {
		return fmt.Errorf("database not initialized. Call Connect first")
	}

	// Get table names from models
	tableNames, err := a.getTableNames()
	if err != nil {
		return err
	}

	// Disable foreign key checks for MySQL
	if a.driverName == dialect.MySQL {
		if _, err := a.db.Exec("SET FOREIGN_KEY_CHECKS = 0"); err != nil {
			return fmt.Errorf("failed to disable foreign key checks: %w", err)
		}
		defer a.db.Exec("SET FOREIGN_KEY_CHECKS = 1")
	}

	// Truncate each table
	for _, table := range tableNames {
		query := fmt.Sprintf("TRUNCATE TABLE %s", table)
		if a.driverName == dialect.SQLite {
			// SQLite uses DELETE for truncation
			query = fmt.Sprintf("DELETE FROM %s", table)
		}
		if _, err := a.db.Exec(query); err != nil {
			return fmt.Errorf("failed to truncate table %s: %w", table, err)
		}
	}

	return nil
}

// getTableNames returns the table names for registered models.
func (a *EntAdapter) getTableNames() ([]string, error) {
	var tables []string
	for _, model := range a.models {
		// Use reflection or a naming strategy to get table name
		// This is a placeholder - you should implement proper table naming
		switch m := model.(type) {
		case interface{ TableName() string }:
			tables = append(tables, m.TableName())
		default:
			// Fallback: use type name as table name with pluralization
			// In practice, use Ent's naming convention or a custom strategy
			tables = append(tables, fmt.Sprintf("%T", m))
		}
	}
	return tables, nil
}

// DB returns the underlying *sql.DB connection.
//
// This provides access to raw database operations when needed.
//
// Returns:
//
//	*sql.DB - The underlying database connection
func (a *EntAdapter) DB() *sql.DB {
	return a.db
}

// Driver returns the Ent SQL driver.
//
// Returns:
//
//	*entsql.Driver - The Ent SQL driver
func (a *EntAdapter) Driver() *entsql.Driver {
	return a.driver
}

// ============================================================================
// DSN Detection Helpers
// ============================================================================

// isMySQLDSN checks if a DSN string is for MySQL.
//
// MySQL DSN format: user:password@tcp(host:port)/dbname?params
func isMySQLDSN(dsn string) bool {
	return strings.Contains(dsn, "@tcp(") || strings.Contains(dsn, "@unix(")
}

// isPostgresDSN checks if a DSN string is for PostgreSQL.
//
// PostgreSQL DSN format: host=... port=... user=... dbname=...
func isPostgresDSN(dsn string) bool {
	return strings.Contains(dsn, "host=") && strings.Contains(dsn, "port=")
}

// isSQLiteDSN checks if a DSN string is for SQLite.
//
// SQLite DSN format: file:path/to/db?params or just a file path
func isSQLiteDSN(dsn string) bool {
	return strings.HasPrefix(dsn, "file:") || strings.HasSuffix(dsn, ".db") || strings.HasSuffix(dsn, ".sqlite")
}

// ============================================================================
// Helper Functions
// ============================================================================

// WithDebug returns a new EntConfig with debug enabled.
func WithDebug(debug bool) *EntConfig {
	return &EntConfig{
		Debug:              debug,
		LogQueries:         debug,
		SlowQueryThreshold: 100 * time.Millisecond,
	}
}

// WithLogging returns a new EntConfig with query logging enabled.
func WithLogging(logQueries bool, threshold time.Duration) *EntConfig {
	return &EntConfig{
		Debug:              false,
		LogQueries:         logQueries,
		SlowQueryThreshold: threshold,
	}
}
