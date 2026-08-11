package types

import (
	"context"
	"database/sql"
)

/**
 * DatabaseAdapter defines the interface for database operations
 * used by the assertions package and the tester.
 *
 * This interface provides all database operations needed for testing.
 * Implementations include GORM, SQLx, and raw MySQL adapters.
 */
//goland:noinspection ALL
type DatabaseAdapter interface {
	// Connection management
	Connect(dsn string) error
	Close() error
	SetMaxOpenConns(n int)
	SetMaxIdleConns(n int)

	// Transaction management
	BeginTx(ctx context.Context) (interface{}, error)
	Commit(tx interface{}) error
	Rollback(tx interface{}) error
	Exec(query string) error

	// CRUD operations
	Insert(ctx context.Context, table string, data map[string]interface{}) error
	Update(ctx context.Context, table string, conditions map[string]interface{}, data map[string]interface{}) error
	Delete(ctx context.Context, table string, conditions map[string]interface{}) error

	// Query operations
	HasRecord(ctx context.Context, table string, conditions map[string]interface{}) (bool, error)
	GetRecord(ctx context.Context, table string, conditions map[string]interface{}) (map[string]interface{}, error)
	GetRecords(ctx context.Context, table string, conditions map[string]interface{}, limit int) ([]map[string]interface{}, error)
	Count(ctx context.Context, table string, conditions map[string]interface{}) (int, error)

	// Soft delete operations
	IsSoftDeleted(ctx context.Context, table string, conditions map[string]interface{}) (bool, error)

	// Migration operations
	AutoMigrate() error
	DropAll() error
	TruncateAll() error

	// Raw database access
	DB() *sql.DB
}
