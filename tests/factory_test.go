package tests

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"os"
	"testing"

	"github.com/mwangaben/graphqltester/pkg/factory"
	"github.com/stretchr/testify/assert"
	_ "github.com/stretchr/testify/require"
)

// ============================================================================
// Test Models
// ============================================================================

type testUser struct {
	ID    string
	Name  string
	Email string
	Age   int
	Role  string
}

type testRole struct {
	Name      string
	GuardName string
}

type testPermission struct {
	Name      string
	GuardName string
}

// ============================================================================
// Helper: Create a configured factory
// ============================================================================

func newTestFactory() *factory.Factory {
	f := factory.NewFactory()

	// Define User factory
	f.Define("User", func(overrides map[string]interface{}) interface{} {
		user := &testUser{
			ID:    "default-id",
			Name:  "Default User",
			Email: "default@example.com",
			Age:   25,
		}
		if v, ok := overrides["id"]; ok {
			user.ID = v.(string)
		}
		if v, ok := overrides["name"]; ok {
			user.Name = v.(string)
		}
		if v, ok := overrides["email"]; ok {
			user.Email = v.(string)
		}
		if v, ok := overrides["age"]; ok {
			user.Age = v.(int)
		}
		if v, ok := overrides["role"]; ok {
			user.Role = v.(string)
		}
		return user
	})

	// Define Role factory
	f.Define("Role", func(overrides map[string]interface{}) interface{} {
		role := &testRole{
			Name:      "default-role",
			GuardName: "api",
		}
		if v, ok := overrides["name"]; ok {
			role.Name = v.(string)
		}
		if v, ok := overrides["guard_name"]; ok {
			role.GuardName = v.(string)
		}
		return role
	})

	// Define Permission factory
	f.Define("Permission", func(overrides map[string]interface{}) interface{} {
		perm := &testPermission{
			Name:      "default-permission",
			GuardName: "api",
		}
		if v, ok := overrides["name"]; ok {
			perm.Name = v.(string)
		}
		if v, ok := overrides["guard_name"]; ok {
			perm.GuardName = v.(string)
		}
		return perm
	})

	// Define a raw/map-based model for testing Raw()
	f.Define("Config", func(overrides map[string]interface{}) interface{} {
		config := map[string]interface{}{
			"debug":  true,
			"port":   8080,
			"host":   "localhost",
			"region": "us-east-1",
		}
		for k, v := range overrides {
			config[k] = v
		}
		return config
	})

	// Register states
	f.State("User", "admin", func(model interface{}) interface{} {
		user := model.(*testUser)
		user.Role = "admin"
		return user
	})

	f.State("User", "verified", func(model interface{}) interface{} {
		user := model.(*testUser)
		user.Email = "verified_" + user.Email
		return user
	})

	f.State("User", "senior", func(model interface{}) interface{} {
		user := model.(*testUser)
		user.Age = 65
		return user
	})

	return f
}

// ============================================================================
// Create Tests
// ============================================================================

func TestFactory_Create_DefaultAttributes(t *testing.T) {
	f := newTestFactory()

	user := f.Of("User").Create().(*testUser)

	assert.NotNil(t, user)
	assert.Equal(t, "default-id", user.ID)
	assert.Equal(t, "Default User", user.Name)
	assert.Equal(t, "default@example.com", user.Email)
	assert.Equal(t, 25, user.Age)
}

func TestFactory_Create_WithOverrides(t *testing.T) {
	f := newTestFactory()

	user := f.Of("User").Overrides(map[string]interface{}{
		"name":  "John Doe",
		"email": "john@example.com",
		"age":   30,
	}).Create().(*testUser)

	assert.Equal(t, "John Doe", user.Name)
	assert.Equal(t, "john@example.com", user.Email)
	assert.Equal(t, 30, user.Age)
	assert.Equal(t, "default-id", user.ID) // Not overridden, uses default
}

func TestFactory_Create_ChainedOverrides(t *testing.T) {
	f := newTestFactory()

	user := f.Of("User").
		Overrides(map[string]interface{}{"name": "Jane"}).
		Overrides(map[string]interface{}{"email": "jane@example.com"}).
		Create().(*testUser)

	assert.Equal(t, "Jane", user.Name)
	assert.Equal(t, "jane@example.com", user.Email)
}

func TestFactory_Create_MultipleWithTimes(t *testing.T) {
	f := newTestFactory()

	users := f.Of("User").Times(5).Create().([]interface{})

	assert.Len(t, users, 5)
	for _, u := range users {
		user := u.(*testUser)
		assert.Equal(t, "Default User", user.Name)
	}
}

func TestFactory_Create_Role(t *testing.T) {
	f := newTestFactory()

	role := f.Of("Role").Create().(*testRole)

	assert.Equal(t, "default-role", role.Name)
	assert.Equal(t, "api", role.GuardName)
}

func TestFactory_Create_RoleWithOverrides(t *testing.T) {
	f := newTestFactory()

	role := f.Of("Role").Overrides(map[string]interface{}{
		"name":       "admin",
		"guard_name": "web",
	}).Create().(*testRole)

	assert.Equal(t, "admin", role.Name)
	assert.Equal(t, "web", role.GuardName)
}

func TestFactory_Create_Permission(t *testing.T) {
	f := newTestFactory()

	perm := f.Of("Permission").Create().(*testPermission)

	assert.Equal(t, "default-permission", perm.Name)
	assert.Equal(t, "api", perm.GuardName)
}

func TestFactory_Create_PermissionWithOverrides(t *testing.T) {
	f := newTestFactory()

	perm := f.Of("Permission").Overrides(map[string]interface{}{
		"name":       "posts.create",
		"guard_name": "api",
	}).Create().(*testPermission)

	assert.Equal(t, "posts.create", perm.Name)
	assert.Equal(t, "api", perm.GuardName)
}

// ============================================================================
// Make Tests (alias for Create)
// ============================================================================

func TestFactory_Make_AliasForCreate(t *testing.T) {
	f := newTestFactory()

	user := f.Of("User").Make().(*testUser)

	assert.NotNil(t, user)
	assert.Equal(t, "Default User", user.Name)
}

func TestFactory_Make_WithOverrides(t *testing.T) {
	f := newTestFactory()

	user := f.Of("User").Overrides(map[string]interface{}{
		"name": "Made User",
	}).Make().(*testUser)

	assert.Equal(t, "Made User", user.Name)
}

// ============================================================================
// State Tests
// ============================================================================

func TestFactory_State_Admin(t *testing.T) {
	f := newTestFactory()

	user := f.Of("User").State("admin").Create().(*testUser)

	assert.Equal(t, "admin", user.Role)
}

func TestFactory_State_Verified(t *testing.T) {
	f := newTestFactory()

	user := f.Of("User").State("verified").Create().(*testUser)

	assert.Equal(t, "verified_default@example.com", user.Email)
}

func TestFactory_State_MultipleStates(t *testing.T) {
	f := newTestFactory()

	user := f.Of("User").
		State("admin").
		State("verified").
		State("senior").
		Create().(*testUser)

	assert.Equal(t, "admin", user.Role)
	assert.Equal(t, "verified_default@example.com", user.Email)
	assert.Equal(t, 65, user.Age)
}

func TestFactory_State_WithOverrides(t *testing.T) {
	f := newTestFactory()

	user := f.Of("User").
		State("admin").
		Overrides(map[string]interface{}{"name": "Admin User"}).
		Create().(*testUser)

	assert.Equal(t, "Admin User", user.Name)
	assert.Equal(t, "admin", user.Role)
}

// ============================================================================
// Raw Tests
// ============================================================================

func TestFactory_Raw_AllAttributes(t *testing.T) {
	f := newTestFactory()

	config := f.Of("Config").Raw(map[string]interface{}{
		"debug":  false,
		"port":   9090,
		"host":   "example.com",
		"region": "eu-west-1",
	}).(map[string]interface{})

	assert.Equal(t, false, config["debug"])
	assert.Equal(t, 9090, config["port"])
	assert.Equal(t, "example.com", config["host"])
	assert.Equal(t, "eu-west-1", config["region"])
}

func TestFactory_Raw_PartialAttributes(t *testing.T) {
	f := newTestFactory()

	config := f.Of("Config").Raw(map[string]interface{}{
		"port": 3000,
	}).(map[string]interface{})

	assert.Equal(t, 3000, config["port"])
	assert.Equal(t, true, config["debug"])       // Default
	assert.Equal(t, "localhost", config["host"]) // Default
}

func TestFactory_Raw_User(t *testing.T) {
	f := newTestFactory()

	user := f.Of("User").Raw(map[string]interface{}{
		"name":  "Raw User",
		"email": "raw@example.com",
		"age":   40,
	}).(*testUser)

	assert.Equal(t, "Raw User", user.Name)
	assert.Equal(t, "raw@example.com", user.Email)
	assert.Equal(t, 40, user.Age)
}

// ============================================================================
// CreateMany Tests
// ============================================================================

func TestFactory_CreateMany(t *testing.T) {
	f := newTestFactory()

	users := f.Of("User").CreateMany(3)

	assert.Len(t, users, 3)
	for _, u := range users {
		assert.NotNil(t, u.(*testUser))
	}
}

func TestFactory_CreateMany_WithOverrides(t *testing.T) {
	f := newTestFactory()

	users := f.Of("User").
		Overrides(map[string]interface{}{"role": "member"}).
		CreateMany(4)

	assert.Len(t, users, 4)
	for _, u := range users {
		assert.Equal(t, "member", u.(*testUser).Role)
	}
}

// ============================================================================
// Error Cases
// ============================================================================

func TestFactory_UndefinedModel_Panics(t *testing.T) {
	f := newTestFactory()

	assert.Panics(t, func() {
		f.Of("NonExistentModel")
	}, "Should panic for undefined model")
}

func TestFactory_Define_OverwritesExisting(t *testing.T) {
	f := newTestFactory()

	// Redefine User with different defaults
	f.Define("User", func(overrides map[string]interface{}) interface{} {
		return &testUser{
			ID:    "new-id",
			Name:  "New Default",
			Email: "new@example.com",
			Age:   99,
		}
	})

	user := f.Of("User").Create().(*testUser)

	assert.Equal(t, "new-id", user.ID)
	assert.Equal(t, "New Default", user.Name)
	assert.Equal(t, "new@example.com", user.Email)
	assert.Equal(t, 99, user.Age)
}

// ============================================================================
// Integration: Simulating SignInAdmin flow
// ============================================================================

func TestFactory_SimulateSignInAdmin(t *testing.T) {
	f := newTestFactory()

	// This simulates what SignInAdmin does internally:
	// 1. Create a user
	user := f.Of("User").Create().(*testUser)

	// 2. Create an admin role
	adminRole := f.Of("Role").Overrides(map[string]interface{}{
		"name": "admin",
	}).Create().(*testRole)

	// 3. Assign role to user (simulated)
	user.Role = adminRole.Name

	assert.NotNil(t, user)
	assert.Equal(t, "admin", user.Role)
	assert.Equal(t, "admin", adminRole.Name)
}

func TestFactory_SimulateSignInUser(t *testing.T) {
	f := newTestFactory()

	// This simulates what SignInUser does internally:
	// 1. Create a user
	user := f.Of("User").Create().(*testUser)

	// 2. Create a permission
	permission := f.Of("Permission").Overrides(map[string]interface{}{
		"name": "posts.edit",
	}).Create().(*testPermission)

	// 3. Create a role
	role := f.Of("Role").Overrides(map[string]interface{}{
		"name": "editor",
	}).Create().(*testRole)

	// 4. Assign role to user
	user.Role = role.Name

	assert.Equal(t, "editor", user.Role)
	assert.Equal(t, "editor", role.Name)
	assert.Equal(t, "posts.edit", permission.Name)
}

// ============================================================================
// Thread Safety Tests
// ============================================================================

func TestFactory_ConcurrentAccess(t *testing.T) {
	f := newTestFactory()

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			user := f.Of("User").Create().(*testUser)
			assert.NotNil(t, user)
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestFactory_ConcurrentDefineAndCreate(t *testing.T) {
	f := newTestFactory()

	done := make(chan bool)

	// Concurrent defines
	go func() {
		f.Define("Concurrent", func(overrides map[string]interface{}) interface{} {
			return map[string]string{"key": "value"}
		})
		done <- true
	}()

	// Concurrent creates
	go func() {
		user := f.Of("User").Create().(*testUser)
		assert.NotNil(t, user)
		done <- true
	}()

	<-done
	<-done
}

// ============================================================================
// Benchmark Tests
// ============================================================================

func BenchmarkFactory_Create(b *testing.B) {
	f := newTestFactory()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.Of("User").Create()
	}
}

func BenchmarkFactory_CreateWithOverrides(b *testing.B) {
	f := newTestFactory()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.Of("User").Overrides(map[string]interface{}{
			"name": "Bench User",
		}).Create()
	}
}

func BenchmarkFactory_CreateMany(b *testing.B) {
	f := newTestFactory()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.Of("User").Times(10).Create()
	}
}

// ============================================================================
// Database-Backed Models for Integration Tests
// ============================================================================

// DBUser is a GORM model for database testing
type DBUser struct {
	ID    uint   `gorm:"primaryKey"`
	Name  string `gorm:"size:255"`
	Email string `gorm:"size:255;unique"`
	Age   int
}

func (DBUser) TableName() string {
	return "factory_test_users"
}

// DBRole is a GORM model for database testing
type DBRole struct {
	ID        uint   `gorm:"primaryKey"`
	Name      string `gorm:"size:255"`
	GuardName string `gorm:"size:50"`
}

func (DBRole) TableName() string {
	return "factory_test_roles"
}

// DBPermission is a GORM model for database testing
type DBPermission struct {
	ID        uint   `gorm:"primaryKey"`
	Name      string `gorm:"size:255"`
	GuardName string `gorm:"size:50"`
}

func (DBPermission) TableName() string {
	return "factory_test_permissions"
}

// ============================================================================
// Database Helper
// ============================================================================

/**
 * getTestDSN returns the database DSN from environment or defaults.
 */
func getTestDSN() string {
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		// Default test database - change this to your test database
		dsn = "root:root@tcp(localhost:3306)/graphql_tester_test?charset=utf8mb4&parseTime=True&loc=Local"
	}
	return dsn
}

/**
 * setupTestDB creates a database connection and migrates tables.
 */
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := getTestDSN()
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Skipf("Skipping database test: cannot connect to database: %v", err)
	}

	// Auto-migrate test tables
	if err := db.AutoMigrate(&DBUser{}, &DBRole{}, &DBPermission{}); err != nil {
		t.Skipf("Skipping database test: migration failed: %v", err)
	}

	// Clean up before test
	db.Exec("DELETE FROM factory_test_users")
	db.Exec("DELETE FROM factory_test_roles")
	db.Exec("DELETE FROM factory_test_permissions")

	t.Cleanup(func() {
		// Clean up after test
		db.Exec("DELETE FROM factory_test_users")
		db.Exec("DELETE FROM factory_test_roles")
		db.Exec("DELETE FROM factory_test_permissions")
	})

	return db
}

/**
 * newDatabaseFactory creates a factory that persists to the database.
 */
func newDatabaseFactory(db *gorm.DB) *factory.Factory {
	f := factory.NewFactory()

	// Use a counter for unique emails
	var emailCounter int64

	// Define User factory that saves to database
	f.Define("User", func(overrides map[string]interface{}) interface{} {
		emailCounter++
		user := DBUser{
			Name:  "Default User",
			Email: fmt.Sprintf("default_%d_%d@example.com", emailCounter, db.NowFunc().UnixNano()),
			Age:   25,
		}
		if v, ok := overrides["name"]; ok {
			user.Name = v.(string)
		}
		if v, ok := overrides["email"]; ok {
			user.Email = v.(string)
		}
		if v, ok := overrides["age"]; ok {
			user.Age = v.(int)
		}
		// Save to database
		result := db.Create(&user)
		if result.Error != nil {
			panic(fmt.Sprintf("failed to create user: %v", result.Error))
		}
		return &user
	})

	// Define Role factory that saves to database
	f.Define("Role", func(overrides map[string]interface{}) interface{} {
		role := DBRole{
			Name:      "default-role",
			GuardName: "api",
		}
		if v, ok := overrides["name"]; ok {
			role.Name = v.(string)
		}
		if v, ok := overrides["guard_name"]; ok {
			role.GuardName = v.(string)
		}
		result := db.Create(&role)
		if result.Error != nil {
			panic(fmt.Sprintf("failed to create role: %v", result.Error))
		}
		return &role
	})

	// Define Permission factory that saves to database
	f.Define("Permission", func(overrides map[string]interface{}) interface{} {
		perm := DBPermission{
			Name:      "default-permission",
			GuardName: "api",
		}
		if v, ok := overrides["name"]; ok {
			perm.Name = v.(string)
		}
		if v, ok := overrides["guard_name"]; ok {
			perm.GuardName = v.(string)
		}
		result := db.Create(&perm)
		if result.Error != nil {
			panic(fmt.Sprintf("failed to create permission: %v", result.Error))
		}
		return &perm
	})

	return f
}

// ============================================================================
// Database Integration Tests
// ============================================================================

/**
 * TestFactory_Create_PersistsToDatabase verifies that Factory().Create()
 * actually saves the record to the MySQL database.
 */
func TestFactory_Create_PersistsToDatabase(t *testing.T) {
	db := setupTestDB(t)
	f := newDatabaseFactory(db)

	// Create a user via factory
	user := f.Of("User").Overrides(map[string]interface{}{
		"name":  "Database User",
		"email": "dbuser@example.com",
		"age":   30,
	}).Create().(*DBUser)

	// Verify the user was returned
	require.NotNil(t, user)
	assert.Equal(t, "Database User", user.Name)
	assert.Equal(t, "dbuser@example.com", user.Email)
	assert.Equal(t, 30, user.Age)
	assert.NotZero(t, user.ID, "User should have a database ID after creation")

	// Verify the user exists in the database
	var dbUser DBUser
	result := db.First(&dbUser, user.ID)
	require.NoError(t, result.Error)
	assert.Equal(t, "Database User", dbUser.Name)
	assert.Equal(t, "dbuser@example.com", dbUser.Email)
	assert.Equal(t, 30, dbUser.Age)

	t.Logf("✅ User persisted to database with ID: %d", user.ID)
}

/**
 * TestFactory_Create_MultiplePersistsToDatabase verifies multiple records
 * are saved to the database.
 */
func TestFactory_Create_MultiplePersistsToDatabase(t *testing.T) {
	db := setupTestDB(t)
	f := newDatabaseFactory(db)

	// Create multiple users
	users := f.Of("User").Times(5).Create().([]interface{})
	assert.Len(t, users, 5)

	// Verify all users are in the database
	var count int64
	db.Model(&DBUser{}).Count(&count)
	assert.Equal(t, int64(5), count, "Should have 5 users in database")

	// Verify each user has an ID
	for _, u := range users {
		user := u.(*DBUser)
		assert.NotZero(t, user.ID, "Each user should have a database ID")

		// Verify can fetch from database
		var dbUser DBUser
		result := db.First(&dbUser, user.ID)
		assert.NoError(t, result.Error)
	}

	t.Logf("✅ All %d users persisted to database", count)
}

/**
 * TestFactory_Create_RolePersistsToDatabase verifies roles are saved.
 */
func TestFactory_Create_RolePersistsToDatabase(t *testing.T) {
	db := setupTestDB(t)
	f := newDatabaseFactory(db)

	// Create a role
	role := f.Of("Role").Overrides(map[string]interface{}{
		"name":       "admin",
		"guard_name": "api",
	}).Create().(*DBRole)

	require.NotNil(t, role)
	assert.NotZero(t, role.ID)

	// Verify in database
	var dbRole DBRole
	result := db.First(&dbRole, role.ID)
	require.NoError(t, result.Error)
	assert.Equal(t, "admin", dbRole.Name)
	assert.Equal(t, "api", dbRole.GuardName)

	t.Logf("✅ Role persisted to database with ID: %d", role.ID)
}

/**
 * TestFactory_Create_PermissionPersistsToDatabase verifies permissions are saved.
 */
func TestFactory_Create_PermissionPersistsToDatabase(t *testing.T) {
	db := setupTestDB(t)
	f := newDatabaseFactory(db)

	// Create a permission
	perm := f.Of("Permission").Overrides(map[string]interface{}{
		"name":       "posts.create",
		"guard_name": "api",
	}).Create().(*DBPermission)

	require.NotNil(t, perm)
	assert.NotZero(t, perm.ID)

	// Verify in database
	var dbPerm DBPermission
	result := db.First(&dbPerm, perm.ID)
	require.NoError(t, result.Error)
	assert.Equal(t, "posts.create", dbPerm.Name)
	assert.Equal(t, "api", dbPerm.GuardName)

	t.Logf("✅ Permission persisted to database with ID: %d", perm.ID)
}

/**
 * TestFactory_SimulateSignInAdmin_PersistsToDatabase verifies the full
 * SignInAdmin flow persists to the database.
 */
func TestFactory_SimulateSignInAdmin_PersistsToDatabase(t *testing.T) {
	db := setupTestDB(t)
	f := newDatabaseFactory(db)

	// Step 1: Create a user
	user := f.Of("User").Overrides(map[string]interface{}{
		"name":  "Admin User",
		"email": "admin@example.com",
	}).Create().(*DBUser)
	assert.NotZero(t, user.ID)

	// Step 2: Create admin role
	adminRole := f.Of("Role").Overrides(map[string]interface{}{
		"name": "admin",
	}).Create().(*DBRole)
	assert.NotZero(t, adminRole.ID)

	// Verify both exist in database
	var dbUser DBUser
	db.First(&dbUser, user.ID)
	assert.Equal(t, "Admin User", dbUser.Name)

	var dbRole DBRole
	db.First(&dbRole, adminRole.ID)
	assert.Equal(t, "admin", dbRole.Name)

	// Verify counts
	var userCount, roleCount int64
	db.Model(&DBUser{}).Count(&userCount)
	db.Model(&DBRole{}).Count(&roleCount)
	assert.Equal(t, int64(1), userCount)
	assert.Equal(t, int64(1), roleCount)

	t.Logf("✅ SignInAdmin simulation: User ID=%d, Role ID=%d persisted", user.ID, adminRole.ID)
}

/**
 * TestFactory_SimulateSignInUser_PersistsToDatabase verifies the full
 * SignInUser flow persists to the database.
 */
func TestFactory_SimulateSignInUser_PersistsToDatabase(t *testing.T) {
	db := setupTestDB(t)
	f := newDatabaseFactory(db)

	// Step 1: Create a user
	user := f.Of("User").Overrides(map[string]interface{}{
		"name":  "Editor User",
		"email": "editor@example.com",
	}).Create().(*DBUser)

	// Step 2: Create permission
	permission := f.Of("Permission").Overrides(map[string]interface{}{
		"name": "posts.edit",
	}).Create().(*DBPermission)

	// Step 3: Create role
	role := f.Of("Role").Overrides(map[string]interface{}{
		"name": "editor",
	}).Create().(*DBRole)

	// Verify all three exist in database
	assert.NotZero(t, user.ID)
	assert.NotZero(t, permission.ID)
	assert.NotZero(t, role.ID)

	var userCount, roleCount, permCount int64
	db.Model(&DBUser{}).Count(&userCount)
	db.Model(&DBRole{}).Count(&roleCount)
	db.Model(&DBPermission{}).Count(&permCount)

	assert.Equal(t, int64(1), userCount)
	assert.Equal(t, int64(1), roleCount)
	assert.Equal(t, int64(1), permCount)

	t.Logf("✅ SignInUser simulation: User=%d, Role=%d, Permission=%d all persisted",
		user.ID, role.ID, permission.ID)
}

/**
 * TestFactory_Create_UniqueEmailConstraint verifies database constraints work.
 */
func TestFactory_Create_UniqueEmailConstraint(t *testing.T) {
	db := setupTestDB(t)
	f := newDatabaseFactory(db)

	// Create first user
	user1 := f.Of("User").Overrides(map[string]interface{}{
		"email": "unique@example.com",
	}).Create().(*DBUser)
	assert.NotZero(t, user1.ID)

	// Try to create second user with same email - should panic
	assert.Panics(t, func() {
		f.Of("User").Overrides(map[string]interface{}{
			"email": "unique@example.com", // Duplicate!
		}).Create()
	}, "Should panic on duplicate email")

	// Verify only one user exists
	var count int64
	db.Model(&DBUser{}).Where("email = ?", "unique@example.com").Count(&count)
	assert.Equal(t, int64(1), count)

	t.Logf("✅ Unique constraint enforced: only 1 user with duplicate email")
}

/**
 * TestFactory_Create_EmptyTable_AfterCreate verifies table is populated.
 */
func TestFactory_Create_EmptyTable_AfterCreate(t *testing.T) {
	db := setupTestDB(t)
	f := newDatabaseFactory(db)

	// Verify table is empty before
	var countBefore int64
	db.Model(&DBUser{}).Count(&countBefore)
	assert.Equal(t, int64(0), countBefore, "Table should be empty before test")

	// Create users
	f.Of("User").Times(3).Create()

	// Verify table has records after
	var countAfter int64
	db.Model(&DBUser{}).Count(&countAfter)
	assert.Equal(t, int64(3), countAfter, "Table should have 3 users after create")

	t.Logf("✅ Table went from %d to %d records", countBefore, countAfter)
}
