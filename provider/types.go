package provider

import "time"

// IncomingMessage represents a message received from a provider.
type IncomingMessage struct {
	// ID is the unique message identifier.
	ID string

	// ProviderName is the provider this message came from (e.g., "telegram").
	ProviderName string

	// ChatID is the chat/conversation identifier.
	ChatID string

	// ChatType is the type of chat (dm, group, channel, thread).
	ChatType ChatType

	// SenderID is the sender's identifier.
	SenderID string

	// SenderName is the sender's display name.
	SenderName string

	// Content is the message text content.
	Content string

	// Media contains any attached media.
	Media []Media

	// ReplyTo is the ID of the message being replied to, if any.
	ReplyTo string

	// Timestamp is when the message was sent.
	Timestamp time.Time

	// Metadata contains provider-specific metadata.
	Metadata map[string]any
}

// OutgoingMessage represents a message to send through a provider.
type OutgoingMessage struct {
	// Content is the message text content.
	Content string

	// Media contains media to attach.
	Media []Media

	// ReplyTo is the ID of the message to reply to, if any.
	ReplyTo string

	// Format specifies the message format.
	Format MessageFormat

	// Metadata contains provider-specific options.
	Metadata map[string]any
}

// Media represents attached media.
type Media struct {
	// Type is the media type (image, video, audio, document).
	Type MediaType

	// URL is the media URL (for remote media).
	URL string

	// Data is the raw media data (for local media).
	Data []byte

	// MimeType is the MIME type.
	MimeType string

	// Filename is the file name.
	Filename string

	// Caption is an optional caption.
	Caption string
}

// MediaType represents the type of media.
type MediaType string

const (
	MediaTypeImage    MediaType = "image"
	MediaTypeVideo    MediaType = "video"
	MediaTypeAudio    MediaType = "audio"
	MediaTypeDocument MediaType = "document"
	MediaTypeSticker  MediaType = "sticker"
	MediaTypeVoice    MediaType = "voice"
)

// MessageFormat represents the message format.
type MessageFormat string

const (
	MessageFormatPlain    MessageFormat = "plain"
	MessageFormatMarkdown MessageFormat = "markdown"
	MessageFormatHTML     MessageFormat = "html"
)

// ChatType represents the type of chat.
type ChatType string

const (
	ChatTypeDM      ChatType = "dm"
	ChatTypeGroup   ChatType = "group"
	ChatTypeChannel ChatType = "channel"
	ChatTypeThread  ChatType = "thread"
)

// Event represents a provider event.
type Event struct {
	// Type is the event type.
	Type EventType

	// ProviderName is the provider this event came from.
	ProviderName string

	// ChatID is the chat/conversation identifier.
	ChatID string

	// Data contains event-specific data.
	Data map[string]any

	// Timestamp is when the event occurred.
	Timestamp time.Time
}

// EventType represents the type of provider event.
type EventType string

const (
	EventTypeMessageEdited  EventType = "message_edited"
	EventTypeMessageDeleted EventType = "message_deleted"
	EventTypeReaction       EventType = "reaction"
	EventTypeTyping         EventType = "typing"
	EventTypePresence       EventType = "presence"
	EventTypeMemberJoined   EventType = "member_joined"
	EventTypeMemberLeft     EventType = "member_left"
	EventTypeChannelCreated EventType = "channel_created"
	EventTypeChannelDeleted EventType = "channel_deleted"
)

// Config is the base configuration for providers.
type Config struct {
	Enabled bool
	Token   string
}
