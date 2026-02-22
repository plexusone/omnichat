// Package telegram provides a Telegram provider for omnichat.
package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"gopkg.in/telebot.v3"

	"github.com/agentplexus/omnichat/provider"
)

// Provider implements the Provider interface for Telegram.
type Provider struct {
	bot            *telebot.Bot
	token          string
	logger         *slog.Logger
	messageHandler provider.MessageHandler
	eventHandler   provider.EventHandler
}

// Config configures the Telegram provider.
type Config struct {
	Token  string
	Logger *slog.Logger
}

// New creates a new Telegram provider.
func New(config Config) (*Provider, error) {
	if config.Token == "" {
		return nil, fmt.Errorf("telegram token required")
	}
	if config.Logger == nil {
		config.Logger = slog.Default()
	}

	return &Provider{
		token:  config.Token,
		logger: config.Logger,
	}, nil
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "telegram"
}

// Connect establishes connection to Telegram.
func (p *Provider) Connect(ctx context.Context) error {
	pref := telebot.Settings{
		Token:  p.token,
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
	}

	bot, err := telebot.NewBot(pref)
	if err != nil {
		return fmt.Errorf("create telegram bot: %w", err)
	}

	p.bot = bot

	// Set up message handler
	p.bot.Handle(telebot.OnText, func(c telebot.Context) error {
		if p.messageHandler == nil {
			return nil
		}

		msg := p.convertIncoming(c.Message())
		return p.messageHandler(ctx, msg)
	})

	// Start bot in background
	go func() {
		p.logger.Info("starting telegram bot")
		p.bot.Start()
	}()

	return nil
}

// Disconnect closes the Telegram connection.
func (p *Provider) Disconnect(ctx context.Context) error {
	if p.bot != nil {
		p.bot.Stop()
		p.logger.Info("telegram bot stopped")
	}
	return nil
}

// Send sends a message to a Telegram chat.
func (p *Provider) Send(ctx context.Context, chatID string, msg provider.OutgoingMessage) error {
	if p.bot == nil {
		return fmt.Errorf("telegram bot not connected")
	}

	// Parse chat ID
	chatIDInt, err := strconv.ParseInt(chatID, 10, 64)
	if err != nil {
		return fmt.Errorf("parse chat ID: %w", err)
	}
	chat, err := p.bot.ChatByID(chatIDInt)
	if err != nil {
		return fmt.Errorf("get chat: %w", err)
	}

	// Send text message
	opts := &telebot.SendOptions{}
	switch msg.Format {
	case provider.MessageFormatMarkdown:
		opts.ParseMode = telebot.ModeMarkdown
	case provider.MessageFormatHTML:
		opts.ParseMode = telebot.ModeHTML
	}

	_, err = p.bot.Send(chat, msg.Content, opts)
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

// convertIncoming converts a Telegram message to an IncomingMessage.
func (p *Provider) convertIncoming(msg *telebot.Message) provider.IncomingMessage {
	var chatType provider.ChatType
	switch msg.Chat.Type {
	case telebot.ChatGroup, telebot.ChatSuperGroup:
		chatType = provider.ChatTypeGroup
	case telebot.ChatChannel:
		chatType = provider.ChatTypeChannel
	default:
		chatType = provider.ChatTypeDM
	}

	senderName := msg.Sender.FirstName
	if msg.Sender.LastName != "" {
		senderName += " " + msg.Sender.LastName
	}
	if senderName == "" {
		senderName = msg.Sender.Username
	}

	return provider.IncomingMessage{
		ID:           fmt.Sprintf("%d", msg.ID),
		ProviderName: "telegram",
		ChatID:       fmt.Sprintf("%d", msg.Chat.ID),
		ChatType:     chatType,
		SenderID:     fmt.Sprintf("%d", msg.Sender.ID),
		SenderName:   senderName,
		Content:      msg.Text,
		Timestamp:    msg.Time(),
		Metadata: map[string]any{
			"chat_title": msg.Chat.Title,
			"username":   msg.Sender.Username,
		},
	}
}

// Ensure Provider implements provider.Provider interface.
var _ provider.Provider = (*Provider)(nil)
