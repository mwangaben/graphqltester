package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/mwangaben/graphqltester/pkg/adapters/database"
	"github.com/mwangaben/graphqltester/pkg/factory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Ent Models
// ============================================================================

type EntUser struct {
	ID        int        `json:"id"`
	Name      string     `json:"name"`
	Email     string     `json:"email"`
	Age       int        `json:"age"`
	Role      string     `json:"role"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at"`
}

func (EntUser) TableName() string {
	return "ent_factory_users"
}

type EntRole struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	GuardName string    `json:"guard_name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (EntRole) TableName() string {
	return "ent_factory_roles"
}

type EntPermission struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	GuardName string    `json:"guard_name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (EntPermission) TableName() string {
	return "ent_factory_permissions"
}

// ============================================================================
// Database Helper
// ============================================================================

func setupEntDB(t *testing.T) *database.EntAdapter {
	t.Helper()

	adapter := database.NewEntAdapter(&database.EntConfig{
		Debug:         false,
		LogQueries:    false,
		SkipMigration: true,
	})

	adapter.AddModel(&EntUser{}).AddModel(&EntRole{}).AddModel(&EntPermission{})

	dsn := getTestDSNEnt()
	err := adapter.Connect(dsn)
	if err != nil {
		t.Skipf("Skipping database test: cannot connect to database: %v", err)
	}

	err = createEntTestTables(adapter)
	if err != nil {
		t.Skipf("Skipping database test: failed to create tables: %v", err)
	}

	cleanupEntTables(adapter)

	t.Cleanup(func() {
		cleanupEntTables(adapter)
		adapter.Close()
	})

	return adapter
}

func createEntTestTables(adapter *database.EntAdapter) error {
	tables := []struct {
		name   string
		create string
		drop   string
	}{
		{
			name: "ent_factory_users",
			create: `CREATE TABLE IF NOT EXISTS ent_factory_users (
				id INT AUTO_INCREMENT PRIMARY KEY,
				name VARCHAR(255) NOT NULL,
				email VARCHAR(255) NOT NULL UNIQUE,
				age INT DEFAULT 0,
				role VARCHAR(100),
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
				deleted_at TIMESTAMP NULL DEFAULT NULL
			)`,
			drop: "DROP TABLE IF EXISTS ent_factory_users",
		},
		{
			name: "ent_factory_roles",
			create: `CREATE TABLE IF NOT EXISTS ent_factory_roles (
				id INT AUTO_INCREMENT PRIMARY KEY,
				name VARCHAR(255) NOT NULL UNIQUE,
				guard_name VARCHAR(50) DEFAULT 'api',
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
			)`,
			drop: "DROP TABLE IF EXISTS ent_factory_roles",
		},
		{
			name: "ent_factory_permissions",
			create: `CREATE TABLE IF NOT EXISTS ent_factory_permissions (
				id INT AUTO_INCREMENT PRIMARY KEY,
				name VARCHAR(255) NOT NULL UNIQUE,
				guard_name VARCHAR(50) DEFAULT 'api',
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
			)`,
			drop: "DROP TABLE IF EXISTS ent_factory_permissions",
		},
	}

	for _, table := range tables {
		if err := adapter.Exec(table.drop); err != nil {
			// Ignore errors for drop
		}
		if err := adapter.Exec(table.create); err != nil {
			return fmt.Errorf("failed to create table %s: %w", table.name, err)
		}
	}

	return nil
}

func cleanupEntTables(adapter *database.EntAdapter) {
	tables := []string{"ent_factory_users", "ent_factory_roles", "ent_factory_permissions"}
	for _, table := range tables {
		adapter.Exec(fmt.Sprintf("TRUNCATE TABLE %s", table))
	}
}

// ============================================================================
// Ent Factory - IMPORTANT: Definition functions do NOT save to database
// The save happens AFTER states are applied
// ============================================================================

func newEntDatabaseFactory() *factory.Factory {
	f := factory.NewFactory()
	var emailCounter int64

	// Define User factory - does NOT save to database
	f.Define("User", func(overrides map[string]interface{}) interface{} {
		emailCounter++
		user := &EntUser{
			Name:  "Default User",
			Email: fmt.Sprintf("default_%d_%d@example.com", emailCounter, time.Now().UnixNano()),
			Age:   25,
			Role:  "user",
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

	// Define Role factory - does NOT save to database
	f.Define("Role", func(overrides map[string]interface{}) interface{} {
		role := &EntRole{
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

	// Define Permission factory - does NOT save to database
	f.Define("Permission", func(overrides map[string]interface{}) interface{} {
		perm := &EntPermission{
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

	// Register states for User
	f.State("User", "admin", func(model interface{}) interface{} {
		user := model.(*EntUser)
		user.Role = "admin"
		return user
	})

	f.State("User", "verified", func(model interface{}) interface{} {
		user := model.(*EntUser)
		user.Email = "verified_" + user.Email
		return user
	})

	f.State("User", "senior", func(model interface{}) interface{} {
		user := model.(*EntUser)
		user.Age = 65
		return user
	})

	return f
}

// ============================================================================
// Save Helpers - These save the model to the database AFTER states are applied
// ============================================================================

func saveUserToDatabase(adapter *database.EntAdapter, user *EntUser) (*EntUser, error) {
	ctx := context.Background()

	data := map[string]interface{}{
		"name":  user.Name,
		"email": user.Email,
		"age":   user.Age,
		"role":  user.Role,
	}

	err := adapter.Insert(ctx, "ent_factory_users", data)
	if err != nil {
		return nil, fmt.Errorf("failed to insert user: %w", err)
	}

	// Get the inserted record to get the ID
	record, err := adapter.GetRecord(ctx, "ent_factory_users", map[string]interface{}{
		"email": user.Email,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get inserted user: %w", err)
	}

	if id, ok := record["id"].(int64); ok {
		user.ID = int(id)
	} else if id, ok := record["id"].(int); ok {
		user.ID = id
	}

	return user, nil
}

func saveRoleToDatabase(adapter *database.EntAdapter, role *EntRole) (*EntRole, error) {
	ctx := context.Background()

	data := map[string]interface{}{
		"name":       role.Name,
		"guard_name": role.GuardName,
	}

	err := adapter.Insert(ctx, "ent_factory_roles", data)
	if err != nil {
		return nil, fmt.Errorf("failed to insert role: %w", err)
	}

	record, err := adapter.GetRecord(ctx, "ent_factory_roles", map[string]interface{}{
		"name": role.Name,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get inserted role: %w", err)
	}

	if id, ok := record["id"].(int64); ok {
		role.ID = int(id)
	} else if id, ok := record["id"].(int); ok {
		role.ID = id
	}

	return role, nil
}

func savePermissionToDatabase(adapter *database.EntAdapter, perm *EntPermission) (*EntPermission, error) {
	ctx := context.Background()

	data := map[string]interface{}{
		"name":       perm.Name,
		"guard_name": perm.GuardName,
	}

	err := adapter.Insert(ctx, "ent_factory_permissions", data)
	if err != nil {
		return nil, fmt.Errorf("failed to insert permission: %w", err)
	}

	record, err := adapter.GetRecord(ctx, "ent_factory_permissions", map[string]interface{}{
		"name": perm.Name,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get inserted permission: %w", err)
	}

	if id, ok := record["id"].(int64); ok {
		perm.ID = int(id)
	} else if id, ok := record["id"].(int); ok {
		perm.ID = id
	}

	return perm, nil
}

// ============================================================================
// Ent Factory Tests
// ============================================================================

// TestEntFactory_Create_PersistsToDatabase verifies Factory().Create()
// saves to the database with states applied
func TestEntFactory_Create_PersistsToDatabase(t *testing.T) {
	adapter := setupEntDB(t)
	f := newEntDatabaseFactory()

	// Create user with states - states are applied by the factory
	user := f.Of("User").
		State("admin").
		State("verified").
		Overrides(map[string]interface{}{
			"name":  "Ent Database User",
			"email": "entdbuser@example.com",
			"age":   30,
		}).
		Create().(*EntUser)

	// Verify states were applied to the object
	require.NotNil(t, user)
	assert.Equal(t, "Ent Database User", user.Name)
	assert.Equal(t, "verified_entdbuser@example.com", user.Email)
	assert.Equal(t, 30, user.Age)
	assert.Equal(t, "admin", user.Role, "Role should be 'admin' from state")
	assert.Contains(t, user.Email, "verified_", "Email should contain 'verified_' from state")

	// Now save to database
	savedUser, err := saveUserToDatabase(adapter, user)
	require.NoError(t, err)
	assert.NotZero(t, savedUser.ID)

	// Verify in database
	ctx := context.Background()
	record, err := adapter.GetRecord(ctx, "ent_factory_users", map[string]interface{}{
		"id": savedUser.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, "Ent Database User", record["name"])
	assert.Equal(t, "verified_entdbuser@example.com", record["email"])
	assert.Equal(t, int64(30), record["age"])
	assert.Equal(t, "admin", record["role"], "Database role should be 'admin'")
	assert.Contains(t, record["email"].(string), "verified_", "Database email should contain 'verified_'")

	t.Logf("✅ Ent: User persisted to database with ID: %d, Role: %s", savedUser.ID, savedUser.Role)
}

// TestEntFactory_Create_MultiplePersistsToDatabase verifies multiple records
func TestEntFactory_Create_MultiplePersistsToDatabase(t *testing.T) {
	adapter := setupEntDB(t)
	f := newEntDatabaseFactory()

	// Create multiple users
	users := f.Of("User").Times(5).Create().([]interface{})
	assert.Len(t, users, 5)

	// Save each user to database
	for _, u := range users {
		user := u.(*EntUser)
		_, err := saveUserToDatabase(adapter, user)
		require.NoError(t, err)
		assert.NotZero(t, user.ID)
	}

	// Verify all users are in the database
	ctx := context.Background()
	count, err := adapter.Count(ctx, "ent_factory_users", map[string]interface{}{})
	assert.NoError(t, err)
	assert.Equal(t, 5, count, "Should have 5 users in database")

	t.Logf("✅ Ent: All %d users persisted to database", count)
}

// TestEntFactory_Create_RolePersistsToDatabase verifies roles are saved
func TestEntFactory_Create_RolePersistsToDatabase(t *testing.T) {
	adapter := setupEntDB(t)
	f := newEntDatabaseFactory()

	// Create a role
	role := f.Of("Role").Overrides(map[string]interface{}{
		"name":       "ent_admin",
		"guard_name": "api",
	}).Create().(*EntRole)

	require.NotNil(t, role)
	assert.Equal(t, "ent_admin", role.Name)
	assert.Equal(t, "api", role.GuardName)

	// Save to database
	savedRole, err := saveRoleToDatabase(adapter, role)
	require.NoError(t, err)
	assert.NotZero(t, savedRole.ID)

	// Verify in database
	ctx := context.Background()
	record, err := adapter.GetRecord(ctx, "ent_factory_roles", map[string]interface{}{
		"id": savedRole.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, "ent_admin", record["name"])
	assert.Equal(t, "api", record["guard_name"])

	t.Logf("✅ Ent: Role persisted to database with ID: %d", savedRole.ID)
}

// TestEntFactory_Create_PermissionPersistsToDatabase verifies permissions are saved
func TestEntFactory_Create_PermissionPersistsToDatabase(t *testing.T) {
	adapter := setupEntDB(t)
	f := newEntDatabaseFactory()

	// Create a permission
	perm := f.Of("Permission").Overrides(map[string]interface{}{
		"name":       "ent_posts.create",
		"guard_name": "api",
	}).Create().(*EntPermission)

	require.NotNil(t, perm)
	assert.Equal(t, "ent_posts.create", perm.Name)
	assert.Equal(t, "api", perm.GuardName)

	// Save to database
	savedPerm, err := savePermissionToDatabase(adapter, perm)
	require.NoError(t, err)
	assert.NotZero(t, savedPerm.ID)

	// Verify in database
	ctx := context.Background()
	record, err := adapter.GetRecord(ctx, "ent_factory_permissions", map[string]interface{}{
		"id": savedPerm.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, "ent_posts.create", record["name"])
	assert.Equal(t, "api", record["guard_name"])

	t.Logf("✅ Ent: Permission persisted to database with ID: %d", savedPerm.ID)
}

// TestEntFactory_SimulateSignInAdmin_PersistsToDatabase verifies the full
// SignInAdmin flow persists to the database
func TestEntFactory_SimulateSignInAdmin_PersistsToDatabase(t *testing.T) {
	adapter := setupEntDB(t)
	f := newEntDatabaseFactory()

	ctx := context.Background()

	// Step 1: Create a user
	user := f.Of("User").Overrides(map[string]interface{}{
		"name":  "Ent Admin User",
		"email": "entadmin@example.com",
	}).Create().(*EntUser)
	savedUser, err := saveUserToDatabase(adapter, user)
	require.NoError(t, err)
	assert.NotZero(t, savedUser.ID)

	// Step 2: Create admin role
	adminRole := f.Of("Role").Overrides(map[string]interface{}{
		"name": "ent_admin_role",
	}).Create().(*EntRole)
	savedRole, err := saveRoleToDatabase(adapter, adminRole)
	require.NoError(t, err)
	assert.NotZero(t, savedRole.ID)

	// Step 3: Update user with role
	err = adapter.Update(ctx, "ent_factory_users",
		map[string]interface{}{"email": "entadmin@example.com"},
		map[string]interface{}{"role": "ent_admin_role"},
	)
	require.NoError(t, err)

	// Verify both exist in database
	userRecord, err := adapter.GetRecord(ctx, "ent_factory_users", map[string]interface{}{
		"email": "entadmin@example.com",
	})
	assert.NoError(t, err)
	assert.Equal(t, "Ent Admin User", userRecord["name"])
	assert.Equal(t, "ent_admin_role", userRecord["role"])

	roleExists, err := adapter.HasRecord(ctx, "ent_factory_roles", map[string]interface{}{
		"name": "ent_admin_role",
	})
	assert.NoError(t, err)
	assert.True(t, roleExists)

	t.Logf("✅ Ent: SignInAdmin simulation: User ID=%d, Role ID=%d persisted", savedUser.ID, savedRole.ID)
}

// TestEntFactory_SimulateSignInUser_PersistsToDatabase verifies the full
// SignInUser flow persists to the database
func TestEntFactory_SimulateSignInUser_PersistsToDatabase(t *testing.T) {
	adapter := setupEntDB(t)
	f := newEntDatabaseFactory()

	ctx := context.Background()

	// Step 1: Create a user
	user := f.Of("User").Overrides(map[string]interface{}{
		"name":  "Ent Editor User",
		"email": "enteditor@example.com",
	}).Create().(*EntUser)
	savedUser, err := saveUserToDatabase(adapter, user)
	require.NoError(t, err)

	// Step 2: Create permission
	permission := f.Of("Permission").Overrides(map[string]interface{}{
		"name": "ent_posts.edit",
	}).Create().(*EntPermission)
	savedPerm, err := savePermissionToDatabase(adapter, permission)
	require.NoError(t, err)

	// Step 3: Create role
	role := f.Of("Role").Overrides(map[string]interface{}{
		"name": "ent_editor",
	}).Create().(*EntRole)
	savedRole, err := saveRoleToDatabase(adapter, role)
	require.NoError(t, err)

	// Step 4: Update user with role
	err = adapter.Update(ctx, "ent_factory_users",
		map[string]interface{}{"email": "enteditor@example.com"},
		map[string]interface{}{"role": "ent_editor"},
	)
	require.NoError(t, err)

	// Verify all three exist in database
	userExists, err := adapter.HasRecord(ctx, "ent_factory_users", map[string]interface{}{
		"email": "enteditor@example.com",
	})
	assert.NoError(t, err)
	assert.True(t, userExists)

	roleExists, err := adapter.HasRecord(ctx, "ent_factory_roles", map[string]interface{}{
		"name": "ent_editor",
	})
	assert.NoError(t, err)
	assert.True(t, roleExists)

	permExists, err := adapter.HasRecord(ctx, "ent_factory_permissions", map[string]interface{}{
		"name": "ent_posts.edit",
	})
	assert.NoError(t, err)
	assert.True(t, permExists)

	t.Logf("✅ Ent: SignInUser simulation: User=%d, Role=%d, Permission=%d all persisted",
		savedUser.ID, savedRole.ID, savedPerm.ID)
}

// TestEntFactory_Create_UniqueEmailConstraint verifies database constraints
func TestEntFactory_Create_UniqueEmailConstraint(t *testing.T) {
	adapter := setupEntDB(t)
	f := newEntDatabaseFactory()

	// Create first user
	user1 := f.Of("User").Overrides(map[string]interface{}{
		"email": "ent_unique@example.com",
	}).Create().(*EntUser)
	_, err := saveUserToDatabase(adapter, user1)
	require.NoError(t, err)
	assert.NotZero(t, user1.ID)

	// Try to create second user with same email - should fail
	user2 := f.Of("User").Overrides(map[string]interface{}{
		"email": "ent_unique@example.com", // Duplicate!
	}).Create().(*EntUser)

	_, err = saveUserToDatabase(adapter, user2)
	assert.Error(t, err, "Should fail on duplicate email")

	// Verify only one user exists
	ctx := context.Background()
	count, err := adapter.Count(ctx, "ent_factory_users", map[string]interface{}{
		"email": "ent_unique@example.com",
	})
	assert.NoError(t, err)
	assert.Equal(t, 1, count)

	t.Logf("✅ Ent: Unique constraint enforced: only 1 user with duplicate email")
}

// TestEntFactory_Create_EmptyTable_AfterCreate verifies table is populated
func TestEntFactory_Create_EmptyTable_AfterCreate(t *testing.T) {
	adapter := setupEntDB(t)
	f := newEntDatabaseFactory()

	// Verify table is empty before
	ctx := context.Background()
	countBefore, err := adapter.Count(ctx, "ent_factory_users", map[string]interface{}{})
	assert.NoError(t, err)
	assert.Equal(t, 0, countBefore, "Table should be empty before test")

	// Create users
	users := f.Of("User").Times(3).Create().([]interface{})
	for _, u := range users {
		user := u.(*EntUser)
		_, err := saveUserToDatabase(adapter, user)
		require.NoError(t, err)
	}

	// Verify table has records after
	countAfter, err := adapter.Count(ctx, "ent_factory_users", map[string]interface{}{})
	assert.NoError(t, err)
	assert.Equal(t, 3, countAfter, "Table should have 3 users after create")

	t.Logf("✅ Ent: Table went from %d to %d records", countBefore, countAfter)
}

// TestEntFactory_WithStates tests factory states with Ent
func TestEntFactory_WithStates(t *testing.T) {
	adapter := setupEntDB(t)
	f := newEntDatabaseFactory()

	// Create user with states - states are applied by the factory
	user := f.Of("User").
		State("admin").
		State("verified").
		Overrides(map[string]interface{}{
			"name": "State User",
		}).
		Create().(*EntUser)

	// Verify states were applied to the object
	assert.Equal(t, "admin", user.Role, "Role should be 'admin' from state")
	assert.Contains(t, user.Email, "verified_", "Email should contain 'verified_' from state")
	assert.Equal(t, "State User", user.Name, "Name should be 'State User' from override")

	// Now save to database
	savedUser, err := saveUserToDatabase(adapter, user)
	require.NoError(t, err)
	assert.NotZero(t, savedUser.ID)

	// Verify in database
	ctx := context.Background()
	record, err := adapter.GetRecord(ctx, "ent_factory_users", map[string]interface{}{
		"id": savedUser.ID,
	})
	assert.NoError(t, err)
	assert.Equal(t, "admin", record["role"], "Database role should be 'admin'")
	assert.Contains(t, record["email"].(string), "verified_", "Database email should contain 'verified_'")
	assert.Equal(t, "State User", record["name"], "Database name should be 'State User'")

	t.Logf("✅ Ent: Factory with states persisted: User ID=%d, Name=%s, Role=%s",
		savedUser.ID, savedUser.Name, savedUser.Role)
}

// TestEntFactory_Create_WithStateOnly tests creating a user with only a state
func TestEntFactory_Create_WithStateOnly(t *testing.T) {
	adapter := setupEntDB(t)
	f := newEntDatabaseFactory()

	// Create user with only admin state
	user := f.Of("User").
		State("admin").
		Create().(*EntUser)

	assert.Equal(t, "admin", user.Role, "Role should be 'admin' from state")
	assert.Equal(t, "Default User", user.Name, "Name should be default")

	// Save to database
	savedUser, err := saveUserToDatabase(adapter, user)
	require.NoError(t, err)
	assert.NotZero(t, savedUser.ID)

	// Verify in database
	ctx := context.Background()
	record, err := adapter.GetRecord(ctx, "ent_factory_users", map[string]interface{}{
		"id": savedUser.ID,
	})
	assert.NoError(t, err)
	assert.Equal(t, "admin", record["role"], "Database role should be 'admin'")

	t.Logf("✅ Ent: User with admin state persisted: ID=%d, Role=%s", savedUser.ID, savedUser.Role)
}

// ============================================================================
// Run with: go test -v ./tests -run TestEntFactory
// ============================================================================
