// pkg/adapters/database/ent.go
package database

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/mwangaben/graphqltester/types"
	"strings"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

// EntAdapter implements DatabaseAdapter for Ent ORM.
type EntAdapter struct {
	// db is the underlying sql.DB connection.
	db *sql.DB

	// driver is the Ent SQL driver.
	driver *entsql.Driver

	// config holds Ent-specific configuration.
	config *EntConfig

	// models holds the models for auto-migration.
	models []interface{}

	// dialect is the database driver name.
	dialect string

	// currentTx holds the current transaction (if any)
	currentTx *sql.Tx
}

// EntConfig holds Ent-specific configuration options.
type EntConfig struct {
	Debug              bool
	LogQueries         bool
	SlowQueryThreshold time.Duration
	SkipMigration      bool
	MaxOpenConns       int
	MaxIdleConns       int
	ConnMaxLifetime    time.Duration
	TablePrefix        string
}

// NewEntAdapter creates a new Ent adapter.
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
func (a *EntAdapter) AddModel(model interface{}) *EntAdapter {
	a.models = append(a.models, model)
	return a
}

// Connect establishes an Ent database connection.
func (a *EntAdapter) Connect(dsn string) error {
	var dialectName string
	var drv *entsql.Driver
	var err error

	// Determine dialect from DSN
	switch {
	case strings.Contains(dsn, "@tcp(") || strings.Contains(dsn, "@unix("):
		dialectName = "mysql"
		drv, err = a.connectMySQL(dsn)
	case strings.Contains(dsn, "host=") && strings.Contains(dsn, "port="):
		dialectName = "postgres"
		drv, err = a.connectPostgres(dsn)
	case strings.HasPrefix(dsn, "file:") || strings.HasSuffix(dsn, ".db") || strings.HasSuffix(dsn, ".sqlite"):
		dialectName = "sqlite3"
		drv, err = a.connectSQLite(dsn)
	default:
		dialectName = "mysql"
		drv, err = a.connectMySQL(dsn)
	}

	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	a.driver = drv
	a.dialect = dialectName

	// Get underlying sql.DB
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

	return nil
}

// connectMySQL establishes a MySQL connection.
func (a *EntAdapter) connectMySQL(dsn string) (*entsql.Driver, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	return entsql.OpenDB("mysql", db), nil
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

	return entsql.OpenDB("postgres", db), nil
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

	return entsql.OpenDB("sqlite3", db), nil
}

// Close closes the database connection.
func (a *EntAdapter) Close() error {
	if a.db != nil {
		return a.db.Close()
	}
	return nil
}

// SetMaxOpenConns sets the maximum number of open connections.
func (a *EntAdapter) SetMaxOpenConns(n int) {
	if a.db != nil {
		a.db.SetMaxOpenConns(n)
	}
}

// SetMaxIdleConns sets the maximum number of idle connections.
func (a *EntAdapter) SetMaxIdleConns(n int) {
	if a.db != nil {
		a.db.SetMaxIdleConns(n)
	}
}

// getExecer returns the appropriate execer (transaction or regular)
func (a *EntAdapter) getExecer(ctx context.Context) interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
} {
	if a.currentTx != nil {
		return a.currentTx
	}
	return a.db
}

// BeginTx starts a new transaction.
func (a *EntAdapter) BeginTx(ctx context.Context) (interface{}, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	// Start a transaction
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Store the transaction for use by other methods
	a.currentTx = tx

	return tx, nil
}

// Commit commits a transaction.
func (a *EntAdapter) Commit(tx interface{}) error {
	sqlTx, ok := tx.(*sql.Tx)
	if !ok {
		return fmt.Errorf("invalid transaction type: expected *sql.Tx")
	}

	err := sqlTx.Commit()
	if err != nil {
		return err
	}

	// Clear the current transaction
	a.currentTx = nil
	return nil
}

// Rollback rolls back a transaction.
func (a *EntAdapter) Rollback(tx interface{}) error {
	sqlTx, ok := tx.(*sql.Tx)
	if !ok {
		return fmt.Errorf("invalid transaction type: expected *sql.Tx")
	}

	err := sqlTx.Rollback()
	if err != nil {
		return err
	}

	// Clear the current transaction
	a.currentTx = nil
	return nil
}

// Exec executes a raw SQL statement.
func (a *EntAdapter) Exec(query string) error {
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}
	_, err := a.db.Exec(query)
	return err
}

// Insert inserts a new record using raw SQL.
func (a *EntAdapter) Insert(ctx context.Context, table string, data map[string]interface{}) error {
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}

	// Build INSERT query
	columns := make([]string, 0, len(data))
	placeholders := make([]string, 0, len(data))
	args := make([]interface{}, 0, len(data))

	for key, value := range data {
		columns = append(columns, key)
		placeholders = append(placeholders, "?")
		args = append(args, value)
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		table,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	// Use the appropriate execer (transaction or regular connection)
	execer := a.getExecer(ctx)
	_, err := execer.ExecContext(ctx, query, args...)
	return err
}

// Update updates records matching conditions using raw SQL.
func (a *EntAdapter) Update(ctx context.Context, table string, conditions map[string]interface{}, data map[string]interface{}) error {
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}

	// Build SET clause
	setClauses := make([]string, 0, len(data))
	args := make([]interface{}, 0, len(data)+len(conditions))

	for key, value := range data {
		setClauses = append(setClauses, fmt.Sprintf("%s = ?", key))
		args = append(args, value)
	}

	// Build WHERE clause
	whereClauses := make([]string, 0, len(conditions))
	for key, value := range conditions {
		whereClauses = append(whereClauses, fmt.Sprintf("%s = ?", key))
		args = append(args, value)
	}

	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s",
		table,
		strings.Join(setClauses, ", "),
		strings.Join(whereClauses, " AND "),
	)

	// Use the appropriate execer (transaction or regular connection)
	execer := a.getExecer(ctx)
	_, err := execer.ExecContext(ctx, query, args...)
	return err
}

// Delete removes records matching conditions using raw SQL.
func (a *EntAdapter) Delete(ctx context.Context, table string, conditions map[string]interface{}) error {
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}

	// Build WHERE clause
	whereClauses := make([]string, 0, len(conditions))
	args := make([]interface{}, 0, len(conditions))

	for key, value := range conditions {
		whereClauses = append(whereClauses, fmt.Sprintf("%s = ?", key))
		args = append(args, value)
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE %s",
		table,
		strings.Join(whereClauses, " AND "),
	)

	// Use the appropriate execer (transaction or regular connection)
	execer := a.getExecer(ctx)
	_, err := execer.ExecContext(ctx, query, args...)
	return err
}

// HasRecord checks if a record exists using raw SQL.
func (a *EntAdapter) HasRecord(ctx context.Context, table string, conditions map[string]interface{}) (bool, error) {
	if a.db == nil {
		return false, fmt.Errorf("database not initialized")
	}

	// Build the base query
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)

	// Build WHERE clause if there are conditions
	var args []interface{}
	if len(conditions) > 0 {
		whereClauses := make([]string, 0, len(conditions))
		for key, value := range conditions {
			whereClauses = append(whereClauses, fmt.Sprintf("%s = ?", key))
			args = append(args, value)
		}
		query = fmt.Sprintf("%s WHERE %s", query, strings.Join(whereClauses, " AND "))
	}

	var count int64

	// Use the appropriate execer (transaction or regular connection)
	execer := a.getExecer(ctx)
	err := execer.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("has record query failed: %w (query: %s)", err, query)
	}
	return count > 0, nil
}

// GetRecord retrieves a single record using raw SQL.
func (a *EntAdapter) GetRecord(ctx context.Context, table string, conditions map[string]interface{}) (map[string]interface{}, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	// Build the base query
	query := fmt.Sprintf("SELECT * FROM %s", table)

	// Build WHERE clause if there are conditions
	var args []interface{}
	if len(conditions) > 0 {
		whereClauses := make([]string, 0, len(conditions))
		for key, value := range conditions {
			whereClauses = append(whereClauses, fmt.Sprintf("%s = ?", key))
			args = append(args, value)
		}
		query = fmt.Sprintf("%s WHERE %s", query, strings.Join(whereClauses, " AND "))
	}

	// Always add LIMIT 1
	query = fmt.Sprintf("%s LIMIT 1", query)

	// Use the appropriate execer (transaction or regular connection)
	execer := a.getExecer(ctx)
	rows, err := execer.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w (query: %s)", err, query)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, &RecordNotFoundError{Table: table, Conditions: conditions}
	}

	// Get column names
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	// Scan the row
	values := make([]interface{}, len(cols))
	valuePtrs := make([]interface{}, len(cols))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	if err := rows.Scan(valuePtrs...); err != nil {
		return nil, err
	}

	// Build result map
	result := make(map[string]interface{})
	for i, col := range cols {
		val := values[i]
		// Convert []byte to string for better readability
		if b, ok := val.([]byte); ok {
			val = string(b)
		}
		result[col] = val
	}

	return result, nil
}

// GetRecords retrieves multiple records using raw SQL.
func (a *EntAdapter) GetRecords(ctx context.Context, table string, conditions map[string]interface{}, limit int) ([]map[string]interface{}, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	// Build the base query
	query := fmt.Sprintf("SELECT * FROM %s", table)

	// Build WHERE clause if there are conditions
	var args []interface{}
	if len(conditions) > 0 {
		whereClauses := make([]string, 0, len(conditions))
		for key, value := range conditions {
			whereClauses = append(whereClauses, fmt.Sprintf("%s = ?", key))
			args = append(args, value)
		}
		query = fmt.Sprintf("%s WHERE %s", query, strings.Join(whereClauses, " AND "))
	}

	// Add LIMIT clause if limit > 0
	if limit > 0 {
		query = fmt.Sprintf("%s LIMIT %d", query, limit)
	}

	// Use the appropriate execer (transaction or regular connection)
	execer := a.getExecer(ctx)
	rows, err := execer.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w (query: %s)", err, query)
	}
	defer rows.Close()

	// Get column names
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}

	// Scan each row
	for rows.Next() {
		values := make([]interface{}, len(cols))
		valuePtrs := make([]interface{}, len(cols))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		// Build result map
		result := make(map[string]interface{})
		for i, col := range cols {
			val := values[i]
			// Convert []byte to string for better readability
			if b, ok := val.([]byte); ok {
				val = string(b)
			}
			result[col] = val
		}
		results = append(results, result)
	}

	return results, nil
}

// Count returns the number of matching records using raw SQL.
func (a *EntAdapter) Count(ctx context.Context, table string, conditions map[string]interface{}) (int, error) {
	if a.db == nil {
		return 0, fmt.Errorf("database not initialized")
	}

	// Build the base query
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)

	// Build WHERE clause if there are conditions
	var args []interface{}
	if len(conditions) > 0 {
		whereClauses := make([]string, 0, len(conditions))
		for key, value := range conditions {
			whereClauses = append(whereClauses, fmt.Sprintf("%s = ?", key))
			args = append(args, value)
		}
		query = fmt.Sprintf("%s WHERE %s", query, strings.Join(whereClauses, " AND "))
	}

	var count int64

	// Use the appropriate execer (transaction or regular connection)
	execer := a.getExecer(ctx)
	err := execer.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count query failed: %w (query: %s)", err, query)
	}
	return int(count), nil
}

// IsSoftDeleted checks if a record is soft deleted.
func (a *EntAdapter) IsSoftDeleted(ctx context.Context, table string, conditions map[string]interface{}) (bool, error) {
	record, err := a.GetRecord(ctx, table, conditions)
	if err != nil {
		return false, err
	}

	// Check for Ent's delete_time or GORM's deleted_at
	for _, field := range []string{"delete_time", "deleted_at"} {
		if val, exists := record[field]; exists && val != nil {
			return true, nil
		}
	}
	return false, nil
}

// AutoMigrate runs auto-migration.
// For Ent, this would use the generated client.
// For testing, we skip or use raw SQL.
func (a *EntAdapter) AutoMigrate() error {
	// In a real implementation, you would use:
	// client := ent.NewClient(ent.Driver(a.driver))
	// return client.Schema.Create(context.Background())

	// For testing with raw SQL, we skip
	return nil
}

// DropAll drops all tables.
func (a *EntAdapter) DropAll() error {
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}

	tableNames, err := a.getTableNames()
	if err != nil {
		return err
	}

	for _, table := range tableNames {
		_, err := a.db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", table))
		if err != nil {
			return fmt.Errorf("failed to drop table %s: %w", table, err)
		}
	}
	return nil
}

// TruncateAll truncates all tables.
func (a *EntAdapter) TruncateAll() error {
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}

	tableNames, err := a.getTableNames()
	if err != nil {
		return err
	}

	// Disable foreign key checks for MySQL
	if a.dialect == "mysql" {
		if _, err := a.db.Exec("SET FOREIGN_KEY_CHECKS = 0"); err != nil {
			return fmt.Errorf("failed to disable foreign key checks: %w", err)
		}
		defer func() {
			_, _ = a.db.Exec("SET FOREIGN_KEY_CHECKS = 1")
		}()
	}

	for _, table := range tableNames {
		var query string
		if a.dialect == "sqlite3" {
			query = fmt.Sprintf("DELETE FROM %s", table)
		} else {
			query = fmt.Sprintf("TRUNCATE TABLE %s", table)
		}
		if _, err := a.db.Exec(query); err != nil {
			return fmt.Errorf("failed to truncate table %s: %w", table, err)
		}
	}
	return nil
}

// getTableNames returns table names from registered models.
func (a *EntAdapter) getTableNames() ([]string, error) {
	var tables []string
	for _, model := range a.models {
		switch m := model.(type) {
		case interface{ TableName() string }:
			tables = append(tables, m.TableName())
		default:
			// Fallback to type name with pluralization
			tables = append(tables, fmt.Sprintf("%T", m))
		}
	}
	return tables, nil
}

// DB returns the underlying *sql.DB connection.
func (a *EntAdapter) DB() *sql.DB {
	return a.db
}

// Driver returns the Ent SQL driver.
func (a *EntAdapter) Driver() *entsql.Driver {
	return a.driver
}

// Compile-time check
var _ types.DatabaseAdapter = (*EntAdapter)(nil)
