package graphqltester

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/mwangaben/graphqltester/assertions"
	"github.com/mwangaben/graphqltester/types"
)

// Compile-time interface compliance checks
var (
	_ types.ResponseInterface  = (*Response)(nil)
	_ types.AssertionResponse  = (*Response)(nil)
	_ types.DebuggableResponse = (*Response)(nil)
)

/**
 * Response wraps a GraphQL HTTP response with assertion methods.
 *
 * Implements types.ResponseInterface, types.AssertionResponse, and
 * types.DebuggableResponse to work with the assertions package
 * without creating cyclic imports.
 */
type Response struct {
	data       interface{}
	errors     []*types.GraphQLError
	extensions map[string]interface{}

	tester     *Tester
	statusCode int
	rawBody    []byte
	elapsed    time.Duration
}

// ============================================================================
// types.ResponseInterface Implementation
// ============================================================================

func (r *Response) Status() int                        { return r.statusCode }
func (r *Response) Data() interface{}                  { return r.data }
func (r *Response) Errors() []*types.GraphQLError      { return r.errors }
func (r *Response) Extensions() map[string]interface{} { return r.extensions }
func (r *Response) RawBody() []byte                    { return r.rawBody }
func (r *Response) Elapsed() time.Duration             { return r.elapsed }

// ============================================================================
// types.AssertionResponse Implementation
// ============================================================================

func (r *Response) Logf(format string, args ...interface{})   { r.tester.t.Logf(format, args...) }
func (r *Response) Errorf(format string, args ...interface{}) { r.tester.t.Errorf(format, args...) }
func (r *Response) Fatalf(format string, args ...interface{}) { r.tester.t.Fatalf(format, args...) }

// ============================================================================
// types.DebuggableResponse Implementation
// ============================================================================

func (r *Response) Debug() {
	if r.tester.config.debugEnabled {
		r.tester.t.Logf("   Raw body: %s", string(r.rawBody))
	}
}

func (r *Response) Dump() types.DebuggableResponse {
	r.tester.t.Log("╔══════════════════════════════════════════════════╗")
	r.tester.t.Log("║              GraphQL Response Dump               ║")
	r.tester.t.Log("╠══════════════════════════════════════════════════╣")
	r.tester.t.Logf("║ Status: %d %s (took %v)", r.statusCode, http.StatusText(r.statusCode), r.elapsed)
	r.tester.t.Log("╠══════════════════════════════════════════════════╣")

	if r.data != nil {
		dataJSON, _ := json.MarshalIndent(r.data, "║ ", "  ")
		r.tester.t.Logf("║ Data:\n║ %s", string(dataJSON))
	} else {
		r.tester.t.Log("║ Data: nil")
	}

	if len(r.errors) > 0 {
		r.tester.t.Log("╠══════════════════════════════════════════════════╣")
		r.tester.t.Logf("║ Errors (%d):", len(r.errors))
		for i, err := range r.errors {
			r.tester.t.Logf("║ [%d] %s", i+1, err.Message)
			if err.Extensions != nil {
				extJSON, _ := json.MarshalIndent(err.Extensions, "║   ", "  ")
				r.tester.t.Logf("║     Extensions: %s", string(extJSON))
			}
		}
	}

	r.tester.t.Log("╚══════════════════════════════════════════════════╝")
	return r
}

// ============================================================================
// JSON Path Navigation
// ============================================================================

/**
 * JSON retrieves a value from the response data using dot notation.
 *
 * Navigates through nested maps and slices using dot-separated paths.
 * Numeric path segments are treated as array indices.
 *
 * Parameters:
 *   path - Dot-separated path (e.g., "data.user.name", "data.users.0.email")
 *
 * Returns:
 *   The value at the path, or nil if not found
 */
func (r *Response) JSON(path string) interface{} {
	if r.data == nil {
		return nil
	}

	parts := strings.Split(path, ".")
	current := r.data

	for _, part := range parts {
		if idx, err := strconv.Atoi(part); err == nil {
			if arr, ok := current.([]interface{}); ok && idx < len(arr) {
				current = arr[idx]
			} else {
				return nil
			}
		} else {
			if m, ok := current.(map[string]interface{}); ok {
				if val, exists := m[part]; exists {
					current = val
				} else {
					return nil
				}
			} else {
				return nil
			}
		}
	}

	return current
}

func (r *Response) JSONString(path string) string {
	val := r.JSON(path)
	if str, ok := val.(string); ok {
		return str
	}
	return ""
}

func (r *Response) JSONInt(path string) int {
	val := r.JSON(path)
	switch v := val.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	}
	return 0
}

func (r *Response) JSONFloat(path string) float64 {
	val := r.JSON(path)
	if f, ok := val.(float64); ok {
		return f
	}
	return 0
}

func (r *Response) JSONBool(path string) bool {
	val := r.JSON(path)
	if b, ok := val.(bool); ok {
		return b
	}
	return false
}

func (r *Response) JSONArray(path string) []interface{} {
	val := r.JSON(path)
	if arr, ok := val.([]interface{}); ok {
		return arr
	}
	return nil
}

func (r *Response) JSONMap(path string) map[string]interface{} {
	val := r.JSON(path)
	if m, ok := val.(map[string]interface{}); ok {
		return m
	}
	return nil
}

// ============================================================================
// Diff Helpers for Better Assertion Output
// ============================================================================

// DiffResult represents a difference between expected and actual values
type DiffResult struct {
	Path     string
	Expected interface{}
	Actual   interface{}
	Message  string
}

// findDifferences recursively finds differences between expected and actual
func findDifferences(path string, expected, actual interface{}) []DiffResult {
	var diffs []DiffResult

	switch exp := expected.(type) {
	case map[string]interface{}:
		act, ok := actual.(map[string]interface{})
		if !ok {
			diffs = append(diffs, DiffResult{
				Path:    path,
				Message: fmt.Sprintf("expected object but got %T", actual),
			})
			return diffs
		}

		for key, expVal := range exp {
			newPath := path
			if path == "" {
				newPath = key
			} else {
				newPath = path + "." + key
			}

			actVal, exists := act[key]
			if !exists {
				diffs = append(diffs, DiffResult{
					Path:    newPath,
					Message: fmt.Sprintf("key not found in response (expected: %v)", expVal),
				})
				continue
			}

			childDiffs := findDifferences(newPath, expVal, actVal)
			diffs = append(diffs, childDiffs...)
		}

	case []interface{}:
		act, ok := actual.([]interface{})
		if !ok {
			diffs = append(diffs, DiffResult{
				Path:    path,
				Message: fmt.Sprintf("expected array but got %T", actual),
			})
			return diffs
		}

		if len(exp) != len(act) {
			diffs = append(diffs, DiffResult{
				Path:    path,
				Message: fmt.Sprintf("array length mismatch: expected %d, got %d", len(exp), len(act)),
			})
		}

		// Check each element
		for i := range exp {
			if i >= len(act) {
				diffs = append(diffs, DiffResult{
					Path:    fmt.Sprintf("%s[%d]", path, i),
					Message: "array element missing",
				})
				continue
			}

			childPath := fmt.Sprintf("%s[%d]", path, i)
			childDiffs := findDifferences(childPath, exp[i], act[i])
			diffs = append(diffs, childDiffs...)
		}

	default:
		// Simple value comparison
		if !reflect.DeepEqual(expected, actual) {
			diffs = append(diffs, DiffResult{
				Path:     path,
				Expected: expected,
				Actual:   actual,
			})
		}
	}

	return diffs
}

// formatValue formats a value for display
func formatValue(val interface{}) string {
	if val == nil {
		return "null"
	}

	switch v := val.(type) {
	case string:
		return fmt.Sprintf("'%s'", v)
	case float64:
		if v == float64(int(v)) {
			return fmt.Sprintf("%d", int(v))
		}
		return fmt.Sprintf("%.1f", v)
	case bool:
		return fmt.Sprintf("%t", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// isSubset checks if subset is a subset of superset
func isSubset(subset, superset interface{}) bool {
	// Convert to JSON and back to normalize
	subsetJSON, _ := json.Marshal(subset)
	supersetJSON, _ := json.Marshal(superset)

	var subsetNorm interface{}
	var supersetNorm interface{}
	json.Unmarshal(subsetJSON, &subsetNorm)
	json.Unmarshal(supersetJSON, &supersetNorm)

	return isSubsetRecursive(subsetNorm, supersetNorm)
}

func isSubsetRecursive(subset, superset interface{}) bool {
	switch sub := subset.(type) {
	case map[string]interface{}:
		super, ok := superset.(map[string]interface{})
		if !ok {
			return false
		}
		for key, subVal := range sub {
			superVal, exists := super[key]
			if !exists {
				return false
			}
			if !isSubsetRecursive(subVal, superVal) {
				return false
			}
		}
		return true

	case []interface{}:
		super, ok := superset.([]interface{})
		if !ok {
			return false
		}
		if len(sub) > len(super) {
			return false
		}
		// Check if all elements in subset exist in superset (order matters)
		for i, subVal := range sub {
			if i >= len(super) {
				return false
			}
			if !isSubsetRecursive(subVal, super[i]) {
				return false
			}
		}
		return true

	default:
		return reflect.DeepEqual(subset, superset)
	}
}

// ============================================================================
// Assertion Helpers (delegates to assertions package)
// ============================================================================

/**
 * AssertStatus asserts the HTTP status code.
 */
func (r *Response) AssertStatus(code int) *Response {
	assertions.NewResponseAssertions(r).AssertStatus(code)
	return r
}

/**
 * AssertOK asserts HTTP 200 OK.
 */
func (r *Response) AssertOK() *Response {
	assertions.NewResponseAssertions(r).AssertOK()
	return r
}

/**
 * AssertCreated asserts HTTP 201 Created.
 */
func (r *Response) AssertCreated() *Response {
	assertions.NewResponseAssertions(r).AssertCreated()
	return r
}

/**
 * AssertNoErrors asserts no GraphQL errors.
 */
func (r *Response) AssertNoErrors() *Response {
	assertions.NewResponseAssertions(r).AssertNoErrors()
	return r
}

/**
 * AssertHasErrors asserts GraphQL errors exist.
 */
func (r *Response) AssertHasErrors() *Response {
	assertions.NewResponseAssertions(r).AssertHasErrors()
	return r
}

/**
 * AssertErrorCount asserts the number of GraphQL errors.
 */
func (r *Response) AssertErrorCount(count int) *Response {
	assertions.NewResponseAssertions(r).AssertErrorCount(count)
	return r
}

/**
 * AssertErrorMessage asserts an exact error message.
 */
func (r *Response) AssertErrorMessage(message string) *Response {
	assertions.NewResponseAssertions(r).AssertErrorMessage(message)
	return r
}

/**
 * AssertErrorContains asserts an error contains a substring.
 */
func (r *Response) AssertErrorContains(substring string) *Response {
	assertions.NewResponseAssertions(r).AssertErrorContains(substring)
	return r
}

/**
 * AssertErrorCategory asserts an error category.
 */
func (r *Response) AssertErrorCategory(category string) *Response {
	assertions.NewResponseAssertions(r).AssertErrorCategory(category)
	return r
}

/**
 * AssertJSON asserts exact JSON match.
 *
 * This will show a detailed diff when the assertion fails.
 */
func (r *Response) AssertJSON(expected interface{}) *Response {
	// Normalize both sides
	expectedJSON, _ := json.Marshal(expected)
	var expectedNorm interface{}
	json.Unmarshal(expectedJSON, &expectedNorm)

	actualJSON, _ := json.Marshal(r.data)
	var actualNorm interface{}
	json.Unmarshal(actualJSON, &actualNorm)

	if !reflect.DeepEqual(expectedNorm, actualNorm) {
		r.tester.t.Errorf("❌ JSON mismatch")

		// Find differences
		diffs := findDifferences("", expectedNorm, actualNorm)

		// Show expected
		expectedPretty, _ := json.MarshalIndent(expected, "", "  ")
		r.tester.t.Logf("📋 Expected:")
		r.tester.t.Logf("%s", string(expectedPretty))

		// Show actual
		actualPretty, _ := json.MarshalIndent(r.data, "", "  ")
		r.tester.t.Logf("📋 Actual:")
		r.tester.t.Logf("%s", string(actualPretty))

		// Show differences
		if len(diffs) > 0 {
			r.tester.t.Logf("🔍 Differences:")
			for _, diff := range diffs {
				if diff.Message != "" {
					r.tester.t.Logf("   ❌ %s: %s", diff.Path, diff.Message)
				} else {
					expStr := formatValue(diff.Expected)
					actStr := formatValue(diff.Actual)
					r.tester.t.Logf("   ❌ %s: expected %s, got %s", diff.Path, expStr, actStr)
				}
			}
		}

		r.tester.t.Logf("")
		r.tester.t.Logf("Failed asserting that JSON matches exactly:")
		expectedPretty, _ = json.MarshalIndent(expected, "", "  ")
		r.tester.t.Logf("%s", string(expectedPretty))
	}
	return r
}

/**
 * AssertJSONSubset asserts partial JSON match.
 *
 * Unlike AssertJSON, extra fields in the response are ignored.
 * This will show a detailed diff when the assertion fails.
 */
func (r *Response) AssertJSONSubset(expected interface{}) *Response {
	// Normalize both sides
	expectedJSON, _ := json.Marshal(expected)
	var expectedNorm interface{}
	json.Unmarshal(expectedJSON, &expectedNorm)

	if !isSubset(expectedNorm, r.data) {
		r.tester.t.Errorf("❌ Expected subset not found in response")

		// Find differences
		diffs := findDifferences("", expectedNorm, r.data)

		// Show expected
		expectedPretty, _ := json.MarshalIndent(expected, "", "  ")
		r.tester.t.Logf("📋 Expected:")
		r.tester.t.Logf("%s", string(expectedPretty))

		// Show actual
		actualPretty, _ := json.MarshalIndent(r.data, "", "  ")
		r.tester.t.Logf("📋 Actual:")
		r.tester.t.Logf("%s", string(actualPretty))

		// Show differences
		if len(diffs) > 0 {
			r.tester.t.Logf("🔍 Differences:")
			for _, diff := range diffs {
				if diff.Message != "" {
					r.tester.t.Logf("   ❌ %s: %s", diff.Path, diff.Message)
				} else {
					expStr := formatValue(diff.Expected)
					actStr := formatValue(diff.Actual)
					r.tester.t.Logf("   ❌ %s: expected %s, got %s", diff.Path, expStr, actStr)
				}
			}
		}

		r.tester.t.Logf("")
		r.tester.t.Logf("Failed asserting that an array has the subset:")
		expectedPretty, _ = json.MarshalIndent(expected, "", "  ")
		r.tester.t.Logf("%s", string(expectedPretty))
	}
	return r
}

/**
 * AssertJSONPath asserts a value at a JSON path.
 */
func (r *Response) AssertJSONPath(path string, expected interface{}) *Response {
	assertions.NewResponseAssertions(r).AssertJSONPath(path, expected)
	return r
}

/**
 * AssertJSONCount asserts array item count.
 */
func (r *Response) AssertJSONCount(path string, count int) *Response {
	assertions.NewResponseAssertions(r).AssertJSONCount(path, count)
	return r
}

/**
 * AssertJSONNotEmpty asserts array is not empty.
 */
func (r *Response) AssertJSONNotEmpty(path string) *Response {
	assertions.NewResponseAssertions(r).AssertJSONNotEmpty(path)
	return r
}

/**
 * AssertJSONEmpty asserts array is empty.
 */
func (r *Response) AssertJSONEmpty(path string) *Response {
	assertions.NewResponseAssertions(r).AssertJSONEmpty(path)
	return r
}

/**
 * AssertValidationError asserts a validation error.
 */
func (r *Response) AssertValidationError(field, message string) *Response {
	assertions.NewValidationAssertions(r).AssertValidationError(field, message)
	return r
}

/**
 * AssertValidationRules asserts multiple validation rules.
 */
func (r *Response) AssertValidationRules(expected map[string]string) *Response {
	assertions.NewValidationAssertions(r).AssertValidationRules(expected)
	return r
}

/**
 * AssertValidationErrors asserts validation error count.
 */
func (r *Response) AssertValidationErrors(count int) *Response {
	assertions.NewValidationAssertions(r).AssertValidationErrors(count)
	return r
}

/**
 * AssertValidationFields asserts fields have validation errors.
 */
func (r *Response) AssertValidationFields(fields ...string) *Response {
	assertions.NewValidationAssertions(r).AssertValidationFields(fields...)
	return r
}

/**
 * AssertNoValidationErrors asserts no validation errors.
 */
func (r *Response) AssertNoValidationErrors() *Response {
	assertions.NewValidationAssertions(r).AssertNoValidationErrors()
	return r
}

/**
 * AssertValidationErrorCode asserts a validation error code.
 */
func (r *Response) AssertValidationErrorCode(field, code string) *Response {
	assertions.NewValidationAssertions(r).AssertValidationErrorCode(field, code)
	return r
}

/**
 * AssertUnauthenticated asserts authentication error.
 */
func (r *Response) AssertUnauthenticated() *Response {
	assertions.NewPermissionAssertions(r, r.tester).AssertUnauthenticated()
	return r
}

/**
 * AssertForbidden asserts forbidden error.
 */
func (r *Response) AssertForbidden() *Response {
	assertions.NewPermissionAssertions(r, r.tester).AssertForbidden()
	return r
}

/**
 * AssertPermissionError asserts permission denied error.
 */
func (r *Response) AssertPermissionError() *Response {
	assertions.NewPermissionAssertions(r, r.tester).AssertPermissionError()
	return r
}

/**
 * AssertPermissionDenied asserts specific permission denied.
 */
func (r *Response) AssertPermissionDenied(permission string) *Response {
	assertions.NewPermissionAssertions(r, r.tester).AssertPermissionDenied(permission)
	return r
}

/**
 * AssertDatabaseHas asserts record exists.
 */
func (r *Response) AssertDatabaseHas(table string, conditions map[string]interface{}) *Response {
	assertions.NewDatabaseAssertions(r, r.tester).AssertDatabaseHas(table, conditions)
	return r
}

/**
 * AssertDatabaseMissing asserts record does not exist.
 */
func (r *Response) AssertDatabaseMissing(table string, conditions map[string]interface{}) *Response {
	assertions.NewDatabaseAssertions(r, r.tester).AssertDatabaseMissing(table, conditions)
	return r
}

/**
 * AssertSoftDeleted asserts record is soft deleted.
 */
func (r *Response) AssertSoftDeleted(table string, conditions map[string]interface{}) *Response {
	assertions.NewDatabaseAssertions(r, r.tester).AssertSoftDeleted(table, conditions)
	return r
}

/**
 * AssertNotSoftDeleted asserts record is not soft deleted.
 */
func (r *Response) AssertNotSoftDeleted(table string, conditions map[string]interface{}) *Response {
	assertions.NewDatabaseAssertions(r, r.tester).AssertNotSoftDeleted(table, conditions)
	return r
}

/**
 * AssertDatabaseCount asserts record count.
 */
func (r *Response) AssertDatabaseCount(table string, expectedCount int) *Response {
	assertions.NewDatabaseAssertions(r, r.tester).AssertDatabaseCount(table, expectedCount)
	return r
}

/**
 * AssertDatabaseCountWhere asserts filtered record count.
 */
func (r *Response) AssertDatabaseCountWhere(table string, conditions map[string]interface{}, expectedCount int) *Response {
	assertions.NewDatabaseAssertions(r, r.tester).AssertDatabaseCountWhere(table, conditions, expectedCount)
	return r
}

/**
 * AssertDatabaseValue asserts a specific column value.
 */
func (r *Response) AssertDatabaseValue(table string, conditions map[string]interface{}, column string, expected interface{}) *Response {
	assertions.NewDatabaseAssertions(r, r.tester).AssertDatabaseValue(table, conditions, column, expected)
	return r
}

/**
 * AssertData asserts that the response data field is not nil.
 *
 * Use this as a quick check that the server returned data.
 * For more specific checks, use AssertJSON or AssertJSONPath.
 *
 * Returns:
 *   *Response for chaining
 *
 * Example:
 *   response.AssertData().AssertJSONPath("data.user.name", "John Doe")
 */
func (r *Response) AssertData() *Response {
	if r.data == nil {
		r.tester.t.Errorf("❌ Expected data field, got nil")
		r.Debug()
	}
	return r
}

/**
 * AssertDataNil asserts that the response data field is nil.
 *
 * Use this when you expect no data to be returned (e.g., certain error scenarios).
 *
 * Returns:
 *   *Response for chaining
 */
func (r *Response) AssertDataNil() *Response {
	if r.data != nil {
		r.tester.t.Errorf("❌ Expected data to be nil")
		r.Debug()
	}
	return r
}
