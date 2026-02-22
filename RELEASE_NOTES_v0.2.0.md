# Release Notes: v0.2.0

**Release Date:** 2026-02-22

## Summary

This release adds voice message support, enabling transcription and synthesis integration for voice-enabled applications.

## Highlights

- **Voice message support** with transcription and synthesis integration

## New Features

### VoiceProcessor Interface

New `VoiceProcessor` interface for integrating speech-to-text (STT) and text-to-speech (TTS) providers:

```go
type VoiceProcessor interface {
    TranscribeAudio(ctx context.Context, audio []byte, mimeType string) (string, error)
    SynthesizeSpeech(ctx context.Context, text string) ([]byte, string, error)
    ResponseMode() string  // "auto", "always", "never"
}
```

### ProcessWithVoice Handler

New router handler that automatically:

- Transcribes incoming voice messages to text
- Processes the text through your agent
- Synthesizes the response back to voice
- Includes text alongside voice when URLs are detected (for clickable links)

```go
router.SetAgent(agent)
router.OnMessage(provider.All(), router.ProcessWithVoice(voiceProcessor))
```

### WhatsApp Audio Support

The WhatsApp provider now supports:

- **Receiving voice notes**: Automatically downloads audio from incoming voice messages
- **Sending voice notes**: Uploads and sends audio with PTT (Push-to-Talk) flag
- **Media type detection**: Distinguishes between `MediaTypeVoice` (PTT) and `MediaTypeAudio`

```go
// Send a voice note
router.Send(ctx, "whatsapp", chatID, provider.OutgoingMessage{
    Media: []provider.Media{{
        Type:     provider.MediaTypeVoice,
        Data:     audioBytes,
        MimeType: "audio/ogg; codecs=opus",
    }},
})
```

### Smart URL Handling

When a voice response contains URLs, both voice and text are sent so users can:

- Hear the response
- Click on links in the accompanying text message

## Upgrade Guide

This release is backwards compatible. To use voice features:

1. Implement the `VoiceProcessor` interface
2. Use `router.ProcessWithVoice(processor)` instead of `router.ProcessWithAgent()`

```bash
go get github.com/agentplexus/omnichat@v0.2.0
```
