package tests

import (
	"testing"
	"time"

	. "github.com/mwangaben/graphqltester"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/**
 * TestDefaultConfig_ReturnsSensibleDefaults verifies that DefaultConfig
 * returns a configuration with production-safe defaults that can be
 * used immediately without causing errors.
 */
func TestDefaultConfig_ReturnsSensibleDefaults(t *testing.T) {
	config := DefaultConfig()

	assert.Equal(t, "/graphql", config.Endpoint, "Default endpoint should be /graphql")
	assert.True(t, config.Debug, "Debug should be enabled by default for development")
	assert.Equal(t, 30*time.Second, config.Timeout, "Default timeout should be 30s")
	assert.NotNil(t, config.Context, "Context should not be nil")
	assert.NotNil(t, config.Schema, "Schema config should not be nil")
	assert.True(t, config.Schema.RefreshCache, "Schema cache should refresh by default")
	assert.True(t, config.Schema.ValidateOnLoad, "Schema validation should be on by default")

	if config.Database != nil {
		assert.True(t, config.Database.UseTransactions, "Transactions should be used by default")
		assert.True(t, config.Database.AutoMigrate, "AutoMigrate should be enabled by default")
		assert.Equal(t, 25, config.Database.MaxOpenConns, "Default max open conns should be 25")
	}
}

/**
 * TestConfig_Validate_ValidConfig validates that a properly configured
 * Config passes validation without errors.
 */
func TestConfig_Validate_ValidConfig(t *testing.T) {
	config := DefaultConfig()
	config.Schema.Path = "./testdata/schema.graphql"
	config.Schema.Resolvers = &mockResolver{}

	err := config.Validate()
	require.NoError(t, err, "Valid config should not produce validation errors")
}

/**
 * TestConfig_Validate_MissingSchema tests that validation catches
 * the case where no schema is configured.
 */
func TestConfig_Validate_MissingSchema(t *testing.T) {
	config := DefaultConfig()
	config.Schema = nil

	err := config.Validate()
	require.Error(t, err, "Missing schema should produce validation error")
	assert.Contains(t, err.Error(), "schema configuration is required")
}

/**
 * TestConfig_Validate_MissingResolvers tests that validation catches
 * the case where resolvers are not provided.
 */
func TestConfig_Validate_MissingResolvers(t *testing.T) {
	config := DefaultConfig()
	config.Schema.Path = "./testdata/schema.graphql"
	config.Schema.Resolvers = nil

	err := config.Validate()
	require.Error(t, err, "Missing resolvers should produce validation error")
	assert.Contains(t, err.Error(), "schema resolvers are required")
}

/**
 * TestConfig_Validate_MissingSchemaSource tests that validation requires
 * either schema path or schema string to be set.
 */
func TestConfig_Validate_MissingSchemaSource(t *testing.T) {
	config := DefaultConfig()
	config.Schema.Path = ""
	config.Schema.String = ""
	config.Schema.Resolvers = &mockResolver{}

	err := config.Validate()
	require.Error(t, err, "Missing schema source should produce validation error")
	assert.Contains(t, err.Error(), "either schema path or schema string must be provided")
}

/**
 * TestConfig_Validate_DatabaseWithoutAdapter tests that database
 * configuration requires an adapter when DSN is provided.
 */
func TestConfig_Validate_DatabaseWithoutAdapter(t *testing.T) {
	config := DefaultConfig()
	config.Schema.Path = "./testdata/schema.graphql"
	config.Schema.Resolvers = &mockResolver{}
	config.Database = &DatabaseConfig{
		DSN:     "test:test@tcp(localhost:3306)/testdb",
		Adapter: nil, // Missing adapter
	}

	err := config.Validate()
	require.Error(t, err, "Database without adapter should produce validation error")
	assert.Contains(t, err.Error(), "database adapter is required")
}

/**
 * TestConfig_Validate_PermissionWithoutAuth tests that permission
 * middleware requires authentication middleware to be enabled.
 */
func TestConfig_Validate_PermissionWithoutAuth(t *testing.T) {
	config := DefaultConfig()
	config.Schema.Path = "./testdata/schema.graphql"
	config.Schema.Resolvers = &mockResolver{}
	config.Middleware.PermissionEnabled = true
	config.Middleware.AuthEnabled = false

	err := config.Validate()
	require.Error(t, err, "Permission without auth should produce validation error")
	assert.Contains(t, err.Error(), "permission middleware requires authentication")
}

// Mock resolver for testing
type mockResolver struct{}
