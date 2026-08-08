package codex

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

const maxAppServerMessageBytes = 32 << 20

type rpcClient struct {
	writer io.Writer
	scan   *bufio.Scanner
	nextID int64
}

type rpcResponse struct {
	ID     json.RawMessage `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *rpcError       `json:"error"`
}

type rpcError struct {
	Code int `json:"code"`
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

	for c.scan.Scan() {
		line := c.scan.Bytes()
		if len(line) == 0 {
			continue
		}

		var response rpcResponse
		if err := json.Unmarshal(line, &response); err != nil {
			return errors.New("codex app-server emitted invalid JSON")
		}
		if len(response.ID) == 0 {
			// Notifications are expected while app-server is running.
			continue
		}

		var responseID int64
		if err := json.Unmarshal(response.ID, &responseID); err != nil || responseID != id {
			// Ignore responses to unrelated requests. mcp-interop performs its own
			// requests sequentially, but app-server may serve other internal work.
			continue
		}
		if response.Error != nil {
			return fmt.Errorf("codex app-server JSON-RPC error %d", response.Error.Code)
		}
		if out == nil || len(response.Result) == 0 || string(response.Result) == "null" {
			return nil
		}
		if err := json.Unmarshal(response.Result, out); err != nil {
			return errors.New("decode codex app-server response")
		}
		return nil
	}

	if err := c.scan.Err(); err != nil {
		return fmt.Errorf("read codex app-server response: %w", err)
	}
	return errors.New("codex app-server closed before responding")
}
