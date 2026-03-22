# Discord

The Discord provider uses [discordgo](https://github.com/bwmarrin/discordgo) to connect to Discord's Gateway API.

## Installation

```bash
go get github.com/plexusone/omnichat/providers/discord
```

## Configuration

```go
import "github.com/plexusone/omnichat/providers/discord"

p, err := discord.New(discord.Config{
    Token:   "Bot MTIzNDU2Nzg5...",  // Bot token (with "Bot " prefix)
    GuildID: "123456789012345678",   // Optional: limit to one server
    Logger:  slog.Default(),
})
```

### Config Options

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `Token` | `string` | Yes | Discord bot token |
| `GuildID` | `string` | No | Restrict to specific guild |
| `Logger` | `*slog.Logger` | No | Logger instance |

## Getting a Bot Token

1. Go to [Discord Developer Portal](https://discord.com/developers/applications)
2. Click **New Application**
3. Go to **Bot** section
4. Click **Reset Token** to generate a new token
5. Enable required **Privileged Gateway Intents**:
   - Message Content Intent (to read message text)
   - Server Members Intent (optional, for member events)

## Bot Permissions

Your bot needs these permissions:

- **Send Messages** - To send responses
- **Read Message History** - To read messages
- **Attach Files** - To send media

Generate an invite URL:

1. Go to **OAuth2 > URL Generator**
2. Select scopes: `bot`, `applications.commands`
3. Select permissions needed
4. Use the generated URL to invite the bot

## Usage

### Basic Setup

```go
package main

import (
    "context"
    "log/slog"
    "os"

    "github.com/plexusone/omnichat/provider"
    "github.com/plexusone/omnichat/providers/discord"
)

func main() {
    logger := slog.Default()

    p, err := discord.New(discord.Config{
        Token:  os.Getenv("DISCORD_TOKEN"),
        Logger: logger,
    })
    if err != nil {
        panic(err)
    }

    router := provider.NewRouter(logger)
    router.Register(p)

    router.OnMessage(provider.All(), func(ctx context.Context, msg provider.IncomingMessage) error {
        return router.Send(ctx, "discord", msg.ChatID, provider.OutgoingMessage{
            Content: "Hello from Discord!",
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
// Send an image
router.Send(ctx, "discord", channelID, provider.OutgoingMessage{
    Content: "Check out this image!",
    Media: []provider.Media{{
        Type:     provider.MediaTypeImage,
        URL:      "https://example.com/image.png",
        Filename: "image.png",
    }},
})

// Send a file
router.Send(ctx, "discord", channelID, provider.OutgoingMessage{
    Media: []provider.Media{{
        Type:     provider.MediaTypeDocument,
        Data:     fileBytes,
        Filename: "document.pdf",
        MimeType: "application/pdf",
    }},
})
```

### Reply to Messages

```go
router.OnMessage(provider.All(), func(ctx context.Context, msg provider.IncomingMessage) error {
    return router.Send(ctx, "discord", msg.ChatID, provider.OutgoingMessage{
        Content: "This is a reply!",
        ReplyTo: msg.ID,  // Reply to the original message
    })
})
```

## Message Mapping

| Discord | OmniChat |
|---------|----------|
| Channel ID | `ChatID` |
| Message ID | `ID` |
| Author ID | `SenderID` |
| Author Username | `SenderName` |
| Content | `Content` |
| Referenced Message | `ReplyTo` |
| Attachments | `Media` |
| DM Channel | `IsDM = true` |

## Filtering Messages

```go
// Only handle messages from specific guild
router.OnMessage(provider.FromProviders("discord"), func(ctx context.Context, msg provider.IncomingMessage) error {
    // Discord-specific handling
    return nil
})

// Only DMs
router.OnMessage(provider.DMOnly(), handler)

// Only guild channels
router.OnMessage(provider.GroupOnly(), handler)
```

## Environment Variables

```bash
DISCORD_TOKEN=your_discord_bot_token_here
```

## Troubleshooting

### Bot not receiving messages

1. Ensure **Message Content Intent** is enabled in Developer Portal
2. Verify bot has **Read Message History** permission
3. Check that the bot is in the channel

### Permission errors

1. Re-invite the bot with correct permissions
2. Check channel-specific permission overrides

### Rate limiting

Discord rate limits are handled automatically by discordgo. For high-volume bots, consider implementing message queuing.

## Next Steps

- [Telegram](telegram.md) - Add Telegram support
- [Router](../reference/router.md) - Advanced routing patterns
