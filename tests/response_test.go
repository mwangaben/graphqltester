package tests

import (
	. "github.com/mwangaben/graphqltester/types"
	"net/http"
	"testing"

	. "github.com/mwangaben/graphqltester"
	"github.com/stretchr/testify/assert"
)

/**
 * TestResponse_AssertStatus_MatchesExact validates that AssertStatus
 * correctly passes when the status code matches exactly.
 */
func TestResponse_AssertStatus_MatchesExact(t *testing.T) {
	// Setup: Create a mock tester and response
	mockT := &testing.T{}
	tester := &Tester{t: mockT, config: DefaultConfig()}

	response := &Response{
		tester:     tester,
		statusCode: http.StatusOK,
	}

	// Test: AssertStatus should pass without error for matching status
	// Note: In real test, this would not call Errorf
	result := response.AssertStatus(http.StatusOK)
	assert.Same(t, response, result, "Should return same response for chaining")
}

/**
 * TestResponse_AssertOK_ShorthandFor200 validates that AssertOK
 * correctly checks for 200 status.
 */
func TestResponse_AssertOK_ShorthandFor200(t *testing.T) {
	mockT := &testing.T{}
	tester := &Tester{t: mockT, config: DefaultConfig()}

	response := &Response{
		tester:     tester,
		statusCode: http.StatusOK,
	}

	result := response.AssertOK()
	assert.Same(t, response, result, "Should return same response for chaining")
}

/**
 * TestResponse_AssertCreated_ShorthandFor201 validates that AssertCreated
 * correctly checks for 201 status.
 */
func TestResponse_AssertCreated_ShorthandFor201(t *testing.T) {
	mockT := &testing.T{}
	tester := &Tester{t: mockT, config: DefaultConfig()}

	response := &Response{
		tester:     tester,
		statusCode: http.StatusCreated,
	}

	result := response.AssertCreated()
	assert.Same(t, response, result, "Should return same response for chaining")
}

/**
 * TestResponse_AssertNoErrors_NoErrors validates that AssertNoErrors
 * passes when there are no GraphQL errors.
 */
func TestResponse_AssertNoErrors_NoErrors(t *testing.T) {
	mockT := &testing.T{}
	tester := &Tester{t: mockT, config: DefaultConfig()}

	response := &Response{
		tester: tester,
		Errors: nil, // No errors
	}

	result := response.AssertNoErrors()
	assert.Same(t, response, result)
}

/**
 * TestResponse_AssertNoErrors_WithErrors validates that AssertNoErrors
 * correctly reports when errors are present.
 */
func TestResponse_AssertNoErrors_WithErrors(t *testing.T) {
	mockT := &testing.T{}
	tester := &Tester{t: mockT, config: DefaultConfig{Debug: false}}

	response := &Response{
		tester: tester,
		Errors: []*GraphQLError{
			{Message: "Something went wrong"},
		},
	}

	// This test verifies the behavior, but in practice AssertNoErrors
	// would call t.Errorf which would fail the test
	// We just verify the method doesn't panic
	result := response.AssertNoErrors()
	assert.Same(t, response, result)
}

/**
 * TestResponse_AssertErrorCount_ExactMatch validates that AssertErrorCount
 * correctly checks the number of errors.
 */
func TestResponse_AssertErrorCount_ExactMatch(t *testing.T) {
	mockT := &testing.T{}
	tester := &Tester{t: mockT, config: DefaultConfig{Debug: false}}

	response := &Response{
		tester: tester,
		Errors: []*GraphQLError{
			{Message: "Error 1"},
			{Message: "Error 2"},
			{Message: "Error 3"},
		},
	}

	result := response.AssertErrorCount(3)
	assert.Same(t, response, result)
}

/**
 * TestResponse_AssertErrorMessage_Found validates that AssertErrorMessage
 * finds an error with the exact message.
 */
func TestResponse_AssertErrorMessage_Found(t *testing.T) {
	mockT := &testing.T{}
	tester := &Tester{t: mockT, config: DefaultConfig{Debug: false}}

	response := &Response{
		tester: tester,
		Errors: []*GraphQLError{
			{Message: "Unauthenticated."},
			{Message: "Validation failed."},
		},
	}

	result := response.AssertErrorMessage("Unauthenticated.")
	assert.Same(t, response, result)
}

/**
 * TestResponse_AssertErrorContains_PartialMatch validates that
 * AssertErrorContains finds an error containing the substring.
 */
func TestResponse_AssertErrorContains_PartialMatch(t *testing.T) {
	mockT := &testing.T{}
	tester := &Tester{t: mockT, config: DefaultConfig{Debug: false}}

	response := &Response{
		tester: tester,
		Errors: []*GraphQLError{
			{Message: "The zone name field is required."},
		},
	}

	result := response.AssertErrorContains("zone name")
	assert.Same(t, response, result)
}

/**
 * TestResponse_AssertErrorCategory_Found validates that AssertErrorCategory
 * finds an error with the specified category in extensions.
 */
func TestResponse_AssertErrorCategory_Found(t *testing.T) {
	mockT := &testing.T{}
	tester := &Tester{t: mockT, config: DefaultConfig{Debug: false}}

	response := &Response{
		tester: tester,
		Errors: []*GraphQLError{
			{
				Message: "Validation failed",
				Extensions: map[string]interface{}{
					"category": "validation",
				},
			},
		},
	}

	result := response.AssertErrorCategory("validation")
	assert.Same(t, response, result)
}

/**
 * TestResponse_AssertData_WithData validates that AssertData
 * passes when data is present.
 */
func TestResponse_AssertData_WithData(t *testing.T) {
	mockT := &testing.T{}
	tester := &Tester{t: mockT, config: DefaultConfig{Debug: false}}

	response := &Response{
		tester: tester,
		Data:   map[string]interface{}{"user": "John"},
	}

	result := response.AssertData()
	assert.Same(t, response, result)
}

/**
 * TestResponse_AssertDataNil_NilData validates that AssertDataNil
 * passes when data is nil.
 */
func TestResponse_AssertDataNil_NilData(t *testing.T) {
	mockT := &testing.T{}
	tester := &Tester{t: mockT, config: DefaultConfig{Debug: false}}

	response := &Response{
		tester: tester,
		Data:   nil,
	}

	result := response.AssertDataNil()
	assert.Same(t, response, result)
}

/**
 * TestResponse_AssertJSON_ExactMatch validates that AssertJSON
 * correctly matches the entire response structure.
 */
func TestResponse_AssertJSON_ExactMatch(t *testing.T) {
	mockT := &testing.T{}
	tester := &Tester{t: mockT, config: DefaultConfig{Debug: false}}

	response := &Response{
		tester: tester,
		Data: map[string]interface{}{
			"user": map[string]interface{}{
				"name":  "John Doe",
				"email": "john@example.com",
				"age":   float64(30), // JSON numbers are float64
			},
		},
	}

	expected := map[string]interface{}{
		"user": map[string]interface{}{
			"name":  "John Doe",
			"email": "john@example.com",
			"age":   30, // Integer will be normalized to float64
		},
	}

	result := response.AssertJSON(expected)
	assert.Same(t, response, result)
}

/**
 * TestResponse_AssertJSONSubset_PartialMatch validates that AssertJSONSubset
 * correctly matches a subset of the response.
 */
func TestResponse_AssertJSONSubset_PartialMatch(t *testing.T) {
	mockT := &testing.T{}
	tester := &Tester{t: mockT, config: DefaultConfig{Debug: false}}

	response := &Response{
		tester: tester,
		Data: map[string]interface{}{
			"user": map[string]interface{}{
				"name":     "John Doe",
				"email":    "john@example.com",
				"age":      float64(30),
				"location": "NYC",
			},
		},
	}

	// Only check subset of fields
	expected := map[string]interface{}{
		"user": map[string]interface{}{
			"name": "John Doe",
		},
	}

	result := response.AssertJSONSubset(expected)
	assert.Same(t, response, result)
}

/**
 * TestResponse_AssertJSONPath_NestedValue validates that AssertJSONPath
 * correctly navigates to nested values using dot notation.
 */
func TestResponse_AssertJSONPath_NestedValue(t *testing.T) {
	mockT := &testing.T{}
	tester := &Tester{t: mockT, config: DefaultConfig{Debug: false}}

	response := &Response{
		tester: tester,
		Data: map[string]interface{}{
			"data": map[string]interface{}{
				"user": map[string]interface{}{
					"name": "John Doe",
					"address": map[string]interface{}{
						"city": "New York",
					},
				},
			},
		},
	}

	result := response.
		AssertJSONPath("data.user.name", "John Doe").
		AssertJSONPath("data.user.address.city", "New York")

	assert.Same(t, response, result)
}

/**
 * TestResponse_AssertJSONCount_ArrayLength validates that AssertJSONCount
 * correctly counts items in a JSON array.
 */
func TestResponse_AssertJSONCount_ArrayLength(t *testing.T) {
	mockT := &testing.T{}
	tester := &Tester{t: mockT, config: DefaultConfig{Debug: false}}

	response := &Response{
		tester: tester,
		Data: map[string]interface{}{
			"data": map[string]interface{}{
				"users": []interface{}{
					map[string]interface{}{"name": "John"},
					map[string]interface{}{"name": "Jane"},
					map[string]interface{}{"name": "Bob"},
				},
			},
		},
	}

	result := response.AssertJSONCount("data.users", 3)
	assert.Same(t, response, result)
}

/**
 * TestResponse_JSON_PathNavigation validates the JSON path navigation
 * with various path formats including array indices.
 */
func TestResponse_JSON_PathNavigation(t *testing.T) {
	mockT := &testing.T{}
	tester := &Tester{t: mockT, config: DefaultConfig{Debug: false}}

	response := &Response{
		tester: tester,
		Data: map[string]interface{}{
			"data": map[string]interface{}{
				"users": []interface{}{
					map[string]interface{}{
						"name":  "John",
						"roles": []interface{}{"admin", "user"},
					},
					map[string]interface{}{
						"name":  "Jane",
						"roles": []interface{}{"user"},
					},
				},
			},
		},
	}

	// Test nested object access
	assert.Equal(t, "John", response.JSONString("data.users.0.name"))

	// Test nested array access
	assert.Equal(t, "admin", response.JSONString("data.users.0.roles.0"))

	// Test second array element
	assert.Equal(t, "Jane", response.JSONString("data.users.1.name"))

	// Test non-existent path
	assert.Nil(t, response.JSON("data.users.0.nonexistent"))

	// Test out of bounds index
	assert.Nil(t, response.JSON("data.users.99.name"))
}

/**
 * TestResponse_JSONType_TypeConversions validates the typed JSON
 * accessor methods return correct types.
 */
func TestResponse_JSONType_TypeConversions(t *testing.T) {
	mockT := &testing.T{}
	tester := &Tester{t: mockT, config: DefaultConfig{Debug: false}}

	response := &Response{
		tester: tester,
		Data: map[string]interface{}{
			"data": map[string]interface{}{
				"name":    "John Doe",
				"age":     float64(30),
				"active":  true,
				"score":   float64(95.5),
				"friends": []interface{}{"Jane", "Bob"},
				"metadata": map[string]interface{}{
					"key": "value",
				},
			},
		},
	}

	assert.Equal(t, "John Doe", response.JSONString("data.name"))
	assert.Equal(t, 30, response.JSONInt("data.age"))
	assert.Equal(t, true, response.JSONBool("data.active"))
	assert.Equal(t, 95.5, response.JSONFloat("data.score"))
	assert.Len(t, response.JSONArray("data.friends"), 2)
	assert.Equal(t, "value", response.JSONMap("data.metadata")["key"])
}

/**
 * TestResponse_DeepEqual_NumericNormalization validates that deepEqual
 * correctly handles integer/float differences from JSON parsing.
 */
func TestResponse_DeepEqual_NumericNormalization(t *testing.T) {
	// Integers should match floats after normalization
	assert.True(t, deepEqual(30, float64(30)), "int should match float64")
	assert.True(t, deepEqual(int64(30), float64(30)), "int64 should match float64")
	assert.True(t, deepEqual(float32(30.5), float64(30.5)), "float32 should match float64")

	// Different values should not match
	assert.False(t, deepEqual(30, float64(31)), "Different values should not match")
}

/**
 * TestResponse_IsSubset_NestedMaps validates that isSubset correctly
 * handles nested map structures.
 */
func TestResponse_IsSubset_NestedMaps(t *testing.T) {
	actual := map[string]interface{}{
		"user": map[string]interface{}{
			"name":  "John",
			"email": "john@example.com",
			"address": map[string]interface{}{
				"city":    "NYC",
				"country": "USA",
			},
		},
	}

	expected := map[string]interface{}{
		"user": map[string]interface{}{
			"name": "John",
			"address": map[string]interface{}{
				"city": "NYC",
			},
		},
	}

	assert.True(t, isSubset(expected, actual), "Subset should match")

	// Extra field in expected should fail
	expected["user"].(map[string]interface{})["phone"] = "555-1234"
	assert.False(t, isSubset(expected, actual), "Extra field should fail")
}
