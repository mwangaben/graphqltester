package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

/**
 * GORMAdapter provides database testing support using GORM ORM.
 *
 * GORM is the most popular ORM for Go, providing a rich feature set
 * including auto-migration, hooks, associations, and soft deletes.
 * This adapter wraps GORM's functionality to implement the DatabaseAdapter
 * interface.
 *
 * Features:
 * - Full GORM feature support (associations, hooks, scopes)
 * - Auto-migration for test database setup
 * - Transaction support for test isolation
 * - Soft delete support via GORM's deleted_at
 * - Support for MySQL, PostgreSQL, and SQLite
 *
 * Usage:
 *   config := &Config{
 *       Database: &DatabaseConfig{
 *           Adapter: &database.GORMAdapter{},
 *           DSN: "user:pass@tcp(localhost:3306)/testdb?parseTime=true",
 *       },
 *   }
 */

// GORMAdapter implements DatabaseAdapter for GORM.
type GORMAdapter struct {
	// db is the GORM database instance.
	db *gorm.DB

	// config holds GORM-specific configuration.
	config *GORMConfig

	// models holds the models for auto-migration.
	models []interface{}

	// driver is the database driver name.
	driver string
}

/**
 * GORMConfig holds GORM-specific configuration options.
 */
type GORMConfig struct {
	// SkipDefaultTransaction disables GORM's default transaction for single operations.
	SkipDefaultTransaction bool

	// PrepareStmt caches prepared statements for performance.
	PrepareStmt bool

	// DryRun enables dry run mode (SQL is generated but not executed).
	DryRun bool

	// Logger is a custom GORM logger.
	Logger gormLogger.Interface
}

/**
 * NewGORMAdapter creates a new GORM adapter with the given configuration.
 *
 * Parameters:
 *   config - GORM-specific configuration (can be nil for defaults)
 *
 * Returns:
 *   *GORMAdapter ready for connection
 *
 * Example:
 *   adapter := NewGORMAdapter(&GORMConfig{
 *       PrepareStmt: true,
 *   })
 */
func NewGORMAdapter(config *GORMConfig) *GORMAdapter {
	if config == nil {
		config = &GORMConfig{
			SkipDefaultTransaction: true, // Recommended for testing
		}
	}

	return &GORMAdapter{
		config: config,
		models: make([]interface{}, 0),
	}
}

/**
 * AddModel registers a model for auto-migration.
 *
 * Models must be registered before calling Connect or AutoMigrate.
 *
 * Parameters:
 *   model - The model struct to register
 *
 * Returns:
 *   *GORMAdapter for fluent method chaining
 *
 * Example:
 *   adapter.AddModel(&User{}).AddModel(&Zone{}).AddModel(&Role{})
 */
func (a *GORMAdapter) AddModel(model interface{}) *GORMAdapter {
	a.models = append(a.models, model)
	return a
}

/**
 * Connect establishes a GORM database connection.
 *
 * The driver is determined from the DSN format:
 * - "user:pass@tcp(...)" -> MySQL
 * - "host=... port=..." -> PostgreSQL
 * - "file:..." -> SQLite
 *
 * Parameters:
 *   dsn - Data Source Name (connection string)
 *
 * Returns:
 *   error if connection fails
 */
func (a *GORMAdapter) Connect(dsn string) error {
	var dialector gorm.Dialector

	// Determine driver from DSN format
	// This is a simplified detection - you may want to make this configurable
	switch {
	case isMySQLDSN(dsn):
		a.driver = "mysql"
		dialector = mysql.Open(dsn)
	case isPostgresDSN(dsn):
		a.driver = "postgres"
		dialector = postgres.Open(dsn)
	case isSQLiteDSN(dsn):
		a.driver = "sqlite"
		dialector = sqlite.Open(dsn)
	default:
		// Default to MySQL
		a.driver = "mysql"
		dialector = mysql.Open(dsn)
	}

	gormConfig := &gorm.Config{
		SkipDefaultTransaction: a.config.SkipDefaultTransaction,
		PrepareStmt:            a.config.PrepareStmt,
		DryRun:                 a.config.DryRun,
	}

	if a.config.Logger != nil {
		gormConfig.Logger = a.config.Logger
	}

	db, err := gorm.Open(dialector, gormConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	a.db = db
	return nil
}

/**
 * Close closes the GORM database connection.
 *
 * Returns:
 *   error if closing fails
 */
func (a *GORMAdapter) Close() error {
	if a.db == nil {
		return nil
	}

	sqlDB, err := a.db.DB()
	if err != nil {
		return err
	}

	return sqlDB.Close()
}

/**
 * SetMaxOpenConns sets the maximum number of open connections.
 *
 * Parameters:
 *   n - Maximum open connections
 */
func (a *GORMAdapter) SetMaxOpenConns(n int) {
	if a.db == nil {
		return
	}

	sqlDB, err := a.db.DB()
	if err != nil {
		return
	}

	sqlDB.SetMaxOpenConns(n)
}

/**
 * SetMaxIdleConns sets the maximum number of idle connections.
 *
 * Parameters:
 *   n - Maximum idle connections
 */
func (a *GORMAdapter) SetMaxIdleConns(n int) {
	if a.db == nil {
		return
	}

	sqlDB, err := a.db.DB()
	if err != nil {
		return
	}

	sqlDB.SetMaxIdleConns(n)
}

/**
 * BeginTx starts a new GORM transaction.
 *
 * Parameters:
 *   ctx - Context for the transaction
 *
 * Returns:
 *   interface{} containing *gorm.DB (transaction)
 *   error if transaction start fails
 */
func (a *GORMAdapter) BeginTx(ctx context.Context) (interface{}, error) {
	tx := a.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}
	return tx, nil
}

/**
 * Commit commits a GORM transaction.
 *
 * Parameters:
 *   tx - The transaction object (*gorm.DB)
 *
 * Returns:
 *   error if commit fails
 */
func (a *GORMAdapter) Commit(tx interface{}) error {
	gormTx, ok := tx.(*gorm.DB)
	if !ok {
		return fmt.Errorf("invalid transaction type: expected *gorm.DB")
	}

	if err := gormTx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

/**
 * Rollback rolls back a GORM transaction.
 *
 * Parameters:
 *   tx - The transaction object (*gorm.DB)
 *
 * Returns:
 *   error if rollback fails
 */
func (a *GORMAdapter) Rollback(tx interface{}) error {
	gormTx, ok := tx.(*gorm.DB)
	if !ok {
		return fmt.Errorf("invalid transaction type: expected *gorm.DB")
	}

	if err := gormTx.Rollback().Error; err != nil {
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}

	return nil
}

/**
 * Exec executes a raw SQL statement.
 *
 * Parameters:
 *   query - SQL statement
 *
 * Returns:
 *   error if execution fails
 */
func (a *GORMAdapter) Exec(query string) error {
	return a.db.Exec(query).Error
}

/**
 * Insert inserts a new record using GORM.
 *
 * Uses GORM's Create method which handles timestamps, soft deletes,
 * and hooks automatically.
 *
 * Parameters:
 *   ctx   - Context for the operation
 *   table - Table name
 *   data  - Column-value pairs
 *
 * Returns:
 *   error if insertion fails
 */
func (a *GORMAdapter) Insert(ctx context.Context, table string, data map[string]interface{}) error {
	return a.db.WithContext(ctx).Table(table).Create(data).Error
}

/**
 * Update updates records matching conditions using GORM.
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
func (a *GORMAdapter) Update(ctx context.Context, table string, conditions map[string]interface{}, data map[string]interface{}) error {
	return a.db.WithContext(ctx).Table(table).Where(conditions).Updates(data).Error
}

/**
 * Delete removes records matching conditions using GORM.
 *
 * Note: If the model uses GORM's soft delete (gorm.DeletedAt),
 * records will be soft deleted instead of permanently removed.
 *
 * Parameters:
 *   ctx        - Context for the operation
 *   table      - Table name
 *   conditions - WHERE conditions
 *
 * Returns:
 *   error if deletion fails
 */
func (a *GORMAdapter) Delete(ctx context.Context, table string, conditions map[string]interface{}) error {
	return a.db.WithContext(ctx).Table(table).Where(conditions).Delete(nil).Error
}

/**
 * HasRecord checks if a record exists using GORM.
 *
 * Parameters:
 *   ctx        - Context for the operation
 *   table      - Table name
 *   conditions - WHERE conditions
 *
 * Returns:
 *   bool - true if record exists
 *   error if query fails
 */
func (a *GORMAdapter) HasRecord(ctx context.Context, table string, conditions map[string]interface{}) (bool, error) {
	var count int64
	err := a.db.WithContext(ctx).Table(table).Where(conditions).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

/**
 * GetRecord retrieves a single record using GORM.
 *
 * Parameters:
 *   ctx        - Context for the operation
 *   table      - Table name
 *   conditions - WHERE conditions
 *
 * Returns:
 *   map[string]interface{} - The record
 *   error if query fails or record not found
 */
func (a *GORMAdapter) GetRecord(ctx context.Context, table string, conditions map[string]interface{}) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := a.db.WithContext(ctx).Table(table).Where(conditions).Take(&result).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &RecordNotFoundError{Table: table, Conditions: conditions}
		}
		return nil, err
	}
	return result, nil
}

/**
 * GetRecords retrieves multiple records using GORM.
 *
 * Parameters:
 *   ctx        - Context for the operation
 *   table      - Table name
 *   conditions - WHERE conditions
 *   limit      - Maximum records to return
 *
 * Returns:
 *   []map[string]interface{} - Slice of records
 *   error if query fails
 */
func (a *GORMAdapter) GetRecords(ctx context.Context, table string, conditions map[string]interface{}, limit int) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	query := a.db.WithContext(ctx).Table(table).Where(conditions)

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&results).Error
	if err != nil {
		return nil, err
	}

	return results, nil
}

/**
 * Count returns the number of matching records using GORM.
 *
 * Parameters:
 *   ctx        - Context for the operation
 *   table      - Table name
 *   conditions - WHERE conditions
 *
 * Returns:
 *   int - Number of matching records
 *   error if query fails
 */
func (a *GORMAdapter) Count(ctx context.Context, table string, conditions map[string]interface{}) (int, error) {
	var count int64
	err := a.db.WithContext(ctx).Table(table).Where(conditions).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

/**
 * IsSoftDeleted checks if a record is soft deleted using GORM.
 *
 * GORM soft deletes set the deleted_at column. This method uses
 * Unscoped() to include soft deleted records in the query, then
 * checks if deleted_at is non-null.
 *
 * Parameters:
 *   ctx        - Context for the operation
 *   table      - Table name
 *   conditions - WHERE conditions
 *
 * Returns:
 *   bool - true if record is soft deleted
 *   error if query fails
 */
func (a *GORMAdapter) IsSoftDeleted(ctx context.Context, table string, conditions map[string]interface{}) (bool, error) {
	var result map[string]interface{}

	// Use Unscoped to include soft deleted records
	err := a.db.WithContext(ctx).Table(table).Unscoped().Where(conditions).Take(&result).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, &RecordNotFoundError{Table: table, Conditions: conditions}
		}
		return false, err
	}

	// Check if deleted_at is non-null
	deletedAt, exists := result["deleted_at"]
	if !exists {
		return false, nil
	}

	// deleted_at should be non-nil for soft deleted records
	return deletedAt != nil, nil
}

/**
 * AutoMigrate runs GORM's auto-migration for registered models.
 *
 * This creates or updates database tables based on the model structs
 * that were registered via AddModel().
 *
 * Returns:
 *   error if migration fails
 */
func (a *GORMAdapter) AutoMigrate() error {
	if len(a.models) == 0 {
		return fmt.Errorf("no models registered for auto-migration. Use AddModel() to register models")
	}

	if err := a.db.AutoMigrate(a.models...); err != nil {
		return fmt.Errorf("auto-migration failed: %w", err)
	}

	return nil
}

/**
 * DropAll drops all tables in the database.
 *
 * Uses GORM's Migrator to drop all registered tables.
 *
 * Returns:
 *   error if dropping fails
 */
func (a *GORMAdapter) DropAll() error {
	for _, model := range a.models {
		if a.db.Migrator().HasTable(model) {
			if err := a.db.Migrator().DropTable(model); err != nil {
				return fmt.Errorf("failed to drop table for %T: %w", model, err)
			}
		}
	}

	return nil
}

/**
 * TruncateAll truncates all tables (removes data, keeps structure).
 *
 * Uses DELETE instead of TRUNCATE for compatibility with transactions
 * and foreign key constraints.
 *
 * Returns:
 *   error if truncation fails
 */
func (a *GORMAdapter) TruncateAll() error {
	for _, model := range a.models {
		if a.db.Migrator().HasTable(model) {
			// Use Unscoped to include soft deleted records
			if err := a.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(model).Error; err != nil {
				return fmt.Errorf("failed to truncate table for %T: %w", model, err)
			}
		}
	}

	return nil
}

/**
 * DB returns the underlying *sql.DB connection.
 *
 * This provides access to raw database operations when needed.
 *
 * Returns:
 *   *sql.DB - The underlying database connection
 */
func (a *GORMAdapter) DB() *sql.DB {
	if a.db == nil {
		return nil
	}

	sqlDB, err := a.db.DB()
	if err != nil {
		return nil
	}

	return sqlDB
}

/**
 * GormDB returns the *gorm.DB instance for direct GORM operations.
 *
 * This allows tests to use GORM-specific features directly when needed.
 *
 * Returns:
 *   *gorm.DB - The GORM database instance
 */
func (a *GORMAdapter) GormDB() *gorm.DB {
	return a.db
}

// ============================================================================
// DSN Detection Helpers
// ============================================================================

/**
 * isMySQLDSN checks if a DSN string is for MySQL.
 *
 * MySQL DSN format: user:password@tcp(host:port)/dbname?params
 */
func isMySQLDSN(dsn string) bool {
	return strings.Contains(dsn, "@tcp(") || strings.Contains(dsn, "@unix(")
}

/**
 * isPostgresDSN checks if a DSN string is for PostgreSQL.
 *
 * PostgreSQL DSN format: host=... port=... user=... dbname=...
 */
func isPostgresDSN(dsn string) bool {
	return strings.Contains(dsn, "host=") && strings.Contains(dsn, "port=")
}

/**
 * isSQLiteDSN checks if a DSN string is for SQLite.
 *
 * SQLite DSN format: file:path/to/db?params or just a file path
 */
func isSQLiteDSN(dsn string) bool {
	return strings.HasPrefix(dsn, "file:") || strings.HasSuffix(dsn, ".db") || strings.HasSuffix(dsn, ".sqlite")
}
