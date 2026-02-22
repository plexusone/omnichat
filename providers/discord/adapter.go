// Package discord provides a Discord provider for omnichat.
package discord

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/bwmarrin/discordgo"

	"github.com/agentplexus/omnichat/provider"
)

// Provider implements the Provider interface for Discord.
type Provider struct {
	session        *discordgo.Session
	token          string
	guildID        string
	logger         *slog.Logger
	messageHandler provider.MessageHandler
	eventHandler   provider.EventHandler
}

// Config configures the Discord provider.
type Config struct {
	Token   string
	GuildID string
	Logger  *slog.Logger
}

// New creates a new Discord provider.
func New(config Config) (*Provider, error) {
	if config.Token == "" {
		return nil, fmt.Errorf("discord token required")
	}
	if config.Logger == nil {
		config.Logger = slog.Default()
	}

	return &Provider{
		token:   config.Token,
		guildID: config.GuildID,
		logger:  config.Logger,
	}, nil
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "discord"
}

// Connect establishes connection to Discord.
func (p *Provider) Connect(ctx context.Context) error {
	session, err := discordgo.New("Bot " + p.token)
	if err != nil {
		return fmt.Errorf("create discord session: %w", err)
	}

	p.session = session

	// Set up message handler
	p.session.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		// Ignore messages from the bot itself
		if m.Author.ID == s.State.User.ID {
			return
		}

		if p.messageHandler != nil {
			msg := p.convertIncoming(m)
			if err := p.messageHandler(ctx, msg); err != nil {
				p.logger.Error("message handler error", "error", err)
			}
		}
	})

	// Set intents
	p.session.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsDirectMessages | discordgo.IntentsMessageContent

	// Open connection
	if err := p.session.Open(); err != nil {
		return fmt.Errorf("open discord session: %w", err)
	}

	p.logger.Info("discord bot connected", "user", p.session.State.User.Username)
	return nil
}

// Disconnect closes the Discord connection.
func (p *Provider) Disconnect(ctx context.Context) error {
	if p.session != nil {
		if err := p.session.Close(); err != nil {
			return fmt.Errorf("close discord session: %w", err)
		}
		p.logger.Info("discord bot disconnected")
	}
	return nil
}

// Send sends a message to a Discord channel.
func (p *Provider) Send(ctx context.Context, channelID string, msg provider.OutgoingMessage) error {
	if p.session == nil {
		return fmt.Errorf("discord session not connected")
	}

	// Build message send options
	data := &discordgo.MessageSend{
		Content: msg.Content,
	}

	if msg.ReplyTo != "" {
		data.Reference = &discordgo.MessageReference{
			MessageID: msg.ReplyTo,
		}
	}

	_, err := p.session.ChannelMessageSendComplex(channelID, data)
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

// convertIncoming converts a Discord message to an IncomingMessage.
func (p *Provider) convertIncoming(m *discordgo.MessageCreate) provider.IncomingMessage {
	chatType := provider.ChatTypeGroup
	// Check if it's a DM
	if m.GuildID == "" {
		chatType = provider.ChatTypeDM
	}

	// Check for thread
	if m.Thread != nil {
		chatType = provider.ChatTypeThread
	}

	return provider.IncomingMessage{
		ID:           m.ID,
		ProviderName: "discord",
		ChatID:       m.ChannelID,
		ChatType:     chatType,
		SenderID:     m.Author.ID,
		SenderName:   m.Author.Username,
		Content:      m.Content,
		ReplyTo:      getReplyTo(m),
		Timestamp:    m.Timestamp,
		Metadata: map[string]any{
			"guild_id":      m.GuildID,
			"discriminator": m.Author.Discriminator,
		},
	}
}

// getReplyTo extracts the reply-to message ID if present.
func getReplyTo(m *discordgo.MessageCreate) string {
	if m.MessageReference != nil {
		return m.MessageReference.MessageID
	}
	return ""
}

// Ensure Provider implements provider.Provider interface.
var _ provider.Provider = (*Provider)(nil)
