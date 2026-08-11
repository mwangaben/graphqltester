package graphqltester

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/mwangaben/graphqltester/types"
)

/**
 * GraphQLClient handles all HTTP communication with the GraphQL test server.
 *
 * This client manages request construction, header propagation, authentication
 * token injection, and response parsing. It acts as the bridge between the
 * tester's state (auth, tenant, context) and the actual HTTP requests.
 *
 * The client is designed to be:
 * - Stateful: Carries authentication and tenant context automatically
 * - Debuggable: Logs all requests and responses when debug mode is enabled
 * - Flexible: Supports custom headers, file uploads, and batch operations
 *
 * Internal Usage:
 *   Created automatically by NewTester, not typically instantiated directly.
 */
type GraphQLClient struct {
	// tester is the parent tester instance, providing access to configuration
	// and state like authentication tokens and tenant context.
	tester *Tester

	// httpClient is the underlying HTTP client for making requests.
	// Uses default transport with no timeout (controlled by context).
	httpClient *http.Client

	// baseURL is the test server's base URL, set automatically from the server.
	baseURL string

	// defaultTimeout is the default request timeout when no context deadline is set.
	defaultTimeout time.Duration
}

/**
 * NewGraphQLClient creates a new GraphQL client for the given tester.
 *
 * The client inherits configuration from the tester and sets up the
 * HTTP client with appropriate defaults for testing.
 *
 * Parameters:
 *   tester - The parent tester instance
 *
 * Returns:
 *   *GraphQLClient configured for the test environment
 */
func NewGraphQLClient(tester *Tester) *GraphQLClient {
	return &GraphQLClient{
		tester: tester,
		httpClient: &http.Client{
			// Disable redirects for test predictability
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		baseURL:        tester.config.BaseURL,
		defaultTimeout: tester.config.Timeout,
	}
}

// ============================================================================
// Public Methods
// ============================================================================

/**
 * GraphQL sends a GraphQL query or mutation to the server.
 *
 * This is the primary method for executing GraphQL operations. It handles:
 * - Building the HTTP request with proper headers
 * - Injecting authentication tokens from tester state
 * - Adding tenant headers for multi-tenancy
 * - Parsing the response into a structured format
 *
 * Parameters:
 *   query - The GraphQL query or mutation string
 *   vars  - Optional variables map (only the first map is used if multiple provided)
 *
 * Returns:
 *   *Response wrapping the GraphQL response with assertion methods
 *
 * Example:
 *   response := client.GraphQL(`
 *       query GetUser($id: ID!) {
 *           user(id: $id) {
 *               name
 *               email
 *           }
 *       }
 *   `, map[string]interface{}{
 *       "id": "123",
 *   })
 *
 *   response.AssertNoErrors().
 *       AssertJSONPath("data.user.name", "John Doe")
 */
func (c *GraphQLClient) GraphQL(query string, vars ...map[string]interface{}) *Response {
	// Extract variables from variadic parameter
	variables := map[string]interface{}{}
	if len(vars) > 0 {
		variables = vars[0]
	}

	return c.doRequest(query, variables, nil, "")
}

/**
 * GraphQLWithHeaders sends a GraphQL request with custom HTTP headers.
 *
 * Use this when you need to test header-specific behavior like:
 * - Custom authentication schemes
 * - API version headers
 * - Content negotiation
 *
 * Parameters:
 *   query     - The GraphQL query string
 *   variables - Query variables
 *   headers   - Custom HTTP headers to include in the request
 *
 * Returns:
 *   *Response wrapping the GraphQL response
 *
 * Example:
 *   client.GraphQLWithHeaders(`{ version }`, nil, map[string]string{
 *       "X-API-Version": "2024-01",
 *   }).AssertJSONPath("version", "2024-01")
 */
func (c *GraphQLClient) GraphQLWithHeaders(
	query string,
	variables map[string]interface{},
	headers map[string]string,
) *Response {
	return c.doRequest(query, variables, headers, "")
}

/**
 * GraphQLNamed executes a named operation from a document with multiple operations.
 *
 * GraphQL documents can contain multiple operations. This method allows
 * specifying which operation to execute by name.
 *
 * Parameters:
 *   query         - GraphQL document containing multiple operations
 *   operationName - Name of the operation to execute
 *   variables     - Variables for the operation
 *
 * Returns:
 *   *Response wrapping the GraphQL response
 *
 * Example:
 *   query := `
 *       query GetUsers { users { name } }
 *       query GetAdmins { admins { name } }
 *   `
 *
 *   client.GraphQLNamed(query, "GetAdmins", nil).
 *       AssertJSONCount("admins", 2)
 */
func (c *GraphQLClient) GraphQLNamed(
	query string,
	operationName string,
	variables map[string]interface{},
) *Response {
	return c.doRequest(query, variables, nil, operationName)
}

/**
 * GraphQLFile reads a GraphQL query from a file and executes it.
 *
 * This is useful for:
 * - Keeping test queries organized in .graphql files
 * - Getting syntax highlighting in your IDE
 * - Reusing queries across multiple tests
 *
 * Parameters:
 *   path      - File path relative to test execution directory
 *   variables - Optional variables for the query
 *
 * Returns:
 *   *Response wrapping the GraphQL response
 *
 * Example:
 *   client.GraphQLFile("./testdata/queries/getUser.graphql",
 *       map[string]interface{}{"id": "123"})
 */
func (c *GraphQLClient) GraphQLFile(path string, variables ...map[string]interface{}) *Response {
	content, err := os.ReadFile(path)
	if err != nil {
		c.tester.t.Fatalf("❌ Failed to read GraphQL file %s: %v", path, err)
	}

	vars := map[string]interface{}{}
	if len(variables) > 0 {
		vars = variables[0]
	}

	return c.doRequest(string(content), vars, nil, "")
}

/**
 * Query is semantic sugar for read operations.
 *
 * Functionally identical to GraphQL() but signals intent in test code.
 *
 * Example:
 *   client.Query(`{ users { name } }`).AssertNoErrors()
 */
func (c *GraphQLClient) Query(query string, vars ...map[string]interface{}) *Response {
	return c.GraphQL(query, vars...)
}

/**
 * Mutation is semantic sugar for write operations.
 *
 * Functionally identical to GraphQL() but signals intent in test code.
 *
 * Example:
 *   client.Mutation(`mutation { createUser(name: "Jane") { id } }`)
 */
func (c *GraphQLClient) Mutation(query string, vars ...map[string]interface{}) *Response {
	return c.GraphQL(query, vars...)
}

/**
 * Batch sends multiple GraphQL queries in a single HTTP request.
 *
 * GraphQL supports sending an array of queries in a single request.
 * This is useful for testing batch query performance and behavior.
 *
 * Parameters:
 *   queries - Array of GraphQL requests to batch together
 *
 * Returns:
 *   []*Response array with each query's response
 *
 * Example:
 *   responses := client.Batch([]GraphQLRequest{
 *       {Query: "{ users { name } }"},
 *       {Query: "{ posts { title } }"},
 *   })
 */
func (c *GraphQLClient) Batch(queries []GraphQLRequest) []*Response {
	body, err := json.Marshal(queries)
	if err != nil {
		c.tester.t.Fatalf("❌ Failed to marshal batch request: %v", err)
	}

	url := c.baseURL + c.tester.config.Endpoint
	httpReq, _ := http.NewRequestWithContext(c.tester.ctx, "POST", url, bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")

	// Add auth if present
	if c.tester.currentToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.tester.currentToken)
	}

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		c.tester.t.Fatalf("❌ Batch request failed: %v", err)
	}
	defer httpResp.Body.Close()

	respBody, _ := io.ReadAll(httpResp.Body)

	// Parse batch response using intermediate struct
	var rawBatchResponses []struct {
		Data       json.RawMessage        `json:"data"`
		Errors     []*types.GraphQLError  `json:"errors"`
		Extensions map[string]interface{} `json:"extensions"`
	}

	if err := json.Unmarshal(respBody, &rawBatchResponses); err != nil {
		c.tester.t.Fatalf("❌ Failed to parse batch response: %v", err)
	}

	responses := make([]*Response, len(rawBatchResponses))
	for i, raw := range rawBatchResponses {
		var data interface{}
		if raw.Data != nil {
			json.Unmarshal(raw.Data, &data)
		}
		responses[i] = &Response{
			tester:     c.tester,
			statusCode: httpResp.StatusCode,
			data:       data,
			errors:     raw.Errors,
			extensions: raw.Extensions,
		}
	}

	return responses
}

/**
 * Upload sends a GraphQL mutation with file upload support.
 *
 * Implements the GraphQL multipart request specification for file uploads.
 * This handles the multipart/form-data encoding required for file transfers.
 *
 * Parameters:
 *   query     - The GraphQL mutation containing file upload
 *   variables - Mutation variables (use null for file variables)
 *   files     - Map of variable names to file readers
 *
 * Returns:
 *   *Response wrapping the GraphQL response
 *
 * Example:
 *   file, _ := os.Open("test.jpg")
 *   defer file.Close()
 *
 *   client.Upload(`
 *       mutation UploadAvatar($file: Upload!) {
 *           uploadAvatar(file: $file) {
 *               url
 *           }
 *       }
 *   `, map[string]interface{}{
 *       "file": nil, // Will be replaced by actual file
 *   }, map[string]io.Reader{
 *       "file": file,
 *   })
 */
func (c *GraphQLClient) Upload(
	query string,
	variables map[string]interface{},
	files map[string]io.Reader,
) *Response {
	// Create multipart writer
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Write the operations part
	operations := map[string]interface{}{
		"query":     query,
		"variables": variables,
	}
	operationsJSON, _ := json.Marshal(operations)
	writer.WriteField("operations", string(operationsJSON))

	// Write the map part (maps file variables to file indices)
	fileMap := make(map[string][]string)
	i := 0
	for varName := range files {
		fileMap[fmt.Sprintf("%d", i)] = []string{fmt.Sprintf("variables.%s", varName)}
		i++
	}
	fileMapJSON, _ := json.Marshal(fileMap)
	writer.WriteField("map", string(fileMapJSON))

	// Write the file parts
	i = 0
	for varName, reader := range files {
		part, _ := writer.CreateFormFile(fmt.Sprintf("%d", i), varName)
		io.Copy(part, reader)
		i++
	}

	writer.Close()

	// Send the multipart request
	url := c.baseURL + c.tester.config.Endpoint
	httpReq, _ := http.NewRequestWithContext(c.tester.ctx, "POST", url, &body)
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())

	if c.tester.currentToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.tester.currentToken)
	}

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		c.tester.t.Fatalf("❌ Upload request failed: %v", err)
	}
	defer httpResp.Body.Close()

	respBody, _ := io.ReadAll(httpResp.Body)

	// Parse response using intermediate struct
	var rawResponse struct {
		Data       json.RawMessage        `json:"data"`
		Errors     []*types.GraphQLError  `json:"errors"`
		Extensions map[string]interface{} `json:"extensions"`
	}

	json.Unmarshal(respBody, &rawResponse)

	var data interface{}
	if rawResponse.Data != nil {
		json.Unmarshal(rawResponse.Data, &data)
	}

	return &Response{
		tester:     c.tester,
		statusCode: httpResp.StatusCode,
		rawBody:    respBody,
		data:       data,
		errors:     rawResponse.Errors,
		extensions: rawResponse.Extensions,
	}
}

// ============================================================================
// Internal Methods
// ============================================================================

/**
 * doRequest is the internal method that handles the complete request lifecycle.
 *
 * This method:
 * 1. Constructs the GraphQL request payload
 * 2. Sets up HTTP headers (auth, tenant, custom)
 * 3. Executes the HTTP request
 * 4. Parses and returns the response
 *
 * Parameters:
 *   query         - The GraphQL query string
 *   variables     - Query variables
 *   headers       - Additional HTTP headers
 *   operationName - Named operation to execute (empty string if none)
 *
 * Returns:
 *   *Response with the parsed GraphQL response
 */
func (c *GraphQLClient) doRequest(
	query string,
	variables map[string]interface{},
	headers map[string]string,
	operationName string,
) *Response {
	// Construct the GraphQL request payload
	request := GraphQLRequest{
		Query:     query,
		Variables: variables,
	}

	if operationName != "" {
		request.OperationName = operationName
	}

	// Marshal the request to JSON
	body, err := json.Marshal(request)
	if err != nil {
		c.tester.t.Fatalf("❌ Failed to marshal GraphQL request: %v", err)
	}

	// Build the URL
	url := c.baseURL + c.tester.config.Endpoint

	// Create the HTTP request
	httpReq, err := http.NewRequestWithContext(
		c.tester.ctx,
		"POST",
		url,
		bytes.NewBuffer(body),
	)
	if err != nil {
		c.tester.t.Fatalf("❌ Failed to create HTTP request: %v", err)
	}

	// Set default headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	// Inject authentication token if the tester has one
	if c.tester.currentToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.tester.currentToken)
		if c.tester.config.Debug() {
			prefixLen := len(c.tester.currentToken)
			if prefixLen > 10 {
				prefixLen = 10
			}
			c.tester.t.Logf("🔑 Using auth token: %s...", c.tester.currentToken[:prefixLen])
		}
	}

	// Add tenant header if multi-tenancy is enabled and a tenant is selected
	if c.tester.config.Tenancy != nil && c.tester.config.Tenancy.Enabled && c.tester.tenant != nil {
		headerName := c.tester.config.Tenancy.HeaderName
		httpReq.Header.Set(headerName, c.tester.tenant.ID)
		if c.tester.config.Debug() {
			c.tester.t.Logf("🏢 Using tenant: %s", c.tester.tenant.ID)
		}
	}

	// Apply custom headers
	for key, value := range headers {
		httpReq.Header.Set(key, value)
	}

	// Log the request in debug mode
	if c.tester.config.Debug() {
		c.logRequest(httpReq, query, variables)
	}

	// Execute the HTTP request
	startTime := time.Now()
	httpResp, err := c.httpClient.Do(httpReq)
	elapsed := time.Since(startTime)

	if err != nil {
		c.tester.t.Fatalf("❌ HTTP request failed after %v: %v", elapsed, err)
	}
	defer httpResp.Body.Close()

	// Read the response body
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		c.tester.t.Fatalf("❌ Failed to read response body: %v", err)
	}

	// Log the response in debug mode
	if c.tester.config.Debug() {
		c.logResponse(httpResp, respBody, elapsed)
	}

	// Parse the GraphQL response using an intermediate struct with EXPORTED fields
	// This is necessary because Response.data is private and json.Unmarshal
	// can only set exported (uppercase) fields
	var rawResponse struct {
		Data       json.RawMessage        `json:"data"`
		Errors     []*types.GraphQLError  `json:"errors"`
		Extensions map[string]interface{} `json:"extensions"`
	}

	if err := json.Unmarshal(respBody, &rawResponse); err != nil {
		// If parsing fails, create a response with the raw body for debugging
		c.tester.t.Logf("⚠️  Failed to parse response as GraphQL: %v", err)
		c.tester.t.Logf("   Raw response: %s", string(respBody))

		return &Response{
			tester:     c.tester,
			statusCode: httpResp.StatusCode,
			rawBody:    respBody,
			elapsed:    elapsed,
			errors: []*types.GraphQLError{
				{
					Message: fmt.Sprintf("Failed to parse response: %v", err),
					Extensions: map[string]interface{}{
						"raw_body": string(respBody),
					},
				},
			},
		}
	}

	// Parse the inner data field into an interface{}
	var data interface{}
	if rawResponse.Data != nil {
		if err := json.Unmarshal(rawResponse.Data, &data); err != nil {
			c.tester.t.Logf("⚠️  Failed to parse data field: %v", err)
		}
	}

	// Create the properly parsed response
	return &Response{
		tester:     c.tester,
		statusCode: httpResp.StatusCode,
		rawBody:    respBody,
		elapsed:    elapsed,
		data:       data,
		errors:     rawResponse.Errors,
		extensions: rawResponse.Extensions,
	}
}

// ============================================================================
// Logging Methods
// ============================================================================

/**
 * logRequest logs the GraphQL request details in debug mode.
 *
 * This provides visibility into:
 * - The full query being sent
 * - Variables being passed
 * - HTTP headers being sent
 *
 * Parameters:
 *   req       - The HTTP request
 *   query     - The GraphQL query
 *   variables - The query variables
 */
func (c *GraphQLClient) logRequest(req *http.Request, query string, variables map[string]interface{}) {
	c.tester.t.Logf("📤 GraphQL Request:")
	c.tester.t.Logf("   URL: %s", req.URL.String())
	c.tester.t.Logf("   Headers: %v", req.Header)

	// Truncate long queries for readability
	if len(query) > 500 {
		c.tester.t.Logf("   Query: %s... (truncated, total %d chars)", query[:500], len(query))
	} else {
		c.tester.t.Logf("   Query: %s", strings.TrimSpace(query))
	}

	if len(variables) > 0 {
		varsJSON, _ := json.MarshalIndent(variables, "   ", "  ")
		c.tester.t.Logf("   Variables: %s", string(varsJSON))
	}
}

/**
 * logResponse logs the GraphQL response details in debug mode.
 *
 * Provides visibility into:
 * - HTTP status code
 * - Response time
 * - Response body (truncated for large responses)
 * - GraphQL errors if present
 *
 * Parameters:
 *   resp     - The HTTP response
 *   body     - The response body bytes
 *   elapsed  - Time taken for the request
 */
func (c *GraphQLClient) logResponse(resp *http.Response, body []byte, elapsed time.Duration) {
	c.tester.t.Logf("📥 GraphQL Response: (took %v)", elapsed)
	c.tester.t.Logf("   Status: %d %s", resp.StatusCode, resp.Status)

	// Truncate long responses for readability
	if len(body) > 1000 {
		c.tester.t.Logf("   Body: %s... (truncated, total %d bytes)",
			string(body[:1000]), len(body))
	} else {
		// Pretty print JSON if possible
		var prettyJSON bytes.Buffer
		if err := json.Indent(&prettyJSON, body, "   ", "  "); err == nil {
			c.tester.t.Logf("   Body:\n%s", prettyJSON.String())
		} else {
			c.tester.t.Logf("   Body: %s", string(body))
		}
	}
}

// ============================================================================
// Types
// ============================================================================

/**
 * GraphQLRequest represents the structure of a GraphQL HTTP request body.
 *
 * This follows the standard GraphQL over HTTP specification:
 * https://graphql.org/learn/serving-over-http/
 */
type GraphQLRequest struct {
	// Query is the GraphQL query or mutation string.
	Query string `json:"query"`

	// Variables contains the variables for the query.
	// Optional: Can be omitted if the query has no variables.
	Variables map[string]interface{} `json:"variables,omitempty"`

	// OperationName specifies which operation to execute when the document
	// contains multiple named operations.
	// Optional: Required only for multi-operation documents.
	OperationName string `json:"operationName,omitempty"`
}
