# Telegram

The Telegram provider uses [telebot](https://github.com/tucnak/telebot) to connect to the Telegram Bot API.

## Installation

```bash
go get github.com/plexusone/omnichat/providers/telegram
```

## Configuration

```go
import "github.com/plexusone/omnichat/providers/telegram"

p, err := telegram.New(telegram.Config{
    Token:  "123456789:ABCdefGHIjklMNOpqrsTUVwxyz",
    Logger: slog.Default(),
})
```

### Config Options

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `Token` | `string` | Yes | Bot token from BotFather |
| `Logger` | `*slog.Logger` | No | Logger instance |

## Getting a Bot Token

1. Open Telegram and search for [@BotFather](https://t.me/botfather)
2. Send `/newbot` command
3. Follow prompts to name your bot
4. Copy the token provided

## Usage

### Basic Setup

```go
package main

import (
    "context"
    "log/slog"
    "os"

    "github.com/plexusone/omnichat/provider"
    "github.com/plexusone/omnichat/providers/telegram"
)

func main() {
    logger := slog.Default()

    p, err := telegram.New(telegram.Config{
        Token:  os.Getenv("TELEGRAM_TOKEN"),
        Logger: logger,
    })
    if err != nil {
        panic(err)
    }

    router := provider.NewRouter(logger)
    router.Register(p)

    router.OnMessage(provider.All(), func(ctx context.Context, msg provider.IncomingMessage) error {
        return router.Send(ctx, "telegram", msg.ChatID, provider.OutgoingMessage{
            Content: "Hello from Telegram!",
        })
    })

    ctx := context.Background()
    router.ConnectAll(ctx)
    defer router.DisconnectAll(ctx)

    select {}
}
```

### Sending Media

```go
// Send a photo
router.Send(ctx, "telegram", chatID, provider.OutgoingMessage{
    Content: "Check out this photo!",
    Media: []provider.Media{{
        Type: provider.MediaTypeImage,
        URL:  "https://example.com/photo.jpg",
    }},
})

// Send a document
router.Send(ctx, "telegram", chatID, provider.OutgoingMessage{
    Media: []provider.Media{{
        Type:     provider.MediaTypeDocument,
        Data:     fileBytes,
        Filename: "report.pdf",
        MimeType: "application/pdf",
    }},
})

// Send a voice message
router.Send(ctx, "telegram", chatID, provider.OutgoingMessage{
    Media: []provider.Media{{
        Type:     provider.MediaTypeVoice,
        Data:     audioBytes,
        MimeType: "audio/ogg",
    }},
})
```

### Reply to Messages

```go
router.OnMessage(provider.All(), func(ctx context.Context, msg provider.IncomingMessage) error {
    return router.Send(ctx, "telegram", msg.ChatID, provider.OutgoingMessage{
        Content: "Replying to your message!",
        ReplyTo: msg.ID,
    })
})
```

## Message Mapping

| Telegram | OmniChat |
|----------|----------|
| Chat ID | `ChatID` |
| Message ID | `ID` |
| From ID | `SenderID` |
| From FirstName | `SenderName` |
| Text | `Content` |
| ReplyToMessage | `ReplyTo` |
| Photo/Document/Audio | `Media` |
| Private chat | `IsDM = true` |

## Bot Commands

Configure commands via BotFather:

1. Send `/setcommands` to @BotFather
2. Send your bot's commands:

```
start - Start the bot
help - Show help
settings - Bot settings
```

Handle commands in your message handler:

```go
router.OnMessage(provider.All(), func(ctx context.Context, msg provider.IncomingMessage) error {
    switch {
    case strings.HasPrefix(msg.Content, "/start"):
        return router.Send(ctx, "telegram", msg.ChatID, provider.OutgoingMessage{
            Content: "Welcome! Send me a message.",
        })
    case strings.HasPrefix(msg.Content, "/help"):
        return router.Send(ctx, "telegram", msg.ChatID, provider.OutgoingMessage{
            Content: "Available commands:\n/start - Start\n/help - Help",
        })
    default:
        // Handle regular messages
    }
    return nil
})
```

## Group Chats

The bot can be added to groups:

1. Search for your bot in Telegram
2. Add to group
3. Optionally make it admin for full access

Group messages have `IsDM = false`:

```go
router.OnMessage(provider.GroupOnly(), func(ctx context.Context, msg provider.IncomingMessage) error {
    // Handle group messages
    return nil
})
```

## Inline Mode

For inline queries, use telebot directly:

```go
// Access underlying telebot instance if needed
// This requires type assertion to access provider internals
```

## Environment Variables

```bash
TELEGRAM_TOKEN=123456789:ABCdefGHIjklMNOpqrsTUVwxyz
```

## Troubleshooting

### Bot not responding

1. Verify token is correct
2. Check bot privacy mode with `/setprivacy` (disable for group messages)
3. Ensure bot is started (users must send `/start` first in DMs)

### Media upload fails

1. Check file size limits (photos: 10MB, documents: 50MB)
2. Verify MIME type is correct
3. Ensure file data is valid

### Rate limiting

Telegram allows ~30 messages/second. For broadcasts, implement delays:

```go
for _, chatID := range chatIDs {
    router.Send(ctx, "telegram", chatID, msg)
    time.Sleep(50 * time.Millisecond)
}
```

## Next Steps

- [WhatsApp](whatsapp.md) - Add WhatsApp support
- [Voice](../reference/voice.md) - Voice message processing
