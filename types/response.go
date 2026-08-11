package types

import "time"

/**
 * ResponseInterface defines the contract that the main package's Response
 * struct must implement.
 *
 * This interface is used by the assertions package to work with response
 * objects without importing the main package, thus avoiding cyclic imports.
 *
 * The interface provides:
 * - Access to response data, errors, and extensions
 * - HTTP status code
 * - JSON path navigation for assertions
 * - Raw body access for debugging
 *
 * Implemented by: main package's Response struct
 */
type ResponseInterface interface {
	// Status returns the HTTP status code from the response.
	Status() int

	// Data returns the parsed response data.
	// Returns nil if no data was returned.
	Data() interface{}

	// Errors returns any GraphQL errors from the response.
	// Returns nil or empty slice if no errors.
	Errors() []*GraphQLError

	// Extensions returns protocol extensions from the response.
	// Returns nil or empty map if no extensions.
	Extensions() map[string]interface{}

	// RawBody returns the raw response body bytes.
	// Useful for debugging unexpected responses.
	RawBody() []byte

	// Elapsed returns the time taken for the request to complete.
	Elapsed() time.Duration

	// JSON returns a value from the response data at the given dot-separated path.
	// Example: JSON("data.user.name") returns the user's name.
	// Returns nil if the path doesn't exist.
	JSON(path string) interface{}

	// JSONString returns a string value at the given path.
	// Returns empty string if the path doesn't exist or value is not a string.
	JSONString(path string) string

	// JSONInt returns an integer value at the given path.
	// Returns 0 if the path doesn't exist or value is not numeric.
	JSONInt(path string) int

	// JSONFloat returns a float value at the given path.
	// Returns 0.0 if the path doesn't exist or value is not numeric.
	JSONFloat(path string) float64

	// JSONBool returns a boolean value at the given path.
	// Returns false if the path doesn't exist or value is not a boolean.
	JSONBool(path string) bool

	// JSONArray returns an array value at the given path.
	// Returns nil if the path doesn't exist or value is not an array.
	JSONArray(path string) []interface{}

	// JSONMap returns a map value at the given path.
	// Returns nil if the path doesn't exist or value is not a map.
	JSONMap(path string) map[string]interface{}
}

/**
 * AssertionResponse extends ResponseInterface with testing.T-like methods.
 *
 * This allows the assertions package to report test failures and log
 * messages without importing the testing package directly.
 *
 * Implemented by: main package's Response struct
 */
type AssertionResponse interface {
	ResponseInterface

	// Logf logs a formatted message for debugging.
	Logf(format string, args ...interface{})

	// Errorf reports a test error with a formatted message.
	Errorf(format string, args ...interface{})

	// Fatalf reports a fatal test error with a formatted message.
	Fatalf(format string, args ...interface{})
}

/**
 * DebuggableResponse extends ResponseInterface with debug capabilities.
 *
 * Implemented by responses that support debug output for troubleshooting
 * test failures.
 */
type DebuggableResponse interface {
	ResponseInterface

	// Debug prints debug information about the response.
	Debug()

	// Dump prints the full response details for inspection.
	Dump()
}
