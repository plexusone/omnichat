# Twilio

The Twilio provider uses [twilio-go](https://github.com/plexusone/twilio-go) for SMS messaging via Twilio's REST API.

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
// Send to a phone number
router.Send(ctx, "twilio", "+15559876543", provider.OutgoingMessage{
    Content: "Hello from OmniChat!",
})
```

### Receiving SMS

Incoming SMS messages are received via Twilio webhooks. Configure your Twilio phone number to POST to your webhook endpoint:

1. Go to **Phone Numbers** > **Manage** > **Active numbers**
2. Click your phone number
3. Under **Messaging**, set the webhook URL to your endpoint (e.g., `https://yourapp.com/sms`)

```go
// The webhook handler is provided by the provider
http.Handle("/sms", p.WebhookHandler())
```

## Message Mapping

| Twilio | OmniChat |
|--------|----------|
| MessageSid | `ID` |
| From | `SenderID` |
| To | `ChatID` |
| Body | `Content` |

## Environment Variables

```bash
TWILIO_ACCOUNT_SID=ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
TWILIO_AUTH_TOKEN=your_auth_token_here
TWILIO_PHONE_NUMBER=+15551234567
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

- SMS only (no MMS support through OmniChat interface)
- No media attachments
- Character limits apply (160 for GSM-7, 70 for Unicode)

## Next Steps

- [IRC](irc.md) - Add IRC support
- [Router](../reference/router.md) - Message routing
