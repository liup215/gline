package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/liup215/gline/internal/log"
)

// Transport is the interface for MCP transports
type Transport interface {
	// Start initializes the transport
	Start(ctx context.Context) error
	// Send sends a message
	Send(msg *JSONRPCMessage) error
	// Receive receives a message (blocking)
	Receive() (*JSONRPCMessage, error)
	// Close closes the transport
	Close() error
	// IsConnected returns true if the transport is connected
	IsConnected() bool
}

// StdioTransport implements transport over stdin/stdout of a subprocess
type StdioTransport struct {
	command string
	args    []string
	env     map[string]string

	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser
	reader *bufio.Reader

	mu        sync.RWMutex
	connected bool
	closed    atomic.Bool
}

// NewStdioTransport creates a new stdio transport
func NewStdioTransport(command string, args []string, env map[string]string) *StdioTransport {
	return &StdioTransport{
		command: command,
		args:    args,
		env:     env,
	}
}

// Start starts the subprocess and initializes the transport
func (t *StdioTransport) Start(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.connected {
		return fmt.Errorf("transport already started")
	}

	// Create command with a long-lived background context so the subprocess
	// survives beyond the initialization timeout. Process lifetime is managed
	// explicitly in Close().
	t.cmd = exec.CommandContext(context.Background(), t.command, t.args...)

	// Set environment variables
	if len(t.env) > 0 {
		env := t.cmd.Environ()
		for k, v := range t.env {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		t.cmd.Env = env
	}

	// Get pipes
	stdin, err := t.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	t.stdin = stdin

	stdout, err := t.cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	t.stdout = stdout

	stderr, err := t.cmd.StderrPipe()
	if err != nil {
		stdin.Close()
		stdout.Close()
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}
	t.stderr = stderr

	// Start the process
	if err := t.cmd.Start(); err != nil {
		stdin.Close()
		stdout.Close()
		stderr.Close()
		return fmt.Errorf("failed to start subprocess: %w", err)
	}

	// Start stderr reader for logging
	go t.readStderr()

	t.reader = bufio.NewReader(t.stdout)
	t.connected = true

	return nil
}

// readStderr reads stderr and logs it
func (t *StdioTransport) readStderr() {
	scanner := bufio.NewScanner(t.stderr)
	for scanner.Scan() {
		line := scanner.Text()
		// Log stderr output (could be redirected to a proper logger)
		fmt.Printf("[MCP Server stderr] %s\n", line)
	}
}

// Send sends a JSON-RPC message
func (t *StdioTransport) Send(msg *JSONRPCMessage) error {
	t.mu.RLock()
	if !t.connected {
		t.mu.RUnlock()
		return fmt.Errorf("transport not connected")
	}
	stdin := t.stdin
	t.mu.RUnlock()

	if t.closed.Load() {
		return fmt.Errorf("transport closed")
	}

	// Marshal message
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Write with newline delimiter
	data = append(data, '\n')

	if _, err := stdin.Write(data); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}

// Receive receives a JSON-RPC message
func (t *StdioTransport) Receive() (*JSONRPCMessage, error) {
	t.mu.RLock()
	if !t.connected {
		t.mu.RUnlock()
		return nil, fmt.Errorf("transport not connected")
	}
	reader := t.reader
	t.mu.RUnlock()

	if t.closed.Load() {
		return nil, fmt.Errorf("transport closed")
	}

	// Read line
	line, err := reader.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			return nil, fmt.Errorf("connection closed")
		}
		return nil, fmt.Errorf("failed to read message: %w", err)
	}

	// Trim whitespace
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, fmt.Errorf("empty message received")
	}

	// Parse JSON
	var msg JSONRPCMessage
	if err := json.Unmarshal([]byte(line), &msg); err != nil {
		return nil, fmt.Errorf("failed to parse message: %w", err)
	}

	return &msg, nil
}

// Close closes the transport
func (t *StdioTransport) Close() error {
	if !t.closed.CompareAndSwap(false, true) {
		return nil // Already closed
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.connected {
		return nil
	}

	// Close stdin to signal EOF to the subprocess
	if t.stdin != nil {
		t.stdin.Close()
	}

	// Wait for process to exit (with timeout)
	if t.cmd != nil && t.cmd.Process != nil {
		done := make(chan error, 1)
		go func() {
			done <- t.cmd.Wait()
		}()

		select {
		case <-done:
			// Process exited normally
		case <-time.After(5 * time.Second):
			// Timeout, kill the process
			t.cmd.Process.Kill()
		}
	}

	t.connected = false
	return nil
}

// IsConnected returns true if the transport is connected
func (t *StdioTransport) IsConnected() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.connected && !t.closed.Load()
}

// HTTPTransport implements simple HTTP transport for MCP
// Uses synchronous HTTP POST request-response (like LtEdu implementation)
type HTTPTransport struct {
	url     string
	headers map[string]string

	client      *http.Client
	ctx         context.Context
	cancel      context.CancelFunc
	mu          sync.RWMutex
	connected   bool
	closed      atomic.Bool
	pendingResp *JSONRPCMessage
	respCond    *sync.Cond // Condition variable for signaling response availability
}

// NewHTTPTransport creates a new HTTP transport
func NewHTTPTransport(urlStr string, headers map[string]string) (*HTTPTransport, error) {
	// Validate URL
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("URL must use http or https scheme")
	}

	t := &HTTPTransport{
		url:     urlStr,
		headers: headers,
		client:  &http.Client{Timeout: 60 * time.Second},
	}
	t.respCond = sync.NewCond(&t.mu)
	return t, nil
}

// Start initializes the HTTP transport
func (t *HTTPTransport) Start(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.connected {
		return fmt.Errorf("transport already started")
	}

	// Use a long-lived background context so the transport survives beyond the
	// initialization timeout. It is cancelled only when Close() is called.
	t.ctx, t.cancel = context.WithCancel(context.Background())
	t.connected = true
	return nil
}

// Send sends a JSON-RPC message via HTTP POST
// For stateless HTTP: this method directly returns the response via the transport's internal storage
func (t *HTTPTransport) Send(msg *JSONRPCMessage) error {
	log.Logger.Info().Str("method", msg.Method).Interface("id", msg.ID).Msg("[HTTPTransport] Send called")
	if t.closed.Load() {
		return fmt.Errorf("transport closed")
	}

	t.mu.RLock()
	connected := t.connected
	t.mu.RUnlock()

	if !connected {
		return fmt.Errorf("transport not connected")
	}

	// Marshal message
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Create POST request
	req, err := http.NewRequestWithContext(t.ctx, "POST", t.url, strings.NewReader(string(data)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	for k, v := range t.headers {
		req.Header.Set(k, v)
	}

	log.Logger.Info().Str("url", t.url).Msg("[HTTPTransport] Sending HTTP POST request")
	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	log.Logger.Info().Int("status", resp.StatusCode).Msg("[HTTPTransport] Received HTTP response")
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	// Read and store response for Receive to pick up
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	log.Logger.Debug().Str("body", string(body)).Msg("[HTTPTransport] Response body")

	var response JSONRPCMessage
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	fmt.Printf("[HTTPTransport] Parsed response: ID=%v, Method='%s', Result=%v, Error=%v\n",
		response.ID, response.Method, response.Result != nil, response.Error != nil)

	// Store response and signal Receive
	t.mu.Lock()
	t.pendingResp = &response
	if t.respCond != nil {
		log.Logger.Debug().Msg("[HTTPTransport] Broadcasting to respCond")
		t.respCond.Broadcast()
	}
	t.mu.Unlock()

	log.Logger.Info().Msg("[HTTPTransport] Send completed successfully")
	return nil
}

// Receive receives a JSON-RPC message
// For stateless HTTP: returns the pending response if available
func (t *HTTPTransport) Receive() (*JSONRPCMessage, error) {
	if t.closed.Load() {
		return nil, fmt.Errorf("transport closed")
	}

	// Wait for a response to be available using condition variable
	t.mu.Lock()
	defer t.mu.Unlock()

	for t.pendingResp == nil && !t.closed.Load() {
		if t.ctx != nil && t.ctx.Err() != nil {
			return nil, t.ctx.Err()
		}
		// Wait for signal with timeout check
		if t.respCond != nil {
			t.respCond.Wait()
		} else {
			t.mu.Unlock()
			time.Sleep(10 * time.Millisecond)
			t.mu.Lock()
		}
	}

	if t.closed.Load() {
		return nil, fmt.Errorf("transport closed")
	}

	msg := t.pendingResp
	t.pendingResp = nil
	return msg, nil
}

// Close closes the transport
func (t *HTTPTransport) Close() error {
	if !t.closed.CompareAndSwap(false, true) {
		return nil
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if t.cancel != nil {
		t.cancel()
	}

	t.connected = false
	return nil
}

// IsConnected returns true if the transport is connected
func (t *HTTPTransport) IsConnected() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.connected && !t.closed.Load()
}

// SSETransport implements transport over HTTP Server-Sent Events (legacy, kept for compatibility)
type SSETransport struct {
	url     string
	headers map[string]string

	client   *http.Client
	eventCh  chan *sseEvent
	msgCh    chan *JSONRPCMessage
	ctx      context.Context
	cancel   context.CancelFunc
	mu       sync.RWMutex
	connected bool
	closed    atomic.Bool
	sessionID string
}

// sseEvent represents an SSE event (internal type)
type sseEvent struct {
	Event string
	Data  string
	ID    string
}

// NewSSETransport creates a new SSE transport (legacy, redirects to HTTP transport)
func NewSSETransport(urlStr string, headers map[string]string) (*SSETransport, error) {
	// For now, keep legacy SSE implementation for backwards compatibility
	// Validate URL
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("URL must use http or https scheme")
	}

	return &SSETransport{
		url:     urlStr,
		headers: headers,
		client:  &http.Client{Timeout: 0}, // No timeout for streaming
		eventCh: make(chan *sseEvent, 100),
		msgCh:   make(chan *JSONRPCMessage, 100),
	}, nil
}

// Start initializes the SSE connection
func (t *SSETransport) Start(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.connected {
		return fmt.Errorf("transport already started")
	}

	// Use a long-lived background context so the transport survives beyond the
	// initialization timeout. It is cancelled only when Close() is called.
	t.ctx, t.cancel = context.WithCancel(context.Background())

	// Start SSE reader
	go t.readSSE()

	t.connected = true
	return nil
}

// readSSE reads SSE events from the server
func (t *SSETransport) readSSE() {
	req, err := http.NewRequestWithContext(t.ctx, "GET", t.url, nil)
	if err != nil {
		fmt.Printf("[MCP SSE] Failed to create request: %v\n", err)
		return
	}

	// Set headers
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	for k, v := range t.headers {
		req.Header.Set(k, v)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		fmt.Printf("[MCP SSE] Failed to connect: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("[MCP SSE] Unexpected status: %d\n", resp.StatusCode)
		return
	}

	// Read SSE events
	reader := bufio.NewReader(resp.Body)
	var currentEvent *sseEvent

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				fmt.Printf("[MCP SSE] Read error: %v\n", err)
			}
			return
		}

		line = strings.TrimRight(line, "\r\n")

		if line == "" {
			// Empty line means end of event
			if currentEvent != nil {
				t.handleEvent(currentEvent)
				currentEvent = nil
			}
			continue
		}

		if currentEvent == nil {
			currentEvent = &sseEvent{}
		}

		// Parse SSE field
		if strings.HasPrefix(line, "event:") {
			currentEvent.Event = strings.TrimSpace(line[6:])
		} else if strings.HasPrefix(line, "data:") {
			if currentEvent.Data != "" {
				currentEvent.Data += "\n"
			}
			currentEvent.Data += strings.TrimSpace(line[5:])
		} else if strings.HasPrefix(line, "id:") {
			currentEvent.ID = strings.TrimSpace(line[3:])
		}
	}
}

// handleEvent processes an SSE event
func (t *SSETransport) handleEvent(event *sseEvent) {
	switch event.Event {
	case "message":
		// Parse JSON-RPC message
		var msg JSONRPCMessage
		if err := json.Unmarshal([]byte(event.Data), &msg); err != nil {
			fmt.Printf("[MCP SSE] Failed to parse message: %v\n", err)
			return
		}
		t.msgCh <- &msg

	case "endpoint":
		// Store the endpoint for POST requests
		t.sessionID = event.Data

	default:
		// Store other events
		t.eventCh <- event
	}
}

// Send sends a JSON-RPC message via POST
func (t *SSETransport) Send(msg *JSONRPCMessage) error {
	if t.closed.Load() {
		return fmt.Errorf("transport closed")
	}

	t.mu.RLock()
	connected := t.connected
	t.mu.RUnlock()

	if !connected {
		return fmt.Errorf("transport not connected")
	}

	// Marshal message
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Build POST URL
	postURL := t.url
	if t.sessionID != "" {
		// Append session ID if provided
		postURL = t.sessionID
	}

	req, err := http.NewRequestWithContext(t.ctx, "POST", postURL, strings.NewReader(string(data)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range t.headers {
		req.Header.Set(k, v)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	return nil
}

// Receive receives a JSON-RPC message
func (t *SSETransport) Receive() (*JSONRPCMessage, error) {
	if t.closed.Load() {
		return nil, fmt.Errorf("transport closed")
	}

	select {
	case msg := <-t.msgCh:
		return msg, nil
	case <-t.ctx.Done():
		return nil, fmt.Errorf("context cancelled")
	}
}

// Close closes the transport
func (t *SSETransport) Close() error {
	if !t.closed.CompareAndSwap(false, true) {
		return nil
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if t.cancel != nil {
		t.cancel()
	}

	t.connected = false
	return nil
}

// IsConnected returns true if the transport is connected
func (t *SSETransport) IsConnected() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.connected && !t.closed.Load()
}
