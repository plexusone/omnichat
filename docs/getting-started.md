# Getting Started

This guide walks you through setting up OmniChat and sending your first message.

## Installation

```bash
go get github.com/plexusone/omnichat
```

## Prerequisites

You'll need credentials for each platform you want to use:

| Platform | Required |
|----------|----------|
| Discord | Bot token from [Discord Developer Portal](https://discord.com/developers/applications) |
| Telegram | Bot token from [@BotFather](https://t.me/botfather) |
| WhatsApp | Phone number (QR code authentication) |
| Slack | Bot token and App token from [Slack API](https://api.slack.com/apps) |
| Gmail | OAuth credentials from [Google Cloud Console](https://console.cloud.google.com/) |

## Basic Setup

### 1. Create a Router

The router manages providers and routes messages:

```go
import (
    "log/slog"
    "github.com/plexusone/omnichat/provider"
)

logger := slog.Default()
router := provider.NewRouter(logger)
```

### 2. Register Providers

Add the platforms you want to support:

```go
import "github.com/plexusone/omnichat/providers/discord"

discordProvider, err := discord.New(discord.Config{
    Token:  os.Getenv("DISCORD_TOKEN"),
    Logger: logger,
})
if err != nil {
    log.Fatal(err)
}
router.Register(discordProvider)
```

### 3. Handle Messages

Register handlers with routing patterns:

```go
// Handle all messages
router.OnMessage(provider.All(), func(ctx context.Context, msg provider.IncomingMessage) error {
    log.Printf("Received: %s from %s", msg.Content, msg.ProviderName)
    return nil
})

// Handle DMs only
router.OnMessage(provider.DMOnly(), func(ctx context.Context, msg provider.IncomingMessage) error {
    return router.Send(ctx, msg.ProviderName, msg.ChatID, provider.OutgoingMessage{
        Content: "Thanks for your DM!",
    })
})
```

### 4. Connect and Run

```go
ctx := context.Background()
router.ConnectAll(ctx)
defer router.DisconnectAll(ctx)

// Keep the application running
select {}
```

## Complete Example

```go
package main

import (
    "context"
    "log"
    "log/slog"
    "os"
    "os/signal"
    "syscall"

    "github.com/plexusone/omnichat/provider"
    "github.com/plexusone/omnichat/providers/discord"
    "github.com/plexusone/omnichat/providers/telegram"
)

func main() {
    logger := slog.Default()
    router := provider.NewRouter(logger)

    // Discord
    if token := os.Getenv("DISCORD_TOKEN"); token != "" {
        p, err := discord.New(discord.Config{
            Token:  token,
            Logger: logger,
        })
        if err != nil {
            log.Fatal(err)
        }
        router.Register(p)
    }

    // Telegram
    if token := os.Getenv("TELEGRAM_TOKEN"); token != "" {
        p, err := telegram.New(telegram.Config{
            Token:  token,
            Logger: logger,
        })
        if err != nil {
            log.Fatal(err)
        }
        router.Register(p)
    }

    // Echo handler
    router.OnMessage(provider.All(), func(ctx context.Context, msg provider.IncomingMessage) error {
        return router.Send(ctx, msg.ProviderName, msg.ChatID, provider.OutgoingMessage{
            Content: "Echo: " + msg.Content,
        })
    })

    // Connect
    ctx := context.Background()
    router.ConnectAll(ctx)
    defer router.DisconnectAll(ctx)

    log.Println("Bot is running. Press Ctrl+C to exit.")

    // Wait for shutdown
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
    <-sigCh

    log.Println("Shutting down...")
}
```

## Environment Variables

Create a `.env` file:

```bash
DISCORD_TOKEN=your_discord_bot_token
TELEGRAM_TOKEN=your_telegram_bot_token
SLACK_BOT_TOKEN=xoxb-your-bot-token
SLACK_APP_TOKEN=xapp-your-app-token
```

## Next Steps

- [Providers](providers/index.md) - Configure each platform
- [Router](reference/router.md) - Advanced routing patterns
- [Voice](reference/voice.md) - Voice message support
