# WhatsApp

The WhatsApp provider uses [whatsmeow](https://github.com/tulir/whatsmeow) to connect to WhatsApp Web.

## Installation

```bash
go get github.com/plexusone/omnichat/providers/whatsapp
```

## Configuration

```go
import "github.com/plexusone/omnichat/providers/whatsapp"

p, err := whatsapp.New(whatsapp.Config{
    DBPath: "whatsapp.db",
    Logger: slog.Default(),
    QRCallback: func(qr string) {
        // Display QR code for scanning
        qrterminal.GenerateHalfBlock(qr, qrterminal.L, os.Stdout)
    },
})
```

### Config Options

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `DBPath` | `string` | Yes | SQLite database path for session |
| `Logger` | `*slog.Logger` | No | Logger instance |
| `QRCallback` | `func(string)` | Yes | Called when QR code is available |

## Authentication

WhatsApp requires QR code authentication:

1. Run your application
2. Scan the QR code with WhatsApp mobile app
3. Session is stored in the database for future connections

```go
import "github.com/mdp/qrterminal/v3"

p, err := whatsapp.New(whatsapp.Config{
    DBPath: "whatsapp.db",
    Logger: logger,
    QRCallback: func(qr string) {
        fmt.Println("Scan this QR code with WhatsApp:")
        qrterminal.GenerateHalfBlock(qr, qrterminal.L, os.Stdout)
    },
})
```

## Usage

### Basic Setup

```go
package main

import (
    "context"
    "fmt"
    "log/slog"
    "os"

    "github.com/mdp/qrterminal/v3"
    "github.com/plexusone/omnichat/provider"
    "github.com/plexusone/omnichat/providers/whatsapp"
)

func main() {
    logger := slog.Default()

    p, err := whatsapp.New(whatsapp.Config{
        DBPath: "whatsapp.db",
        Logger: logger,
        QRCallback: func(qr string) {
            fmt.Println("Scan QR code:")
            qrterminal.GenerateHalfBlock(qr, qrterminal.L, os.Stdout)
        },
    })
    if err != nil {
        panic(err)
    }

    router := provider.NewRouter(logger)
    router.Register(p)

    router.OnMessage(provider.All(), func(ctx context.Context, msg provider.IncomingMessage) error {
        return router.Send(ctx, "whatsapp", msg.ChatID, provider.OutgoingMessage{
            Content: "Hello from WhatsApp!",
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
router.Send(ctx, "whatsapp", chatID, provider.OutgoingMessage{
    Content: "Check this out!",
    Media: []provider.Media{{
        Type:     provider.MediaTypeImage,
        Data:     imageBytes,
        MimeType: "image/jpeg",
    }},
})

// Send a document
router.Send(ctx, "whatsapp", chatID, provider.OutgoingMessage{
    Media: []provider.Media{{
        Type:     provider.MediaTypeDocument,
        Data:     pdfBytes,
        Filename: "document.pdf",
        MimeType: "application/pdf",
    }},
})
```

### Voice Messages

WhatsApp supports voice notes (PTT - Push to Talk):

```go
// Receive voice messages
router.OnMessage(provider.All(), func(ctx context.Context, msg provider.IncomingMessage) error {
    for _, media := range msg.Media {
        if media.Type == provider.MediaTypeVoice {
            // media.Data contains audio bytes
            // media.MimeType is typically "audio/ogg; codecs=opus"
            transcription := transcribe(media.Data)
            // Process transcription...
        }
    }
    return nil
})

// Send voice messages
router.Send(ctx, "whatsapp", chatID, provider.OutgoingMessage{
    Media: []provider.Media{{
        Type:     provider.MediaTypeVoice,
        Data:     audioBytes,
        MimeType: "audio/ogg; codecs=opus",
    }},
})
```

## Message Mapping

| WhatsApp | OmniChat |
|----------|----------|
| JID | `ChatID` |
| Message ID | `ID` |
| Sender JID | `SenderID` |
| Push Name | `SenderName` |
| Text/Caption | `Content` |
| Quoted Message | `ReplyTo` |
| Image/Document/Audio | `Media` |
| Chat type | `IsDM` (true for individual chats) |

## Chat ID Format

WhatsApp JIDs (Jabber IDs) follow this format:

- Individual: `1234567890@s.whatsapp.net`
- Group: `123456789-1234567890@g.us`
- Newsletter: `123456789@newsletter`
- Status: `status@broadcast`

```go
// Send to individual
router.Send(ctx, "whatsapp", "1234567890@s.whatsapp.net", msg)

// Send to group
router.Send(ctx, "whatsapp", "123456789-1234567890@g.us", msg)
```

## Chat Types

The provider detects different chat types automatically:

```go
router.OnMessage(provider.All(), func(ctx context.Context, msg provider.IncomingMessage) error {
    switch msg.ChatType {
    case provider.ChatTypeDM:
        // Individual chat
    case provider.ChatTypeGroup:
        // Group chat
    case provider.ChatTypeNewsletter:
        // WhatsApp Channel/Newsletter message
    case provider.ChatTypeStatus:
        // Status broadcast
    }
    return nil
})
```

## Newsletters (Channels)

WhatsApp Channels/Newsletters are one-way broadcast channels:

```go
// Cast to whatsapp provider for extended methods
waProvider := p.(*whatsapp.Provider)

// Get subscribed newsletters
newsletters, err := waProvider.GetNewsletters(ctx)
for _, nl := range newsletters {
    fmt.Printf("Newsletter: %s (%s)\n", nl.Name, nl.ID)
}

// Subscribe to a newsletter
err = waProvider.FollowNewsletter(ctx, "123456789@newsletter")

// Unsubscribe from a newsletter
err = waProvider.UnfollowNewsletter(ctx, "123456789@newsletter")
```

### Newsletter Events

```go
router.OnEvent(func(ctx context.Context, evt provider.Event) error {
    switch evt.Type {
    case provider.EventType("newsletter_join"):
        name := evt.Data["name"].(string)
        log.Printf("Joined newsletter: %s", name)
    case provider.EventType("newsletter_leave"):
        log.Printf("Left newsletter: %s", evt.ChatID)
    }
    return nil
})
```

## Reactions

Send and receive emoji reactions on messages:

```go
// Cast to whatsapp provider for extended methods
waProvider := p.(*whatsapp.Provider)

// Send a reaction
err := waProvider.SendReaction(ctx, chatID, messageID, "👍")

// Remove a reaction
err = waProvider.RemoveReaction(ctx, chatID, messageID)
```

### Receiving Reactions

```go
router.OnEvent(func(ctx context.Context, evt provider.Event) error {
    if evt.Type == provider.EventTypeReaction {
        messageID := evt.Data["message_id"].(string)
        reaction := evt.Data["reaction"].(string)
        added := evt.Data["added"].(bool)

        if added {
            log.Printf("Reaction %s added to message %s", reaction, messageID)
        } else {
            log.Printf("Reaction removed from message %s", messageID)
        }
    }
    return nil
})
```

## Session Management

The SQLite database stores:

- Authentication keys
- Message history (optional)
- Contact information

```go
// Different database per account
p1, _ := whatsapp.New(whatsapp.Config{
    DBPath: "account1.db",
    // ...
})

p2, _ := whatsapp.New(whatsapp.Config{
    DBPath: "account2.db",
    // ...
})
```

## Group Features

```go
router.OnMessage(provider.GroupOnly(), func(ctx context.Context, msg provider.IncomingMessage) error {
    // msg.ChatID is the group JID
    // msg.SenderID is the member who sent the message
    return nil
})
```

## Environment Variables

No tokens needed - authentication is via QR code.

```bash
WHATSAPP_DB_PATH=./data/whatsapp.db
```

## Troubleshooting

### QR code not appearing

1. Ensure `QRCallback` is set in config
2. Check terminal supports the QR output format
3. Try `qrterminal.Generate()` instead of `GenerateHalfBlock()`

### Session expired

1. Delete the database file
2. Re-scan QR code
3. WhatsApp Web sessions expire after ~14 days of inactivity

### Messages not sending

1. Verify chat ID format (must include `@s.whatsapp.net` or `@g.us`)
2. Check you're connected (`Connect()` completed successfully)
3. Verify the recipient hasn't blocked you

### Media download fails

1. Media URLs expire after some time
2. Download media immediately when received
3. Store locally if needed for later processing

## Limitations

- One phone number per connection
- WhatsApp may restrict automated messaging
- Media size limits apply
- No official API - uses reverse-engineered protocol
- Newsletter posting requires admin permissions (reading is supported)

## Next Steps

- [Slack](slack.md) - Add Slack support
- [Voice](../reference/voice.md) - Voice message transcription
