package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/auxdata-ai/mcp-golang/transport"
	"github.com/gin-gonic/gin"
)

// GinTransport implements a stateless HTTP transport for MCP using Gin
type GinTransport struct {
	*baseTransport
}

// NewGinTransport creates a new Gin transport
func NewGinTransport() *GinTransport {
	return &GinTransport{
		baseTransport: newBaseTransport(),
	}
}

// Start implements Transport.Start - no-op for Gin transport as it's handled by Gin
func (t *GinTransport) Start(ctx context.Context) error {
	return nil
}

// Send implements Transport.Send
func (t *GinTransport) Send(ctx context.Context, message *transport.BaseJsonRpcMessage) error {
	key := message.JsonRpcResponse.Id
	responseChannel := t.responseMap[int64(key)]
	if responseChannel == nil {
		return fmt.Errorf("no response channel found for key: %d", key)
	}
	responseChannel <- message
	return nil
}

// Close implements Transport.Close
func (t *GinTransport) Close() error {
	if t.closeHandler != nil {
		t.closeHandler()
	}
	return nil
}

// SetCloseHandler implements Transport.SetCloseHandler
func (t *GinTransport) SetCloseHandler(handler func()) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.closeHandler = handler
}

// SetErrorHandler implements Transport.SetErrorHandler
func (t *GinTransport) SetErrorHandler(handler func(error)) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.errorHandler = handler
}

// SetMessageHandler implements Transport.SetMessageHandler
func (t *GinTransport) SetMessageHandler(handler func(ctx context.Context, message *transport.BaseJsonRpcMessage)) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.messageHandler = handler
}

// Handler returns a Gin handler function that can be used with Gin's router
func (t *GinTransport) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.Background()
		ctx = context.WithValue(ctx, "ginContext", c)
		if c.Request.Method != http.MethodPost {
			c.String(http.StatusMethodNotAllowed, "Only POST method is supported")
			return
		}

		body, err := t.readBody(c.Request.Body)
		if err != nil {
			c.String(http.StatusBadRequest, err.Error())
			return
		}

		response, err := t.handleMessage(ctx, body)
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}

		jsonData, err := json.Marshal(response)
		if err != nil {
			if t.errorHandler != nil {
				t.errorHandler(fmt.Errorf("failed to marshal response: %w", err))
			}
			c.String(http.StatusInternalServerError, "Failed to marshal response")
			return
		}

		c.Data(http.StatusOK, "application/json", jsonData)
	}
}
