package assertions

import (
	"strings"

	"github.com/mwangaben/graphqltester/types"
)

/**
 * PermissionAssertions provides methods for asserting GraphQL authorization errors.
 *
 * Handles common permission error patterns:
 * - Authentication errors ("Unauthenticated.")
 * - Authorization errors ("This action is unauthorized.")
 * - Permission-specific denial messages
 *
 * Works with the types.AssertionResponse and types.TesterInterface
 * to avoid cyclic imports with the main package.
 */
type PermissionAssertions struct {
	response types.AssertionResponse
	tester   types.TesterInterface
}

/**
 * NewPermissionAssertions creates permission assertion helpers.
 *
 * Parameters:
 *   response - The response to assert on
 *   tester   - The tester instance for permission checking
 *
 * Returns:
 *   *PermissionAssertions for chaining
 */
func NewPermissionAssertions(response types.AssertionResponse, tester types.TesterInterface) *PermissionAssertions {
	return &PermissionAssertions{
		response: response,
		tester:   tester,
	}
}

// ============================================================================
// Authentication Assertions
// ============================================================================

/**
 * AssertUnauthenticated asserts that the response indicates authentication failure.
 *
 * Checks for:
 * - Error message "Unauthenticated."
 * - Error category "authentication"
 * - HTTP 401 status
 *
 * Returns:
 *   *PermissionAssertions for chaining
 */
func (pa *PermissionAssertions) AssertUnauthenticated() *PermissionAssertions {
	// Check GraphQL errors
	for _, err := range pa.response.Errors() {
		if err.Message == "Unauthenticated." || err.Message == "Unauthenticated" {
			return pa
		}
		if cat, ok := err.Extensions["category"].(string); ok && cat == "authentication" {
			return pa
		}
		if strings.Contains(strings.ToLower(err.Message), "unauthenticated") {
			return pa
		}
	}

	// Check HTTP status
	if pa.response.Status() == 401 {
		return pa
	}

	pa.response.Errorf("❌ Expected 'Unauthenticated' error")
	pa.response.Logf("   Available errors:")
	for i, err := range pa.response.Errors() {
		pa.response.Logf("   [%d] %s", i+1, err.Message)
	}
	return pa
}

/**
 * AssertForbidden asserts the response indicates a forbidden action (403).
 *
 * Returns:
 *   *PermissionAssertions for chaining
 */
func (pa *PermissionAssertions) AssertForbidden() *PermissionAssertions {
	for _, err := range pa.response.Errors() {
		if err.Message == "Forbidden" || err.Message == "Forbidden." {
			return pa
		}
		if cat, ok := err.Extensions["category"].(string); ok {
			if cat == "authorization" || cat == "forbidden" {
				return pa
			}
		}
		if strings.Contains(strings.ToLower(err.Message), "forbidden") {
			return pa
		}
	}

	if pa.response.Status() == 403 {
		return pa
	}

	pa.response.Errorf("❌ Expected 'Forbidden' error")
	return pa
}

// ============================================================================
// Permission-Specific Assertions
// ============================================================================

/**
 * AssertPermissionError asserts the standard permission denied error.
 *
 * Checks for "This action is unauthorized." message.
 *
 * Returns:
 *   *PermissionAssertions for chaining
 */
func (pa *PermissionAssertions) AssertPermissionError() *PermissionAssertions {
	for _, err := range pa.response.Errors() {
		if err.Message == "This action is unauthorized." {
			return pa
		}
		if strings.Contains(err.Message, "unauthorized") ||
			strings.Contains(err.Message, "not authorized") {
			return pa
		}
	}

	pa.response.Errorf("❌ Expected permission error 'This action is unauthorized.'")
	pa.response.Logf("   Available errors:")
	for i, err := range pa.response.Errors() {
		pa.response.Logf("   [%d] %s", i+1, err.Message)
	}
	return pa
}

/**
 * AssertPermissionDenied asserts a specific permission was denied.
 *
 * Parameters:
 *   permission - The permission that was denied (e.g., "zones.delete")
 *
 * Returns:
 *   *PermissionAssertions for chaining
 */
func (pa *PermissionAssertions) AssertPermissionDenied(permission string) *PermissionAssertions {
	for _, err := range pa.response.Errors() {
		if strings.Contains(err.Message, permission) {
			return pa
		}
		if perm, ok := err.Extensions["permission"].(string); ok && perm == permission {
			return pa
		}
	}

	pa.response.Errorf("❌ Expected permission denied for '%s'", permission)
	return pa
}

/**
 * AssertHasPermission asserts the current user has a specific permission.
 *
 * This checks the tester's permission state, not the response.
 *
 * Parameters:
 *   permission - The permission to check
 *
 * Returns:
 *   *PermissionAssertions for chaining
 */
func (pa *PermissionAssertions) AssertHasPermission(permission string) *PermissionAssertions {
	if pa.tester.CurrentUser() == nil {
		pa.tester.Fatalf("❌ No authenticated user. Use ActingAs() or GivenUser() first.")
	}

	if !pa.tester.HasPermission(permission) {
		pa.tester.Errorf("❌ User does not have permission '%s'", permission)
	}

	return pa
}

/**
 * AssertHasRole asserts the current user has a specific role.
 *
 * Parameters:
 *   role - The role to check
 *
 * Returns:
 *   *PermissionAssertions for chaining
 */
func (pa *PermissionAssertions) AssertHasRole(role string) *PermissionAssertions {
	if pa.tester.CurrentUser() == nil {
		pa.tester.Fatalf("❌ No authenticated user. Use ActingAs() or GivenUser() first.")
	}

	if !pa.tester.HasRole(role) {
		pa.tester.Errorf("❌ User does not have role '%s'", role)
	}

	return pa
}

/**
 * AssertLacksPermission asserts the current user does NOT have a permission.
 *
 * Parameters:
 *   permission - The permission that should be missing
 *
 * Returns:
 *   *PermissionAssertions for chaining
 */
func (pa *PermissionAssertions) AssertLacksPermission(permission string) *PermissionAssertions {
	if pa.tester.CurrentUser() == nil {
		pa.tester.Fatalf("❌ No authenticated user")
	}

	if pa.tester.HasPermission(permission) {
		pa.tester.Errorf("❌ User should not have permission '%s'", permission)
	}

	return pa
}

/**
 * AssertLacksRole asserts the current user does NOT have a role.
 *
 * Parameters:
 *   role - The role that should be missing
 *
 * Returns:
 *   *PermissionAssertions for chaining
 */
func (pa *PermissionAssertions) AssertLacksRole(role string) *PermissionAssertions {
	if pa.tester.CurrentUser() == nil {
		pa.tester.Fatalf("❌ No authenticated user")
	}

	if pa.tester.HasRole(role) {
		pa.tester.Errorf("❌ User should not have role '%s'", role)
	}

	return pa
}
