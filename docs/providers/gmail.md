# Gmail

The Gmail provider uses [gogoogle](https://github.com/grokify/gogoogle) to send emails via the Gmail API.

## Installation

```bash
go get github.com/plexusone/omnichat/providers/email/gmail
```

## Configuration

```go
import "github.com/plexusone/omnichat/providers/email/gmail"

p, err := gmail.New(gmail.Config{
    CredentialsFile: "credentials.json",
    TokenFile:       "token.json",
    FromAddress:     "bot@example.com",
    Logger:          slog.Default(),
})
```

### Config Options

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `CredentialsFile` | `string` | Yes | OAuth credentials JSON file |
| `TokenFile` | `string` | Yes | Token storage file |
| `FromAddress` | `string` | Yes | Sender email address |
| `Logger` | `*slog.Logger` | No | Logger instance |

## Setting Up OAuth

### 1. Create Google Cloud Project

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select existing
3. Enable the **Gmail API**

### 2. Create OAuth Credentials

1. Go to **APIs & Services > Credentials**
2. Click **Create Credentials > OAuth client ID**
3. Select **Desktop app** as application type
4. Download the JSON file as `credentials.json`

### 3. Configure OAuth Consent Screen

1. Go to **OAuth consent screen**
2. Add required scopes:
   - `https://www.googleapis.com/auth/gmail.send`
3. Add test users if in testing mode

### 4. First Run Authentication

On first run, the provider will:

1. Open a browser for OAuth consent
2. Store the token in `token.json`
3. Subsequent runs use the stored token

## Usage

### Basic Setup

```go
package main

import (
    "context"
    "log/slog"

    "github.com/plexusone/omnichat/provider"
    "github.com/plexusone/omnichat/providers/email/gmail"
)

func main() {
    logger := slog.Default()

    p, err := gmail.New(gmail.Config{
        CredentialsFile: "credentials.json",
        TokenFile:       "token.json",
        FromAddress:     "notifications@example.com",
        Logger:          logger,
    })
    if err != nil {
        panic(err)
    }

    router := provider.NewRouter(logger)
    router.Register(p)

    ctx := context.Background()
    router.ConnectAll(ctx)
    defer router.DisconnectAll(ctx)

    // Send an email
    router.Send(ctx, "gmail", "recipient@example.com", provider.OutgoingMessage{
        Content: "Hello from OmniChat!",
    })
}
```

### Sending Plain Text

```go
router.Send(ctx, "gmail", "user@example.com", provider.OutgoingMessage{
    Content: "This is a plain text email.",
})
```

### Sending HTML

The provider automatically detects HTML content:

```go
router.Send(ctx, "gmail", "user@example.com", provider.OutgoingMessage{
    Content: "<html><body><h1>Hello!</h1><p>This is HTML.</p></body></html>",
})
```

### Sending Markdown

Markdown is converted to HTML automatically:

```go
router.Send(ctx, "gmail", "user@example.com", provider.OutgoingMessage{
    Content: "# Hello\n\nThis is **bold** and this is *italic*.",
})
```

### Setting Subject

The subject is extracted from the first line or HTML title:

```go
// Subject from first line
router.Send(ctx, "gmail", "user@example.com", provider.OutgoingMessage{
    Content: "Weekly Report\n\nHere are the weekly metrics...",
})

// Subject from HTML title
router.Send(ctx, "gmail", "user@example.com", provider.OutgoingMessage{
    Content: "<html><head><title>Weekly Report</title></head><body>...</body></html>",
})
```

### Reply-To Header

Set a reply-to address different from the sender:

```go
router.Send(ctx, "gmail", "user@example.com", provider.OutgoingMessage{
    Content: "Please reply to support.",
    ReplyTo: "support@example.com",
})
```

## Message Mapping

| Gmail | OmniChat |
|-------|----------|
| To address | `ChatID` |
| Subject | Extracted from `Content` |
| Body | `Content` |
| Reply-To | `ReplyTo` |

## Email Format Detection

The provider detects content format automatically:

| Content | Format |
|---------|--------|
| Starts with `<html>` or `<!DOCTYPE` | HTML |
| Contains `#`, `**`, `*`, etc. | Markdown (converted to HTML) |
| Otherwise | Plain text |

## Environment Variables

```bash
GOOGLE_CREDENTIALS_FILE=credentials.json
GOOGLE_TOKEN_FILE=token.json
GMAIL_FROM_ADDRESS=bot@example.com
```

## OAuth Scopes

The provider requires this scope:

```
https://www.googleapis.com/auth/gmail.send
```

For read access (future feature), add:

```
https://www.googleapis.com/auth/gmail.readonly
```

## Troubleshooting

### OAuth errors

1. Verify `credentials.json` is valid
2. Delete `token.json` and re-authenticate
3. Check OAuth consent screen configuration

### Send failures

1. Verify sender address matches authenticated account
2. Check recipient address is valid
3. Review Gmail sending limits (500/day for consumer accounts)

### Token refresh issues

1. Delete `token.json`
2. Re-run authentication flow
3. Ensure refresh token is included

## Limitations

- Send-only (no incoming message support)
- One sender address per provider instance
- Subject extracted from content (no separate field)
- No attachment support yet

## Gmail vs Other Email Providers

Gmail provider is recommended for:

- Google Workspace accounts
- Personal Gmail accounts
- OAuth-based authentication

For SMTP-based sending, consider implementing a generic SMTP provider.

## Next Steps

- [Router](../reference/router.md) - Route messages across providers
- [Testing](../reference/testing.md) - Mock providers for tests
