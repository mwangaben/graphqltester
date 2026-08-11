package tests

import (
	. "github.com/mwangaben/graphqltester"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

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
	config.Database = nil // Remove database config to avoid database validation error
	config.Middleware.PermissionEnabled = true
	config.Middleware.AuthEnabled = false

	err := config.Validate()
	require.Error(t, err, "Permission without auth should produce validation error")
	assert.Contains(t, err.Error(), "permission middleware requires authentication")
}
