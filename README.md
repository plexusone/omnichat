# OmniChat

[![Build Status][build-status-svg]][build-status-url]
[![Lint Status][lint-status-svg]][lint-status-url]
[![Go Report Card][goreport-svg]][goreport-url]
[![Docs][docs-godoc-svg]][docs-godoc-url]
[![Visualization][viz-svg]][viz-url]
[![License][license-svg]][license-url]

A Go library providing a unified interface for messaging platforms (Discord, Telegram, WhatsApp).

## Installation

```bash
go get github.com/agentplexus/omnichat
```

## Quick Start

```go
package main

import (
    "context"
    "log/slog"
    "os"

    "github.com/agentplexus/omnichat/provider"
    "github.com/agentplexus/omnichat/providers/discord"
    "github.com/agentplexus/omnichat/providers/telegram"
)

func main() {
    logger := slog.Default()
    router := provider.NewRouter(logger)

    // Register Discord provider
    discordProvider, _ := discord.New(discord.Config{
        Token:  os.Getenv("DISCORD_TOKEN"),
        Logger: logger,
    })
    router.Register(discordProvider)

    // Register Telegram provider
    telegramProvider, _ := telegram.New(telegram.Config{
        Token:  os.Getenv("TELEGRAM_TOKEN"),
        Logger: logger,
    })
    router.Register(telegramProvider)

    // Handle all messages
    router.OnMessage(provider.All(), func(ctx context.Context, msg provider.IncomingMessage) error {
        // Echo the message back
        return router.Send(ctx, msg.ProviderName, msg.ChatID, provider.OutgoingMessage{
            Content: "Echo: " + msg.Content,
        })
    })

    // Connect and run
    ctx := context.Background()
    router.ConnectAll(ctx)
    defer router.DisconnectAll(ctx)

    // Keep running...
    select {}
}
```

## Providers

### Discord

```go
import "github.com/agentplexus/omnichat/providers/discord"

p, err := discord.New(discord.Config{
    Token:   "bot-token",
    GuildID: "optional-guild-id",
    Logger:  slog.Default(),
})
```

### Telegram

```go
import "github.com/agentplexus/omnichat/providers/telegram"

p, err := telegram.New(telegram.Config{
    Token:  "bot-token",
    Logger: slog.Default(),
})
```

### WhatsApp

```go
import "github.com/agentplexus/omnichat/providers/whatsapp"

p, err := whatsapp.New(whatsapp.Config{
    DBPath: "whatsapp.db",  // Session storage
    Logger: slog.Default(),
    QRCallback: func(qr string) {
        // Display QR code for authentication
        fmt.Println("Scan this QR code:", qr)
    },
})
```

## Router

The router manages multiple providers and routes messages:

```go
router := provider.NewRouter(logger)

// Register providers
router.Register(discordProvider)
router.Register(telegramProvider)

// Route patterns
router.OnMessage(provider.All(), handler)                    // All messages
router.OnMessage(provider.DMOnly(), handler)                 // DMs only
router.OnMessage(provider.GroupOnly(), handler)              // Groups only
router.OnMessage(provider.FromProviders("discord"), handler) // Discord only

// Send messages
router.Send(ctx, "discord", channelID, provider.OutgoingMessage{
    Content: "Hello!",
})

// Broadcast to multiple providers
router.Broadcast(ctx, map[string]string{
    "discord":  channelID,
    "telegram": chatID,
}, provider.OutgoingMessage{
    Content: "Hello everyone!",
})
```

## Testing

Use the mock provider for testing:

```go
import "github.com/agentplexus/omnichat/provider/providertest"

mock := providertest.NewMockProvider("test")
router.Register(mock)

// Simulate incoming message
mock.SimulateMessage(ctx, provider.IncomingMessage{
    ChatID:  "123",
    Content: "Hello",
})

// Check sent messages
sent := mock.SentMessages()
```

## License

MIT

 [build-status-svg]: https://github.com/agentplexus/omnichat/actions/workflows/ci.yaml/badge.svg?branch=main
 [build-status-url]: https://github.com/agentplexus/omnichat/actions/workflows/ci.yaml
 [lint-status-svg]: https://github.com/agentplexus/omnichat/actions/workflows/lint.yaml/badge.svg?branch=main
 [lint-status-url]: https://github.com/agentplexus/omnichat/actions/workflows/lint.yaml
 [goreport-svg]: https://goreportcard.com/badge/github.com/agentplexus/omnichat
 [goreport-url]: https://goreportcard.com/report/github.com/agentplexus/omnichat
 [docs-godoc-svg]: https://pkg.go.dev/badge/github.com/agentplexus/omnichat
 [docs-godoc-url]: https://pkg.go.dev/github.com/agentplexus/omnichat
 [viz-svg]: https://img.shields.io/badge/visualizaton-Go-blue.svg
 [viz-url]: https://mango-dune-07a8b7110.1.azurestaticapps.net/?repo=agentplexus%2Fomnichat
 [loc-svg]: https://tokei.rs/b1/github/agentplexus/omnichat
 [repo-url]: https://github.com/agentplexus/omnichat
 [license-svg]: https://img.shields.io/badge/license-MIT-blue.svg
 [license-url]: https://github.com/agentplexus/omnichat/blob/master/LICENSE
