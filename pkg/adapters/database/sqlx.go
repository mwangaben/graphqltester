package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"

	// Database drivers
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

/**
 * SQLxAdapter provides database testing support using the SQLx library.
 *
 * SQLx extends the standard database/sql library with:
 * - Named parameter support
 * - Struct scanning
 * - Compile-time query checking
 * - Easier result mapping
 *
 * This adapter is a good choice when you want more control than GORM
 * but still want convenience features over raw database/sql.
 *
 * Usage:
 *   config := &Config{
 *       Database: &DatabaseConfig{
 *           Adapter: &database.SQLxAdapter{},
 *           DSN: "user:pass@tcp(localhost:3306)/testdb?parseTime=true",
 *       },
 *   }
 */

// SQLxAdapter implements DatabaseAdapter for SQLx.
type SQLxAdapter struct {
	// db is the SQLx database instance.
	db *sqlx.DB

	// driver is the database driver name.
	driver string

	// tables holds table names for migration operations.
	tables []string
}

/**
 * NewSQLxAdapter creates a new SQLx adapter.
 *
 * Parameters:
 *   driver - Database driver name ("mysql", "postgres", "sqlite3")
 *
 * Returns:
 *   *SQLxAdapter ready for connection
 *
 * Example:
 *   adapter := NewSQLxAdapter("mysql")
 */
func NewSQLxAdapter(driver string) *SQLxAdapter {
	return &SQLxAdapter{
		driver: driver,
		tables: make([]string, 0),
	}
}

/**
 * AddTable registers a table for migration operations.
 *
 * Parameters:
 *   tableName - Name of the table
 *
 * Returns:
 *   *SQLxAdapter for fluent method chaining
 */
func (a *SQLxAdapter) AddTable(tableName string) *SQLxAdapter {
	a.tables = append(a.tables, tableName)
	return a
}

/**
 * Connect establishes a SQLx database connection.
 *
 * Parameters:
 *   dsn - Data Source Name
 *
 * Returns:
 *   error if connection fails
 */
func (a *SQLxAdapter) Connect(dsn string) error {
	db, err := sqlx.Connect(a.driver, dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	a.db = db
	return nil
}

/**
 * Close closes the SQLx database connection.
 *
 * Returns:
 *   error if closing fails
 */
func (a *SQLxAdapter) Close() error {
	if a.db == nil {
		return nil
	}
	return a.db.Close()
}

/**
 * SetMaxOpenConns sets the maximum number of open connections.
 *
 * Parameters:
 *   n - Maximum open connections
 */
func (a *SQLxAdapter) SetMaxOpenConns(n int) {
	if a.db != nil {
		a.db.SetMaxOpenConns(n)
	}
}

/**
 * SetMaxIdleConns sets the maximum number of idle connections.
 *
 * Parameters:
 *   n - Maximum idle connections
 */
func (a *SQLxAdapter) SetMaxIdleConns(n int) {
	if a.db != nil {
		a.db.SetMaxIdleConns(n)
	}
}

/**
 * BeginTx starts a new SQLx transaction.
 *
 * Parameters:
 *   ctx - Context for the transaction
 *
 * Returns:
 *   interface{} containing *sqlx.Tx
 *   error if transaction start fails
 */
func (a *SQLxAdapter) BeginTx(ctx context.Context) (interface{}, error) {
	tx, err := a.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return tx, nil
}

/**
 * Commit commits a SQLx transaction.
 *
 * Parameters:
 *   tx - The transaction object (*sqlx.Tx)
 *
 * Returns:
 *   error if commit fails
 */
func (a *SQLxAdapter) Commit(tx interface{}) error {
	sqlxTx, ok := tx.(*sqlx.Tx)
	if !ok {
		return fmt.Errorf("invalid transaction type: expected *sqlx.Tx")
	}
	return sqlxTx.Commit()
}

/**
 * Rollback rolls back a SQLx transaction.
 *
 * Parameters:
 *   tx - The transaction object (*sqlx.Tx)
 *
 * Returns:
 *   error if rollback fails
 */
func (a *SQLxAdapter) Rollback(tx interface{}) error {
	sqlxTx, ok := tx.(*sqlx.Tx)
	if !ok {
		return fmt.Errorf("invalid transaction type: expected *sqlx.Tx")
	}
	return sqlxTx.Rollback()
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
func (a *SQLxAdapter) Exec(query string) error {
	_, err := a.db.Exec(query)
	return err
}

/**
 * Insert inserts a new record using SQLx named parameters.
 *
 * Parameters:
 *   ctx   - Context for the operation
 *   table - Table name
 *   data  - Column-value pairs
 *
 * Returns:
 *   error if insertion fails
 */
func (a *SQLxAdapter) Insert(ctx context.Context, table string, data map[string]interface{}) error {
	columns := make([]string, 0, len(data))
	placeholders := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))

	for col, val := range data {
		columns = append(columns, col)
		placeholders = append(placeholders, ":"+col)
		values = append(values, sql.Named(col, val))
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		table,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	_, err := a.db.NamedExecContext(ctx, query, data)
	return err
}

/**
 * Update updates records matching conditions using SQLx.
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
func (a *SQLxAdapter) Update(ctx context.Context, table string, conditions map[string]interface{}, data map[string]interface{}) error {
	setClauses := make([]string, 0, len(data))
	whereClauses := make([]string, 0, len(conditions))

	// Build SET clause
	for col := range data {
		setClauses = append(setClauses, fmt.Sprintf("%s = :%s", col, col))
	}

	// Build WHERE clause
	for col := range conditions {
		whereClauses = append(whereClauses, fmt.Sprintf("%s = :where_%s", col, col))
		data["where_"+col] = conditions[col]
	}

	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s",
		table,
		strings.Join(setClauses, ", "),
		strings.Join(whereClauses, " AND "),
	)

	_, err := a.db.NamedExecContext(ctx, query, data)
	return err
}

/**
 * Delete removes records matching conditions using SQLx.
 *
 * Parameters:
 *   ctx        - Context for the operation
 *   table      - Table name
 *   conditions - WHERE conditions
 *
 * Returns:
 *   error if deletion fails
 */
func (a *SQLxAdapter) Delete(ctx context.Context, table string, conditions map[string]interface{}) error {
	whereClauses := make([]string, 0, len(conditions))

	for col := range conditions {
		whereClauses = append(whereClauses, fmt.Sprintf("%s = :%s", col, col))
	}

	query := fmt.Sprintf(
		"DELETE FROM %s WHERE %s",
		table,
		strings.Join(whereClauses, " AND "),
	)

	_, err := a.db.NamedExecContext(ctx, query, conditions)
	return err
}

/**
 * HasRecord checks if a record exists using SQLx.
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
func (a *SQLxAdapter) HasRecord(ctx context.Context, table string, conditions map[string]interface{}) (bool, error) {
	count, err := a.Count(ctx, table, conditions)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

/**
 * GetRecord retrieves a single record using SQLx.
 *
 * Parameters:
 *   ctx        - Context for the operation
 *   table      - Table name
 *   conditions - WHERE conditions
 *
 * Returns:
 *   map[string]interface{} - The record
 *   error if query fails
 */
func (a *SQLxAdapter) GetRecord(ctx context.Context, table string, conditions map[string]interface{}) (map[string]interface{}, error) {
	whereClauses := make([]string, 0, len(conditions))

	for col := range conditions {
		whereClauses = append(whereClauses, fmt.Sprintf("%s = :%s", col, col))
	}

	query := fmt.Sprintf(
		"SELECT * FROM %s WHERE %s LIMIT 1",
		table,
		strings.Join(whereClauses, " AND "),
	)

	rows, err := a.db.NamedQueryContext(ctx, query, conditions)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, &RecordNotFoundError{Table: table, Conditions: conditions}
	}

	result := make(map[string]interface{})
	if err := rows.MapScan(result); err != nil {
		return nil, err
	}

	return result, nil
}

/**
 * GetRecords retrieves multiple records using SQLx.
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
func (a *SQLxAdapter) GetRecords(ctx context.Context, table string, conditions map[string]interface{}, limit int) ([]map[string]interface{}, error) {
	var whereClauses []string

	for col := range conditions {
		whereClauses = append(whereClauses, fmt.Sprintf("%s = :%s", col, col))
	}

	query := fmt.Sprintf("SELECT * FROM %s", table)

	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := a.db.NamedQueryContext(ctx, query, conditions)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}

	for rows.Next() {
		result := make(map[string]interface{})
		if err := rows.MapScan(result); err != nil {
			return nil, err
		}
		results = append(results, result)
	}

	return results, rows.Err()
}

/**
 * Count returns the number of matching records using SQLx.
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
func (a *SQLxAdapter) Count(ctx context.Context, table string, conditions map[string]interface{}) (int, error) {
	whereClauses := make([]string, 0, len(conditions))

	for col := range conditions {
		whereClauses = append(whereClauses, fmt.Sprintf("%s = :%s", col, col))
	}

	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)

	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	var count int
	rows, err := a.db.NamedQueryContext(ctx, query, conditions)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	if rows.Next() {
		if err := rows.Scan(&count); err != nil {
			return 0, err
		}
	}

	return count, nil
}

/**
 * IsSoftDeleted checks if a record is soft deleted using SQLx.
 *
 * Assumes the table has a deleted_at column for soft deletes.
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
func (a *SQLxAdapter) IsSoftDeleted(ctx context.Context, table string, conditions map[string]interface{}) (bool, error) {
	whereClauses := make([]string, 0, len(conditions))

	for col := range conditions {
		whereClauses = append(whereClauses, fmt.Sprintf("%s = :%s", col, col))
	}

	query := fmt.Sprintf(
		"SELECT deleted_at FROM %s WHERE %s LIMIT 1",
		table,
		strings.Join(whereClauses, " AND "),
	)

	rows, err := a.db.NamedQueryContext(ctx, query, conditions)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	if !rows.Next() {
		return false, &RecordNotFoundError{Table: table, Conditions: conditions}
	}

	var deletedAt sql.NullTime
	if err := rows.Scan(&deletedAt); err != nil {
		return false, err
	}

	return deletedAt.Valid, nil
}

/**
 * AutoMigrate runs SQL-based migrations for registered tables.
 *
 * This method relies on pre-defined CREATE TABLE statements
 * or external migration tools. For SQLx, you typically handle
 * migrations separately.
 *
 * Returns:
 *   error if migration fails
 */
func (a *SQLxAdapter) AutoMigrate() error {
	// SQLx doesn't have built-in auto-migration like GORM.
	// You would typically run migration files or have pre-defined
	// CREATE TABLE statements.
	//
	// This is a placeholder that would be implemented based on
	// your migration strategy.
	return nil
}

/**
 * DropAll drops all registered tables.
 *
 * Returns:
 *   error if dropping fails
 */
func (a *SQLxAdapter) DropAll() error {
	for _, table := range a.tables {
		if _, err := a.db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", table)); err != nil {
			return fmt.Errorf("failed to drop table %s: %w", table, err)
		}
	}

	return nil
}

/**
 * TruncateAll truncates all registered tables.
 *
 * Returns:
 *   error if truncation fails
 */
func (a *SQLxAdapter) TruncateAll() error {
	for _, table := range a.tables {
		if _, err := a.db.Exec(fmt.Sprintf("DELETE FROM %s", table)); err != nil {
			return fmt.Errorf("failed to truncate table %s: %w", table, err)
		}
	}

	return nil
}

/**
 * DB returns the underlying *sql.DB connection.
 *
 * Returns:
 *   *sql.DB - The underlying database connection
 */
func (a *SQLxAdapter) DB() *sql.DB {
	if a.db == nil {
		return nil
	}
	return a.db.DB
}

/**
 * SqlxDB returns the *sqlx.DB instance for direct SQLx operations.
 *
 * Returns:
 *   *sqlx.DB - The SQLx database instance
 */
func (a *SQLxAdapter) SqlxDB() *sqlx.DB {
	return a.db
}
