package transport

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"

	mcptransport "github.com/metoro-io/mcp-golang/transport"
)

// StdioServerTransport implements MCP stdio framing using Content-Length headers.
type StdioServerTransport struct {
	mu        sync.Mutex
	started   bool
	reader    *bufio.Reader
	writer    io.Writer
	onClose   func()
	onError   func(error)
	onMessage func(context.Context, *mcptransport.BaseJsonRpcMessage)
	nextID    mcptransport.RequestId
	idMap     map[mcptransport.RequestId]json.RawMessage
	frames    map[mcptransport.RequestId]messageFrame
}

type messageFrame int

const (
	messageFrameContentLength messageFrame = iota
	messageFrameLine
)

func NewStdioServerTransport() *StdioServerTransport {
	return NewStdioServerTransportWithIO(os.Stdin, os.Stdout)
}

func NewStdioServerTransportWithIO(in io.Reader, out io.Writer) *StdioServerTransport {
	return &StdioServerTransport{
		reader: bufio.NewReader(in),
		writer: out,
		nextID: -1,
		idMap:  make(map[mcptransport.RequestId]json.RawMessage),
		frames: make(map[mcptransport.RequestId]messageFrame),
	}
}

func (t *StdioServerTransport) Start(ctx context.Context) error {
	t.mu.Lock()
	if t.started {
		t.mu.Unlock()
		return errors.New("stdio transport already started")
	}
	t.started = true
	t.mu.Unlock()

	go t.readLoop(ctx)
	return nil
}

func (t *StdioServerTransport) Send(_ context.Context, message *mcptransport.BaseJsonRpcMessage) error {
	data, err := t.marshalMessage(message)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	frame := t.frameForMessage(message)
	if frame == messageFrameLine {
		_, err = fmt.Fprintf(t.writer, "%s\n", data)
		return err
	}
	_, err = fmt.Fprintf(t.writer, "Content-Length: %d\r\n\r\n%s", len(data), data)
	return err
}

func (t *StdioServerTransport) Close() error {
	t.mu.Lock()
	wasStarted := t.started
	t.started = false
	handler := t.onClose
	t.mu.Unlock()

	if wasStarted && handler != nil {
		handler()
	}
	return nil
}

func (t *StdioServerTransport) SetCloseHandler(handler func()) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.onClose = handler
}

func (t *StdioServerTransport) SetErrorHandler(handler func(error)) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.onError = handler
}

func (t *StdioServerTransport) SetMessageHandler(handler func(context.Context, *mcptransport.BaseJsonRpcMessage)) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.onMessage = handler
}

func (t *StdioServerTransport) readLoop(ctx context.Context) {
	defer t.Close()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		message, err := t.readMessage()
		if err != nil {
			if !errors.Is(err, io.EOF) {
				t.handleError(err)
			}
			return
		}
		t.handleMessage(message)
	}
}

func (t *StdioServerTransport) readMessage() (*mcptransport.BaseJsonRpcMessage, error) {
	b, err := t.reader.Peek(1)
	if err != nil {
		return nil, err
	}

	// Keep accepting the old line-delimited transport in tests and older clients.
	if b[0] == '{' {
		line, err := t.reader.ReadBytes('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, err
		}
		return t.decodeMessage([]byte(strings.TrimSpace(string(line))), messageFrameLine)
	}

	length, err := t.readContentLength()
	if err != nil {
		return nil, err
	}

	data := make([]byte, length)
	if _, err := io.ReadFull(t.reader, data); err != nil {
		return nil, err
	}
	return t.decodeMessage(data, messageFrameContentLength)
}

func (t *StdioServerTransport) readContentLength() (int, error) {
	contentLength := -1

	for {
		line, err := t.reader.ReadString('\n')
		if err != nil {
			return 0, err
		}

		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			if contentLength < 0 {
				return 0, errors.New("missing Content-Length header")
			}
			return contentLength, nil
		}

		name, value, ok := strings.Cut(line, ":")
		if !ok {
			return 0, fmt.Errorf("invalid MCP stdio header: %q", line)
		}
		if strings.EqualFold(strings.TrimSpace(name), "Content-Length") {
			n, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil || n < 0 {
				return 0, fmt.Errorf("invalid Content-Length: %q", strings.TrimSpace(value))
			}
			contentLength = n
		}
	}
}

func (t *StdioServerTransport) decodeMessage(data []byte, frame messageFrame) (*mcptransport.BaseJsonRpcMessage, error) {
	data, err := t.normalizeRequestID(data, frame)
	if err != nil {
		return nil, err
	}

	var request mcptransport.BaseJSONRPCRequest
	if err := json.Unmarshal(data, &request); err == nil {
		return mcptransport.NewBaseMessageRequest(&request), nil
	}

	var notification mcptransport.BaseJSONRPCNotification
	if err := json.Unmarshal(data, &notification); err == nil {
		return mcptransport.NewBaseMessageNotification(&notification), nil
	}

	var response mcptransport.BaseJSONRPCResponse
	if err := json.Unmarshal(data, &response); err == nil {
		return mcptransport.NewBaseMessageResponse(&response), nil
	}

	var errorResponse mcptransport.BaseJSONRPCError
	if err := json.Unmarshal(data, &errorResponse); err == nil {
		return mcptransport.NewBaseMessageError(&errorResponse), nil
	}

	return nil, errors.New("failed to unmarshal JSON-RPC message")
}

func (t *StdioServerTransport) normalizeRequestID(data []byte, frame messageFrame) ([]byte, error) {
	var envelope struct {
		ID json.RawMessage `json:"id"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil || len(envelope.ID) == 0 {
		return data, nil
	}

	if isJSONNumber(envelope.ID) {
		var id mcptransport.RequestId
		if err := json.Unmarshal(envelope.ID, &id); err == nil {
			t.mu.Lock()
			t.frames[id] = frame
			t.mu.Unlock()
		}
		return data, nil
	}

	t.mu.Lock()
	internalID := t.nextID
	t.nextID--
	t.idMap[internalID] = append(json.RawMessage(nil), envelope.ID...)
	t.frames[internalID] = frame
	t.mu.Unlock()

	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, err
	}
	idBytes, err := json.Marshal(internalID)
	if err != nil {
		return nil, err
	}
	obj["id"] = idBytes
	return json.Marshal(obj)
}

func (t *StdioServerTransport) marshalMessage(message *mcptransport.BaseJsonRpcMessage) ([]byte, error) {
	if message.Type != mcptransport.BaseMessageTypeJSONRPCResponseType && message.Type != mcptransport.BaseMessageTypeJSONRPCErrorType {
		return json.Marshal(message)
	}

	var id mcptransport.RequestId
	if message.Type == mcptransport.BaseMessageTypeJSONRPCResponseType {
		id = message.JsonRpcResponse.Id
	} else {
		id = message.JsonRpcError.Id
	}

	t.mu.Lock()
	originalID, ok := t.idMap[id]
	if ok {
		delete(t.idMap, id)
	}
	t.mu.Unlock()

	if !ok {
		return json.Marshal(message)
	}

	data, err := json.Marshal(message)
	if err != nil {
		return nil, err
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, err
	}
	obj["id"] = originalID
	return json.Marshal(obj)
}

func (t *StdioServerTransport) frameForMessage(message *mcptransport.BaseJsonRpcMessage) messageFrame {
	var id mcptransport.RequestId
	if message.Type == mcptransport.BaseMessageTypeJSONRPCResponseType {
		id = message.JsonRpcResponse.Id
	} else if message.Type == mcptransport.BaseMessageTypeJSONRPCErrorType {
		id = message.JsonRpcError.Id
	} else {
		return messageFrameContentLength
	}

	frame, ok := t.frames[id]
	if ok {
		delete(t.frames, id)
		return frame
	}
	return messageFrameContentLength
}

func isJSONNumber(raw json.RawMessage) bool {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return false
	}
	first := trimmed[0]
	return first == '-' || (first >= '0' && first <= '9')
}

func (t *StdioServerTransport) handleError(err error) {
	t.mu.Lock()
	handler := t.onError
	t.mu.Unlock()

	if handler != nil {
		handler(err)
	}
}

func (t *StdioServerTransport) handleMessage(message *mcptransport.BaseJsonRpcMessage) {
	t.mu.Lock()
	handler := t.onMessage
	t.mu.Unlock()

	if handler != nil {
		handler(context.Background(), message)
	}
}
