package middleware

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

/**
 * ValidationMiddleware provides input validation for GraphQL requests.
 *
 * This middleware can validate GraphQL variables against predefined rules
 * before they reach the resolver. It integrates with validation packages
 * to provide consistent validation across queries and mutations.
 */

// ValidationMiddleware validates GraphQL request inputs.
//
// Parameters:
//
//	tester - The tester instance (must support validation)
//
// Returns:
//
//	Middleware function for the chain
func ValidationMiddleware(tester interface{}) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only validate POST requests with JSON body
			if r.Method == "POST" && strings.Contains(r.Header.Get("Content-Type"), "application/json") {
				validationErrors := validateRequest(r, tester)

				if len(validationErrors) > 0 {
					ctx := context.WithValue(r.Context(), contextKey("validation_errors"), validationErrors)
					r = r.WithContext(ctx)
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// validateRequest validates the GraphQL request body.
func validateRequest(r *http.Request, tester interface{}) map[string][]string {
	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil
	}

	// Restore the body for downstream handlers
	r.Body = io.NopCloser(strings.NewReader(string(body)))

	// Parse the GraphQL request
	var request struct {
		Query     string                 `json:"query"`
		Variables map[string]interface{} `json:"variables"`
	}

	if err := json.Unmarshal(body, &request); err != nil {
		return nil
	}

	// If no variables, nothing to validate
	if request.Variables == nil {
		return nil
	}

	// Validate variables against rules
	errors := make(map[string][]string)

	// Check for common validation patterns
	for field, value := range request.Variables {
		fieldErrors := validateField(field, value, request.Query)
		if len(fieldErrors) > 0 {
			errors[field] = fieldErrors
		}
	}

	return errors
}

// validateField validates a single field value.
func validateField(field string, value interface{}, query string) []string {
	var errors []string

	// Check for required fields (empty strings, nil values)
	if value == nil || value == "" {
		// Only mark as error if the field appears to be required in the query
		if isRequiredField(field, query) {
			errors = append(errors, "The "+formatFieldName(field)+" field is required.")
		}
	}

	// Check string length (example: max 255 characters)
	if str, ok := value.(string); ok && len(str) > 255 {
		errors = append(errors, "The "+formatFieldName(field)+" may not be greater than 255 characters.")
	}

	return errors
}

// isRequiredField checks if a field appears to be required based on the query.
func isRequiredField(field string, query string) bool {
	// Check if the field has a "!" (non-null) type in the query
	// This is a simplified check - production code would parse the schema
	return strings.Contains(query, field+":") && strings.Contains(query, "!")
}

// formatFieldName formats a field name for error messages.
func formatFieldName(field string) string {
	// Remove "input." prefix if present
	field = strings.TrimPrefix(field, "input.")

	// Convert camelCase to space-separated words
	var result strings.Builder
	for i, r := range field {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune(' ')
		}
		result.WriteRune(r)
	}

	return strings.ToLower(result.String())
}
