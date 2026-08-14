package graphqltester

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mwangaben/graphqltester/types"
)

// ============================================================================
// Subscription Client
// ============================================================================

/**
 * SubscriptionClient handles WebSocket connections for GraphQL subscriptions.
 */
type SubscriptionClient struct {
	tester    *Tester
	conn      *websocket.Conn
	url       string
	ctx       context.Context
	cancel    context.CancelFunc
	connected bool

	mu            sync.RWMutex
	subscriptions map[string]*Subscription
	messageLog    []*SubscriptionMessage
}

/**
 * Subscription represents an active GraphQL subscription.
 */
type Subscription struct {
	ID          string
	Query       string
	Variables   map[string]interface{}
	Active      bool
	Messages    []*SubscriptionMessage
	MessageChan chan *SubscriptionMessage
	ErrorChan   chan error
	ctx         context.Context
	cancel      context.CancelFunc
}

/**
 * SubscriptionMessage represents a message over WebSocket.
 */
type SubscriptionMessage struct {
	ID      string                `json:"id,omitempty"`
	Type    string                `json:"type"`
	Payload json.RawMessage       `json:"payload,omitempty"`
	Data    interface{}           `json:"-"`
	Errors  []*types.GraphQLError `json:"-"`
}

// ============================================================================
// Tester Subscription Methods
// ============================================================================

/**
 * Subscribe creates a subscription client and subscription in one call.
 */
func (tester *Tester) Subscribe(query string, variables ...map[string]interface{}) (*SubscriptionClient, *Subscription) {
	client := tester.NewSubscriptionClient()

	vars := map[string]interface{}{}
	if len(variables) > 0 {
		vars = variables[0]
	}

	sub := client.Subscribe(query, vars)
	client.Connect()

	return client, sub
}

/**
 * NewSubscriptionClient creates a new subscription client.
 */
func (tester *Tester) NewSubscriptionClient() *SubscriptionClient {
	ctx, cancel := context.WithCancel(tester.ctx)

	// Log the server URL for debugging
	tester.t.Logf("🔍 Server URL: %s", tester.server.URL)
	tester.t.Logf("🔍 Endpoint: %s", tester.config.Endpoint)

	// Convert HTTP to WebSocket URL
	wsURL := "ws" + tester.server.URL[4:] + tester.config.Endpoint
	tester.t.Logf("🔍 WebSocket URL: %s", wsURL)

	return &SubscriptionClient{
		tester:        tester,
		url:           wsURL,
		ctx:           ctx,
		cancel:        cancel,
		subscriptions: make(map[string]*Subscription),
		messageLog:    make([]*SubscriptionMessage, 0),
	}
}

/**
 * AssertSubscription creates assertion helpers for a subscription.
 */
func (tester *Tester) AssertSubscription(sub *Subscription) *SubscriptionAssertions {
	return &SubscriptionAssertions{
		Subscription: sub,
		Tester:       tester,
		Timeout:      5 * time.Second,
		cachedMsg:    nil,
	}
}

/**
 * AssertDataPathContains asserts a value contains expected data at a path.
 * Uses cached message for multiple assertions.
 */
func (sa *SubscriptionAssertions) AssertDataPathContains(path string, expected interface{}) *SubscriptionAssertions {
	var msg *SubscriptionMessage
	if sa.cachedMsg != nil {
		msg = sa.cachedMsg
	} else {
		msg = sa.Subscription.WaitForMessage(sa.Timeout)
		if msg != nil {
			sa.cachedMsg = msg
		}
	}

	if msg == nil || msg.Data == nil {
		sa.Tester.t.Errorf("❌ Expected data but got nil")
		return sa
	}

	actual := subscriptionGetJSONPath(msg.Data, path)
	if !subscriptionIsSubset(expected, actual) {
		sa.Tester.t.Errorf("❌ Path '%s': expected subset %v, got %v", path, expected, actual)
	}

	return sa
}

// ============================================================================
// SubscriptionClient Methods
// ============================================================================

/**
 * Subscribe registers a new subscription.
 */
func (sc *SubscriptionClient) Subscribe(query string, vars map[string]interface{}) *Subscription {
	sub := &Subscription{
		ID:          fmt.Sprintf("sub_%d", time.Now().UnixNano()),
		Query:       query,
		Variables:   vars,
		Active:      true,
		Messages:    make([]*SubscriptionMessage, 0),
		MessageChan: make(chan *SubscriptionMessage, 100),
		ErrorChan:   make(chan error, 10),
	}
	sub.ctx, sub.cancel = context.WithCancel(sc.ctx)

	sc.mu.Lock()
	sc.subscriptions[sub.ID] = sub
	sc.mu.Unlock()

	if sc.connected {
		sc.sendStart(sub)
	}

	return sub
}

/**
 * Connect establishes the WebSocket connection.
 */
func (sc *SubscriptionClient) Connect() {
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
		Subprotocols:     []string{"graphql-ws"}, // ← ADD THIS
	}
	sc.tester.t.Logf("🔌 Attempting WebSocket connection to: %s", sc.url)
	conn, resp, err := dialer.Dial(sc.url, nil)
	sc.tester.t.Logf("❌ WebSocket dial failed: %v", err)
	if resp != nil {
		sc.tester.t.Logf("   HTTP Status: %d", resp.StatusCode)
		sc.tester.t.Logf("   Headers: %v", resp.Header)
		body, _ := io.ReadAll(resp.Body)
		sc.tester.t.Logf("   Body: %s", string(body))
	}
	if err != nil {
		sc.tester.t.Fatalf("❌ Failed to connect WebSocket: %v", err)
	}

	sc.conn = conn
	sc.connected = true

	// Send connection init
	sc.sendMessage(&SubscriptionMessage{Type: "connection_init"})

	// Start all pending subscriptions
	sc.mu.RLock()
	for _, sub := range sc.subscriptions {
		if sub.Active {
			sc.sendStart(sub)
		}
	}
	sc.mu.RUnlock()

	// Start reading messages
	go sc.readPump()
}

/**
 * Disconnect closes the WebSocket connection.
 */
func (sc *SubscriptionClient) Disconnect() {
	sc.cancel()
	if sc.conn != nil {
		sc.conn.Close()
		sc.connected = false
	}
}

/**
 * sendStart sends a start message for a subscription.
 */
func (sc *SubscriptionClient) sendStart(sub *Subscription) {
	payload := map[string]interface{}{
		"query":     sub.Query,
		"variables": sub.Variables,
	}
	payloadJSON, _ := json.Marshal(payload)

	sc.sendMessage(&SubscriptionMessage{
		ID:      sub.ID,
		Type:    "start",
		Payload: payloadJSON,
	})
}

/**
 * sendMessage sends a message over WebSocket.
 */
func (sc *SubscriptionClient) sendMessage(msg *SubscriptionMessage) {
	if sc.conn == nil {
		return
	}

	data, _ := json.Marshal(msg)
	sc.conn.WriteMessage(websocket.TextMessage, data)
}

/**
 * readPump reads messages from WebSocket.
 */
func (sc *SubscriptionClient) readPump() {
	defer sc.Disconnect()

	for {
		select {
		case <-sc.ctx.Done():
			return
		default:
		}

		_, message, err := sc.conn.ReadMessage()
		if err != nil {
			sc.tester.t.Logf("⚠️  WebSocket read error: %v", err)
			return
		}

		sc.tester.t.Logf("📩 Raw message: %s", string(message))

		var msg SubscriptionMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			sc.tester.t.Logf("⚠️  Failed to parse message: %v", err)
			continue
		}

		switch msg.Type {
		case "connection_ack":
			sc.tester.t.Logf("🔗 WebSocket connection acknowledged")

		case "data":
			// Parse the payload
			var payload struct {
				Data   json.RawMessage       `json:"data"`
				Errors []*types.GraphQLError `json:"errors"`
			}
			if err := json.Unmarshal(msg.Payload, &payload); err != nil {
				sc.tester.t.Logf("⚠️  Failed to parse data payload: %v", err)
				continue
			}

			// Unmarshal the data field
			var data interface{}
			if len(payload.Data) > 0 {
				if err := json.Unmarshal(payload.Data, &data); err != nil {
					sc.tester.t.Logf("⚠️  Failed to unmarshal data: %v", err)
					continue
				}
			}

			msg.Data = data
			msg.Errors = payload.Errors

			sc.tester.t.Logf("📊 Parsed data: %v", data)

			sc.mu.RLock()
			if sub, ok := sc.subscriptions[msg.ID]; ok && sub.Active {
				sub.Messages = append(sub.Messages, &msg)
				select {
				case sub.MessageChan <- &msg:
				default:
				}
			}
			sc.mu.RUnlock()

		case "complete":
			sc.mu.Lock()
			if sub, ok := sc.subscriptions[msg.ID]; ok {
				sub.Active = false
			}
			sc.mu.Unlock()
		}
	}
}

// ============================================================================
// Subscription Methods
// ============================================================================

/**
 * WaitForMessage waits for the next message with timeout.
 */
func (sub *Subscription) WaitForMessage(timeout time.Duration) *SubscriptionMessage {
	select {
	case msg := <-sub.MessageChan:
		return msg
	case err := <-sub.ErrorChan:
		return &SubscriptionMessage{
			Type:   "error",
			Errors: []*types.GraphQLError{{Message: err.Error()}},
		}
	case <-time.After(timeout):
		return &SubscriptionMessage{
			Type:   "error",
			Errors: []*types.GraphQLError{{Message: "timeout waiting for message"}},
		}
	case <-sub.ctx.Done():
		return nil
	}
}

/**
 * WaitForMessages waits for multiple messages.
 */
func (sub *Subscription) WaitForMessages(count int, timeout time.Duration) []*SubscriptionMessage {
	messages := make([]*SubscriptionMessage, 0, count)
	deadline := time.After(timeout)

	for i := 0; i < count; i++ {
		select {
		case msg := <-sub.MessageChan:
			messages = append(messages, msg)
		case <-deadline:
			return messages
		case <-sub.ctx.Done():
			return messages
		}
	}

	return messages
}

/**
 * ExpectMessage asserts the next message matches expected data.
 */

/**
 * ExpectMessage asserts the next message exactly matches expected data.
 */
func (sub *Subscription) ExpectMessage(expected interface{}, timeout time.Duration) *Subscription {
	msg := sub.WaitForMessage(timeout)
	if msg == nil || msg.Data == nil {
		sub.ErrorChan <- fmt.Errorf("expected message but got nil")
		return sub
	}

	expectedJSON, _ := json.Marshal(expected)
	actualJSON, _ := json.Marshal(msg.Data)

	var expectedNorm, actualNorm interface{}
	json.Unmarshal(expectedJSON, &expectedNorm)
	json.Unmarshal(actualJSON, &actualNorm)

	if !subscriptionDeepEqual(expectedNorm, actualNorm) {
		sub.ErrorChan <- fmt.Errorf("message mismatch:\nExpected: %s\nActual: %s", expectedJSON, actualJSON)
	}

	return sub
}

/**
 * ExpectMessageContains asserts the next message contains the expected data as a subset.
 * Extra fields in the actual data are ignored.
 */
func (sub *Subscription) ExpectMessageContains(expected interface{}, timeout time.Duration) *Subscription {
	msg := sub.WaitForMessage(timeout)
	if msg == nil || msg.Data == nil {
		sub.ErrorChan <- fmt.Errorf("expected message but got nil")
		return sub
	}

	expectedJSON, _ := json.Marshal(expected)
	actualJSON, _ := json.Marshal(msg.Data)

	var expectedNorm, actualNorm interface{}
	json.Unmarshal(expectedJSON, &expectedNorm)
	json.Unmarshal(actualJSON, &actualNorm)

	if !subscriptionIsSubset(expectedNorm, actualNorm) {
		sub.ErrorChan <- fmt.Errorf("message subset mismatch:\nExpected subset: %s\nActual: %s", expectedJSON, actualJSON)
	}

	return sub
}

/**
 * ExpectMessageExact is an alias for ExpectMessage (exact match).
 */
func (sub *Subscription) ExpectMessageExact(expected interface{}, timeout time.Duration) *Subscription {
	return sub.ExpectMessage(expected, timeout)
}

/**
 * subscriptionIsSubset checks if expected is a subset of actual.
 * Recursively compares maps and slices.
 */
func subscriptionIsSubset(expected, actual interface{}) bool {
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
			if !subscriptionIsSubset(expectedVal, actualVal) {
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
			if !subscriptionIsSubset(expectedVal, actualSlice[i]) {
				return false
			}
		}
		return true
	}

	return subscriptionDeepEqual(expected, actual)
}

/**
 * ExpectNoMessage asserts no message is received within timeout.
 */
func (sub *Subscription) ExpectNoMessage(timeout time.Duration) *Subscription {
	select {
	case msg := <-sub.MessageChan:
		sub.ErrorChan <- fmt.Errorf("expected no message but got: %v", msg.Data)
	case <-time.After(timeout):
		// Expected - no message
	case <-sub.ctx.Done():
		// Subscription ended
	}
	return sub
}

/**
 * Stop terminates the subscription.
 */
func (sub *Subscription) Stop() {
	sub.cancel()
	sub.Active = false
}

// ============================================================================
// SubscriptionAssertions
// ============================================================================

/**
 * SubscriptionAssertions provides assertion methods for subscriptions.
 */
type SubscriptionAssertions struct {
	Subscription *Subscription
	Tester       *Tester
	Timeout      time.Duration
	cachedMsg    *SubscriptionMessage // Cache the last received message
}

/**
 * WithTimeout sets the default timeout.
 */
func (sa *SubscriptionAssertions) WithTimeout(timeout time.Duration) *SubscriptionAssertions {
	sa.Timeout = timeout
	return sa
}

/**
 * AssertDataPath asserts a value at a specific path in the data.
 */
func (sa *SubscriptionAssertions) AssertDataPath(path string, expected interface{}) *SubscriptionAssertions {
	// Use cached message if available, otherwise wait for new message
	var msg *SubscriptionMessage
	if sa.cachedMsg != nil {
		msg = sa.cachedMsg
	} else {
		msg = sa.Subscription.WaitForMessage(sa.Timeout)
		if msg != nil {
			sa.cachedMsg = msg
		}
	}

	if msg == nil || msg.Data == nil {
		sa.Tester.t.Errorf("❌ Expected data but got nil")
		return sa
	}

	actual := subscriptionGetJSONPath(msg.Data, path)
	if !subscriptionDeepEqual(actual, expected) {
		sa.Tester.t.Errorf("❌ Path '%s': expected %v, got %v", path, expected, actual)
	}

	return sa
}

/**
 * AssertNoErrors asserts no errors occurred.
 */
func (sa *SubscriptionAssertions) AssertNoErrors() *SubscriptionAssertions {
	select {
	case err := <-sa.Subscription.ErrorChan:
		sa.Tester.t.Errorf("❌ Unexpected error: %v", err)
	default:
		// No errors
	}
	return sa
}

/**
 * AssertActive asserts the subscription is still active.
 */
func (sa *SubscriptionAssertions) AssertActive() *SubscriptionAssertions {
	if !sa.Subscription.Active {
		sa.Tester.t.Errorf("❌ Expected subscription to be active")
	}
	return sa
}

/**
 * AssertClosed asserts the subscription is closed.
 */
func (sa *SubscriptionAssertions) AssertClosed() *SubscriptionAssertions {
	if sa.Subscription.Active {
		sa.Tester.t.Errorf("❌ Expected subscription to be closed")
	}
	return sa
}

/**
 * ClearCache clears the cached message for a new assertion sequence.
 */
func (sa *SubscriptionAssertions) ClearCache() *SubscriptionAssertions {
	sa.cachedMsg = nil
	return sa
}

// ============================================================================
// Helper Functions
// ============================================================================

/**
 * subscriptionDeepEqual performs deep comparison with numeric normalization.
 */
func subscriptionDeepEqual(a, b interface{}) bool {
	a = subscriptionNormalizeNumeric(a)
	b = subscriptionNormalizeNumeric(b)
	return reflect.DeepEqual(a, b)
}

/**
 * subscriptionNormalizeNumeric converts int types to float64 for consistent comparison.
 */
func subscriptionNormalizeNumeric(val interface{}) interface{} {
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
 * subscriptionGetJSONPath navigates a JSON structure using dot notation.
 */
func subscriptionGetJSONPath(data interface{}, path string) interface{} {
	parts := strings.Split(path, ".")
	current := data

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
