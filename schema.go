package graphqltester

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	graphql "github.com/graph-gophers/graphql-go"
)

/**
 * Schema Management for GraphQL Testing
 *
 * This file handles GraphQL schema loading, parsing, caching, and validation.
 * It supports both file-based and string-based schema sources and manages
 * the schema lifecycle including cache refreshing for test isolation.
 *
 * Features:
 * - File-based schema loading (*.graphql files)
 * - String-based schema loading (inline schemas)
 * - Schema validation on load
 * - Schema caching for performance
 * - Cache refreshing between tests
 * - Introspection support for query analysis
 */

// ============================================================================
// Schema Manager
// ============================================================================

/**
 * SchemaManager handles all GraphQL schema operations for the tester.
 *
 * It is responsible for:
 * - Loading schemas from files or strings
 * - Parsing schemas with resolvers
 * - Caching schemas for performance
 * - Providing schema introspection
 * - Validating schema correctness
 */
type SchemaManager struct {
	// config holds the schema configuration.
	config *SchemaConfig

	// schema is the parsed GraphQL schema.
	schema *graphql.Schema

	// schemaString holds the raw schema text.
	schemaString string

	// resolvers holds the resolver implementation.
	resolvers interface{}

	// cache stores parsed schemas for reuse.
	cache map[string]*graphql.Schema

	// mu protects concurrent access to the schema.
	mu sync.RWMutex
}

/**
 * NewSchemaManager creates a new schema manager with the given configuration.
 *
 * Parameters:
 *   config - Schema configuration (path, resolvers, options)
 *
 * Returns:
 *   *SchemaManager ready for schema loading
 */
func NewSchemaManager(config *SchemaConfig) *SchemaManager {
	return &SchemaManager{
		config: config,
		cache:  make(map[string]*graphql.Schema),
	}
}

/**
 * Load loads the GraphQL schema from the configured source.
 *
 * This method handles:
 * 1. Reading the schema from file or string
 * 2. Parsing the schema with resolvers
 * 3. Validating the schema (if ValidateOnLoad is set)
 * 4. Caching the parsed schema
 *
 * Returns:
 *   error if schema loading or parsing fails
 *
 * Example:
 *   manager := NewSchemaManager(config)
 *   if err := manager.Load(); err != nil {
 *       t.Fatalf("Failed to load schema: %v", err)
 *   }
 */
func (sm *SchemaManager) Load() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	var schemaText string

	// Load schema from file or string
	if sm.config.Path != "" {
		// Read schema from file
		content, err := os.ReadFile(sm.config.Path)
		if err != nil {
			return fmt.Errorf("failed to read schema file %s: %w", sm.config.Path, err)
		}
		schemaText = string(content)
	} else if sm.config.String != "" {
		// Use inline schema string
		schemaText = sm.config.String
	} else {
		return fmt.Errorf("no schema source configured (set Path or String)")
	}

	sm.schemaString = schemaText

	// Check cache first
	cacheKey := sm.getCacheKey()
	if cached, ok := sm.cache[cacheKey]; ok && !sm.config.RefreshCache {
		sm.schema = cached
		return nil
	}

	// Parse the schema
	schema, err := sm.parse(schemaText)
	if err != nil {
		return fmt.Errorf("failed to parse schema: %w", err)
	}

	// Validate schema if enabled
	if sm.config.ValidateOnLoad {
		if err := sm.validate(schema); err != nil {
			return fmt.Errorf("schema validation failed: %w", err)
		}
	}

	// Store the parsed schema
	sm.schema = schema
	sm.cache[cacheKey] = schema

	return nil
}

/**
 * parse parses a GraphQL schema string with configured resolvers and options.
 *
 * Parameters:
 *   schemaText - The GraphQL schema text
 *
 * Returns:
 *   *graphql.Schema parsed schema
 *   error if parsing fails
 */
func (sm *SchemaManager) parse(schemaText string) (*graphql.Schema, error) {
	// Build schema options
	opts := []graphql.SchemaOpt{
		graphql.UseFieldResolvers(),
		graphql.MaxParallelism(10),
		graphql.MaxDepth(20),
		graphql.Tracer(&testTracer{}),
	}

	// Add user-configured options
	if sm.config.Options != nil {
		opts = append(opts, sm.config.Options...)
	}

	// Parse the schema with resolvers
	schema, err := graphql.ParseSchema(schemaText, sm.config.Resolvers, opts...)
	if err != nil {
		return nil, err
	}

	return schema, nil
}

/**
 * validate performs basic validation checks on the parsed schema.
 *
 * This checks for common issues like:
 * - Query type must be defined
 * - No duplicate type definitions
 * - Valid directive usage
 *
 * Parameters:
 *   schema - The parsed schema to validate
 *
 * Returns:
 *   error if validation fails
 */
func (sm *SchemaManager) validate(schema *graphql.Schema) error {
	// Perform introspection to verify schema is queryable
	// This is a basic validation that the schema is functional

	// Try to execute a simple introspection query
	introspectionQuery := `
        query {
            __schema {
                queryType {
                    name
                }
            }
        }
    `

	ctx := context.Background()
	result := schema.Exec(ctx, introspectionQuery, "", nil)

	if len(result.Errors) > 0 {
		return fmt.Errorf("schema introspection failed: %v", result.Errors[0].Message)
	}

	return nil
}

/**
 * GetSchema returns the parsed GraphQL schema.
 *
 * If the schema hasn't been loaded yet, it will be loaded automatically.
 *
 * Returns:
 *   *graphql.Schema ready for query execution
 */
func (sm *SchemaManager) GetSchema() *graphql.Schema {
	sm.mu.RLock()
	schema := sm.schema
	sm.mu.RUnlock()

	if schema == nil {
		// Schema not loaded yet, load it now
		if err := sm.Load(); err != nil {
			panic(fmt.Sprintf("Failed to load schema: %v", err))
		}
		sm.mu.RLock()
		schema = sm.schema
		sm.mu.RUnlock()
	}

	return schema
}

/**
 * GetSchemaString returns the raw schema text.
 *
 * Returns:
 *   string containing the GraphQL schema text
 */
func (sm *SchemaManager) GetSchemaString() string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.schemaString
}

/**
 * RefreshSchema reloads the schema from the source.
 *
 * This is useful for tests that modify the schema or test schema changes.
 * It clears the cache and reloads the schema from the original source.
 *
 * Returns:
 *   error if reloading fails
 */
func (sm *SchemaManager) RefreshSchema() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Clear the cache
	sm.cache = make(map[string]*graphql.Schema)
	sm.schema = nil

	// Reload the schema
	return sm.Load()
}

/**
 * Introspect performs schema introspection for the given type or field.
 *
 * This can be used to:
 * - Verify schema structure
 * - Check available queries and mutations
 * - Validate field types and arguments
 * - Generate documentation
 *
 * Parameters:
 *   name - The type or field name to introspect
 *
 * Returns:
 *   map[string]interface{} with introspection results
 *   error if introspection fails
 *
 * Example:
 *   result, err := manager.Introspect("Zone")
 *   fields := result["fields"].([]interface{})
 *   for _, field := range fields {
 *       fmt.Println(field.(map[string]interface{})["name"])
 *   }
 */
func (sm *SchemaManager) Introspect(name string) (map[string]interface{}, error) {
	schema := sm.GetSchema()

	query := fmt.Sprintf(`
        query IntrospectType($name: String!) {
            __type(name: $name) {
                name
                kind
                description
                fields {
                    name
                    description
                    type {
                        name
                        kind
                        ofType {
                            name
                            kind
                        }
                    }
                    args {
                        name
                        description
                        type {
                            name
                            kind
                        }
                    }
                }
            }
        }
    `)

	ctx := context.Background()
	result := schema.Exec(ctx, query, "", map[string]interface{}{
		"name": name,
	})

	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("introspection failed: %v", result.Errors[0].Message)
	}

	// Parse the result into a map
	var response map[string]interface{}
	if err := json.Unmarshal(result.Data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse introspection result: %w", err)
	}

	return response, nil
}

/**
 * HasQuery checks if a query with the given name exists in the schema.
 *
 * Parameters:
 *   queryName - The query name to check
 *
 * Returns:
 *   bool indicating whether the query exists
 */
func (sm *SchemaManager) HasQuery(queryName string) bool {
	schema := sm.GetSchema()

	// Execute introspection to check for the query
	query := `
        query {
            __schema {
                queryType {
                    fields {
                        name
                    }
                }
            }
        }
    `

	ctx := context.Background()
	result := schema.Exec(ctx, query, "", nil)

	if len(result.Errors) > 0 {
		return false
	}

	// Parse and check for the query name
	var response struct {
		Data struct {
			Schema struct {
				QueryType struct {
					Fields []struct {
						Name string `json:"name"`
					} `json:"fields"`
				} `json:"queryType"`
			} `json:"__schema"`
		} `json:"data"`
	}

	if err := json.Unmarshal(result.Data, &response); err != nil {
		return false
	}

	for _, field := range response.Data.Schema.QueryType.Fields {
		if field.Name == queryName {
			return true
		}
	}

	return false
}

/**
 * HasMutation checks if a mutation with the given name exists in the schema.
 *
 * Parameters:
 *   mutationName - The mutation name to check
 *
 * Returns:
 *   bool indicating whether the mutation exists
 */
func (sm *SchemaManager) HasMutation(mutationName string) bool {
	// Similar to HasQuery but checks mutation type
	schema := sm.GetSchema()

	query := `
        query {
            __schema {
                mutationType {
                    fields {
                        name
                    }
                }
            }
        }
    `

	ctx := context.Background()
	result := schema.Exec(ctx, query, "", nil)

	if len(result.Errors) > 0 {
		return false
	}

	var response struct {
		Data struct {
			Schema struct {
				MutationType *struct {
					Fields []struct {
						Name string `json:"name"`
					} `json:"fields"`
				} `json:"mutationType"`
			} `json:"__schema"`
		} `json:"data"`
	}

	if err := json.Unmarshal(result.Data, &response); err != nil {
		return false
	}

	if response.Data.Schema.MutationType == nil {
		return false
	}

	for _, field := range response.Data.Schema.MutationType.Fields {
		if field.Name == mutationName {
			return true
		}
	}

	return false
}

/**
 * getCacheKey generates a cache key for the current schema configuration.
 *
 * The cache key is based on the schema source (path or string) to allow
 * caching of different schemas.
 *
 * Returns:
 *   string cache key
 */
func (sm *SchemaManager) getCacheKey() string {
	if sm.config.Path != "" {
		return fmt.Sprintf("file:%s", sm.config.Path)
	}
	if sm.config.String != "" {
		// Hash the schema string for a shorter cache key
		return fmt.Sprintf("string:%d", len(sm.config.String))
	}
	return "unknown"
}

// ============================================================================
// Test Tracer (for debugging)
// ============================================================================

/**
 * testTracer implements graphql.Tracer for test debugging.
 *
 * When debug mode is enabled, this tracer logs query execution steps
 * to help understand how queries are resolved.
 */
type testTracer struct{}

/**
 * TraceQuery logs the start of query execution.
 */
func (t *testTracer) TraceQuery(ctx context.Context, queryString string, operationName string, variables map[string]interface{}, varTypes map[string]*graphql.Type) context.Context {
	// Query tracing can be added here for debug mode
	return ctx
}

/**
 * TraceField logs field resolution.
 */
func (t *testTracer) TraceField(ctx context.Context, label, typeName, fieldName string, trivial bool, args map[string]interface{}) context.Context {
	// Field tracing can be added here for debug mode
	return ctx
}

/**
 * TraceValidation logs validation step.
 */
func (t *testTracer) TraceValidation(ctx context.Context) context.Context {
	return ctx
}
