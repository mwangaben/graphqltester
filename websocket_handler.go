package graphqltester

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	graphql "github.com/graph-gophers/graphql-go"
)

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	Subprotocols: []string{"graphql-ws"},
}

func createWSHandler(schema *graphql.Schema) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				fmt.Printf("❌ PANIC in WebSocket handler: %v\n", rec)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()

		conn, err := wsUpgrader.Upgrade(w, r, nil)
		if err != nil {
			fmt.Printf("❌ WebSocket upgrade failed: %v\n", err)
			return
		}
		defer conn.Close()

		fmt.Printf("✅ WebSocket connection established\n")

		subscriptions := make(map[string]context.CancelFunc)
		var mu sync.Mutex

		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				fmt.Printf("⚠️  WebSocket read error: %v\n", err)
				mu.Lock()
				for _, cancel := range subscriptions {
					cancel()
				}
				mu.Unlock()
				return
			}

			var msg SubscriptionMessage
			if err := json.Unmarshal(message, &msg); err != nil {
				continue
			}

			switch msg.Type {
			case "connection_init":
				ackJSON, _ := json.Marshal(map[string]string{"type": "connection_ack"})
				conn.WriteMessage(websocket.TextMessage, ackJSON)
				fmt.Printf("✅ Sent connection_ack\n")

			case "start":
				var payload struct {
					Query     string                 `json:"query"`
					Variables map[string]interface{} `json:"variables"`
				}
				json.Unmarshal(msg.Payload, &payload)

				ctx, cancel := context.WithCancel(r.Context())
				mu.Lock()
				subscriptions[msg.ID] = cancel
				mu.Unlock()

				go func(id string, ctx context.Context, query string, vars map[string]interface{}) {
					// schema.Subscribe returns (<-chan any, error)
					ch, err := schema.Subscribe(ctx, query, "", vars)
					if err != nil {
						fmt.Printf("❌ Subscription error: %v\n", err)
						errJSON, _ := json.Marshal(map[string]interface{}{
							"id":   id,
							"type": "error",
							"payload": map[string]interface{}{
								"errors": []map[string]string{{"message": err.Error()}},
							},
						})
						conn.WriteMessage(websocket.TextMessage, errJSON)
						return
					}

					fmt.Printf("✅ Subscription started for %s\n", id)

					for {
						select {
						case <-ctx.Done():
							fmt.Printf("🛑 Subscription %s cancelled\n", id)
							return
						case result, ok := <-ch:
							if !ok {
								fmt.Printf("🔚 Channel closed for %s\n", id)
								completeJSON, _ := json.Marshal(map[string]string{
									"id":   id,
									"type": "complete",
								})
								conn.WriteMessage(websocket.TextMessage, completeJSON)
								return
							}

							// result is *graphql.Response (from chan any)
							resp, ok := result.(*graphql.Response)
							if !ok {
								fmt.Printf("⚠️  Unexpected result type: %T\n", result)
								continue
							}

							if len(resp.Errors) > 0 {
								fmt.Printf("⚠️  Event error: %v\n", resp.Errors)
								errJSON, _ := json.Marshal(map[string]interface{}{
									"id":   id,
									"type": "error",
									"payload": map[string]interface{}{
										"errors": resp.Errors,
									},
								})
								conn.WriteMessage(websocket.TextMessage, errJSON)
								continue
							}

							// resp.Data is json.RawMessage
							fmt.Printf("📤 Sending event data: %s\n", string(resp.Data))
							dataJSON, _ := json.Marshal(map[string]interface{}{
								"id":   id,
								"type": "data",
								"payload": map[string]interface{}{
									"data": resp.Data,
								},
							})
							conn.WriteMessage(websocket.TextMessage, dataJSON)
						}
					}
				}(msg.ID, ctx, payload.Query, payload.Variables)

			case "stop":
				mu.Lock()
				if cancel, ok := subscriptions[msg.ID]; ok {
					cancel()
					delete(subscriptions, msg.ID)
				}
				mu.Unlock()

			case "connection_terminate":
				return
			}
		}
	})
}
