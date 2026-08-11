package types

import "context"

/**
 * TesterInterface defines the contract that the main Tester struct must implement.
 *
 * This interface provides the assertions package with access to tester
 * functionality without importing the main package, avoiding cyclic imports.
 */
type TesterInterface interface {
	// Database returns the database adapter for database assertions.
	Database() DatabaseAdapter

	// Config returns the tester configuration for checking settings.
	Config() TesterConfigInterface

	// Context returns the tester's base context.
	Context() context.Context

	// CurrentUser returns the currently authenticated user.
	CurrentUser() interface{}

	// CurrentToken returns the current authentication token.
	CurrentToken() string

	// HasPermission checks if the current user has a specific permission.
	HasPermission(permission string) bool

	// HasRole checks if the current user has a specific role.
	HasRole(role string) bool

	// Logf logs a formatted message for debugging.
	Logf(format string, args ...interface{})

	// Errorf reports a test error with a formatted message.
	Errorf(format string, args ...interface{})

	// Fatalf reports a fatal test error with a formatted message.
	Fatalf(format string, args ...interface{})

	// Helper marks the calling function as a test helper.
	Helper()

	// Name returns the name of the current test.
	Name() string
}

/**
 * TesterConfigInterface exposes configuration settings to the assertions package.
 */
type TesterConfigInterface interface {
	// Debug returns whether debug mode is enabled.
	Debug() bool

	// TenancyEnabled returns whether multi-tenancy is enabled.
	TenancyEnabled() bool
}
