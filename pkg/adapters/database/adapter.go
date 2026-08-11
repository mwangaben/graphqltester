package database

import (
	"context"
	"database/sql"
	"fmt"
)

/**
 * Database Adapter Interface for GraphQL Testing
 *
 * This file defines the common interface that all database adapters must implement.
 * The adapter pattern allows the tester to work with different ORMs and database
 * drivers through a unified API, making it easy to switch between GORM, SQLx,
 * raw MySQL, or any other database library.
 *
 * Each adapter is responsible for:
 * - Connection management (connect, close, pool configuration)
 * - Transaction management (begin, commit, rollback, savepoints)
 * - CRUD operations (insert, update, delete, query)
 * - Assertion helpers (has record, count, soft delete checks)
 * - Migration support (auto-migrate, drop all, truncate)
 *
 * Design Pattern:
 *   This follows the Adapter pattern, where each database library gets its own
 *   adapter that translates its specific API into the common DatabaseAdapter interface.
 *   This allows the tester to work with any database library without code changes.
 */

/**
 * DatabaseAdapter defines the interface for database operations in tests.
 *
 * All database adapters must implement this interface to provide consistent
 * database access for testing. The interface is designed to support common
 * testing patterns like:
 * - Verifying data was persisted (HasRecord)
 * - Checking record counts (Count)
 * - Testing soft deletes (IsSoftDeleted)
 * - Transaction-based test isolation
 *
 * Implementations:
 * - GORMAdapter: For GORM ORM
 * - SQLxAdapter: For SQLx library
 * - MySQLAdapter: For raw database/sql with MySQL
 */
type DatabaseAdapter interface {
	// ========================================================================
	// Connection Management
	// ========================================================================

	/**
	 * Connect establishes a connection to the database.
	 *
	 * Parameters:
	 *   dsn - Data Source Name (connection string)
	 *         Format varies by driver:
	 *         MySQL: "user:password@tcp(host:port)/dbname?parseTime=true"
	 *         PostgreSQL: "host=localhost port=5432 user=test dbname=test sslmode=disable"
	 *
	 * Returns:
	 *   error if connection fails
	 */
	Connect(dsn string) error

	/**
	 * Close terminates the database connection.
	 *
	 * Should be called during cleanup to release resources.
	 *
	 * Returns:
	 *   error if closing fails
	 */
	Close() error

	/**
	 * SetMaxOpenConns sets the maximum number of open connections.
	 *
	 * Parameters:
	 *   n - Maximum open connections
	 */
	SetMaxOpenConns(n int)

	/**
	 * SetMaxIdleConns sets the maximum number of idle connections.
	 *
	 * Parameters:
	 *   n - Maximum idle connections
	 */
	SetMaxIdleConns(n int)

	// ========================================================================
	// Transaction Management
	// ========================================================================

	/**
	 * BeginTx starts a new database transaction.
	 *
	 * Transactions are used for test isolation - changes made within
	 * a transaction can be rolled back after the test completes.
	 *
	 * Parameters:
	 *   ctx - Context for the transaction
	 *
	 * Returns:
	 *   interface{} - The transaction object (type varies by adapter)
	 *   error if the transaction cannot be started
	 */
	BeginTx(ctx context.Context) (interface{}, error)

	/**
	 * Commit commits the current transaction.
	 *
	 * Parameters:
	 *   tx - The transaction object from BeginTx
	 *
	 * Returns:
	 *   error if commit fails
	 */
	Commit(tx interface{}) error

	/**
	 * Rollback rolls back the current transaction.
	 *
	 * Parameters:
	 *   tx - The transaction object from BeginTx
	 *
	 * Returns:
	 *   error if rollback fails
	 */
	Rollback(tx interface{}) error

	/**
	 * Exec executes a raw SQL statement.
	 *
	 * Used for savepoints and other transaction control statements.
	 *
	 * Parameters:
	 *   query - SQL statement to execute
	 *
	 * Returns:
	 *   error if execution fails
	 */
	Exec(query string) error

	// ========================================================================
	// CRUD Operations
	// ========================================================================

	/**
	 * Insert inserts a new record into the database.
	 *
	 * Parameters:
	 *   ctx   - Context for the operation
	 *   table - Table name
	 *   data  - Column-value pairs to insert
	 *
	 * Returns:
	 *   error if insertion fails
	 */
	Insert(ctx context.Context, table string, data map[string]interface{}) error

	/**
	 * Update updates records matching the conditions.
	 *
	 * Parameters:
	 *   ctx        - Context for the operation
	 *   table      - Table name
	 *   conditions - WHERE conditions
	 *   data       - Column-value pairs to update
	 *
	 * Returns:
	 *   error if update fails
	 */
	Update(ctx context.Context, table string, conditions map[string]interface{}, data map[string]interface{}) error

	/**
	 * Delete removes records matching the conditions.
	 *
	 * Parameters:
	 *   ctx        - Context for the operation
	 *   table      - Table name
	 *   conditions - WHERE conditions
	 *
	 * Returns:
	 *   error if deletion fails
	 */
	Delete(ctx context.Context, table string, conditions map[string]interface{}) error

	// ========================================================================
	// Query Operations
	// ========================================================================

	/**
	 * HasRecord checks if at least one record matches the conditions.
	 *
	 * Parameters:
	 *   ctx        - Context for the operation
	 *   table      - Table name
	 *   conditions - WHERE conditions
	 *
	 * Returns:
	 *   bool - true if at least one record exists
	 *   error if query fails
	 */
	HasRecord(ctx context.Context, table string, conditions map[string]interface{}) (bool, error)

	/**
	 * GetRecord retrieves the first record matching the conditions.
	 *
	 * Parameters:
	 *   ctx        - Context for the operation
	 *   table      - Table name
	 *   conditions - WHERE conditions
	 *
	 * Returns:
	 *   map[string]interface{} - The record as a map
	 *   error if query fails
	 */
	GetRecord(ctx context.Context, table string, conditions map[string]interface{}) (map[string]interface{}, error)

	/**
	 * GetRecords retrieves multiple records matching the conditions.
	 *
	 * Parameters:
	 *   ctx        - Context for the operation
	 *   table      - Table name
	 *   conditions - WHERE conditions (nil for all records)
	 *   limit      - Maximum records to return (0 for all)
	 *
	 * Returns:
	 *   []map[string]interface{} - Slice of records
	 *   error if query fails
	 */
	GetRecords(ctx context.Context, table string, conditions map[string]interface{}, limit int) ([]map[string]interface{}, error)

	/**
	 * Count returns the number of records matching the conditions.
	 *
	 * Parameters:
	 *   ctx        - Context for the operation
	 *   table      - Table name
	 *   conditions - WHERE conditions (nil for all records)
	 *
	 * Returns:
	 *   int - Number of matching records
	 *   error if query fails
	 */
	Count(ctx context.Context, table string, conditions map[string]interface{}) (int, error)

	// ========================================================================
	// Soft Delete Operations
	// ========================================================================

	/**
	 * IsSoftDeleted checks if a record is soft deleted.
	 *
	 * Soft deleted records have a non-null deleted_at timestamp.
	 * This checks if the record matching the conditions (including
	 * soft deleted records) has a deleted_at value.
	 *
	 * Parameters:
	 *   ctx        - Context for the operation
	 *   table      - Table name
	 *   conditions - WHERE conditions to identify the record
	 *
	 * Returns:
	 *   bool - true if the record is soft deleted
	 *   error if query fails
	 */
	IsSoftDeleted(ctx context.Context, table string, conditions map[string]interface{}) (bool, error)

	// ========================================================================
	// Migration Operations
	// ========================================================================

	/**
	 * AutoMigrate automatically migrates the database schema.
	 *
	 * This should create or update tables based on the model definitions.
	 * The exact behavior depends on the adapter (GORM's AutoMigrate vs raw SQL).
	 *
	 * Returns:
	 *   error if migration fails
	 */
	AutoMigrate() error

	/**
	 * DropAll drops all tables in the database.
	 *
	 * Use with caution - this removes all data. Typically used
	 * in MigrateFresh scenarios.
	 *
	 * Returns:
	 *   error if dropping tables fails
	 */
	DropAll() error

	/**
	 * TruncateAll truncates all tables (removes data, keeps structure).
	 *
	 * Faster than DropAll + AutoMigrate for resetting test data.
	 *
	 * Returns:
	 *   error if truncation fails
	 */
	TruncateAll() error

	// ========================================================================
	// Raw Database Access
	// ========================================================================

	/**
	 * DB returns the underlying database connection.
	 *
	 * This provides access to the raw *sql.DB for operations that
	 * aren't covered by the interface.
	 *
	 * Returns:
	 *   *sql.DB - The underlying database connection
	 */
	DB() *sql.DB
}

/**
 * RecordNotFoundError indicates that a record was not found.
 *
 * This error type allows callers to distinguish between "not found"
 * and other database errors.
 */
type RecordNotFoundError struct {
	Table      string
	Conditions map[string]interface{}
}

/**
 * Error implements the error interface for RecordNotFoundError.
 */
func (e *RecordNotFoundError) Error() string {
	return fmt.Sprintf("record not found in '%s' with conditions: %v", e.Table, e.Conditions)
}
