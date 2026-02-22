// Package providertest provides conformance test helpers for provider implementations.
package providertest

import (
	"context"
	"testing"
	"time"

	"github.com/agentplexus/omnichat/provider"
)

// TestProvider runs conformance tests for a provider implementation.
// This is a basic test suite that verifies a provider implements the interface correctly.
func TestProvider(t *testing.T, p provider.Provider, skipConnect bool) {
	t.Helper()

	// Test Name() returns non-empty string
	t.Run("Name", func(t *testing.T) {
		name := p.Name()
		if name == "" {
			t.Error("Name() returned empty string")
		}
	})

	// Test OnMessage accepts handler without panic
	t.Run("OnMessage", func(t *testing.T) {
		p.OnMessage(func(ctx context.Context, msg provider.IncomingMessage) error {
			return nil
		})
		// Handler registration should not panic
	})

	// Test OnEvent accepts handler without panic
	t.Run("OnEvent", func(t *testing.T) {
		p.OnEvent(func(ctx context.Context, evt provider.Event) error {
			return nil
		})
		// Handler registration should not panic
	})

	if skipConnect {
		t.Log("skipping Connect/Disconnect tests (skipConnect=true)")
		return
	}

	// Test Connect/Disconnect lifecycle
	t.Run("ConnectDisconnect", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := p.Connect(ctx); err != nil {
			t.Fatalf("Connect() error = %v", err)
		}

		// Allow some time for connection to stabilize
		time.Sleep(100 * time.Millisecond)

		if err := p.Disconnect(ctx); err != nil {
			t.Fatalf("Disconnect() error = %v", err)
		}
	})
}

// MessageRecorder records incoming messages for testing.
type MessageRecorder struct {
	Messages []provider.IncomingMessage
}

// Handler returns a message handler that records messages.
func (r *MessageRecorder) Handler() provider.MessageHandler {
	return func(ctx context.Context, msg provider.IncomingMessage) error {
		r.Messages = append(r.Messages, msg)
		return nil
	}
}

// Clear clears recorded messages.
func (r *MessageRecorder) Clear() {
	r.Messages = nil
}

// WaitForMessage waits for a message to be recorded with a timeout.
func (r *MessageRecorder) WaitForMessage(timeout time.Duration) (*provider.IncomingMessage, bool) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if len(r.Messages) > 0 {
			msg := r.Messages[0]
			return &msg, true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return nil, false
}

// EventRecorder records events for testing.
type EventRecorder struct {
	Events []provider.Event
}

// Handler returns an event handler that records events.
func (r *EventRecorder) Handler() provider.EventHandler {
	return func(ctx context.Context, evt provider.Event) error {
		r.Events = append(r.Events, evt)
		return nil
	}
}

// Clear clears recorded events.
func (r *EventRecorder) Clear() {
	r.Events = nil
}

// MockProvider is a mock provider for testing routers and handlers.
type MockProvider struct {
	name           string
	messageHandler provider.MessageHandler
	eventHandler   provider.EventHandler
	connected      bool
	sentMessages   []SentMessage
}

// SentMessage records a message that was sent via the mock provider.
type SentMessage struct {
	ChatID  string
	Message provider.OutgoingMessage
}

// NewMockProvider creates a new mock provider.
func NewMockProvider(name string) *MockProvider {
	return &MockProvider{name: name}
}

func (m *MockProvider) Name() string {
	return m.name
}

func (m *MockProvider) Connect(ctx context.Context) error {
	m.connected = true
	return nil
}

func (m *MockProvider) Disconnect(ctx context.Context) error {
	m.connected = false
	return nil
}

func (m *MockProvider) Send(ctx context.Context, chatID string, msg provider.OutgoingMessage) error {
	m.sentMessages = append(m.sentMessages, SentMessage{ChatID: chatID, Message: msg})
	return nil
}

func (m *MockProvider) OnMessage(handler provider.MessageHandler) {
	m.messageHandler = handler
}

func (m *MockProvider) OnEvent(handler provider.EventHandler) {
	m.eventHandler = handler
}

// SimulateMessage simulates receiving a message.
func (m *MockProvider) SimulateMessage(ctx context.Context, msg provider.IncomingMessage) error {
	if m.messageHandler != nil {
		msg.ProviderName = m.name
		return m.messageHandler(ctx, msg)
	}
	return nil
}

// SimulateEvent simulates receiving an event.
func (m *MockProvider) SimulateEvent(ctx context.Context, evt provider.Event) error {
	if m.eventHandler != nil {
		evt.ProviderName = m.name
		return m.eventHandler(ctx, evt)
	}
	return nil
}

// SentMessages returns all messages sent via this mock.
func (m *MockProvider) SentMessages() []SentMessage {
	return m.sentMessages
}

// ClearSent clears the sent messages buffer.
func (m *MockProvider) ClearSent() {
	m.sentMessages = nil
}

// IsConnected returns whether the mock is connected.
func (m *MockProvider) IsConnected() bool {
	return m.connected
}

// Ensure MockProvider implements Provider.
var _ provider.Provider = (*MockProvider)(nil)
