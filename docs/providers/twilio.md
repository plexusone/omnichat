# Twilio

The Twilio provider uses [omni-twilio](https://github.com/plexusone/omni-twilio) for SMS, MMS, and RCS messaging via Twilio's REST API.

## Installation

```bash
go get github.com/plexusone/omnichat/providers/twilio
```

## Configuration

```go
import "github.com/plexusone/omnichat/providers/twilio"

p, err := twilio.New(twilio.Config{
    AccountSID:  "ACxxxxxxxx",
    AuthToken:   "your-auth-token",
    PhoneNumber: "+15551234567",  // Your Twilio phone number
    Logger:      slog.Default(),
})
```

### Config Options

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `AccountSID` | `string` | Yes | Twilio Account SID |
| `AuthToken` | `string` | Yes | Twilio Auth Token |
| `PhoneNumber` | `string` | No | Default outbound phone number (E.164 format) |
| `MessagingServiceSid` | `string` | No | Messaging Service SID for RCS (enables RCS with SMS/MMS fallback) |
| `Logger` | `*slog.Logger` | No | Logger instance |

## Twilio Setup

1. Sign up at [Twilio](https://www.twilio.com/)
2. Get your Account SID and Auth Token from the Console
3. Buy or configure a phone number with SMS capability

### Finding Your Credentials

1. Go to [Twilio Console](https://console.twilio.com/)
2. Your **Account SID** and **Auth Token** are on the dashboard
3. Click **Phone Numbers** > **Manage** > **Active numbers** for your phone numbers

## Usage

### Basic Setup

```go
package main

import (
    "context"
    "log/slog"
    "net/http"
    "os"

    "github.com/plexusone/omnichat/provider"
    "github.com/plexusone/omnichat/providers/twilio"
)

func main() {
    logger := slog.Default()

    p, err := twilio.New(twilio.Config{
        AccountSID:  os.Getenv("TWILIO_ACCOUNT_SID"),
        AuthToken:   os.Getenv("TWILIO_AUTH_TOKEN"),
        PhoneNumber: os.Getenv("TWILIO_PHONE_NUMBER"),
        Logger:      logger,
    })
    if err != nil {
        panic(err)
    }

    router := provider.NewRouter(logger)
    router.Register(p)

    router.OnMessage(provider.All(), func(ctx context.Context, msg provider.IncomingMessage) error {
        return router.Send(ctx, "twilio", msg.SenderID, provider.OutgoingMessage{
            Content: "Thanks for your message!",
        })
    })

    ctx := context.Background()
    router.ConnectAll(ctx)
    defer router.DisconnectAll(ctx)

    // Set up webhook for incoming SMS
    http.Handle("/sms", p.WebhookHandler())
    http.ListenAndServe(":8080", nil)
}
```

### Sending SMS

```go
// Send SMS to a phone number
router.Send(ctx, "twilio", "+15559876543", provider.OutgoingMessage{
    Content: "Hello from OmniChat!",
})
```

### Sending MMS

Send media attachments (images, video, audio) via MMS:

```go
// Send MMS with an image
router.Send(ctx, "twilio", "+15559876543", provider.OutgoingMessage{
    Content: "Check out this photo!",
    Media: []provider.Media{
        {
            Type: provider.MediaTypeImage,
            URL:  "https://example.com/photo.jpg",
        },
    },
})

// Send multiple media attachments
router.Send(ctx, "twilio", "+15559876543", provider.OutgoingMessage{
    Content: "Here are the documents",
    Media: []provider.Media{
        {Type: provider.MediaTypeImage, URL: "https://example.com/image.jpg"},
        {Type: provider.MediaTypeDocument, URL: "https://example.com/doc.pdf"},
    },
})
```

**Note**: MMS requires a URL-accessible media file. The URL must be publicly accessible by Twilio's servers.

### Sending RCS

RCS (Rich Communication Services) provides rich messaging with automatic fallback to SMS/MMS:

```go
import "github.com/plexusone/omnichat/providers/twilio"

// Enable RCS by setting MessagingServiceSid
p, err := twilio.New(twilio.Config{
    AccountSID:          os.Getenv("TWILIO_ACCOUNT_SID"),
    AuthToken:           os.Getenv("TWILIO_AUTH_TOKEN"),
    MessagingServiceSid: os.Getenv("TWILIO_MESSAGING_SERVICE_SID"), // RCS-enabled
    Logger:              logger,
})

// Send message - RCS attempted, falls back to SMS/MMS if unavailable
router.Send(ctx, "twilio", "+15559876543", provider.OutgoingMessage{
    Content: "Hello via RCS!",
})

// Send RCS with content template (rich cards, carousels)
router.Send(ctx, "twilio", "+15559876543", provider.OutgoingMessage{
    Content: "Order update",
    Metadata: map[string]any{
        "content_sid":       "HXxxxxxxxx", // Pre-created content template
        "content_variables": `{"1": "John", "2": "#12345"}`,
    },
})
```

**RCS Setup:**

1. Create a Messaging Service in the Twilio Console
2. Add an RCS sender to your Messaging Service (requires carrier approval)
3. Set `MessagingServiceSid` in your configuration

**RCS Features:**

- Branded sender identity
- Rich cards and carousels (via Content API templates)
- Suggested replies and actions
- Read receipts and typing indicators
- Automatic fallback to SMS/MMS when RCS unavailable

### Receiving SMS/MMS

Incoming SMS and MMS messages are received via Twilio webhooks. Configure your Twilio phone number to POST to your webhook endpoint:

1. Go to **Phone Numbers** > **Manage** > **Active numbers**
2. Click your phone number
3. Under **Messaging**, set the webhook URL to your endpoint (e.g., `https://yourapp.com/sms`)

```go
// The webhook handler is provided by the provider
http.Handle("/sms", p.WebhookHandler())
```

### Handling Incoming Media (MMS)

When receiving MMS messages, media attachments are automatically extracted:

```go
router.OnMessage(provider.All(), func(ctx context.Context, msg provider.IncomingMessage) error {
    // Check for media attachments
    for _, media := range msg.Media {
        fmt.Printf("Received %s: %s\n", media.Type, media.URL)
        fmt.Printf("MIME type: %s\n", media.MimeType)
    }

    // Text content is still available
    fmt.Println("Message:", msg.Content)
    return nil
})
```

Media types supported:

| Type | Description |
|------|-------------|
| `MediaTypeImage` | Images (JPEG, PNG, GIF, etc.) |
| `MediaTypeVideo` | Videos (MP4, etc.) |
| `MediaTypeAudio` | Audio files (MP3, WAV, etc.) |
| `MediaTypeDocument` | Other files (PDF, etc.) |

## Message Mapping

| Twilio | OmniChat |
|--------|----------|
| MessageSid | `ID` |
| From | `SenderID` |
| To | `ChatID` |
| Body | `Content` |
| MediaUrl0, MediaUrl1, ... | `Media` (slice of attachments) |
| MediaContentType0, ... | `Media[].MimeType` |
| NumMedia | Number of items in `Media` |

## Environment Variables

```bash
TWILIO_ACCOUNT_SID=ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
TWILIO_AUTH_TOKEN=your_auth_token_here
TWILIO_PHONE_NUMBER=+15551234567
TWILIO_MESSAGING_SERVICE_SID=MGxxxxxxxx  # Optional: for RCS
```

## Webhook Security

For production, validate incoming webhooks using Twilio's request signature:

```go
// The webhook handler validates signatures when AuthToken is set
```

## Troubleshooting

### SMS not sending

1. Verify Account SID and Auth Token are correct
2. Check phone number is SMS-capable
3. Verify recipient number format (E.164: `+15551234567`)
4. Check Twilio Console for error logs

### Webhooks not receiving

1. Verify webhook URL is publicly accessible
2. Check Twilio Console for webhook errors
3. Ensure endpoint returns 200 status code
4. Verify TLS certificate is valid (Twilio requires HTTPS)

### Phone number format

Always use E.164 format:

- Correct: `+15551234567`
- Incorrect: `555-123-4567`, `(555) 123-4567`

## Limitations

- SMS character limits apply (160 for GSM-7, 70 for Unicode per segment)
- MMS media must be URL-accessible (no direct byte uploads)
- MMS availability depends on carrier and region support
- Maximum 10 media attachments per MMS

## Next Steps

- [IRC](irc.md) - Add IRC support
- [Router](../reference/router.md) - Message routing
