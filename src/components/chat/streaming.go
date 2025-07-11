// streaming.go - Per-chat streaming worker for OpenRouter API
// Handles streaming, cancellation, and updates to chat state.

package chat

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
)

// StreamEventType represents the type of event sent from the worker
// to the main thread.
type StreamEventType int

const (
	StreamEventChunk  StreamEventType = iota // Streaming chunk received
	StreamEventDone                          // Streaming finished
	StreamEventCancel                        // Streaming cancelled
	StreamEventError                         // Error occurred
)

// StreamEvent is sent from the worker to the main thread
// to update chat state.
type StreamEvent struct {
	Type    StreamEventType
	Content string // For chunk
	Err     error  // For error
}

// StreamWorker manages streaming for a single chat
type StreamWorker struct {
	ChatID      string
	Model       string
	APIKey      string
	CancelFunc  context.CancelFunc
	EventChan   chan StreamEvent // Worker → main thread
	InputChan   chan string      // Main thread → worker (user message)
	active      bool
	activeMutex sync.Mutex
}

// StartStreamWorker starts a new streaming worker for a chat
func StartStreamWorker(chatID, model, apiKey string) *StreamWorker {
	ctx, cancel := context.WithCancel(context.Background())
	w := &StreamWorker{
		ChatID:     chatID,
		Model:      model,
		APIKey:     apiKey,
		CancelFunc: cancel,
		EventChan:  make(chan StreamEvent, 10),
		InputChan:  make(chan string, 1),
		active:     true,
	}
	go w.run(ctx)
	return w
}

// Stop stops the worker and cancels any ongoing stream
func (w *StreamWorker) Stop() {
	w.activeMutex.Lock()
	if w.active {
		w.CancelFunc()
		w.active = false
	}
	w.activeMutex.Unlock()
}

// run is the main loop for the worker
func (w *StreamWorker) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case userMsg := <-w.InputChan:
			w.streamMessage(ctx, userMsg)
		}
	}
}

// streamMessage streams a response from the OpenRouter API
func (w *StreamWorker) streamMessage(ctx context.Context, userMsg string) {
	url := "https://openrouter.ai/api/v1/chat/completions"
	payload := map[string]interface{}{
		"model":    w.Model,
		"messages": []map[string]string{{"role": "user", "content": userMsg}},
		"stream":   true,
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(body)))
	req.Header.Set("Authorization", "Bearer "+w.APIKey)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		w.EventChan <- StreamEvent{Type: StreamEventError, Err: err}
		return
	}
	defer resp.Body.Close()
	reader := bufio.NewReader(resp.Body)
	var buffer strings.Builder
	for {
		select {
		case <-ctx.Done():
			w.EventChan <- StreamEvent{Type: StreamEventCancel}
			return
		default:
			line, err := reader.ReadString('\n')
			if err != nil {
				w.EventChan <- StreamEvent{Type: StreamEventError, Err: err}
				return
			}
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "data: ") {
				data := line[6:]
				if data == "[DONE]" {
					w.EventChan <- StreamEvent{Type: StreamEventDone}
					return
				}
				var obj map[string]interface{}
				if err := json.Unmarshal([]byte(data), &obj); err == nil {
					choices, ok := obj["choices"].([]interface{})
					if ok && len(choices) > 0 {
						choice := choices[0].(map[string]interface{})
						delta, ok := choice["delta"].(map[string]interface{})
						if ok {
							content, _ := delta["content"].(string)
							if content != "" {
								buffer.WriteString(content)
								w.EventChan <- StreamEvent{Type: StreamEventChunk, Content: content}
							}
						}
					}
				}
			}
		}
	}
}
