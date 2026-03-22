# Voice Processing

OmniChat supports voice message handling with transcription and synthesis.

## Overview

Voice processing enables:

- **Transcription** - Convert incoming voice messages to text
- **Synthesis** - Convert text responses to voice notes
- **Smart routing** - Respond with voice to voice input

## VoiceProcessor Interface

```go
type VoiceProcessor interface {
    // TranscribeAudio converts audio to text
    TranscribeAudio(ctx context.Context, audio []byte, mimeType string) (string, error)

    // SynthesizeSpeech converts text to audio
    SynthesizeSpeech(ctx context.Context, text string) ([]byte, string, error)

    // ResponseMode controls when to use voice responses
    // "auto" - voice reply to voice input
    // "always" - always respond with voice
    // "never" - never respond with voice
    ResponseMode() string
}
```

## Basic Usage

```go
// Implement VoiceProcessor
type MyVoiceProcessor struct {
    // STT/TTS clients
}

func (p *MyVoiceProcessor) TranscribeAudio(ctx context.Context, audio []byte, mimeType string) (string, error) {
    // Use your STT service
    return transcribe(audio, mimeType)
}

func (p *MyVoiceProcessor) SynthesizeSpeech(ctx context.Context, text string) ([]byte, string, error) {
    audio, err := synthesize(text)
    return audio, "audio/ogg; codecs=opus", err
}

func (p *MyVoiceProcessor) ResponseMode() string {
    return "auto"
}

// Use with router
router.SetAgent(agent)
router.OnMessage(provider.All(), router.ProcessWithVoice(voiceProcessor))
```

## ProcessWithVoice Flow

```
Incoming Message
       │
       ▼
┌──────────────┐
│ Has Voice?   │──No──▶ Process as text
└──────┬───────┘
       │ Yes
       ▼
┌──────────────┐
│ Transcribe   │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ Process text │
│ (via Agent)  │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ ResponseMode │
└──────┬───────┘
       │
  ┌────┴────┐
  │         │
auto     always/never
  │         │
  ▼         ▼
Voice    Based on mode
reply
```

## Response Modes

### Auto Mode

Respond with voice only when the input was voice:

```go
func (p *MyVoiceProcessor) ResponseMode() string {
    return "auto"
}
```

- Voice input → Voice response
- Text input → Text response

### Always Mode

Always respond with voice:

```go
func (p *MyVoiceProcessor) ResponseMode() string {
    return "always"
}
```

### Never Mode

Never respond with voice (transcription only):

```go
func (p *MyVoiceProcessor) ResponseMode() string {
    return "never"
}
```

## Audio Formats

### Incoming Audio

Common formats from platforms:

| Platform | Format | MIME Type |
|----------|--------|-----------|
| WhatsApp | Opus in Ogg | `audio/ogg; codecs=opus` |
| Telegram | Ogg Opus | `audio/ogg` |
| Discord | Opus | `audio/opus` |

### Outgoing Audio

Recommended format for voice notes:

```go
func (p *MyVoiceProcessor) SynthesizeSpeech(ctx context.Context, text string) ([]byte, string, error) {
    audio, err := synthesize(text, "ogg_opus")
    return audio, "audio/ogg; codecs=opus", err
}
```

## URL Handling

When responses contain URLs, both voice and text are sent:

```go
// Response: "Check out https://example.com"
// Results in:
// 1. Voice note with "Check out example.com"
// 2. Text message with clickable URL
```

This ensures URLs remain clickable while providing voice response.

## Integration with STT/TTS Services

### Google Cloud Speech

```go
import (
    speech "cloud.google.com/go/speech/apiv1"
    texttospeech "cloud.google.com/go/texttospeech/apiv1"
)

type GoogleVoiceProcessor struct {
    sttClient *speech.Client
    ttsClient *texttospeech.Client
}

func (p *GoogleVoiceProcessor) TranscribeAudio(ctx context.Context, audio []byte, mimeType string) (string, error) {
    req := &speechpb.RecognizeRequest{
        Config: &speechpb.RecognitionConfig{
            Encoding:        speechpb.RecognitionConfig_OGG_OPUS,
            LanguageCode:    "en-US",
        },
        Audio: &speechpb.RecognitionAudio{
            AudioSource: &speechpb.RecognitionAudio_Content{Content: audio},
        },
    }
    resp, err := p.sttClient.Recognize(ctx, req)
    // Extract transcript...
}
```

### OpenAI Whisper

```go
type OpenAIVoiceProcessor struct {
    client *openai.Client
}

func (p *OpenAIVoiceProcessor) TranscribeAudio(ctx context.Context, audio []byte, mimeType string) (string, error) {
    resp, err := p.client.CreateTranscription(ctx, openai.AudioRequest{
        Model:    openai.Whisper1,
        Reader:   bytes.NewReader(audio),
        FilePath: "audio.ogg",
    })
    return resp.Text, err
}
```

### ElevenLabs TTS

```go
type ElevenLabsVoiceProcessor struct {
    apiKey  string
    voiceID string
}

func (p *ElevenLabsVoiceProcessor) SynthesizeSpeech(ctx context.Context, text string) ([]byte, string, error) {
    // Call ElevenLabs API
    audio, err := elevenlabs.TextToSpeech(p.apiKey, p.voiceID, text)
    return audio, "audio/mpeg", err
}
```

## Receiving Voice Messages

Voice messages arrive in the `Media` field:

```go
router.OnMessage(provider.All(), func(ctx context.Context, msg provider.IncomingMessage) error {
    for _, media := range msg.Media {
        if media.Type == provider.MediaTypeVoice {
            // media.Data - audio bytes
            // media.MimeType - format info
            transcript, err := voiceProcessor.TranscribeAudio(ctx, media.Data, media.MimeType)
            if err != nil {
                return err
            }
            log.Printf("Voice message said: %s", transcript)
        }
    }
    return nil
})
```

## Sending Voice Messages

```go
audio, mimeType, err := voiceProcessor.SynthesizeSpeech(ctx, "Hello!")
if err != nil {
    return err
}

router.Send(ctx, "whatsapp", chatID, provider.OutgoingMessage{
    Media: []provider.Media{{
        Type:     provider.MediaTypeVoice,
        Data:     audio,
        MimeType: mimeType,
    }},
})
```

## Platform Support

| Platform | Receive Voice | Send Voice |
|----------|--------------|------------|
| WhatsApp | Yes | Yes (PTT) |
| Telegram | Yes | Yes |
| Discord | Limited | Limited |
| Slack | No | No |
| Gmail | No | No |

## Best Practices

1. **Handle transcription errors gracefully**

```go
transcript, err := processor.TranscribeAudio(ctx, audio, mimeType)
if err != nil {
    // Fall back to asking user to type
    return router.Send(ctx, provider, chatID, provider.OutgoingMessage{
        Content: "Sorry, I couldn't understand the audio. Please type your message.",
    })
}
```

2. **Limit audio length**

```go
const maxAudioBytes = 10 * 1024 * 1024 // 10MB

if len(audio) > maxAudioBytes {
    return errors.New("audio too long")
}
```

3. **Cache synthesized audio**

```go
var audioCache = make(map[string][]byte)

func (p *CachedProcessor) SynthesizeSpeech(ctx context.Context, text string) ([]byte, string, error) {
    if cached, ok := audioCache[text]; ok {
        return cached, "audio/ogg", nil
    }
    audio, mimeType, err := p.underlying.SynthesizeSpeech(ctx, text)
    if err == nil {
        audioCache[text] = audio
    }
    return audio, mimeType, err
}
```

## Next Steps

- [Testing](testing.md) - Mock voice processor for tests
- [WhatsApp](../providers/whatsapp.md) - WhatsApp voice notes
