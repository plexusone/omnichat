// Package twilio provides a Twilio SMS provider for omnichat.
//
// This package wraps github.com/plexusone/omni-twilio/omnichat to provide
// a consistent interface with other omnichat providers.
package twilio

import (
	"log/slog"

	twiliosms "github.com/plexusone/omni-twilio/omnichat"
	"github.com/plexusone/omnichat/provider"
)

// Provider is an alias for the Twilio SMS provider.
type Provider = twiliosms.Provider

// Config configures the Twilio SMS provider.
type Config struct {
	// AccountSID is the Twilio Account SID.
	AccountSID string

	// AuthToken is the Twilio Auth Token.
	AuthToken string

	// PhoneNumber is the default outbound phone number in E.164 format.
	PhoneNumber string

	// Logger is the logger instance.
	Logger *slog.Logger
}

// New creates a new Twilio SMS provider.
func New(config Config) (*Provider, error) {
	opts := []twiliosms.Option{
		twiliosms.WithAccountSID(config.AccountSID),
		twiliosms.WithAuthToken(config.AuthToken),
	}

	if config.PhoneNumber != "" {
		opts = append(opts, twiliosms.WithPhoneNumber(config.PhoneNumber))
	}

	if config.Logger != nil {
		opts = append(opts, twiliosms.WithLogger(config.Logger))
	}

	return twiliosms.New(opts...)
}

// Ensure Provider implements provider.Provider interface.
var _ provider.Provider = (*Provider)(nil)
