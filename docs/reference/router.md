# Router

The Router is the central component for managing providers and routing messages.

## Creating a Router

```go
import (
    "log/slog"
    "github.com/plexusone/omnichat/provider"
)

logger := slog.Default()
router := provider.NewRouter(logger)
```

## Registering Providers

```go
router.Register(discordProvider)
router.Register(telegramProvider)
router.Register(whatsappProvider)
```

## Connection Management

### Connect All Providers

```go
ctx := context.Background()
router.ConnectAll(ctx)
```

### Disconnect All Providers

```go
router.DisconnectAll(ctx)
```

### Connect Individual Provider

```go
router.Connect(ctx, "discord")
```

## Message Handling

### OnMessage

Register handlers for incoming messages:

```go
router.OnMessage(filter, handler)
```

**Parameters:**

- `filter` - A `FilterFunc` that determines which messages to handle
- `handler` - A `MessageHandler` function

**Example:**

```go
router.OnMessage(provider.All(), func(ctx context.Context, msg provider.IncomingMessage) error {
    log.Printf("Received: %s", msg.Content)
    return nil
})
```

## Filters

### All

Match all messages:

```go
router.OnMessage(provider.All(), handler)
```

### DMOnly

Match only direct messages:

```go
router.OnMessage(provider.DMOnly(), handler)
```

### GroupOnly

Match only group/channel messages:

```go
router.OnMessage(provider.GroupOnly(), handler)
```

### FromProviders

Match messages from specific providers:

```go
// Single provider
router.OnMessage(provider.FromProviders("discord"), handler)

// Multiple providers
router.OnMessage(provider.FromProviders("discord", "telegram"), handler)
```

### Custom Filters

Create custom filters:

```go
// Only messages containing "help"
helpFilter := func(msg provider.IncomingMessage) bool {
    return strings.Contains(strings.ToLower(msg.Content), "help")
}

router.OnMessage(helpFilter, helpHandler)
```

### Combining Filters

```go
// DMs from Discord only
discordDMs := func(msg provider.IncomingMessage) bool {
    return msg.ProviderName == "discord" && msg.IsDM
}

router.OnMessage(discordDMs, handler)
```

## Sending Messages

### Send

Send to a specific provider and chat:

```go
err := router.Send(ctx, providerName, chatID, message)
```

**Parameters:**

- `ctx` - Context for cancellation
- `providerName` - Provider identifier (e.g., "discord", "telegram")
- `chatID` - Chat/channel identifier
- `message` - `OutgoingMessage` to send

**Example:**

```go
router.Send(ctx, "discord", "123456789", provider.OutgoingMessage{
    Content: "Hello!",
})
```

### Broadcast

Send to multiple providers simultaneously:

```go
targets := map[string]string{
    "discord":  "123456789",
    "telegram": "987654321",
    "whatsapp": "1234567890@s.whatsapp.net",
}

router.Broadcast(ctx, targets, provider.OutgoingMessage{
    Content: "Announcement to all platforms!",
})
```

## Reply to Messages

Reply to the incoming message:

```go
router.OnMessage(provider.All(), func(ctx context.Context, msg provider.IncomingMessage) error {
    return router.Send(ctx, msg.ProviderName, msg.ChatID, provider.OutgoingMessage{
        Content: "This is a reply!",
        ReplyTo: msg.ID,
    })
})
```

## Agent Integration

### SetAgent

Set an agent for processing messages:

```go
type Agent interface {
    Process(ctx context.Context, input string) (string, error)
}

router.SetAgent(myAgent)
```

### ProcessWithVoice

Handle voice messages with transcription:

```go
router.OnMessage(provider.All(), router.ProcessWithVoice(voiceProcessor))
```

See [Voice Processing](voice.md) for details.

## Error Handling

Handlers can return errors:

```go
router.OnMessage(provider.All(), func(ctx context.Context, msg provider.IncomingMessage) error {
    if err := processMessage(msg); err != nil {
        return fmt.Errorf("processing failed: %w", err)
    }
    return nil
})
```

Errors are logged by the router but don't stop message processing.

## Multiple Handlers

Register multiple handlers for the same filter:

```go
// Logger handler
router.OnMessage(provider.All(), func(ctx context.Context, msg provider.IncomingMessage) error {
    log.Printf("[%s] %s: %s", msg.ProviderName, msg.SenderName, msg.Content)
    return nil
})

// Response handler
router.OnMessage(provider.All(), func(ctx context.Context, msg provider.IncomingMessage) error {
    return router.Send(ctx, msg.ProviderName, msg.ChatID, provider.OutgoingMessage{
        Content: "Got it!",
    })
})
```

Handlers are called in registration order.

## Provider Access

### Get Provider by Name

```go
p := router.GetProvider("discord")
if p != nil {
    // Use provider directly
}
```

### List All Providers

```go
providers := router.Providers()
for _, p := range providers {
    fmt.Println(p.Name())
}
```

## Context Values

The context passed to handlers may contain:

- Logger from router configuration
- Provider-specific metadata

```go
router.OnMessage(provider.All(), func(ctx context.Context, msg provider.IncomingMessage) error {
    // Access context values if needed
    return nil
})
```

## Best Practices

1. **Use specific filters** - Avoid `All()` when you only need certain messages
2. **Handle errors** - Return errors from handlers for logging
3. **Keep handlers fast** - Use goroutines for long-running operations
4. **Disconnect on shutdown** - Always call `DisconnectAll()` on exit

```go
// Graceful shutdown
sigCh := make(chan os.Signal, 1)
signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

go func() {
    <-sigCh
    router.DisconnectAll(ctx)
    os.Exit(0)
}()
```

## Next Steps

- [Voice Processing](voice.md) - Handle voice messages
- [Testing](testing.md) - Mock providers for tests
