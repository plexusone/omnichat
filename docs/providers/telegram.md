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

### Localized Commands

Set commands for different languages programmatically:

```go
// Cast to telegram provider for extended methods
tgProvider := p.(*telegram.Provider)

// Set default commands
err := tgProvider.SetCommands(ctx, []telegram.Command{
    {Command: "start", Description: "Start the bot"},
    {Command: "help", Description: "Show help"},
})

// Set localized commands for multiple languages
err = tgProvider.SetLocalizedCommands(ctx, telegram.LocalizedCommands{
    "":   {{Command: "start", Description: "Start the bot"}},           // Default
    "es": {{Command: "start", Description: "Iniciar el bot"}},          // Spanish
    "de": {{Command: "start", Description: "Bot starten"}},             // German
    "zh": {{Command: "start", Description: "启动机器人"}},                // Chinese
})

// Delete commands for a specific language
err = tgProvider.DeleteCommands(ctx, "es")
```

## Inline Keyboards & Web App Buttons

Send messages with interactive inline keyboard buttons:

```go
// Send message with inline buttons
router.Send(ctx, "telegram", chatID, provider.OutgoingMessage{
    Content: "Choose an option:",
    Metadata: map[string]any{
        telegram.MetaInlineKeyboard: [][]telegram.InlineButton{
            // First row
            {
                {Text: "Option 1", CallbackData: "opt1"},
                {Text: "Option 2", CallbackData: "opt2"},
            },
            // Second row
            {
                {Text: "Visit Website", URL: "https://example.com"},
            },
        },
    },
})

// Send with Web App button (opens a mini-app)
router.Send(ctx, "telegram", chatID, provider.OutgoingMessage{
    Content: "Open our app:",
    Metadata: map[string]any{
        telegram.MetaInlineKeyboard: [][]telegram.InlineButton{
            {
                {Text: "Open App", WebAppURL: "https://myapp.example.com"},
            },
        },
    },
})
```

### Handling Button Callbacks

```go
// Cast to telegram provider for callback handling
tgProvider := p.(*telegram.Provider)

// Handle inline button callbacks
tgProvider.OnCallback(func(ctx context.Context, cb *telegram.Callback) error {
    switch cb.Data {
    case "opt1":
        // Acknowledge the callback with optional text
        tgProvider.AnswerCallback(ctx, cb.ID, "You chose Option 1!", false)
    case "opt2":
        // Show an alert popup
        tgProvider.AnswerCallback(ctx, cb.ID, "Option 2 selected", true)
    }
    return nil
})

// Handle Web App data responses
tgProvider.OnWebAppData(func(ctx context.Context, data *telegram.WebAppData) error {
    // data.Data contains the data sent from the web app
    // data.ButtonText is the button that opened the web app
    log.Printf("Received from web app: %s", data.Data)
    return nil
})
```

### Button Types

| Field | Description |
|-------|-------------|
| `Text` | Button label text (required) |
| `URL` | Opens URL when pressed |
| `CallbackData` | Data sent to bot when pressed |
| `WebAppURL` | Opens Telegram Web App |
| `SwitchInlineQuery` | Opens inline query in chat picker |
| `SwitchInlineQueryCurrentChat` | Opens inline query in current chat |

### Message Options

```go
// Silent message (no notification)
router.Send(ctx, "telegram", chatID, provider.OutgoingMessage{
    Content: "Silent message",
    Metadata: map[string]any{
        telegram.MetaDisableNotification: true,
    },
})

// Disable link preview
router.Send(ctx, "telegram", chatID, provider.OutgoingMessage{
    Content: "Check https://example.com",
    Metadata: map[string]any{
        telegram.MetaDisablePreview: true,
    },
})

// Protect content from forwarding/saving
router.Send(ctx, "telegram", chatID, provider.OutgoingMessage{
    Content: "Confidential message",
    Metadata: map[string]any{
        telegram.MetaProtectContent: true,
    },
})
```

**Metadata Keys:**

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `telegram_inline_keyboard` | `[][]InlineButton` | none | Inline keyboard rows |
| `telegram_disable_preview` | `bool` | `false` | Disable link preview |
| `telegram_disable_notification` | `bool` | `false` | Send silently |
| `telegram_protect_content` | `bool` | `false` | Prevent forwarding/saving |

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
