package tests

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	_ "entgo.io/ent/dialect"
	_ "entgo.io/ent/dialect/sql"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"
	"github.com/mwangaben/graphqltester/pkg/adapters/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Test Models (Ent-style schemas would normally be generated)
// For testing, we use structs that simulate Ent models
// ============================================================================

// TestUser represents a user model for testing
type TestUser struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   int    `json:"age"`
	Role  string `json:"role"`
}

// TableName returns the table name for TestUser
func (TestUser) TableName() string {
	return "ent_test_users"
}

// TestRole represents a role model for testing
type TestRole struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	GuardName string `json:"guard_name"`
}

// TableName returns the table name for TestRole
func (TestRole) TableName() string {
	return "ent_test_roles"
}

// TestPermission represents a permission model for testing
type TestPermission struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	GuardName string `json:"guard_name"`
}

// TableName returns the table name for TestPermission
func (TestPermission) TableName() string {
	return "ent_test_permissions"
}

// ============================================================================
// Helper Functions
// ============================================================================

// getTestDSN returns the database DSN from environment or defaults
func getTestDSNEnt() string {
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		dsn = "root:root@tcp(localhost:3306)/graphql_tester_test?charset=utf8mb4&parseTime=True&loc=Local"
	}
	return dsn
}

// setupTestEntDB creates a test database connection and sets up tables
func setupTestEntDB(t *testing.T) *database.EntAdapter {
	t.Helper()

	adapter := database.NewEntAdapter(&database.EntConfig{
		Debug:         true,
		LogQueries:    true,
		SkipMigration: true, // We'll handle migration manually for testing
	})

	// Register models for table creation
	adapter.AddModel(&TestUser{}).AddModel(&TestRole{}).AddModel(&TestPermission{})

	// Connect to database
	dsn := getTestDSNEnt()
	err := adapter.Connect(dsn)
	if err != nil {
		t.Skipf("Skipping database test: cannot connect to database: %v", err)
	}

	// Create tables manually (since we don't have Ent schema generation for testing)
	err = createTestTables(adapter)
	if err != nil {
		t.Skipf("Skipping database test: failed to create tables: %v", err)
	}

	// Clean up before test
	cleanupTables(adapter)

	t.Cleanup(func() {
		cleanupTables(adapter)
		adapter.Close()
	})

	return adapter
}

// createTestTables creates the test tables
func createTestTables(adapter *database.EntAdapter) error {
	tables := []struct {
		name   string
		create string
		drop   string
	}{
		{
			name: "ent_test_users",
			create: `CREATE TABLE IF NOT EXISTS ent_test_users (
				id INT AUTO_INCREMENT PRIMARY KEY,
				name VARCHAR(255) NOT NULL,
				email VARCHAR(255) NOT NULL UNIQUE,
				age INT DEFAULT 0,
				role VARCHAR(100),
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
				deleted_at TIMESTAMP NULL DEFAULT NULL
			)`,
			drop: "DROP TABLE IF EXISTS ent_test_users",
		},
		{
			name: "ent_test_roles",
			create: `CREATE TABLE IF NOT EXISTS ent_test_roles (
				id INT AUTO_INCREMENT PRIMARY KEY,
				name VARCHAR(255) NOT NULL UNIQUE,
				guard_name VARCHAR(50) DEFAULT 'api',
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
			)`,
			drop: "DROP TABLE IF EXISTS ent_test_roles",
		},
		{
			name: "ent_test_permissions",
			create: `CREATE TABLE IF NOT EXISTS ent_test_permissions (
				id INT AUTO_INCREMENT PRIMARY KEY,
				name VARCHAR(255) NOT NULL UNIQUE,
				guard_name VARCHAR(50) DEFAULT 'api',
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
			)`,
			drop: "DROP TABLE IF EXISTS ent_test_permissions",
		},
	}

	for _, table := range tables {
		// Drop table if exists
		if err := adapter.Exec(table.drop); err != nil {
			// Ignore errors for drop
		}
		// Create table
		if err := adapter.Exec(table.create); err != nil {
			return fmt.Errorf("failed to create table %s: %w", table.name, err)
		}
	}

	return nil
}

// cleanupTables truncates all test tables
func cleanupTables(adapter *database.EntAdapter) {
	tables := []string{"ent_test_users", "ent_test_roles", "ent_test_permissions"}
	for _, table := range tables {
		adapter.Exec(fmt.Sprintf("TRUNCATE TABLE %s", table))
	}
}

// ============================================================================
// Basic CRUD Tests
// ============================================================================

func TestEntAdapter_Insert(t *testing.T) {
	adapter := setupTestEntDB(t)
	ctx := context.Background()

	userData := map[string]interface{}{
		"name":  "John Doe",
		"email": "john@example.com",
		"age":   30,
		"role":  "user",
	}

	err := adapter.Insert(ctx, "ent_test_users", userData)
	assert.NoError(t, err)

	// Verify the record was inserted
	exists, err := adapter.HasRecord(ctx, "ent_test_users", map[string]interface{}{
		"email": "john@example.com",
	})
	assert.NoError(t, err)
	assert.True(t, exists)

	t.Log("✅ Insert test passed")
}

func TestEntAdapter_Insert_WithDuplicateEmail(t *testing.T) {
	adapter := setupTestEntDB(t)
	ctx := context.Background()

	// Insert first user
	userData := map[string]interface{}{
		"name":  "First User",
		"email": "duplicate@example.com",
		"age":   25,
	}
	err := adapter.Insert(ctx, "ent_test_users", userData)
	require.NoError(t, err)

	// Try to insert second user with same email
	userData2 := map[string]interface{}{
		"name":  "Second User",
		"email": "duplicate@example.com", // Duplicate!
		"age":   30,
	}
	err = adapter.Insert(ctx, "ent_test_users", userData2)
	assert.Error(t, err, "Should fail on duplicate email")

	t.Log("✅ Duplicate email test passed")
}

func TestEntAdapter_Update(t *testing.T) {
	adapter := setupTestEntDB(t)
	ctx := context.Background()

	// Insert a user
	userData := map[string]interface{}{
		"name":  "Update User",
		"email": "update@example.com",
		"age":   25,
	}
	err := adapter.Insert(ctx, "ent_test_users", userData)
	require.NoError(t, err)

	// Update the user
	updateData := map[string]interface{}{
		"name": "Updated Name",
		"age":  30,
	}
	err = adapter.Update(ctx, "ent_test_users",
		map[string]interface{}{"email": "update@example.com"},
		updateData,
	)
	assert.NoError(t, err)

	// Verify the update
	record, err := adapter.GetRecord(ctx, "ent_test_users", map[string]interface{}{
		"email": "update@example.com",
	})
	assert.NoError(t, err)
	assert.Equal(t, "Updated Name", record["name"])
	assert.Equal(t, int64(30), record["age"])

	t.Log("✅ Update test passed")
}

func TestEntAdapter_Delete(t *testing.T) {
	adapter := setupTestEntDB(t)
	ctx := context.Background()

	// Insert a user
	userData := map[string]interface{}{
		"name":  "Delete User",
		"email": "delete@example.com",
		"age":   25,
	}
	err := adapter.Insert(ctx, "ent_test_users", userData)
	require.NoError(t, err)

	// Verify it exists
	exists, err := adapter.HasRecord(ctx, "ent_test_users", map[string]interface{}{
		"email": "delete@example.com",
	})
	require.NoError(t, err)
	require.True(t, exists)

	// Delete the user
	err = adapter.Delete(ctx, "ent_test_users", map[string]interface{}{
		"email": "delete@example.com",
	})
	assert.NoError(t, err)

	// Verify it's gone
	exists, err = adapter.HasRecord(ctx, "ent_test_users", map[string]interface{}{
		"email": "delete@example.com",
	})
	assert.NoError(t, err)
	assert.False(t, exists)

	t.Log("✅ Delete test passed")
}

// ============================================================================
// Query Tests
// ============================================================================

func TestEntAdapter_GetRecord(t *testing.T) {
	adapter := setupTestEntDB(t)
	ctx := context.Background()

	// Insert test data
	userData := map[string]interface{}{
		"name":  "Get User",
		"email": "get@example.com",
		"age":   28,
	}
	err := adapter.Insert(ctx, "ent_test_users", userData)
	require.NoError(t, err)

	// Get the record
	record, err := adapter.GetRecord(ctx, "ent_test_users", map[string]interface{}{
		"email": "get@example.com",
	})
	assert.NoError(t, err)
	assert.Equal(t, "Get User", record["name"])
	assert.Equal(t, int64(28), record["age"])

	t.Log("✅ GetRecord test passed")
}

func TestEntAdapter_GetRecord_NotFound(t *testing.T) {
	adapter := setupTestEntDB(t)
	ctx := context.Background()

	_, err := adapter.GetRecord(ctx, "ent_test_users", map[string]interface{}{
		"email": "nonexistent@example.com",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "record not found")

	t.Log("✅ GetRecord not found test passed")
}

func TestEntAdapter_GetRecords(t *testing.T) {
	adapter := setupTestEntDB(t)
	ctx := context.Background()

	// Insert multiple users
	for i := 1; i <= 5; i++ {
		userData := map[string]interface{}{
			"name":  fmt.Sprintf("User %d", i),
			"email": fmt.Sprintf("user%d@example.com", i),
			"age":   20 + i,
		}
		err := adapter.Insert(ctx, "ent_test_users", userData)
		require.NoError(t, err)
	}

	// Get all users
	records, err := adapter.GetRecords(ctx, "ent_test_users", map[string]interface{}{}, 0)
	assert.NoError(t, err)
	assert.Len(t, records, 5)

	// Get with limit
	records, err = adapter.GetRecords(ctx, "ent_test_users", map[string]interface{}{}, 3)
	assert.NoError(t, err)
	assert.Len(t, records, 3)

	// Get with conditions
	records, err = adapter.GetRecords(ctx, "ent_test_users", map[string]interface{}{
		"age": 23,
	}, 0)
	assert.NoError(t, err)
	assert.Len(t, records, 1)

	t.Log("✅ GetRecords test passed")
}

func TestEntAdapter_Count(t *testing.T) {
	adapter := setupTestEntDB(t)
	ctx := context.Background()

	// Insert users
	for i := 1; i <= 3; i++ {
		userData := map[string]interface{}{
			"name":  fmt.Sprintf("Count User %d", i),
			"email": fmt.Sprintf("count%d@example.com", i),
			"age":   25,
		}
		err := adapter.Insert(ctx, "ent_test_users", userData)
		require.NoError(t, err)
	}

	// Count all
	count, err := adapter.Count(ctx, "ent_test_users", map[string]interface{}{})
	assert.NoError(t, err)
	assert.Equal(t, 3, count)

	// Count with conditions
	count, err = adapter.Count(ctx, "ent_test_users", map[string]interface{}{
		"age": 25,
	})
	assert.NoError(t, err)
	assert.Equal(t, 3, count)

	count, err = adapter.Count(ctx, "ent_test_users", map[string]interface{}{
		"age": 99,
	})
	assert.NoError(t, err)
	assert.Equal(t, 0, count)

	t.Log("✅ Count test passed")
}

// ============================================================================
// Transaction Tests
// ============================================================================

func TestEntAdapter_Transaction_Commit(t *testing.T) {
	adapter := setupTestEntDB(t)
	ctx := context.Background()

	// Begin transaction
	tx, err := adapter.BeginTx(ctx)
	require.NoError(t, err)

	// Insert within transaction
	userData := map[string]interface{}{
		"name":  "TX User",
		"email": "tx@example.com",
		"age":   25,
	}
	err = adapter.Insert(ctx, "ent_test_users", userData)
	require.NoError(t, err)

	// Commit
	err = adapter.Commit(tx)
	assert.NoError(t, err)

	// Verify data is persisted
	exists, err := adapter.HasRecord(ctx, "ent_test_users", map[string]interface{}{
		"email": "tx@example.com",
	})
	assert.NoError(t, err)
	assert.True(t, exists)

	t.Log("✅ Transaction commit test passed")
}

func TestEntAdapter_Transaction_Rollback(t *testing.T) {
	adapter := setupTestEntDB(t)
	ctx := context.Background()

	// Begin transaction
	tx, err := adapter.BeginTx(ctx)
	require.NoError(t, err)

	// Insert within transaction
	userData := map[string]interface{}{
		"name":  "Rollback User",
		"email": "rollback@example.com",
		"age":   25,
	}
	err = adapter.Insert(ctx, "ent_test_users", userData)
	require.NoError(t, err)

	// Rollback
	err = adapter.Rollback(tx)
	assert.NoError(t, err)

	// Verify data was not persisted
	exists, err := adapter.HasRecord(ctx, "ent_test_users", map[string]interface{}{
		"email": "rollback@example.com",
	})
	assert.NoError(t, err)
	assert.False(t, exists)

	t.Log("✅ Transaction rollback test passed")
}

// ============================================================================
// Soft Delete Tests
// ============================================================================

func TestEntAdapter_IsSoftDeleted(t *testing.T) {
	adapter := setupTestEntDB(t)
	ctx := context.Background()

	// Insert a user
	userData := map[string]interface{}{
		"name":  "Soft Delete User",
		"email": "softdelete@example.com",
		"age":   25,
	}
	err := adapter.Insert(ctx, "ent_test_users", userData)
	require.NoError(t, err)

	// Check not soft deleted
	isDeleted, err := adapter.IsSoftDeleted(ctx, "ent_test_users", map[string]interface{}{
		"email": "softdelete@example.com",
	})
	assert.NoError(t, err)
	assert.False(t, isDeleted)

	// Soft delete (set deleted_at)
	err = adapter.Update(ctx, "ent_test_users",
		map[string]interface{}{"email": "softdelete@example.com"},
		map[string]interface{}{"deleted_at": time.Now()},
	)
	require.NoError(t, err)

	// Check soft deleted
	isDeleted, err = adapter.IsSoftDeleted(ctx, "ent_test_users", map[string]interface{}{
		"email": "softdelete@example.com",
	})
	assert.NoError(t, err)
	assert.True(t, isDeleted)

	t.Log("✅ Soft delete test passed")
}

// ============================================================================
// Integration Tests (Simulating SignInAdmin flow)
// ============================================================================

func TestEntAdapter_SimulateSignInAdmin(t *testing.T) {
	adapter := setupTestEntDB(t)
	ctx := context.Background()

	// Step 1: Create a user
	userData := map[string]interface{}{
		"name":  "Admin User",
		"email": "admin@example.com",
		"age":   30,
	}
	err := adapter.Insert(ctx, "ent_test_users", userData)
	require.NoError(t, err)

	// Step 2: Create an admin role
	roleData := map[string]interface{}{
		"name":       "admin",
		"guard_name": "api",
	}
	err = adapter.Insert(ctx, "ent_test_roles", roleData)
	require.NoError(t, err)

	// Step 3: Update user with role
	err = adapter.Update(ctx, "ent_test_users",
		map[string]interface{}{"email": "admin@example.com"},
		map[string]interface{}{"role": "admin"},
	)
	require.NoError(t, err)

	// Verify user has admin role
	user, err := adapter.GetRecord(ctx, "ent_test_users", map[string]interface{}{
		"email": "admin@example.com",
	})
	assert.NoError(t, err)
	assert.Equal(t, "admin", user["role"])

	// Verify role exists
	exists, err := adapter.HasRecord(ctx, "ent_test_roles", map[string]interface{}{
		"name": "admin",
	})
	assert.NoError(t, err)
	assert.True(t, exists)

	t.Log("✅ SignInAdmin simulation test passed")
}

func TestEntAdapter_SimulateSignInUser(t *testing.T) {
	adapter := setupTestEntDB(t)
	ctx := context.Background()

	// Step 1: Create a user
	userData := map[string]interface{}{
		"name":  "Editor User",
		"email": "editor@example.com",
		"age":   28,
	}
	err := adapter.Insert(ctx, "ent_test_users", userData)
	require.NoError(t, err)

	// Step 2: Create a permission
	permData := map[string]interface{}{
		"name":       "posts.edit",
		"guard_name": "api",
	}
	err = adapter.Insert(ctx, "ent_test_permissions", permData)
	require.NoError(t, err)

	// Step 3: Create a role
	roleData := map[string]interface{}{
		"name":       "editor",
		"guard_name": "api",
	}
	err = adapter.Insert(ctx, "ent_test_roles", roleData)
	require.NoError(t, err)

	// Step 4: Update user with role
	err = adapter.Update(ctx, "ent_test_users",
		map[string]interface{}{"email": "editor@example.com"},
		map[string]interface{}{"role": "editor"},
	)
	require.NoError(t, err)

	// Verify all records exist
	exists, err := adapter.HasRecord(ctx, "ent_test_users", map[string]interface{}{
		"email": "editor@example.com",
	})
	assert.NoError(t, err)
	assert.True(t, exists)

	exists, err = adapter.HasRecord(ctx, "ent_test_roles", map[string]interface{}{
		"name": "editor",
	})
	assert.NoError(t, err)
	assert.True(t, exists)

	exists, err = adapter.HasRecord(ctx, "ent_test_permissions", map[string]interface{}{
		"name": "posts.edit",
	})
	assert.NoError(t, err)
	assert.True(t, exists)

	t.Log("✅ SignInUser simulation test passed")
}

// ============================================================================
// Raw SQL Tests
// ============================================================================

func TestEntAdapter_Exec(t *testing.T) {
	adapter := setupTestEntDB(t)

	err := adapter.Exec(`INSERT INTO ent_test_users (name, email, age) VALUES ('Exec User', 'exec@example.com', 25)`)
	assert.NoError(t, err)

	// Verify
	ctx := context.Background()
	exists, err := adapter.HasRecord(ctx, "ent_test_users", map[string]interface{}{
		"email": "exec@example.com",
	})
	assert.NoError(t, err)
	assert.True(t, exists)

	t.Log("✅ Exec test passed")
}

// ============================================================================
// Concurrent Access Tests
// ============================================================================

func TestEntAdapter_ConcurrentAccess(t *testing.T) {
	adapter := setupTestEntDB(t)
	ctx := context.Background()

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			userData := map[string]interface{}{
				"name":  fmt.Sprintf("Concurrent User %d", idx),
				"email": fmt.Sprintf("concurrent%d@example.com", idx),
				"age":   20 + idx,
			}
			err := adapter.Insert(ctx, "ent_test_users", userData)
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all records were inserted
	count, err := adapter.Count(ctx, "ent_test_users", map[string]interface{}{})
	assert.NoError(t, err)
	assert.Equal(t, 10, count)

	t.Log("✅ Concurrent access test passed")
}

// ============================================================================
// Integration: Factory Pattern Test
// ============================================================================

func TestEntAdapter_WithFactoryPattern(t *testing.T) {
	adapter := setupTestEntDB(t)
	ctx := context.Background()

	// This simulates how the factory would work with Ent
	type UserFactory struct {
		adapter *database.EntAdapter
	}

	// Define a helper function to create users
	createTestUser := func(name, email string, age int) error {
		return adapter.Insert(ctx, "ent_test_users", map[string]interface{}{
			"name":  name,
			"email": email,
			"age":   age,
		})
	}

	// Create users using factory pattern
	err := createTestUser("Factory User 1", "factory1@example.com", 25)
	require.NoError(t, err)
	err = createTestUser("Factory User 2", "factory2@example.com", 30)
	require.NoError(t, err)
	err = createTestUser("Factory User 3", "factory3@example.com", 35)
	require.NoError(t, err)

	// Verify all users were created
	count, err := adapter.Count(ctx, "ent_test_users", map[string]interface{}{})
	assert.NoError(t, err)
	assert.Equal(t, 3, count)

	t.Log("✅ Factory pattern test passed")
}

// ============================================================================
// Benchmark Tests
// ============================================================================

func BenchmarkEntAdapter_Insert(b *testing.B) {
	adapter := setupTestEntDB(&testing.T{})
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		userData := map[string]interface{}{
			"name":  fmt.Sprintf("Bench User %d", i),
			"email": fmt.Sprintf("bench%d@example.com", i),
			"age":   25,
		}
		adapter.Insert(ctx, "ent_test_users", userData)
	}
}

func BenchmarkEntAdapter_GetRecord(b *testing.B) {
	adapter := setupTestEntDB(&testing.T{})
	ctx := context.Background()

	// Insert a user for the benchmark
	userData := map[string]interface{}{
		"name":  "Bench Get User",
		"email": "bench_get@example.com",
		"age":   25,
	}
	adapter.Insert(ctx, "ent_test_users", userData)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		adapter.GetRecord(ctx, "ent_test_users", map[string]interface{}{
			"email": "bench_get@example.com",
		})
	}
}

func BenchmarkEntAdapter_Update(b *testing.B) {
	adapter := setupTestEntDB(&testing.T{})
	ctx := context.Background()

	// Insert a user for the benchmark
	userData := map[string]interface{}{
		"name":  "Bench Update User",
		"email": "bench_update@example.com",
		"age":   25,
	}
	adapter.Insert(ctx, "ent_test_users", userData)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		adapter.Update(ctx, "ent_test_users",
			map[string]interface{}{"email": "bench_update@example.com"},
			map[string]interface{}{"age": 30 + i%10},
		)
	}
}

// ============================================================================
// Run with: go test -v ./pkg/adapters/database -run TestEntAdapter
// Run benchmarks: go test -bench=BenchmarkEntAdapter -benchmem ./pkg/adapters/database
// ============================================================================
