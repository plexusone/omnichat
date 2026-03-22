// Package gmail provides a Gmail email provider for omnichat using the Gmail API.
package gmail

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/grokify/goauth/google"
	"github.com/grokify/gogoogle/gmailutil/v1"

	"github.com/plexusone/omnichat/provider"
)

// Provider implements the Provider interface for Gmail.
type Provider struct {
	service        *gmailutil.GmailService
	httpClient     *http.Client
	config         Config
	logger         *slog.Logger
	messageHandler provider.MessageHandler
	eventHandler   provider.EventHandler
	fromAddress    string
}

// Config configures the Gmail provider.
type Config struct {
	// CredentialsJSON is the Google OAuth credentials JSON (client_secret.json contents).
	// This is the application credentials, not the user token.
	CredentialsJSON []byte

	// TokenFile is the path to store/load the OAuth token.
	// If empty, defaults to ~/.agentcomms/gmail_token.json
	TokenFile string

	// FromAddress is the email address to send from.
	// Use "me" to send from the authenticated user's address.
	FromAddress string

	// Scopes defines the OAuth scopes to request.
	// If empty, defaults to gmail.GmailSendScope.
	Scopes []string

	// Logger is the logger instance.
	Logger *slog.Logger

	// ForceNewToken forces re-authentication even if a token exists.
	ForceNewToken bool
}

// New creates a new Gmail provider.
func New(config Config) (*Provider, error) {
	if len(config.CredentialsJSON) == 0 {
		return nil, fmt.Errorf("gmail credentials JSON required")
	}
	if config.Logger == nil {
		config.Logger = slog.Default()
	}
	if config.FromAddress == "" {
		config.FromAddress = gmailutil.UserIDMe
	}
	if len(config.Scopes) == 0 {
		config.Scopes = []string{gmailutil.GmailSendScope}
	}
	if config.TokenFile == "" {
		config.TokenFile = "~/.agentcomms/gmail_token.json"
	}

	return &Provider{
		config:      config,
		logger:      config.Logger,
		fromAddress: config.FromAddress,
	}, nil
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "gmail"
}

// Connect establishes connection to Gmail API via OAuth.
func (p *Provider) Connect(ctx context.Context) error {
	// Create OAuth client using goauth
	client, err := google.NewClientOAuthCLITokenStore(google.ClientOAuthCLITokenStoreConfig{
		Context:       ctx,
		AppConfig:     p.config.CredentialsJSON,
		Scopes:        p.config.Scopes,
		TokenFile:     expandPath(p.config.TokenFile),
		ForceNewToken: p.config.ForceNewToken,
		State:         "gmail-omnichat",
	})
	if err != nil {
		return fmt.Errorf("gmail oauth: %w", err)
	}
	p.httpClient = client

	// Create Gmail service
	service, err := gmailutil.NewGmailService(ctx, client)
	if err != nil {
		return fmt.Errorf("gmail service: %w", err)
	}
	p.service = service

	p.logger.Info("gmail provider connected", "from", p.fromAddress)
	return nil
}

// Disconnect closes the Gmail connection.
func (p *Provider) Disconnect(ctx context.Context) error {
	// Gmail API doesn't have persistent connections to close
	p.service = nil
	p.httpClient = nil
	p.logger.Info("gmail provider disconnected")
	return nil
}

// Send sends an email to a recipient.
// The chatID is the recipient's email address.
func (p *Provider) Send(ctx context.Context, chatID string, msg provider.OutgoingMessage) error {
	if p.service == nil {
		return fmt.Errorf("gmail service not connected")
	}

	// chatID is the recipient email address
	toAddress := chatID

	// Build email options
	opts := gmailutil.SendSimpleOpts{
		To:      toAddress,
		Subject: extractSubject(msg),
		ReplyTo: msg.ReplyTo,
	}

	// Handle content format
	switch msg.Format {
	case provider.MessageFormatHTML:
		opts.BodyHTML = msg.Content
		opts.BodyText = stripHTML(msg.Content) // Fallback text version
	case provider.MessageFormatMarkdown:
		// Convert markdown to HTML (simplified - just wrap in pre for now)
		opts.BodyHTML = "<pre>" + msg.Content + "</pre>"
		opts.BodyText = msg.Content
	default:
		opts.BodyText = msg.Content
	}

	_, err := p.service.SendSimple(ctx, p.fromAddress, opts)
	if err != nil {
		return fmt.Errorf("send email: %w", err)
	}

	p.logger.Info("email sent", "to", toAddress, "subject", opts.Subject)
	return nil
}

// OnMessage registers a message handler.
// Note: Gmail inbound requires Watch API setup which is not yet implemented.
func (p *Provider) OnMessage(handler provider.MessageHandler) {
	p.messageHandler = handler
}

// OnEvent registers an event handler.
func (p *Provider) OnEvent(handler provider.EventHandler) {
	p.eventHandler = handler
}

// extractSubject extracts the email subject from message metadata or generates one.
func extractSubject(msg provider.OutgoingMessage) string {
	// Check metadata for explicit subject
	if msg.Metadata != nil {
		if subject, ok := msg.Metadata["subject"].(string); ok && subject != "" {
			return subject
		}
	}

	// Generate subject from content (first line or truncated)
	content := strings.TrimSpace(msg.Content)
	if content == "" {
		return "Message from AgentComms"
	}

	// Use first line as subject
	if idx := strings.Index(content, "\n"); idx > 0 {
		content = content[:idx]
	}

	// Truncate if too long
	if len(content) > 78 {
		content = content[:75] + "..."
	}

	return content
}

// stripHTML removes HTML tags for plain text fallback (simplified).
func stripHTML(html string) string {
	// Very basic HTML stripping - in production use a proper library
	result := html
	result = strings.ReplaceAll(result, "<br>", "\n")
	result = strings.ReplaceAll(result, "<br/>", "\n")
	result = strings.ReplaceAll(result, "<br />", "\n")
	result = strings.ReplaceAll(result, "</p>", "\n\n")
	result = strings.ReplaceAll(result, "</div>", "\n")

	// Remove remaining tags
	var inTag bool
	var out strings.Builder
	for _, r := range result {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			out.WriteRune(r)
		}
	}
	return strings.TrimSpace(out.String())
}

// expandPath expands ~ to home directory.
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		if home, err := userHomeDir(); err == nil {
			return home + path[1:]
		}
	}
	return path
}

// userHomeDir returns the user's home directory.
func userHomeDir() (string, error) {
	return os.UserHomeDir()
}

// SendEmail is a convenience method for sending emails with explicit parameters.
func (p *Provider) SendEmail(ctx context.Context, to, subject, body string, html bool) error {
	format := provider.MessageFormatPlain
	if html {
		format = provider.MessageFormatHTML
	}

	return p.Send(ctx, to, provider.OutgoingMessage{
		Content: body,
		Format:  format,
		Metadata: map[string]any{
			"subject": subject,
		},
	})
}

// IncomingEmail represents an email received (for future Watch API support).
type IncomingEmail struct {
	ID        string
	From      string
	To        string
	Subject   string
	Body      string
	BodyHTML  string
	Timestamp time.Time
	ThreadID  string
	Labels    []string
}

// Ensure Provider implements provider.Provider interface.
var _ provider.Provider = (*Provider)(nil)
