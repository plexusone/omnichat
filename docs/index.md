# OmniChat

A unified Go library for messaging platforms.

## Overview

OmniChat provides a single interface for building applications that communicate across multiple messaging platforms. Write your message handling logic once and deploy it to Discord, Telegram, WhatsApp, Slack, and Gmail.

## Features

- **Unified Interface** - Single API for all messaging platforms
- **Router** - Pattern-based message routing with filters
- **Voice Support** - Transcription/synthesis for voice messages, Discord voice channels
- **Interactive UI** - Telegram inline keyboards, Web App buttons
- **Newsletters** - WhatsApp Channel/Newsletter support with reactions
- **Provider Registry** - Dynamic provider registration
- **Testing** - Mock provider for unit tests

## Supported Platforms

| Platform | Package | Status |
|----------|---------|--------|
| Discord | `providers/discord` | Stable (voice channels in v0.7.0) |
| Telegram | `providers/telegram` | Stable (inline keyboards in v0.7.0) |
| WhatsApp | `providers/whatsapp` | Stable (newsletters in v0.7.0) |
| Slack | `providers/slack` | Stable |
| Gmail | `providers/email/gmail` | Stable |
| IRC | `providers/irc` | Stable |
| Twilio | `providers/twilio` | Stable |

## Quick Example

```go
package main

import (
    "context"
    "log/slog"
    "os"

    "github.com/plexusone/omnichat/provider"
    "github.com/plexusone/omnichat/providers/discord"
    "github.com/plexusone/omnichat/providers/telegram"
)

func main() {
    logger := slog.Default()
    router := provider.NewRouter(logger)

    // Register providers
    discordProvider, _ := discord.New(discord.Config{
        Token:  os.Getenv("DISCORD_TOKEN"),
        Logger: logger,
    })
    router.Register(discordProvider)

    telegramProvider, _ := telegram.New(telegram.Config{
        Token:  os.Getenv("TELEGRAM_TOKEN"),
        Logger: logger,
    })
    router.Register(telegramProvider)

    // Handle all messages
    router.OnMessage(provider.All(), func(ctx context.Context, msg provider.IncomingMessage) error {
        return router.Send(ctx, msg.ProviderName, msg.ChatID, provider.OutgoingMessage{
            Content: "Echo: " + msg.Content,
        })
    })

    // Connect and run
    ctx := context.Background()
    router.ConnectAll(ctx)
    defer router.DisconnectAll(ctx)

    select {}
}
```

## Installation

```bash
go get github.com/plexusone/omnichat
```

## Next Steps

- [Getting Started](getting-started.md) - Installation and first steps
- [Providers](providers/index.md) - Platform-specific documentation
- [Router](reference/router.md) - Message routing reference
