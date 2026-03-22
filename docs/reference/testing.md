# Testing

OmniChat provides a mock provider for unit testing.

## MockProvider

The mock provider simulates a messaging platform without external connections.

```go
import "github.com/plexusone/omnichat/provider/providertest"

mock := providertest.NewMockProvider("test")
```

## Basic Usage

```go
package myapp

import (
    "context"
    "testing"

    "github.com/plexusone/omnichat/provider"
    "github.com/plexusone/omnichat/provider/providertest"
)

func TestMessageHandler(t *testing.T) {
    // Create mock provider
    mock := providertest.NewMockProvider("test")

    // Create router with mock
    router := provider.NewRouter(nil)
    router.Register(mock)

    // Register handler
    router.OnMessage(provider.All(), func(ctx context.Context, msg provider.IncomingMessage) error {
        return router.Send(ctx, "test", msg.ChatID, provider.OutgoingMessage{
            Content: "Echo: " + msg.Content,
        })
    })

    // Connect
    ctx := context.Background()
    router.ConnectAll(ctx)

    // Simulate incoming message
    mock.SimulateMessage(ctx, provider.IncomingMessage{
        ID:           "msg-1",
        ProviderName: "test",
        ChatID:       "chat-123",
        SenderID:     "user-1",
        SenderName:   "Test User",
        Content:      "Hello",
    })

    // Check sent messages
    sent := mock.SentMessages()
    if len(sent) != 1 {
        t.Errorf("expected 1 sent message, got %d", len(sent))
    }
    if sent[0].Content != "Echo: Hello" {
        t.Errorf("unexpected content: %s", sent[0].Content)
    }
}
```

## MockProvider Methods

### SimulateMessage

Simulate an incoming message:

```go
mock.SimulateMessage(ctx, provider.IncomingMessage{
    ID:           "msg-1",
    ProviderName: "test",
    ChatID:       "chat-123",
    SenderID:     "user-1",
    SenderName:   "Alice",
    Content:      "Hello, world!",
    IsDM:         true,
})
```

### SentMessages

Get all messages sent through the mock:

```go
sent := mock.SentMessages()
for _, msg := range sent {
    fmt.Printf("Sent to %s: %s\n", msg.ChatID, msg.Content)
}
```

### ClearSentMessages

Reset the sent messages list:

```go
mock.ClearSentMessages()
```

### LastSentMessage

Get the most recent sent message:

```go
last := mock.LastSentMessage()
if last != nil {
    fmt.Println(last.Content)
}
```

## Testing Patterns

### Test Echo Bot

```go
func TestEchoBot(t *testing.T) {
    mock := providertest.NewMockProvider("test")
    router := provider.NewRouter(nil)
    router.Register(mock)

    router.OnMessage(provider.All(), func(ctx context.Context, msg provider.IncomingMessage) error {
        return router.Send(ctx, msg.ProviderName, msg.ChatID, provider.OutgoingMessage{
            Content: "Echo: " + msg.Content,
        })
    })

    ctx := context.Background()
    router.ConnectAll(ctx)

    // Test cases
    tests := []struct {
        input    string
        expected string
    }{
        {"Hello", "Echo: Hello"},
        {"World", "Echo: World"},
        {"", "Echo: "},
    }

    for _, tc := range tests {
        mock.ClearSentMessages()
        mock.SimulateMessage(ctx, provider.IncomingMessage{
            ChatID:  "chat-1",
            Content: tc.input,
        })

        sent := mock.LastSentMessage()
        if sent == nil || sent.Content != tc.expected {
            t.Errorf("input %q: expected %q, got %v", tc.input, tc.expected, sent)
        }
    }
}
```

### Test DM-Only Handler

```go
func TestDMOnlyHandler(t *testing.T) {
    mock := providertest.NewMockProvider("test")
    router := provider.NewRouter(nil)
    router.Register(mock)

    router.OnMessage(provider.DMOnly(), func(ctx context.Context, msg provider.IncomingMessage) error {
        return router.Send(ctx, msg.ProviderName, msg.ChatID, provider.OutgoingMessage{
            Content: "DM received",
        })
    })

    ctx := context.Background()
    router.ConnectAll(ctx)

    // DM message - should be handled
    mock.SimulateMessage(ctx, provider.IncomingMessage{
        ChatID:  "dm-1",
        Content: "Hello",
        IsDM:    true,
    })

    if len(mock.SentMessages()) != 1 {
        t.Error("DM should be handled")
    }

    mock.ClearSentMessages()

    // Group message - should be ignored
    mock.SimulateMessage(ctx, provider.IncomingMessage{
        ChatID:  "group-1",
        Content: "Hello",
        IsDM:    false,
    })

    if len(mock.SentMessages()) != 0 {
        t.Error("Group message should be ignored")
    }
}
```

### Test Media Handling

```go
func TestMediaHandler(t *testing.T) {
    mock := providertest.NewMockProvider("test")
    router := provider.NewRouter(nil)
    router.Register(mock)

    var receivedMedia []provider.Media

    router.OnMessage(provider.All(), func(ctx context.Context, msg provider.IncomingMessage) error {
        receivedMedia = msg.Media
        return nil
    })

    ctx := context.Background()
    router.ConnectAll(ctx)

    mock.SimulateMessage(ctx, provider.IncomingMessage{
        ChatID:  "chat-1",
        Content: "Check this out",
        Media: []provider.Media{{
            Type:     provider.MediaTypeImage,
            URL:      "https://example.com/image.png",
            MimeType: "image/png",
        }},
    })

    if len(receivedMedia) != 1 {
        t.Error("expected 1 media item")
    }
    if receivedMedia[0].Type != provider.MediaTypeImage {
        t.Error("expected image type")
    }
}
```

### Test Error Handling

```go
func TestErrorHandler(t *testing.T) {
    mock := providertest.NewMockProvider("test")
    router := provider.NewRouter(nil)
    router.Register(mock)

    var handlerCalled bool

    router.OnMessage(provider.All(), func(ctx context.Context, msg provider.IncomingMessage) error {
        handlerCalled = true
        return errors.New("simulated error")
    })

    ctx := context.Background()
    router.ConnectAll(ctx)

    mock.SimulateMessage(ctx, provider.IncomingMessage{
        ChatID:  "chat-1",
        Content: "Hello",
    })

    if !handlerCalled {
        t.Error("handler should be called")
    }
    // Router logs error but continues processing
}
```

## Testing Multiple Providers

```go
func TestMultiProvider(t *testing.T) {
    discord := providertest.NewMockProvider("discord")
    telegram := providertest.NewMockProvider("telegram")

    router := provider.NewRouter(nil)
    router.Register(discord)
    router.Register(telegram)

    router.OnMessage(provider.FromProviders("discord"), func(ctx context.Context, msg provider.IncomingMessage) error {
        return router.Send(ctx, "discord", msg.ChatID, provider.OutgoingMessage{
            Content: "Discord response",
        })
    })

    router.OnMessage(provider.FromProviders("telegram"), func(ctx context.Context, msg provider.IncomingMessage) error {
        return router.Send(ctx, "telegram", msg.ChatID, provider.OutgoingMessage{
            Content: "Telegram response",
        })
    })

    ctx := context.Background()
    router.ConnectAll(ctx)

    // Test Discord
    discord.SimulateMessage(ctx, provider.IncomingMessage{
        ProviderName: "discord",
        ChatID:       "ch-1",
        Content:      "Hello",
    })

    if discord.LastSentMessage().Content != "Discord response" {
        t.Error("wrong Discord response")
    }

    // Test Telegram
    telegram.SimulateMessage(ctx, provider.IncomingMessage{
        ProviderName: "telegram",
        ChatID:       "ch-2",
        Content:      "Hello",
    })

    if telegram.LastSentMessage().Content != "Telegram response" {
        t.Error("wrong Telegram response")
    }
}
```

## Mock Voice Processor

```go
type MockVoiceProcessor struct {
    TranscribeFunc  func(audio []byte) string
    SynthesizeFunc  func(text string) []byte
    Mode            string
}

func (m *MockVoiceProcessor) TranscribeAudio(ctx context.Context, audio []byte, mimeType string) (string, error) {
    if m.TranscribeFunc != nil {
        return m.TranscribeFunc(audio), nil
    }
    return "transcribed text", nil
}

func (m *MockVoiceProcessor) SynthesizeSpeech(ctx context.Context, text string) ([]byte, string, error) {
    if m.SynthesizeFunc != nil {
        return m.SynthesizeFunc(text), "audio/ogg", nil
    }
    return []byte("audio"), "audio/ogg", nil
}

func (m *MockVoiceProcessor) ResponseMode() string {
    if m.Mode != "" {
        return m.Mode
    }
    return "auto"
}
```

## Best Practices

1. **Clear state between tests**

```go
func TestSomething(t *testing.T) {
    mock.ClearSentMessages()
    // ... test code
}
```

2. **Use table-driven tests**

```go
tests := []struct {
    name     string
    input    provider.IncomingMessage
    expected string
}{
    // ... test cases
}

for _, tc := range tests {
    t.Run(tc.name, func(t *testing.T) {
        // ... test code
    })
}
```

3. **Test edge cases**

```go
// Empty message
mock.SimulateMessage(ctx, provider.IncomingMessage{
    Content: "",
})

// Unicode content
mock.SimulateMessage(ctx, provider.IncomingMessage{
    Content: "Hello 世界 🌍",
})
```

## Next Steps

- [Router](router.md) - Router reference
- [Voice](voice.md) - Voice processing
