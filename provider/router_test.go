package provider

import (
	"context"
	"testing"
)

func TestContainsURL(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected bool
	}{
		{
			name:     "empty string",
			text:     "",
			expected: false,
		},
		{
			name:     "plain text",
			text:     "Hello, this is a message without any links.",
			expected: false,
		},
		{
			name:     "https URL",
			text:     "Check out https://example.com for more info.",
			expected: true,
		},
		{
			name:     "http URL",
			text:     "Visit http://example.com today.",
			expected: true,
		},
		{
			name:     "www URL",
			text:     "Go to www.example.com to learn more.",
			expected: true,
		},
		{
			name:     "URL at start",
			text:     "https://example.com is a great site.",
			expected: true,
		},
		{
			name:     "URL at end",
			text:     "The link is https://example.com",
			expected: true,
		},
		{
			name:     "multiple URLs",
			text:     "Visit https://one.com and https://two.com",
			expected: true,
		},
		{
			name:     "URL with path",
			text:     "See https://example.com/path/to/page for details.",
			expected: true,
		},
		{
			name:     "almost URL - missing colon",
			text:     "Not a URL https//example.com",
			expected: false,
		},
		{
			name:     "partial match - http only",
			text:     "The word http is not a URL",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsURL(tt.text)
			if got != tt.expected {
				t.Errorf("containsURL(%q) = %v, want %v", tt.text, got, tt.expected)
			}
		})
	}
}

// mockVoiceProcessor implements VoiceProcessor for testing.
type mockVoiceProcessor struct {
	transcribeFunc  func(ctx context.Context, audio []byte, mimeType string) (string, error)
	synthesizeFunc  func(ctx context.Context, text string) ([]byte, string, error)
	responseMode    string
	transcribeCalls int
	synthesizeCalls int
}

func (m *mockVoiceProcessor) TranscribeAudio(ctx context.Context, audio []byte, mimeType string) (string, error) {
	m.transcribeCalls++
	if m.transcribeFunc != nil {
		return m.transcribeFunc(ctx, audio, mimeType)
	}
	return "transcribed text", nil
}

func (m *mockVoiceProcessor) SynthesizeSpeech(ctx context.Context, text string) ([]byte, string, error) {
	m.synthesizeCalls++
	if m.synthesizeFunc != nil {
		return m.synthesizeFunc(ctx, text)
	}
	return []byte("audio data"), "audio/mpeg", nil
}

func (m *mockVoiceProcessor) ResponseMode() string {
	if m.responseMode == "" {
		return "auto"
	}
	return m.responseMode
}

// mockAgent implements AgentProcessor for testing.
type mockAgent struct {
	processFunc  func(ctx context.Context, sessionID, content string) (string, error)
	processCalls int
	lastContent  string
}

func (m *mockAgent) Process(ctx context.Context, sessionID, content string) (string, error) {
	m.processCalls++
	m.lastContent = content
	if m.processFunc != nil {
		return m.processFunc(ctx, sessionID, content)
	}
	return "agent response", nil
}

// mockProvider implements Provider for testing.
type mockProvider struct {
	name         string
	sentMessages []OutgoingMessage
}

func (m *mockProvider) Name() string                         { return m.name }
func (m *mockProvider) Connect(ctx context.Context) error    { return nil }
func (m *mockProvider) Disconnect(ctx context.Context) error { return nil }
func (m *mockProvider) Send(ctx context.Context, chatID string, msg OutgoingMessage) error {
	m.sentMessages = append(m.sentMessages, msg)
	return nil
}
func (m *mockProvider) OnMessage(handler MessageHandler) {}
func (m *mockProvider) OnEvent(handler EventHandler)     {}

func TestProcessWithVoice_TextMessage(t *testing.T) {
	router := NewRouter(nil)
	agent := &mockAgent{}
	voice := &mockVoiceProcessor{responseMode: "auto"}
	provider := &mockProvider{name: "test"}

	router.Register(provider)
	router.SetAgent(agent)

	handler := router.ProcessWithVoice(voice)

	msg := IncomingMessage{
		ID:           "1",
		ProviderName: "test",
		ChatID:       "chat1",
		Content:      "Hello",
	}

	err := handler(context.Background(), msg)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}

	// Text message should not trigger transcription
	if voice.transcribeCalls != 0 {
		t.Errorf("transcribeCalls = %d, want 0", voice.transcribeCalls)
	}

	// Agent should be called with original content
	if agent.lastContent != "Hello" {
		t.Errorf("agent content = %q, want %q", agent.lastContent, "Hello")
	}

	// Response should be text (auto mode, text input)
	if voice.synthesizeCalls != 0 {
		t.Errorf("synthesizeCalls = %d, want 0", voice.synthesizeCalls)
	}

	if len(provider.sentMessages) != 1 {
		t.Fatalf("sentMessages = %d, want 1", len(provider.sentMessages))
	}
	if provider.sentMessages[0].Content != "agent response" {
		t.Errorf("sent content = %q, want %q", provider.sentMessages[0].Content, "agent response")
	}
}

func TestProcessWithVoice_VoiceMessage(t *testing.T) {
	router := NewRouter(nil)
	agent := &mockAgent{}
	voice := &mockVoiceProcessor{responseMode: "auto"}
	provider := &mockProvider{name: "test"}

	router.Register(provider)
	router.SetAgent(agent)

	handler := router.ProcessWithVoice(voice)

	msg := IncomingMessage{
		ID:           "1",
		ProviderName: "test",
		ChatID:       "chat1",
		Media: []Media{{
			Type:     MediaTypeVoice,
			Data:     []byte("audio data"),
			MimeType: "audio/ogg",
		}},
	}

	err := handler(context.Background(), msg)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}

	// Voice message should trigger transcription
	if voice.transcribeCalls != 1 {
		t.Errorf("transcribeCalls = %d, want 1", voice.transcribeCalls)
	}

	// Agent should be called with transcribed content
	if agent.lastContent != "transcribed text" {
		t.Errorf("agent content = %q, want %q", agent.lastContent, "transcribed text")
	}

	// Response should be voice (auto mode, voice input)
	if voice.synthesizeCalls != 1 {
		t.Errorf("synthesizeCalls = %d, want 1", voice.synthesizeCalls)
	}

	if len(provider.sentMessages) != 1 {
		t.Fatalf("sentMessages = %d, want 1", len(provider.sentMessages))
	}
	if len(provider.sentMessages[0].Media) != 1 {
		t.Errorf("sent media count = %d, want 1", len(provider.sentMessages[0].Media))
	}
}

func TestProcessWithVoice_VoiceWithURL(t *testing.T) {
	router := NewRouter(nil)
	agent := &mockAgent{
		processFunc: func(ctx context.Context, sessionID, content string) (string, error) {
			return "Check out https://example.com for more info.", nil
		},
	}
	voice := &mockVoiceProcessor{responseMode: "auto"}
	provider := &mockProvider{name: "test"}

	router.Register(provider)
	router.SetAgent(agent)

	handler := router.ProcessWithVoice(voice)

	msg := IncomingMessage{
		ID:           "1",
		ProviderName: "test",
		ChatID:       "chat1",
		Media: []Media{{
			Type:     MediaTypeVoice,
			Data:     []byte("audio data"),
			MimeType: "audio/ogg",
		}},
	}

	err := handler(context.Background(), msg)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}

	// Should still synthesize voice
	if voice.synthesizeCalls != 1 {
		t.Errorf("synthesizeCalls = %d, want 1", voice.synthesizeCalls)
	}

	// Should include both voice and text (URL detected)
	if len(provider.sentMessages) != 1 {
		t.Fatalf("sentMessages = %d, want 1", len(provider.sentMessages))
	}
	sent := provider.sentMessages[0]
	if len(sent.Media) != 1 {
		t.Errorf("sent media count = %d, want 1", len(sent.Media))
	}
	if sent.Content == "" {
		t.Error("sent content is empty, want text with URL")
	}
}

func TestProcessWithVoice_AlwaysMode(t *testing.T) {
	router := NewRouter(nil)
	agent := &mockAgent{}
	voice := &mockVoiceProcessor{responseMode: "always"}
	provider := &mockProvider{name: "test"}

	router.Register(provider)
	router.SetAgent(agent)

	handler := router.ProcessWithVoice(voice)

	// Text message with "always" mode should still get voice response
	msg := IncomingMessage{
		ID:           "1",
		ProviderName: "test",
		ChatID:       "chat1",
		Content:      "Hello",
	}

	err := handler(context.Background(), msg)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}

	// Should synthesize even for text input
	if voice.synthesizeCalls != 1 {
		t.Errorf("synthesizeCalls = %d, want 1", voice.synthesizeCalls)
	}
}

func TestProcessWithVoice_NeverMode(t *testing.T) {
	router := NewRouter(nil)
	agent := &mockAgent{}
	voice := &mockVoiceProcessor{responseMode: "never"}
	provider := &mockProvider{name: "test"}

	router.Register(provider)
	router.SetAgent(agent)

	handler := router.ProcessWithVoice(voice)

	// Voice message with "never" mode should get text response
	msg := IncomingMessage{
		ID:           "1",
		ProviderName: "test",
		ChatID:       "chat1",
		Media: []Media{{
			Type:     MediaTypeVoice,
			Data:     []byte("audio data"),
			MimeType: "audio/ogg",
		}},
	}

	err := handler(context.Background(), msg)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}

	// Should transcribe but not synthesize
	if voice.transcribeCalls != 1 {
		t.Errorf("transcribeCalls = %d, want 1", voice.transcribeCalls)
	}
	if voice.synthesizeCalls != 0 {
		t.Errorf("synthesizeCalls = %d, want 0", voice.synthesizeCalls)
	}

	// Response should be text only
	if len(provider.sentMessages) != 1 {
		t.Fatalf("sentMessages = %d, want 1", len(provider.sentMessages))
	}
	if provider.sentMessages[0].Content == "" {
		t.Error("sent content is empty")
	}
	if len(provider.sentMessages[0].Media) != 0 {
		t.Errorf("sent media count = %d, want 0", len(provider.sentMessages[0].Media))
	}
}
