package whatsapp

import (
	"context"
	"fmt"

	"go.mau.fi/whatsmeow/types"
)

// Session provides session management utilities for WhatsApp.
type Session struct {
	provider *Provider
}

// NewSession creates a session manager wrapping the provider.
func NewSession(p *Provider) *Session {
	return &Session{provider: p}
}

// Logout logs out from WhatsApp and clears the session.
func (s *Session) Logout(ctx context.Context) error {
	if s.provider.client == nil {
		return fmt.Errorf("not connected")
	}

	if err := s.provider.client.Logout(ctx); err != nil {
		return fmt.Errorf("logout: %w", err)
	}

	s.provider.logger.Info("whatsapp session logged out")
	return nil
}

// IsLoggedIn returns true if there's an active session.
func (s *Session) IsLoggedIn() bool {
	return s.provider.IsLoggedIn()
}

// GetPhoneNumber returns the phone number of the logged-in account.
func (s *Session) GetPhoneNumber() (string, error) {
	if s.provider.client == nil || s.provider.client.Store.ID == nil {
		return "", fmt.Errorf("not logged in")
	}

	return s.provider.client.Store.ID.User, nil
}

// SubscribePresence subscribes to presence updates for a contact.
func (s *Session) SubscribePresence(ctx context.Context, phoneNumber string) error {
	if s.provider.client == nil {
		return fmt.Errorf("not connected")
	}

	jid, err := parsePhoneNumber(phoneNumber)
	if err != nil {
		return fmt.Errorf("parse phone number: %w", err)
	}

	return s.provider.client.SubscribePresence(ctx, jid)
}

// parsePhoneNumber converts a phone number to a WhatsApp JID.
func parsePhoneNumber(phone string) (types.JID, error) {
	// Parse the phone number as a WhatsApp JID
	return types.ParseJID(phone + "@s.whatsapp.net")
}
