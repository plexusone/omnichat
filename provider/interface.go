// Package provider defines the core interface for messaging providers.
package provider

import (
	"context"
)

// Provider is the interface that all messaging providers implement.
type Provider interface {
	// Name returns the provider name (e.g., "discord", "telegram", "whatsapp").
	Name() string

	// Connect establishes connection to the messaging platform.
	Connect(ctx context.Context) error

	// Disconnect closes the connection.
	Disconnect(ctx context.Context) error

	// Send sends a message to a specific chat/conversation.
	Send(ctx context.Context, chatID string, msg OutgoingMessage) error

	// OnMessage registers a handler for incoming messages.
	OnMessage(handler MessageHandler)

	// OnEvent registers a handler for platform events.
	OnEvent(handler EventHandler)
}

// StreamingProvider extends Provider with typing indicators and streaming.
type StreamingProvider interface {
	Provider

	// SendTyping sends a typing indicator to a chat.
	SendTyping(ctx context.Context, chatID string) error

	// SendStream sends a message as a stream of chunks.
	SendStream(ctx context.Context, chatID string, chunks <-chan string) error
}

// MessageHandler handles incoming messages.
type MessageHandler func(ctx context.Context, msg IncomingMessage) error

// EventHandler handles provider events.
type EventHandler func(ctx context.Context, event Event) error
