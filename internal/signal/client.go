package signal

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/coder/websocket"
)

// Client is a Signal client that connects to the signal-cli-rest-api.
type Client struct {
	addr   string
	number string
	conn   *websocket.Conn
	db     DB
}

// DB is the interface for the database.
type DB interface {
	SaveMessage(msg *Envelope) error
}

// NewClient creates a new Signal client.
func NewClient(addr, number string, db DB) *Client {
	return &Client{
		addr:   addr,
		number: number,
		db:     db,
	}
}

// Listen connects to the WebSocket and listens for messages.
func (c *Client) Listen(ctx context.Context) error {
	wsURL := fmt.Sprintf("ws://%s/v1/receive/%s", c.addr, c.number)
	log.Printf("Connecting to WebSocket: %s", wsURL)

	var err error
	maxRetries := 5
	retryDelay := 5 * time.Second

	for i := 0; i < maxRetries; i++ {
		c.conn, _, err = websocket.Dial(ctx, wsURL, nil)
		if err == nil {
			break // Success
		}
		log.Printf("WebSocket connection failed (attempt %d/%d): %v. Retrying in %v...", i+1, maxRetries, err, retryDelay)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(retryDelay):
		}
	}

	if err != nil {
		return fmt.Errorf("failed to dial websocket after %d retries: %w", maxRetries, err)
	}

	defer c.conn.Close(websocket.StatusInternalError, "internal error")

	for {
		select {
		case <-ctx.Done():
			c.conn.Close(websocket.StatusNormalClosure, "")
			return nil
		default:
			messageType, data, err := c.conn.Read(ctx)
			if err != nil {
				// Handle close errors
				if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
					return nil
				}
				return fmt.Errorf("failed to read message: %w", err)
			}

			if messageType == websocket.MessageText {
				var envelope Envelope
				if err := json.Unmarshal(data, &envelope); err != nil {
					log.Printf("Error unmarshaling message: %v", err)
					continue
				}

				if err := c.db.SaveMessage(&envelope); err != nil {
					log.Printf("Error saving message: %v", err)
					continue
				}

				log.Printf("Saved message from %s", envelope.DisplayName())
			}
		}
	}
}