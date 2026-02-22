// Package whatsapp provides a WhatsApp provider for omnichat using whatsmeow.
package whatsapp

import (
	"context"
	"fmt"
	"log/slog"

	_ "modernc.org/sqlite" // Pure-Go SQLite driver for session storage

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"

	"github.com/agentplexus/omnichat/provider"
)

// Provider implements the Provider interface for WhatsApp.
type Provider struct {
	client         *whatsmeow.Client
	container      *sqlstore.Container
	logger         *slog.Logger
	messageHandler provider.MessageHandler
	eventHandler   provider.EventHandler
	qrCallback     func(qr string)
	dbPath         string
}

// Config configures the WhatsApp provider.
type Config struct {
	// DBPath is the path to the SQLite database for session storage.
	// Defaults to "whatsapp.db" if empty.
	DBPath string

	// Logger for logging events.
	Logger *slog.Logger

	// QRCallback is called when a QR code needs to be displayed for authentication.
	// The callback receives the QR code string that should be rendered.
	QRCallback func(qr string)
}

// New creates a new WhatsApp provider.
func New(config Config) (*Provider, error) {
	if config.Logger == nil {
		config.Logger = slog.Default()
	}
	if config.DBPath == "" {
		config.DBPath = "whatsapp.db"
	}

	return &Provider{
		logger:     config.Logger,
		qrCallback: config.QRCallback,
		dbPath:     config.DBPath,
	}, nil
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "whatsapp"
}

// Connect establishes connection to WhatsApp.
func (p *Provider) Connect(ctx context.Context) error {
	// Create a whatsmeow logger adapter
	wmLog := waLog.Stdout("whatsmeow", "INFO", true)

	// Initialize the SQL store for credentials
	container, err := sqlstore.New(ctx, "sqlite", "file:"+p.dbPath+"?_pragma=foreign_keys(1)", wmLog)
	if err != nil {
		return fmt.Errorf("create sqlstore: %w", err)
	}
	p.container = container

	// Get or create a new device
	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		return fmt.Errorf("get device: %w", err)
	}

	// Create the client
	p.client = whatsmeow.NewClient(deviceStore, wmLog)

	// Set up event handler
	p.client.AddEventHandler(p.handleEvent)

	// Connect to WhatsApp
	if p.client.Store.ID == nil {
		// Not logged in, need to show QR code
		qrChan, _ := p.client.GetQRChannel(ctx)
		err = p.client.Connect()
		if err != nil {
			return fmt.Errorf("connect to whatsapp: %w", err)
		}

		// Wait for QR code and handle login
		for evt := range qrChan {
			if evt.Event == "code" {
				if p.qrCallback != nil {
					p.qrCallback(evt.Code)
				} else {
					p.logger.Info("QR code received (set QRCallback to display)", "code", evt.Code)
				}
			} else if evt.Event == "success" {
				p.logger.Info("whatsapp logged in successfully")
				break
			} else if evt.Event == "timeout" {
				return fmt.Errorf("QR code login timed out")
			}
		}
	} else {
		// Already logged in
		err = p.client.Connect()
		if err != nil {
			return fmt.Errorf("connect to whatsapp: %w", err)
		}
		p.logger.Info("whatsapp connected with existing session")
	}

	return nil
}

// Disconnect closes the WhatsApp connection.
func (p *Provider) Disconnect(ctx context.Context) error {
	if p.client != nil {
		p.client.Disconnect()
		p.logger.Info("whatsapp disconnected")
	}
	if p.container != nil {
		if err := p.container.Close(); err != nil {
			return fmt.Errorf("close container: %w", err)
		}
	}
	return nil
}

// Send sends a message to a WhatsApp chat.
func (p *Provider) Send(ctx context.Context, chatID string, msg provider.OutgoingMessage) error {
	if p.client == nil {
		return fmt.Errorf("whatsapp client not connected")
	}

	// Parse the JID (phone@s.whatsapp.net or group@g.us)
	jid, err := types.ParseJID(chatID)
	if err != nil {
		return fmt.Errorf("parse chat ID: %w", err)
	}

	// Build the WhatsApp message
	waMsg := &waE2E.Message{
		Conversation: proto.String(msg.Content),
	}

	// Send the message
	_, err = p.client.SendMessage(ctx, jid, waMsg)
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}

	return nil
}

// OnMessage registers a message handler.
func (p *Provider) OnMessage(handler provider.MessageHandler) {
	p.messageHandler = handler
}

// OnEvent registers an event handler.
func (p *Provider) OnEvent(handler provider.EventHandler) {
	p.eventHandler = handler
}

// handleEvent processes WhatsApp events.
func (p *Provider) handleEvent(evt any) {
	switch v := evt.(type) {
	case *events.Message:
		if p.messageHandler != nil {
			msg := p.convertIncoming(v)
			if err := p.messageHandler(context.Background(), msg); err != nil {
				p.logger.Error("message handler error", "error", err)
			}
		}
	case *events.Connected:
		p.logger.Info("whatsapp connected event received")
	case *events.Disconnected:
		p.logger.Info("whatsapp disconnected event received")
	case *events.LoggedOut:
		p.logger.Warn("whatsapp logged out")
	}
}

// convertIncoming converts a WhatsApp message to an IncomingMessage.
func (p *Provider) convertIncoming(evt *events.Message) provider.IncomingMessage {
	chatType := provider.ChatTypeDM
	if evt.Info.IsGroup {
		chatType = provider.ChatTypeGroup
	}

	// Extract text content from the message
	content := ""
	if evt.Message.GetConversation() != "" {
		content = evt.Message.GetConversation()
	} else if evt.Message.GetExtendedTextMessage() != nil {
		content = evt.Message.GetExtendedTextMessage().GetText()
	}

	return provider.IncomingMessage{
		ID:           evt.Info.ID,
		ProviderName: "whatsapp",
		ChatID:       evt.Info.Chat.String(),
		ChatType:     chatType,
		SenderID:     evt.Info.Sender.String(),
		SenderName:   evt.Info.PushName,
		Content:      content,
		Timestamp:    evt.Info.Timestamp,
		Metadata: map[string]any{
			"is_from_me": evt.Info.IsFromMe,
			"is_group":   evt.Info.IsGroup,
		},
	}
}

// IsLoggedIn returns true if the client has an active session.
func (p *Provider) IsLoggedIn() bool {
	return p.client != nil && p.client.Store.ID != nil
}

// Ensure Provider implements provider.Provider interface.
var _ provider.Provider = (*Provider)(nil)
