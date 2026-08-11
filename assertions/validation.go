package assertions

import (
	"encoding/json"
	"strings"

	"github.com/mwangaben/graphqltester/types"
)

/**
 * ValidationAssertions provides methods for asserting GraphQL validation errors.
 *
 * Supports multiple validation error formats:
 *
 * 1. Lighthouse-style (extensions.validation):
 *    extensions.validation = {"input.name": ["The name field is required."]}
 *
 * 2. Category-style (extensions.category = "validation"):
 *    extensions.category = "validation", extensions.field = "input.name"
 *
 * 3. Simple message-style:
 *    error message contains field name and error description
 */
type ValidationAssertions struct {
	response types.AssertionResponse
}

/**
 * NewValidationAssertions creates validation assertion helpers.
 *
 * Parameters:
 *   response - The response to assert on
 *
 * Returns:
 *   *ValidationAssertions for chaining
 */
func NewValidationAssertions(response types.AssertionResponse) *ValidationAssertions {
	return &ValidationAssertions{response: response}
}

/**
 * AssertValidationError asserts a specific validation error for a field.
 *
 * This searches through all GraphQL errors for validation errors matching
 * the given field and message. Supports multiple validation error formats.
 *
 * Parameters:
 *   field   - The field path (e.g., "input.name", "input.email")
 *   message - The expected error message (partial match supported)
 *
 * Returns:
 *   *ValidationAssertions for chaining
 *
 * Example:
 *   assertions.AssertValidationError("input.name", "The name field is required.")
 */
func (va *ValidationAssertions) AssertValidationError(field string, message string) *ValidationAssertions {
	found := false

	for _, err := range va.response.Errors() {
		if checkLighthouseValidationError(err, field, message) {
			found = true
			break
		}
		if checkCategoryValidationError(err, field, message) {
			found = true
			break
		}
		if checkSimpleValidationError(err, field, message) {
			found = true
			break
		}
	}

	if !found {
		va.response.Errorf(
			"❌ Expected validation error for field '%s' with message containing '%s'",
			field, message,
		)
		va.response.Logf("   Available errors:")
		for i, err := range va.response.Errors() {
			va.response.Logf("   [%d] %s", i+1, err.Message)
			if err.Extensions != nil {
				extJSON, _ := json.MarshalIndent(err.Extensions, "   ", "  ")
				va.response.Logf("       Extensions: %s", string(extJSON))
			}
		}
	}

	return va
}

/**
 * AssertValidationRules asserts multiple validation rules at once.
 *
 * Parameters:
 *   expected - Map of field paths to expected error messages
 *
 * Example:
 *   assertions.AssertValidationRules(map[string]string{
 *       "input.name": "required",
 *       "input.code": "required",
 *   })
 */
func (va *ValidationAssertions) AssertValidationRules(expected map[string]string) *ValidationAssertions {
	for field, message := range expected {
		va.AssertValidationError(field, message)
	}
	return va
}

/**
 * AssertValidationErrors asserts the total number of validation errors.
 *
 * Parameters:
 *   count - Expected number of validation errors across all fields
 */
func (va *ValidationAssertions) AssertValidationErrors(count int) *ValidationAssertions {
	actualCount := countValidationErrors(va.response.Errors())
	if actualCount != count {
		va.response.Errorf("❌ Expected %d validation errors, got %d", count, actualCount)
	}
	return va
}

/**
 * AssertValidationFields asserts that specific fields have validation errors.
 *
 * Parameters:
 *   fields - Field paths that should have validation errors
 */
func (va *ValidationAssertions) AssertValidationFields(fields ...string) *ValidationAssertions {
	for _, field := range fields {
		found := false
		for _, err := range va.response.Errors() {
			if hasFieldValidationError(err, field) {
				found = true
				break
			}
		}
		if !found {
			va.response.Errorf("❌ Expected validation error for field '%s'", field)
		}
	}
	return va
}

/**
 * AssertNoValidationErrors asserts there are no validation errors.
 */
func (va *ValidationAssertions) AssertNoValidationErrors() *ValidationAssertions {
	count := countValidationErrors(va.response.Errors())
	if count > 0 {
		va.response.Errorf("❌ Expected no validation errors, got %d", count)
	}
	return va
}

/**
 * AssertValidationErrorCode asserts a validation error with a specific code.
 *
 * Parameters:
 *   field - The field path
 *   code  - Expected error code (e.g., "REQUIRED", "UNIQUE")
 */
func (va *ValidationAssertions) AssertValidationErrorCode(field string, code string) *ValidationAssertions {
	found := false
	for _, err := range va.response.Errors() {
		if ext, ok := err.Extensions["validation"]; ok {
			if validationMap, ok := ext.(map[string]interface{}); ok {
				if details, ok := validationMap[field]; ok {
					if containsErrorCode(details, code) {
						found = true
						break
					}
				}
			}
		}
	}

	if !found {
		va.response.Errorf("❌ Expected validation error code '%s' for field '%s'", code, field)
	}
	return va
}

// ============================================================================
// Validation Error Format Checkers (package-level functions)
// ============================================================================

/**
 * checkLighthouseValidationError checks Lighthouse-style validation errors.
 *
 * Format: extensions.validation = {"input.field": ["error message"]}
 */
func checkLighthouseValidationError(err *types.GraphQLError, field string, message string) bool {
	ext, ok := err.Extensions["validation"]
	if !ok {
		return false
	}

	validationMap, ok := ext.(map[string]interface{})
	if !ok {
		return false
	}

	messages, ok := validationMap[field]
	if !ok {
		return false
	}

	switch msgList := messages.(type) {
	case []interface{}:
		for _, msg := range msgList {
			if msgStr, ok := msg.(string); ok && strings.Contains(msgStr, message) {
				return true
			}
		}
	case []string:
		for _, msgStr := range msgList {
			if strings.Contains(msgStr, message) {
				return true
			}
		}
	case string:
		if strings.Contains(msgList, message) {
			return true
		}
	}

	return false
}

/**
 * checkCategoryValidationError checks category-style validation errors.
 *
 * Format: extensions.category = "validation", extensions.field = "input.field"
 */
func checkCategoryValidationError(err *types.GraphQLError, field string, message string) bool {
	category, ok := err.Extensions["category"]
	if !ok {
		return false
	}

	catStr, ok := category.(string)
	if !ok || catStr != "validation" {
		return false
	}

	if errField, ok := err.Extensions["field"]; ok {
		if fieldStr, ok := errField.(string); ok && fieldStr != field {
			return false
		}
	}

	return strings.Contains(err.Message, message)
}

/**
 * checkSimpleValidationError checks simple message-style validation errors.
 *
 * Format: error message contains field name and error description
 */
func checkSimpleValidationError(err *types.GraphQLError, field string, message string) bool {
	return strings.Contains(err.Message, field) && strings.Contains(err.Message, message)
}

/**
 * hasFieldValidationError checks if an error relates to a specific field.
 */
func hasFieldValidationError(err *types.GraphQLError, field string) bool {
	if ext, ok := err.Extensions["validation"]; ok {
		if validationMap, ok := ext.(map[string]interface{}); ok {
			if _, exists := validationMap[field]; exists {
				return true
			}
		}
	}

	if cat, ok := err.Extensions["category"]; ok {
		if catStr, ok := cat.(string); ok && catStr == "validation" {
			if errField, ok := err.Extensions["field"]; ok {
				if fieldStr, ok := errField.(string); ok && fieldStr == field {
					return true
				}
			}
		}
	}

	if strings.Contains(err.Message, field) {
		return true
	}

	return false
}

/**
 * countValidationErrors counts total validation error messages.
 */
func countValidationErrors(errors []*types.GraphQLError) int {
	count := 0
	for _, err := range errors {
		if ext, ok := err.Extensions["validation"]; ok {
			if validationMap, ok := ext.(map[string]interface{}); ok {
				for _, messages := range validationMap {
					switch msgList := messages.(type) {
					case []interface{}:
						count += len(msgList)
					case []string:
						count += len(msgList)
					case string:
						count++
					}
				}
			}
		}
		if cat, ok := err.Extensions["category"]; ok {
			if catStr, ok := cat.(string); ok && catStr == "validation" {
				count++
			}
		}
	}
	return count
}

/**
 * containsErrorCode checks if validation details contain a specific error code.
 */
func containsErrorCode(details interface{}, code string) bool {
	switch d := details.(type) {
	case map[string]interface{}:
		if c, ok := d["code"]; ok {
			if codeStr, ok := c.(string); ok && codeStr == code {
				return true
			}
		}
		if rules, ok := d["rules"]; ok {
			switch ruleList := rules.(type) {
			case []interface{}:
				for _, rule := range ruleList {
					if ruleMap, ok := rule.(map[string]interface{}); ok {
						if c, ok := ruleMap["code"]; ok {
							if codeStr, ok := c.(string); ok && codeStr == code {
								return true
							}
						}
					}
				}
			}
		}
	}
	return false
}
