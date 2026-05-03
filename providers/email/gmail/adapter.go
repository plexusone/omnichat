// Package gmail provides a Gmail email provider for omnichat.
//
// This package wraps github.com/plexusone/omni-google/omnichat/gmail to provide
// a consistent interface with other omnichat providers.
package gmail

import (
	"log/slog"

	omnigmail "github.com/plexusone/omni-google/omnichat/gmail"
	"github.com/plexusone/omnichat/provider"
)

// Provider is an alias for the Gmail provider.
type Provider = omnigmail.Provider

// Config configures the Gmail provider.
type Config struct {
	// CredentialsJSON is the Google OAuth credentials JSON (client_secret.json contents).
	// This is the application credentials, not the user token.
	CredentialsJSON []byte

	// TokenFile is the path to store/load the OAuth token.
	// If empty, defaults to ~/.omnichat/gmail_token.json
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
	opts := []omnigmail.Option{
		omnigmail.WithCredentialsJSON(config.CredentialsJSON),
	}

	if config.TokenFile != "" {
		opts = append(opts, omnigmail.WithTokenFile(config.TokenFile))
	}

	if config.FromAddress != "" {
		opts = append(opts, omnigmail.WithFromAddress(config.FromAddress))
	}

	if len(config.Scopes) > 0 {
		opts = append(opts, omnigmail.WithScopes(config.Scopes))
	}

	if config.Logger != nil {
		opts = append(opts, omnigmail.WithLogger(config.Logger))
	}

	if config.ForceNewToken {
		opts = append(opts, omnigmail.WithForceNewToken(true))
	}

	return omnigmail.New(opts...)
}

// IncomingEmail is an alias for email data (for future Watch API support).
type IncomingEmail = omnigmail.IncomingEmail

// Ensure Provider implements provider.Provider interface.
var _ provider.Provider = (*Provider)(nil)
