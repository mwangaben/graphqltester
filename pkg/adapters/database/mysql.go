package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

/**
 * MySQLAdapter provides database testing support using raw database/sql with MySQL.
 *
 * This adapter is the lowest-level option, using only the standard library's
 * database/sql package with the MySQL driver. It provides maximum control
 * and minimal dependencies at the cost of more verbose code.
 *
 * Use this adapter when:
 * - You want minimal dependencies
 * - You need direct control over SQL queries
 * - You're not using an ORM
 * - You want maximum performance
 *
 * Usage:
 *   config := &Config{
 *       Database: &DatabaseConfig{
 *           Adapter: &database.MySQLAdapter{},
 *           DSN: "user:pass@tcp(localhost:3306)/testdb?parseTime=true",
 *       },
 *   }
 */

// MySQLAdapter implements DatabaseAdapter for raw MySQL.
type MySQLAdapter struct {
	// db is the standard database/sql connection.
	db *sql.DB

	// tables holds table names for migration operations.
	tables []string

	// migrations holds SQL migration statements.
	migrations []string
}

//NewMySQLAdapter
/**
 * NewMySQLAdapter creates a new MySQL adapter.
 *
 * Returns:
 *   *MySQLAdapter ready for connection
 *
 * Example:
 *   adapter := NewMySQLAdapter()
 *   adapter.AddTable("users").AddTable("zones")
 */
func NewMySQLAdapter() *MySQLAdapter {
	return &MySQLAdapter{
		tables:     make([]string, 0),
		migrations: make([]string, 0),
	}
}

//AddTable
/**
 * AddTable registers a table and its CREATE TABLE migration.
 *
 * Parameters:
 *   tableName       - Name of the table
 *   createStatement - CREATE TABLE SQL statement
 *
 * Returns:
 *   *MySQLAdapter for fluent method chaining
 *
 * Example:
 *   adapter.AddTable("users", `
 *       CREATE TABLE users (
 *           id INT PRIMARY KEY AUTO_INCREMENT,
 *           name VARCHAR(255) NOT NULL,
 *           email VARCHAR(255) UNIQUE NOT NULL,
 *           created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
 *           updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
 *           deleted_at TIMESTAMP NULL
 *       )
 *   `)
 */
func (a *MySQLAdapter) AddTable(tableName string, createStatement string) *MySQLAdapter {
	a.tables = append(a.tables, tableName)
	a.migrations = append(a.migrations, createStatement)
	return a
}

//Connect
/**
 * Connect establishes a MySQL database connection.
 *
 * Parameters:
 *   dsn - MySQL Data Source Name
 *         Format: "user:password@tcp(host:port)/dbname?parseTime=true"
 *
 * Returns:
 *   error if connection fails
 */
func (a *MySQLAdapter) Connect(dsn string) error {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Verify connection
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	a.db = db
	return nil
}

//Close
/**
 * Close closes the MySQL database connection.
 *
 * Returns:
 *   error if closing fails
 */
func (a *MySQLAdapter) Close() error {
	if a.db == nil {
		return nil
	}
	return a.db.Close()
}

//SetMaxOpenConns
/**
 * SetMaxOpenConns sets the maximum number of open connections.
 *
 * Parameters:
 *   n - Maximum open connections
 */
func (a *MySQLAdapter) SetMaxOpenConns(n int) {
	if a.db != nil {
		a.db.SetMaxOpenConns(n)
	}
}

//SetMaxIdleConns
/**
 * SetMaxIdleConns sets the maximum number of idle connections.
 *
 * Parameters:
 *   n - Maximum idle connections
 */
func (a *MySQLAdapter) SetMaxIdleConns(n int) {
	if a.db != nil {
		a.db.SetMaxIdleConns(n)
	}
}

//BeginTx
/**
 * BeginTx starts a new MySQL transaction.
 *
 * Parameters:
 *   ctx - Context for the transaction
 *
 * Returns:
 *   interface{} containing *sql.Tx
 *   error if transaction start fails
 */
func (a *MySQLAdapter) BeginTx(ctx context.Context) (interface{}, error) {
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return tx, nil
}

/**
 * Commit commits a MySQL transaction.
 *
 * Parameters:
 *   tx - The transaction object (*sql.Tx)
 *
 * Returns:
 *   error if commit fails
 */
func (a *MySQLAdapter) Commit(tx interface{}) error {
	sqlTx, ok := tx.(*sql.Tx)
	if !ok {
		return fmt.Errorf("invalid transaction type: expected *sql.Tx")
	}
	return sqlTx.Commit()
}

/**
 * Rollback rolls back a MySQL transaction.
 *
 * Parameters:
 *   tx - The transaction object (*sql.Tx)
 *
 * Returns:
 *   error if rollback fails
 */
func (a *MySQLAdapter) Rollback(tx interface{}) error {
	sqlTx, ok := tx.(*sql.Tx)
	if !ok {
		return fmt.Errorf("invalid transaction type: expected *sql.Tx")
	}
	return sqlTx.Rollback()
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
func (a *MySQLAdapter) Exec(query string) error {
	_, err := a.db.Exec(query)
	return err
}

/**
 * Insert inserts a new record using raw SQL.
 *
 * Parameters:
 *   ctx   - Context for the operation
 *   table - Table name
 *   data  - Column-value pairs
 *
 * Returns:
 *   error if insertion fails
 */
func (a *MySQLAdapter) Insert(ctx context.Context, table string, data map[string]interface{}) error {
	columns := make([]string, 0, len(data))
	placeholders := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))

	for col, val := range data {
		columns = append(columns, col)
		placeholders = append(placeholders, "?")
		values = append(values, val)
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		table,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	_, err := a.db.ExecContext(ctx, query, values...)
	return err
}

/**
 * Update updates records matching conditions using raw SQL.
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
func (a *MySQLAdapter) Update(ctx context.Context, table string, conditions map[string]interface{}, data map[string]interface{}) error {
	setClauses := make([]string, 0, len(data))
	whereClauses := make([]string, 0, len(conditions))
	values := make([]interface{}, 0)

	// Build SET clause
	for col, val := range data {
		setClauses = append(setClauses, fmt.Sprintf("%s = ?", col))
		values = append(values, val)
	}

	// Build WHERE clause
	for col, val := range conditions {
		whereClauses = append(whereClauses, fmt.Sprintf("%s = ?", col))
		values = append(values, val)
	}

	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s",
		table,
		strings.Join(setClauses, ", "),
		strings.Join(whereClauses, " AND "),
	)

	_, err := a.db.ExecContext(ctx, query, values...)
	return err
}

/**
 * Delete removes records matching conditions using raw SQL.
 *
 * Parameters:
 *   ctx        - Context for the operation
 *   table      - Table name
 *   conditions - WHERE conditions
 *
 * Returns:
 *   error if deletion fails
 */
func (a *MySQLAdapter) Delete(ctx context.Context, table string, conditions map[string]interface{}) error {
	whereClauses := make([]string, 0, len(conditions))
	values := make([]interface{}, 0, len(conditions))

	for col, val := range conditions {
		whereClauses = append(whereClauses, fmt.Sprintf("%s = ?", col))
		values = append(values, val)
	}

	query := fmt.Sprintf(
		"DELETE FROM %s WHERE %s",
		table,
		strings.Join(whereClauses, " AND "),
	)

	_, err := a.db.ExecContext(ctx, query, values...)
	return err
}

/**
 * HasRecord checks if a record exists using raw SQL.
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
func (a *MySQLAdapter) HasRecord(ctx context.Context, table string, conditions map[string]interface{}) (bool, error) {
	count, err := a.Count(ctx, table, conditions)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

/**
 * GetRecord retrieves a single record using raw SQL.
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
func (a *MySQLAdapter) GetRecord(ctx context.Context, table string, conditions map[string]interface{}) (map[string]interface{}, error) {
	whereClauses := make([]string, 0, len(conditions))
	values := make([]interface{}, 0, len(conditions))

	for col, val := range conditions {
		whereClauses = append(whereClauses, fmt.Sprintf("%s = ?", col))
		values = append(values, val)
	}

	query := fmt.Sprintf(
		"SELECT * FROM %s WHERE %s LIMIT 1",
		table,
		strings.Join(whereClauses, " AND "),
	)

	row := a.db.QueryRowContext(ctx, query, values...)

	// Get column names
	columns, err := a.getColumns(table)
	if err != nil {
		return nil, err
	}

	// Create scan destinations
	dest := make([]interface{}, len(columns))
	destPtrs := make([]interface{}, len(columns))
	for i := range dest {
		destPtrs[i] = &dest[i]
	}

	if err := row.Scan(destPtrs...); err != nil {
		if err == sql.ErrNoRows {
			return nil, &RecordNotFoundError{Table: table, Conditions: conditions}
		}
		return nil, err
	}

	// Build result map
	result := make(map[string]interface{})
	for i, col := range columns {
		result[col] = dest[i]
	}

	return result, nil
}

/**
 * GetRecords retrieves multiple records using raw SQL.
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
func (a *MySQLAdapter) GetRecords(ctx context.Context, table string, conditions map[string]interface{}, limit int) ([]map[string]interface{}, error) {
	whereClauses := make([]string, 0, len(conditions))
	values := make([]interface{}, 0, len(conditions))

	for col, val := range conditions {
		whereClauses = append(whereClauses, fmt.Sprintf("%s = ?", col))
		values = append(values, val)
	}

	query := fmt.Sprintf("SELECT * FROM %s", table)

	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := a.db.QueryContext(ctx, query, values...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}

	for rows.Next() {
		dest := make([]interface{}, len(columns))
		destPtrs := make([]interface{}, len(columns))
		for i := range dest {
			destPtrs[i] = &dest[i]
		}

		if err := rows.Scan(destPtrs...); err != nil {
			return nil, err
		}

		result := make(map[string]interface{})
		for i, col := range columns {
			result[col] = dest[i]
		}

		results = append(results, result)
	}

	return results, rows.Err()
}

/**
 * Count returns the number of matching records using raw SQL.
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
func (a *MySQLAdapter) Count(ctx context.Context, table string, conditions map[string]interface{}) (int, error) {
	whereClauses := make([]string, 0, len(conditions))
	values := make([]interface{}, 0, len(conditions))

	for col, val := range conditions {
		whereClauses = append(whereClauses, fmt.Sprintf("%s = ?", col))
		values = append(values, val)
	}

	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)

	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	var count int
	err := a.db.QueryRowContext(ctx, query, values...).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

/**
 * IsSoftDeleted checks if a record is soft deleted using raw SQL.
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
func (a *MySQLAdapter) IsSoftDeleted(ctx context.Context, table string, conditions map[string]interface{}) (bool, error) {
	whereClauses := make([]string, 0, len(conditions))
	values := make([]interface{}, 0, len(conditions))

	for col, val := range conditions {
		whereClauses = append(whereClauses, fmt.Sprintf("%s = ?", col))
		values = append(values, val)
	}

	query := fmt.Sprintf(
		"SELECT deleted_at FROM %s WHERE %s LIMIT 1",
		table,
		strings.Join(whereClauses, " AND "),
	)

	var deletedAt sql.NullTime
	err := a.db.QueryRowContext(ctx, query, values...).Scan(&deletedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, &RecordNotFoundError{Table: table, Conditions: conditions}
		}
		return false, err
	}

	return deletedAt.Valid, nil
}

/**
 * AutoMigrate runs the registered CREATE TABLE migrations.
 *
 * This executes all CREATE TABLE statements that were registered
 * via AddTable().
 *
 * Returns:
 *   error if migration fails
 */
func (a *MySQLAdapter) AutoMigrate() error {
	for i, migration := range a.migrations {
		if _, err := a.db.Exec(migration); err != nil {
			return fmt.Errorf("migration %d failed for table %s: %w", i+1, a.tables[i], err)
		}
	}

	return nil
}

/**
 * DropAll drops all registered tables.
 *
 * Returns:
 *   error if dropping fails
 */
func (a *MySQLAdapter) DropAll() error {
	for _, table := range a.tables {
		if _, err := a.db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS `%s`", table)); err != nil {
			return fmt.Errorf("failed to drop table %s: %w", table, err)
		}
	}

	return nil
}

/**
 * TruncateAll truncates all registered tables.
 *
 * Uses DELETE for compatibility with transactions.
 *
 * Returns:
 *   error if truncation fails
 */
func (a *MySQLAdapter) TruncateAll() error {
	// Disable foreign key checks temporarily for truncation
	if _, err := a.db.Exec("SET FOREIGN_KEY_CHECKS = 0"); err != nil {
		return err
	}
	defer a.db.Exec("SET FOREIGN_KEY_CHECKS = 1")

	for _, table := range a.tables {
		if _, err := a.db.Exec(fmt.Sprintf("DELETE FROM `%s`", table)); err != nil {
			return fmt.Errorf("failed to truncate table %s: %w", table, err)
		}
	}

	return nil
}

/**
 * DB returns the underlying *sql.DB connection.
 *
 * Returns:
 *   *sql.DB - The database connection
 */
func (a *MySQLAdapter) DB() *sql.DB {
	return a.db
}

/**
 * getColumns returns the column names for a table.
 *
 * Uses DESCRIBE query to get column information.
 *
 * Parameters:
 *   table - Table name
 *
 * Returns:
 *   []string - Column names
 *   error if query fails
 */
func (a *MySQLAdapter) getColumns(table string) ([]string, error) {
	rows, err := a.db.Query(fmt.Sprintf("DESCRIBE `%s`", table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var field, typ, null, key, defaultVal, extra sql.NullString
		if err := rows.Scan(&field, &typ, &null, &key, &defaultVal, &extra); err != nil {
			return nil, err
		}
		columns = append(columns, field.String)
	}

	return columns, rows.Err()
}
