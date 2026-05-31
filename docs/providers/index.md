# Providers

OmniChat supports multiple messaging platforms through a unified provider interface.

## Available Providers

| Provider | Package | Type | Authentication |
|----------|---------|------|----------------|
| [Discord](discord.md) | `providers/discord` | Chat | Bot token |
| [Telegram](telegram.md) | `providers/telegram` | Chat | Bot token |
| [WhatsApp](whatsapp.md) | `providers/whatsapp` | Chat | QR code |
| [Slack](slack.md) | `providers/slack` | Chat | OAuth tokens |
| [Gmail](gmail.md) | `providers/email/gmail` | Email | OAuth2 |
| [IRC](irc.md) | `providers/irc` | Chat | NickServ (optional) |
| [Twilio](twilio.md) | `providers/twilio` | SMS | Account SID/Auth Token |

## Provider Interface

All providers implement the `Provider` interface:

```go
type Provider interface {
    // Name returns the provider identifier
    Name() string

    // Connect establishes the connection
    Connect(ctx context.Context) error

    // Disconnect closes the connection
    Disconnect(ctx context.Context) error

    // Send sends a message
    Send(ctx context.Context, chatID string, msg OutgoingMessage) error

    // SetMessageHandler sets the incoming message callback
    SetMessageHandler(handler MessageHandler)
}
```

## Message Types

### IncomingMessage

```go
type IncomingMessage struct {
    ID           string            // Message ID
    ProviderName string            // Provider that received the message
    ChatID       string            // Chat/channel/thread ID
    ChatType     ChatType          // dm, group, channel, thread, newsletter, status
    SenderID     string            // Sender's user ID
    SenderName   string            // Sender's display name
    Content      string            // Text content
    ReplyTo      string            // Parent message ID (for threads)
    Media        []Media           // Attachments
    Timestamp    time.Time         // When the message was sent
    Metadata     map[string]any    // Provider-specific metadata
}

type ChatType string

const (
    ChatTypeDM         ChatType = "dm"         // Direct message
    ChatTypeGroup      ChatType = "group"      // Group chat
    ChatTypeChannel    ChatType = "channel"    // Channel (Slack, Discord, Telegram)
    ChatTypeThread     ChatType = "thread"     // Thread reply
    ChatTypeNewsletter ChatType = "newsletter" // WhatsApp Newsletter/Channel
    ChatTypeStatus     ChatType = "status"     // WhatsApp Status broadcast
)
```

### OutgoingMessage

```go
type OutgoingMessage struct {
    Content  string         // Text content
    ReplyTo  string         // Message ID to reply to
    Media    []Media        // Attachments
    Format   MessageFormat  // plain, markdown, html
    Metadata map[string]any // Provider-specific options
}
```

### Provider-Specific Metadata

Each provider supports custom metadata keys:

**Slack:**

- `slack_unfurl_links` - Enable/disable link previews
- `slack_unfurl_media` - Enable/disable media previews
- `slack_reply_broadcast` - Broadcast thread reply to channel

**Telegram:**

- `telegram_inline_keyboard` - Inline keyboard buttons
- `telegram_disable_preview` - Disable link preview
- `telegram_disable_notification` - Send silently
- `telegram_protect_content` - Prevent forwarding/saving

See individual provider docs for details.

### Media

```go
type Media struct {
    Type     MediaType // Image, Audio, Video, Document, Voice
    URL      string    // Remote URL
    Data     []byte    // Binary data
    MimeType string    // MIME type
    Filename string    // Original filename
}

type MediaType string

const (
    MediaTypeImage    MediaType = "image"
    MediaTypeAudio    MediaType = "audio"
    MediaTypeVideo    MediaType = "video"
    MediaTypeDocument MediaType = "document"
    MediaTypeVoice    MediaType = "voice"
)
```

## Registering Providers

```go
router := provider.NewRouter(logger)

// Register multiple providers
router.Register(discordProvider)
router.Register(telegramProvider)
router.Register(whatsappProvider)

// Connect all at once
router.ConnectAll(ctx)
```

## Provider-Specific Features

Each provider may support additional features:

| Feature | Discord | Telegram | WhatsApp | Slack | Gmail | IRC | Twilio |
|---------|---------|----------|----------|-------|-------|-----|--------|
| Text messages | Yes | Yes | Yes | Yes | Yes | Yes | Yes |
| Media attachments | Yes | Yes | Yes | Yes | Yes | No | No |
| Voice messages | No | Yes | Yes | No | No | No | No |
| Voice channels | Yes | No | No | No | No | No | No |
| Reactions | Yes | Yes | Yes | Yes | No | No | No |
| Threads | Yes | Yes | Yes | Yes | Yes | No | No |
| Typing indicators | Yes | Yes | Yes | No | No | No | No |
| Read receipts | No | Yes | Yes | No | No | No | No |
| Inline buttons | No | Yes | No | No | No | No | No |
| Web App | No | Yes | No | No | No | No | No |
| Localized commands | No | Yes | No | No | No | No | No |
| Unfurl controls | No | No | No | Yes | No | No | No |
| Reply broadcast | No | No | No | Yes | No | No | No |
| Newsletters/Channels | No | No | Yes | No | No | No | No |
| Voice channel allowlist | Yes | No | No | No | No | No | No |
| Auto-follow users | Yes | No | No | No | No | No | No |

## Error Handling

Providers return errors for connection and send failures:

```go
err := provider.Connect(ctx)
if err != nil {
    log.Printf("Failed to connect %s: %v", provider.Name(), err)
}

err = provider.Send(ctx, chatID, msg)
if err != nil {
    log.Printf("Failed to send to %s: %v", chatID, err)
}
```

## Next Steps

- [Discord](discord.md) - Discord bot setup
- [Telegram](telegram.md) - Telegram bot setup
- [WhatsApp](whatsapp.md) - WhatsApp Web setup
- [Slack](slack.md) - Slack app setup
- [Gmail](gmail.md) - Gmail API setup
- [IRC](irc.md) - IRC server connection
- [Twilio](twilio.md) - Twilio SMS setup
