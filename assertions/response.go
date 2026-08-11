// Package assertions provides assertion methods for GraphQL test responses.
//
// This package depends only on the types package to avoid cyclic imports
// with the main graphqltester package. All assertion methods work with
// interfaces defined in types/ rather than concrete types.
package assertions

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strings"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/mwangaben/graphqltester/types"
)

/**
 * ResponseAssertions provides HTTP and GraphQL response assertion methods.
 *
 * All methods work with the types.AssertionResponse interface, allowing
 * this package to function without importing the main package.
 *
 * Assertions are chainable - each method returns the same ResponseAssertions
 * instance for fluent test code.
 *
 * Example:
 *   assertions.AssertStatus(http.StatusOK).
 *       AssertNoErrors().
 *       AssertJSONPath("data.user.name", "John Doe")
 */
type ResponseAssertions struct {
	response types.AssertionResponse
}

/**
 * NewResponseAssertions creates response assertion helpers.
 *
 * Parameters:
 *   response - The response to assert on (must implement types.AssertionResponse)
 *
 * Returns:
 *   *ResponseAssertions for chaining assertions
 */
func NewResponseAssertions(response types.AssertionResponse) *ResponseAssertions {
	return &ResponseAssertions{response: response}
}

// ============================================================================
// HTTP Status Assertions
// ============================================================================

/**
 * AssertStatus asserts that the HTTP response status code matches expected.
 *
 * Parameters:
 *   code - Expected HTTP status code
 *
 * Returns:
 *   *ResponseAssertions for chaining
 */
func (ra *ResponseAssertions) AssertStatus(code int) *ResponseAssertions {
	if ra.response.Status() != code {
		ra.response.Errorf(
			"❌ Expected HTTP status %d (%s), got %d",
			code,
			http.StatusText(code),
			ra.response.Status(),
		)
	}
	return ra
}

/**
 * AssertOK asserts HTTP 200 OK status.
 */
func (ra *ResponseAssertions) AssertOK() *ResponseAssertions {
	return ra.AssertStatus(http.StatusOK)
}

/**
 * AssertCreated asserts HTTP 201 Created status.
 */
func (ra *ResponseAssertions) AssertCreated() *ResponseAssertions {
	return ra.AssertStatus(http.StatusCreated)
}

/**
 * AssertNoContent asserts HTTP 204 No Content status.
 */
func (ra *ResponseAssertions) AssertNoContent() *ResponseAssertions {
	return ra.AssertStatus(http.StatusNoContent)
}

/**
 * AssertUnauthorized asserts HTTP 401 Unauthorized status.
 */
func (ra *ResponseAssertions) AssertUnauthorized() *ResponseAssertions {
	return ra.AssertStatus(http.StatusUnauthorized)
}

/**
 * AssertForbidden asserts HTTP 403 Forbidden status.
 */
func (ra *ResponseAssertions) AssertForbidden() *ResponseAssertions {
	return ra.AssertStatus(http.StatusForbidden)
}

/**
 * AssertNotFound asserts HTTP 404 Not Found status.
 */
func (ra *ResponseAssertions) AssertNotFound() *ResponseAssertions {
	return ra.AssertStatus(http.StatusNotFound)
}

/**
 * AssertUnprocessable asserts HTTP 422 Unprocessable Entity status.
 */
func (ra *ResponseAssertions) AssertUnprocessable() *ResponseAssertions {
	return ra.AssertStatus(http.StatusUnprocessableEntity)
}

/**
 * AssertServerError asserts HTTP 5xx server error status.
 */
func (ra *ResponseAssertions) AssertServerError() *ResponseAssertions {
	if ra.response.Status() < 500 || ra.response.Status() >= 600 {
		ra.response.Errorf("❌ Expected 5xx server error, got %d", ra.response.Status())
	}
	return ra
}

// ============================================================================
// GraphQL Error Assertions
// ============================================================================

/**
 * AssertNoErrors asserts that the response contains no GraphQL errors.
 */
func (ra *ResponseAssertions) AssertNoErrors() *ResponseAssertions {
	errs := ra.response.Errors()
	if len(errs) > 0 {
		ra.response.Errorf("❌ Expected no GraphQL errors, got %d:", len(errs))
		for i, err := range errs {
			ra.response.Logf("   [%d] %s", i+1, err.Message)
			if err.Extensions != nil {
				extJSON, _ := json.MarshalIndent(err.Extensions, "   ", "  ")
				ra.response.Logf("       Extensions: %s", string(extJSON))
			}
		}
	}
	return ra
}

/**
 * AssertHasErrors asserts that the response contains at least one GraphQL error.
 */
func (ra *ResponseAssertions) AssertHasErrors() *ResponseAssertions {
	if len(ra.response.Errors()) == 0 {
		ra.response.Errorf("❌ Expected GraphQL errors, but got none")
	}
	return ra
}

/**
 * AssertErrorCount asserts the exact number of GraphQL errors.
 *
 * Parameters:
 *   count - Expected number of errors
 */
func (ra *ResponseAssertions) AssertErrorCount(count int) *ResponseAssertions {
	errs := ra.response.Errors()
	if len(errs) != count {
		ra.response.Errorf("❌ Expected %d GraphQL errors, got %d", count, len(errs))
	}
	return ra
}

/**
 * AssertErrorMessage asserts that at least one error has the exact message.
 *
 * Parameters:
 *   message - Expected error message (exact match)
 */
func (ra *ResponseAssertions) AssertErrorMessage(message string) *ResponseAssertions {
	for _, err := range ra.response.Errors() {
		if err.Message == message {
			return ra
		}
	}

	ra.response.Errorf("❌ Expected error message '%s' not found", message)
	ra.response.Logf("   Actual errors:")
	for _, err := range ra.response.Errors() {
		ra.response.Logf("   - %s", err.Message)
	}
	return ra
}

/**
 * AssertErrorContains asserts that at least one error message contains the substring.
 *
 * Parameters:
 *   substring - Text that should appear in an error message
 */
func (ra *ResponseAssertions) AssertErrorContains(substring string) *ResponseAssertions {
	for _, err := range ra.response.Errors() {
		if strings.Contains(err.Message, substring) {
			return ra
		}
	}

	ra.response.Errorf("❌ No error containing '%s' found", substring)
	ra.response.Logf("   Actual errors:")
	for _, err := range ra.response.Errors() {
		ra.response.Logf("   - %s", err.Message)
	}
	return ra
}

/**
 * AssertErrorCategory asserts that at least one error has the specified category.
 *
 * Error categories are stored in the Extensions map under "category".
 *
 * Parameters:
 *   category - Expected error category (e.g., "validation", "authentication")
 */
func (ra *ResponseAssertions) AssertErrorCategory(category string) *ResponseAssertions {
	for _, err := range ra.response.Errors() {
		if cat, ok := err.Extensions["category"].(string); ok && cat == category {
			return ra
		}
	}

	ra.response.Errorf("❌ No error with category '%s' found", category)
	return ra
}

// ============================================================================
// Data Assertions
// ============================================================================

/**
 * AssertData asserts that the response data is not nil.
 */
func (ra *ResponseAssertions) AssertData() *ResponseAssertions {
	if ra.response.Data() == nil {
		ra.response.Errorf("❌ Expected data field, got nil")
	}
	return ra
}

/**
 * AssertDataNil asserts that the response data is nil.
 */
func (ra *ResponseAssertions) AssertDataNil() *ResponseAssertions {
	if ra.response.Data() != nil {
		ra.response.Errorf("❌ Expected data to be nil")
	}
	return ra
}

/**
 * AssertJSON asserts exact JSON match with the response data.
 *
 * Parameters:
 *   expected - Expected JSON structure (maps, slices, primitives)
 */
func (ra *ResponseAssertions) AssertJSON(expected interface{}) *ResponseAssertions {
	expectedJSON, _ := json.Marshal(expected)
	var expectedNorm interface{}
	json.Unmarshal(expectedJSON, &expectedNorm)

	if !deepEqual(expectedNorm, ra.response.Data()) {
		diff := cmp.Diff(expectedNorm, ra.response.Data(), cmpopts.EquateEmpty())
		ra.response.Errorf("❌ JSON mismatch (-expected +actual):\n%s", diff)
	}
	return ra
}

/**
 * AssertJSONSubset asserts that the response contains the expected subset.
 *
 * Unlike AssertJSON, extra fields in the response are ignored.
 *
 * Parameters:
 *   expected - Expected subset of the response
 */
func (ra *ResponseAssertions) AssertJSONSubset(expected interface{}) *ResponseAssertions {
	expectedJSON, _ := json.Marshal(expected)
	var expectedNorm interface{}
	json.Unmarshal(expectedJSON, &expectedNorm)

	if !isSubset(expectedNorm, ra.response.Data()) {
		ra.response.Errorf("❌ Expected subset not found in response")
		actualJSON, _ := json.MarshalIndent(ra.response.Data(), "   ", "  ")
		ra.response.Logf("   Actual: %s", string(actualJSON))
	}
	return ra
}

/**
 * AssertJSONPath asserts a value at a specific JSON path.
 *
 * Uses dot notation: "data.user.name", "data.users.0.email"
 *
 * Parameters:
 *   path     - Dot-separated path to the value
 *   expected - Expected value at that path
 */
func (ra *ResponseAssertions) AssertJSONPath(path string, expected interface{}) *ResponseAssertions {
	actual := ra.response.JSON(path)
	if !deepEqual(actual, expected) {
		ra.response.Errorf("❌ Path '%s': expected %v, got %v", path, expected, actual)
	}
	return ra
}

/**
 * AssertJSONCount asserts the number of items in a JSON array.
 *
 * Parameters:
 *   path  - Path to the array
 *   count - Expected number of items
 */
func (ra *ResponseAssertions) AssertJSONCount(path string, count int) *ResponseAssertions {
	arr := ra.response.JSONArray(path)
	if len(arr) != count {
		ra.response.Errorf("❌ Path '%s': expected %d items, got %d", path, count, len(arr))
	}
	return ra
}

/**
 * AssertJSONNotEmpty asserts a JSON array is not empty.
 */
func (ra *ResponseAssertions) AssertJSONNotEmpty(path string) *ResponseAssertions {
	arr := ra.response.JSONArray(path)
	if len(arr) == 0 {
		ra.response.Errorf("❌ Path '%s': expected non-empty array", path)
	}
	return ra
}

/**
 * AssertJSONEmpty asserts a JSON array is empty.
 */
func (ra *ResponseAssertions) AssertJSONEmpty(path string) *ResponseAssertions {
	arr := ra.response.JSONArray(path)
	if len(arr) != 0 {
		ra.response.Errorf("❌ Path '%s': expected empty array, got %d items", path, len(arr))
	}
	return ra
}

// ============================================================================
// Helper Functions
// ============================================================================

/**
 * deepEqual performs deep comparison with numeric type normalization.
 */
func deepEqual(a, b interface{}) bool {
	a = normalizeNumeric(a)
	b = normalizeNumeric(b)
	return reflect.DeepEqual(a, b)
}

/**
 * normalizeNumeric converts integer types to float64 for consistent comparison.
 *
 * JSON numbers are always float64 after unmarshaling, but test expectations
 * often use integers. This normalizes both to float64.
 */
func normalizeNumeric(val interface{}) interface{} {
	if val == nil {
		return nil
	}

	switch v := val.(type) {
	case int:
		return float64(v)
	case int8:
		return float64(v)
	case int16:
		return float64(v)
	case int32:
		return float64(v)
	case int64:
		return float64(v)
	case uint:
		return float64(v)
	case uint8:
		return float64(v)
	case uint16:
		return float64(v)
	case uint32:
		return float64(v)
	case uint64:
		return float64(v)
	case float32:
		return float64(v)
	}
	return val
}

/**
 * isSubset checks if expected is a subset of actual.
 *
 * Recursively compares maps and slices, returning true if all
 * keys and values in expected exist in actual.
 */
func isSubset(expected, actual interface{}) bool {
	if expected == nil {
		return actual == nil
	}

	expectedMap, expectedIsMap := expected.(map[string]interface{})
	actualMap, actualIsMap := actual.(map[string]interface{})

	if expectedIsMap && actualIsMap {
		for key, expectedVal := range expectedMap {
			actualVal, ok := actualMap[key]
			if !ok {
				return false
			}
			if !isSubset(expectedVal, actualVal) {
				return false
			}
		}
		return true
	}

	expectedSlice, expectedIsSlice := expected.([]interface{})
	actualSlice, actualIsSlice := actual.([]interface{})

	if expectedIsSlice && actualIsSlice {
		if len(expectedSlice) > len(actualSlice) {
			return false
		}
		for i, expectedVal := range expectedSlice {
			if !isSubset(expectedVal, actualSlice[i]) {
				return false
			}
		}
		return true
	}

	return deepEqual(expected, actual)
}

// Ensure ResponseAssertions is not empty (compile-time check)
var _ = NewResponseAssertions
