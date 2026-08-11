package graphqltester

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	graphql "github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/errors"
	"github.com/graph-gophers/graphql-go/introspection"
	"github.com/graph-gophers/graphql-go/trace/tracer"
)

/**
 * Schema Management for GraphQL Testing
 *
 * This file handles GraphQL schema loading, parsing, caching, and validation.
 * It supports both file-based and string-based schema sources and manages
 * the schema lifecycle including cache refreshing for test isolation.
 */

// ============================================================================
// Schema Manager
// ============================================================================

/**
 * SchemaManager handles all GraphQL schema operations for the tester.
 */
type SchemaManager struct {
	config       *SchemaConfig
	schema       *graphql.Schema
	schemaString string
	resolvers    interface{}
	cache        map[string]*graphql.Schema
	mu           sync.RWMutex
}

/**
 * NewSchemaManager creates a new schema manager with the given configuration.
 */
func NewSchemaManager(config *SchemaConfig) *SchemaManager {
	return &SchemaManager{
		config: config,
		cache:  make(map[string]*graphql.Schema),
	}
}

/**
 * Load loads the GraphQL schema from the configured source.
 */
func (sm *SchemaManager) Load() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	var schemaText string

	if sm.config.Path != "" {
		content, err := os.ReadFile(sm.config.Path)
		if err != nil {
			return fmt.Errorf("failed to read schema file %s: %w", sm.config.Path, err)
		}
		schemaText = string(content)
	} else if sm.config.String != "" {
		schemaText = sm.config.String
	} else {
		return fmt.Errorf("no schema source configured (set Path or String)")
	}

	sm.schemaString = schemaText

	cacheKey := sm.getCacheKey()
	if cached, ok := sm.cache[cacheKey]; ok && !sm.config.RefreshCache {
		sm.schema = cached
		return nil
	}

	schema, err := sm.parse(schemaText)
	if err != nil {
		return fmt.Errorf("failed to parse schema: %w", err)
	}

	if sm.config.ValidateOnLoad {
		if err := sm.validate(schema); err != nil {
			return fmt.Errorf("schema validation failed: %w", err)
		}
	}

	sm.schema = schema
	sm.cache[cacheKey] = schema

	return nil
}

/**
 * parse parses a GraphQL schema string with configured resolvers and options.
 */
func (sm *SchemaManager) parse(schemaText string) (*graphql.Schema, error) {
	opts := []graphql.SchemaOpt{
		graphql.UseFieldResolvers(),
		graphql.MaxParallelism(10),
		graphql.MaxDepth(20),
	}

	if sm.config.Options != nil {
		opts = append(opts, sm.config.Options...)
	}

	schema, err := graphql.ParseSchema(schemaText, sm.config.Resolvers, opts...)
	if err != nil {
		return nil, err
	}

	return schema, nil
}

/**
 * validate performs basic validation checks on the parsed schema.
 */
func (sm *SchemaManager) validate(schema *graphql.Schema) error {
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
 */
func (sm *SchemaManager) GetSchema() *graphql.Schema {
	sm.mu.RLock()
	schema := sm.schema
	sm.mu.RUnlock()

	if schema == nil {
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
 */
func (sm *SchemaManager) GetSchemaString() string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.schemaString
}

/**
 * RefreshSchema reloads the schema from the source.
 */
func (sm *SchemaManager) RefreshSchema() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.cache = make(map[string]*graphql.Schema)
	sm.schema = nil

	return sm.Load()
}

/**
 * Introspect performs schema introspection for the given type or field.
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

	var response map[string]interface{}
	if err := json.Unmarshal(result.Data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse introspection result: %w", err)
	}

	return response, nil
}

/**
 * HasQuery checks if a query with the given name exists in the schema.
 */
func (sm *SchemaManager) HasQuery(queryName string) bool {
	schema := sm.GetSchema()

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
 */
func (sm *SchemaManager) HasMutation(mutationName string) bool {
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
 */
func (sm *SchemaManager) getCacheKey() string {
	if sm.config.Path != "" {
		return fmt.Sprintf("file:%s", sm.config.Path)
	}
	if sm.config.String != "" {
		return fmt.Sprintf("string:%d", len(sm.config.String))
	}
	return "unknown"
}

// ============================================================================
// Test Tracer (for debugging)
// ============================================================================

/**
 * testTracer implements the tracer.Tracer interface for test debugging.
 *
 * Based on the actual tracer.go source for v1.9.0:
 *
 *   type Tracer interface {
 *       TraceQuery(ctx context.Context, queryString string, operationName string,
 *           variables map[string]any, varTypes map[string]*introspection.Type) (context.Context, QueryFinishFunc)
 *       TraceField(ctx context.Context, label, typeName, fieldName string, trivial bool,
 *           args map[string]any) (context.Context, FieldFinishFunc)
 *   }
 *
 *   type QueryFinishFunc = func([]*errors.QueryError)
 *   type FieldFinishFunc = func(*errors.QueryError)
 *
 * Note: The Tracer interface has ONLY two methods. TraceValidation is in a
 * separate ValidationTracer interface, which is optional.
 */
type testTracer struct{}

/**
 * TraceQuery logs the start of query execution.
 *
 * Returns a QueryFinishFunc that is called when the query completes.
 */
func (t *testTracer) TraceQuery(
	ctx context.Context,
	queryString string,
	operationName string,
	variables map[string]any,
	varTypes map[string]*introspection.Type,
) (context.Context, tracer.QueryFinishFunc) {
	// Return a finish function that can log query errors
	finishFunc := func(errs []*errors.QueryError) {
		// Query completed - could log errors here for debugging
		if len(errs) > 0 {
			// Debug logging could be added
		}
	}
	return ctx, finishFunc
}

/**
 * TraceField logs field resolution and returns a field finish function.
 *
 * Returns a FieldFinishFunc that is called when the field resolution completes.
 */
func (t *testTracer) TraceField(
	ctx context.Context,
	label, typeName, fieldName string,
	trivial bool,
	args map[string]any,
) (context.Context, tracer.FieldFinishFunc) {
	// Return a finish function that can log field errors
	finishFunc := func(err *errors.QueryError) {
		// Field resolved - could log errors here for debugging
	}
	return ctx, finishFunc
}

// Compile-time check that testTracer implements tracer.Tracer
var _ tracer.Tracer = (*testTracer)(nil)
