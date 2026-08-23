package codex

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

const (
	maxAppServerMessageBytes   = 32 << 20
	maxQueuedNotifications     = 128
	maxQueuedNotificationBytes = 8 << 20
)

type rpcClient struct {
	writer            io.Writer
	scan              *bufio.Scanner
	nextID            int64
	notifications     []rpcNotification
	notificationBytes int
}

type rpcMessage struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
	Result json.RawMessage `json:"result"`
	Error  *rpcError       `json:"error"`
}

type rpcNotification struct {
	Method string
	Params json.RawMessage
}

type rpcError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// rpcCallError retains app-server error details only for in-process
// classification. Error deliberately omits Message and Data so remote or
// client-generated text cannot leak through ordinary error reporting.
type rpcCallError struct {
	Code    int
	Message string
	Data    json.RawMessage
}

func (e *rpcCallError) Error() string {
	return fmt.Sprintf("codex app-server JSON-RPC error %d", e.Code)
}

func newRPCClient(reader io.Reader, writer io.Writer) *rpcClient {
	scan := bufio.NewScanner(reader)
	scan.Buffer(make([]byte, 64<<10), maxAppServerMessageBytes)
	return &rpcClient{writer: writer, scan: scan}
}

func (c *rpcClient) call(method string, params any, out any) error {
	c.nextID++
	id := c.nextID

	request := struct {
		ID     int64  `json:"id"`
		Method string `json:"method"`
		Params any    `json:"params"`
	}{ID: id, Method: method, Params: params}

	payload, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("encode app-server request: %w", err)
	}
	payload = append(payload, '\n')
	if _, err := c.writer.Write(payload); err != nil {
		return fmt.Errorf("write app-server request: %w", err)
	}

	for {
		message, err := c.readMessage()
		if err != nil {
			return err
		}
		if message.Method != "" {
			if len(message.ID) == 0 {
				c.queueNotification(rpcNotification{Method: message.Method, Params: message.Params})
			}
			// Notifications and server-initiated requests are expected while
			// app-server is running. This adapter does not opt into request
			// capabilities, so neither should satisfy one of our calls.
			continue
		}
		if len(message.ID) == 0 {
			continue
		}

		var responseID int64
		if err := json.Unmarshal(message.ID, &responseID); err != nil || responseID != id {
			// Ignore responses to unrelated requests. mcp-interop performs its own
			// requests sequentially, but app-server may serve other internal work.
			continue
		}
		if message.Error != nil {
			return &rpcCallError{
				Code:    message.Error.Code,
				Message: message.Error.Message,
				Data:    append(json.RawMessage(nil), message.Error.Data...),
			}
		}
		if out == nil || len(message.Result) == 0 || string(message.Result) == "null" {
			return nil
		}
		if err := json.Unmarshal(message.Result, out); err != nil {
			return errors.New("decode codex app-server response")
		}
		return nil
	}
}

func (c *rpcClient) waitNotification(method string, out any) error {
	for {
		for i, notification := range c.notifications {
			if notification.Method != method {
				continue
			}
			c.notificationBytes -= notificationSize(notification)
			if c.notificationBytes < 0 {
				c.notificationBytes = 0
			}
			c.notifications = append(c.notifications[:i], c.notifications[i+1:]...)
			return decodeNotification(notification, out)
		}

		message, err := c.readMessage()
		if err != nil {
			return err
		}
		if message.Method == "" || len(message.ID) != 0 {
			continue
		}
		notification := rpcNotification{Method: message.Method, Params: message.Params}
		if notification.Method == method {
			return decodeNotification(notification, out)
		}
		c.queueNotification(notification)
	}
}

func (c *rpcClient) readMessage() (rpcMessage, error) {
	for c.scan.Scan() {
		line := c.scan.Bytes()
		if len(line) == 0 {
			continue
		}

		var message rpcMessage
		if err := json.Unmarshal(line, &message); err != nil {
			return rpcMessage{}, errors.New("codex app-server emitted invalid JSON")
		}
		return message, nil
	}

	if err := c.scan.Err(); err != nil {
		return rpcMessage{}, fmt.Errorf("read codex app-server response: %w", err)
	}
	return rpcMessage{}, errors.New("codex app-server closed before responding")
}

func (c *rpcClient) queueNotification(notification rpcNotification) {
	size := notificationSize(notification)
	if size > maxQueuedNotificationBytes {
		return
	}
	for len(c.notifications) > 0 && (len(c.notifications) >= maxQueuedNotifications || c.notificationBytes+size > maxQueuedNotificationBytes) {
		c.notificationBytes -= notificationSize(c.notifications[0])
		c.notifications = c.notifications[1:]
	}
	c.notifications = append(c.notifications, notification)
	c.notificationBytes += size
}

func notificationSize(notification rpcNotification) int {
	return len(notification.Method) + len(notification.Params)
}

func decodeNotification(notification rpcNotification, out any) error {
	if out == nil || len(notification.Params) == 0 || string(notification.Params) == "null" {
		return nil
	}
	if err := json.Unmarshal(notification.Params, out); err != nil {
		return errors.New("decode codex app-server notification")
	}
	return nil
}
