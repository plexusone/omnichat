# Release Notes - v0.1.0

**Release Date:** 2026-02-22

## Highlights

- **Unified Go interface for messaging platforms** with Discord, Telegram, and WhatsApp providers
- **Multi-provider message routing** with pattern matching for flexible message handling
- **WhatsApp support via whatsmeow** with QR code authentication and SQLite session persistence (pure Go, no CGO required)

## Overview

OmniChat provides a unified interface for building messaging applications across multiple platforms. Write your message handling logic once and deploy it across Discord, Telegram, and WhatsApp.

## Features

### Core Provider Interface

```go
type Provider interface {
    Name() string
    Connect(ctx context.Context) error
    Disconnect(ctx context.Context) error
    Send(ctx context.Context, chatID string, msg OutgoingMessage) error
    OnMessage(handler MessageHandler)
    OnEvent(handler EventHandler)
}
```

### Message Router

The router manages multiple providers and routes messages using pattern matching:

```go
router := provider.NewRouter(logger)
router.Register(discordProvider)
router.Register(telegramProvider)

// Route patterns
router.OnMessage(provider.All(), handler)           // All messages
router.OnMessage(provider.DMOnly(), handler)        // DMs only
router.OnMessage(provider.GroupOnly(), handler)     // Groups only
router.OnMessage(provider.FromProviders("discord"), handler)
```

### Providers

| Provider | Library | Features |
|----------|---------|----------|
| Discord | discordgo | Guilds, DMs, threads, media |
| Telegram | telebot | Groups, channels, private, media |
| WhatsApp | whatsmeow | QR auth, session persistence, media |

### WhatsApp Session Persistence

WhatsApp sessions are persisted using SQLite with a pure Go driver (modernc.org/sqlite), requiring no CGO:

```go
whatsapp.New(whatsapp.Config{
    DBPath: "whatsapp.db",
    QRCallback: func(qr string) {
        // Display QR code for scanning
    },
})
```

## Installation

```bash
go get github.com/agentplexus/omnichat
```

## Quick Start

See the [README](README.md) for complete usage examples and the `examples/echo/` directory for a working multi-provider bot.

## Dependencies

- `github.com/bwmarrin/discordgo` - Discord API
- `gopkg.in/telebot.v3` - Telegram Bot API
- `go.mau.fi/whatsmeow` - WhatsApp Web API
- `modernc.org/sqlite` - Pure Go SQLite driver

## Contributors

- Claude Opus 4.5 (AI pair programming)
