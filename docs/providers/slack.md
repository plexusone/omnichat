# Slack

The Slack provider uses [slack-go](https://github.com/slack-go/slack) with Socket Mode for real-time events.

## Installation

```bash
go get github.com/plexusone/omnichat/providers/slack
```

## Configuration

```go
import "github.com/plexusone/omnichat/providers/slack"

p, err := slack.New(slack.Config{
    BotToken: "xoxb-...",   // Bot token
    AppToken: "xapp-...",   // App-level token for Socket Mode
    Logger:   slog.Default(),
})
```

### Config Options

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `BotToken` | `string` | Yes | Bot token (`xoxb-...`) |
| `AppToken` | `string` | Yes | App token (`xapp-...`) for Socket Mode |
| `Logger` | `*slog.Logger` | No | Logger instance |

## Creating a Slack App

1. Go to [Slack API](https://api.slack.com/apps)
2. Click **Create New App** > **From scratch**
3. Name your app and select a workspace

### Enable Socket Mode

1. Go to **Socket Mode** in sidebar
2. Enable Socket Mode
3. Create an App-Level Token with `connections:write` scope
4. Save the token (`xapp-...`)

### Bot Token Scopes

Go to **OAuth & Permissions** and add these scopes:

| Scope | Purpose |
|-------|---------|
| `chat:write` | Send messages |
| `channels:history` | Read channel messages |
| `groups:history` | Read private channel messages |
| `im:history` | Read DM messages |
| `mpim:history` | Read group DM messages |
| `users:read` | Get user information |
| `reactions:read` | Read reactions |

### Event Subscriptions

Go to **Event Subscriptions** and subscribe to:

| Event | Description |
|-------|-------------|
| `message.channels` | Messages in public channels |
| `message.groups` | Messages in private channels |
| `message.im` | Direct messages |
| `message.mpim` | Group direct messages |
| `reaction_added` | Reactions added to messages |
| `member_joined_channel` | Users joining channels |

### Install to Workspace

1. Go to **Install App**
2. Click **Install to Workspace**
3. Authorize the app
4. Copy the Bot Token (`xoxb-...`)

## Usage

### Basic Setup

```go
package main

import (
    "context"
    "log/slog"
    "os"

    "github.com/plexusone/omnichat/provider"
    "github.com/plexusone/omnichat/providers/slack"
)

func main() {
    logger := slog.Default()

    p, err := slack.New(slack.Config{
        BotToken: os.Getenv("SLACK_BOT_TOKEN"),
        AppToken: os.Getenv("SLACK_APP_TOKEN"),
        Logger:   logger,
    })
    if err != nil {
        panic(err)
    }

    router := provider.NewRouter(logger)
    router.Register(p)

    router.OnMessage(provider.All(), func(ctx context.Context, msg provider.IncomingMessage) error {
        return router.Send(ctx, "slack", msg.ChatID, provider.OutgoingMessage{
            Content: "Hello from Slack!",
        })
    })

    ctx := context.Background()
    router.ConnectAll(ctx)
    defer router.DisconnectAll(ctx)

    select {}
}
```

### Sending Messages

```go
// Simple message
router.Send(ctx, "slack", channelID, provider.OutgoingMessage{
    Content: "Hello, Slack!",
})

// Reply in thread
router.Send(ctx, "slack", channelID, provider.OutgoingMessage{
    Content: "This is a thread reply",
    ReplyTo: parentMessageTS,  // Thread timestamp
})

// Reply in thread AND broadcast to channel
router.Send(ctx, "slack", channelID, provider.OutgoingMessage{
    Content: "This reply will also appear in the channel",
    ReplyTo: parentMessageTS,
    Metadata: map[string]any{
        slack.MetaReplyBroadcast: true,
    },
})
```

### Controlling Link Previews (Unfurling)

Control how Slack displays link previews in messages:

```go
// Disable link unfurling
router.Send(ctx, "slack", channelID, provider.OutgoingMessage{
    Content: "Check this link: https://example.com",
    Metadata: map[string]any{
        slack.MetaUnfurlLinks: false,  // Don't show link preview
    },
})

// Disable media unfurling (images, videos in links)
router.Send(ctx, "slack", channelID, provider.OutgoingMessage{
    Content: "https://example.com/image.jpg",
    Metadata: map[string]any{
        slack.MetaUnfurlMedia: false,  // Don't expand media
    },
})

// Enable unfurling for non-markdown messages
router.Send(ctx, "slack", channelID, provider.OutgoingMessage{
    Content: "https://example.com",
    Metadata: map[string]any{
        slack.MetaUnfurlLinks: true,  // Show link preview
        slack.MetaUnfurlMedia: true,  // Show media previews
    },
})
```

**Metadata Keys:**

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `slack_unfurl_links` | `bool` | `true` for markdown | Enable text link previews |
| `slack_unfurl_media` | `bool` | `true` | Enable media previews |
| `slack_reply_broadcast` | `bool` | `false` | Broadcast thread replies to channel |

### Sending Media

```go
// Send a file
router.Send(ctx, "slack", channelID, provider.OutgoingMessage{
    Content: "Here's the report:",
    Media: []provider.Media{{
        Type:     provider.MediaTypeDocument,
        Data:     fileBytes,
        Filename: "report.pdf",
        MimeType: "application/pdf",
    }},
})

// Send an image
router.Send(ctx, "slack", channelID, provider.OutgoingMessage{
    Media: []provider.Media{{
        Type:     provider.MediaTypeImage,
        Data:     imageBytes,
        Filename: "screenshot.png",
        MimeType: "image/png",
    }},
})
```

## Message Mapping

| Slack | OmniChat |
|-------|----------|
| Channel ID | `ChatID` |
| Message TS | `ID` |
| User ID | `SenderID` |
| User Name | `SenderName` |
| Text | `Content` |
| Thread TS | `ReplyTo` |
| Files | `Media` |
| IM Channel | `IsDM = true` |

## Event Handling

The provider handles these Slack events:

### Messages

```go
router.OnMessage(provider.All(), func(ctx context.Context, msg provider.IncomingMessage) error {
    // msg.Content contains the message text
    // msg.ChatID is the channel ID
    // msg.ReplyTo is set if this is a thread reply
    return nil
})
```

### Reactions

Reactions are delivered as messages with special content:

```go
router.OnMessage(provider.All(), func(ctx context.Context, msg provider.IncomingMessage) error {
    if strings.HasPrefix(msg.Content, ":") && strings.HasSuffix(msg.Content, ":") {
        // This is a reaction
        emoji := strings.Trim(msg.Content, ":")
        log.Printf("Reaction %s added", emoji)
    }
    return nil
})
```

### Member Joins

Member join events are delivered as system messages.

## Channel Types

| Type | `IsDM` | Description |
|------|--------|-------------|
| Public channel | `false` | `C...` prefix |
| Private channel | `false` | `G...` prefix |
| Direct message | `true` | `D...` prefix |
| Group DM | `true` | `G...` prefix |

## Environment Variables

```bash
SLACK_BOT_TOKEN=xoxb-your-bot-token-here
SLACK_APP_TOKEN=xapp-your-app-token-here
```

## Troubleshooting

### Bot not receiving messages

1. Verify Socket Mode is enabled
2. Check Event Subscriptions are configured
3. Ensure bot is invited to channels (`/invite @botname`)
4. Verify App Token has `connections:write` scope

### Permission errors

1. Check OAuth scopes in app settings
2. Reinstall app after adding scopes
3. Verify bot is a member of the channel

### Connection drops

Socket Mode reconnects automatically. For persistent issues:

1. Check App Token is valid
2. Verify network connectivity
3. Check Slack status page

### Rate limiting

Slack has rate limits (~1 message/second per channel). The provider handles backoff automatically.

## Limitations

- No support for Block Kit (rich messages) through OmniChat interface
- Reactions are simplified to emoji strings
- Thread replies require parent message timestamp

## Next Steps

- [Gmail](gmail.md) - Add email support
- [Router](../reference/router.md) - Message routing
