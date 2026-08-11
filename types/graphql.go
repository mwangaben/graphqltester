// Package types provides shared types and interfaces used across the graphql-tester package.
// This package has no internal dependencies to prevent cyclic imports.
package types

import "encoding/json"

/**
 * GraphQLRequest represents the structure of a GraphQL HTTP request body.
 *
 * This follows the standard GraphQL over HTTP specification:
 * https://graphql.org/learn/serving-over-http/
 *
 * This type is shared between the main package and assertions to avoid
 * cyclic dependencies.
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

/**
 * GraphQLError represents an error in a GraphQL response.
 *
 * Follows the GraphQL error specification with additional extensions
 * for validation, authentication, and custom error categories.
 *
 * This type is shared between the main package and assertions to avoid
 * cyclic dependencies.
 */
type GraphQLError struct {
	// Message is the human-readable error description.
	Message string `json:"message"`

	// Locations points to the location in the GraphQL document where the error occurred.
	// Optional: May be omitted for non-document errors (e.g., network errors).
	Locations []Location `json:"locations,omitempty"`

	// Path indicates the path in the response data where the error occurred.
	// Example: ["user", "email"] means the error is in data.user.email
	Path []interface{} `json:"path,omitempty"`

	// Extensions contains additional error metadata.
	// Common keys: "category", "validation", "code"
	Extensions map[string]interface{} `json:"extensions,omitempty"`
}

/**
 * Location points to a position in a GraphQL document.
 */
type Location struct {
	// Line is the 1-indexed line number.
	Line int `json:"line"`

	// Column is the 1-indexed column number.
	Column int `json:"column"`
}

/**
 * SubscriptionMessageType represents the type of a WebSocket message
 * in the GraphQL subscription protocol.
 */
type SubscriptionMessageType string

// Subscription message type constants as defined by the protocol.
const (
	// GQLConnectionInit is sent by the client to initialize the connection.
	GQLConnectionInit SubscriptionMessageType = "connection_init"

	// GQLConnectionAck is sent by the server to acknowledge the connection.
	GQLConnectionAck SubscriptionMessageType = "connection_ack"

	// GQLConnectionError is sent by the server on connection error.
	GQLConnectionError SubscriptionMessageType = "connection_error"

	// GQLConnectionKeepAlive is sent by the server as a keep-alive.
	GQLConnectionKeepAlive SubscriptionMessageType = "ka"

	// GQLConnectionTerminate is sent by the client to terminate the connection.
	GQLConnectionTerminate SubscriptionMessageType = "connection_terminate"

	// GQLStart is sent by the client to start a subscription.
	GQLStart SubscriptionMessageType = "start"

	// GQLData is sent by the server with subscription data.
	GQLData SubscriptionMessageType = "data"

	// GQLError is sent by the server on subscription error.
	GQLError SubscriptionMessageType = "error"

	// GQLComplete is sent by the server when a subscription is complete.
	GQLComplete SubscriptionMessageType = "complete"

	// GQLStop is sent by the client to stop a subscription.
	GQLStop SubscriptionMessageType = "stop"
)

/**
 * SubscriptionMessage represents a message exchanged over the WebSocket
 * connection for GraphQL subscriptions.
 */
type SubscriptionMessage struct {
	// ID identifies the subscription operation.
	ID string `json:"id,omitempty"`

	// Type indicates the message type.
	Type SubscriptionMessageType `json:"type"`

	// Payload contains the message payload.
	Payload json.RawMessage `json:"payload,omitempty"`
}
