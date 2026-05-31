// Package telegram provides a Telegram provider for omnichat.
package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"gopkg.in/telebot.v3"

	"github.com/plexusone/omnichat/provider"
)

// Provider implements the Provider interface for Telegram.
type Provider struct {
	bot             *telebot.Bot
	token           string
	logger          *slog.Logger
	messageHandler  provider.MessageHandler
	eventHandler    provider.EventHandler
	callbackHandler CallbackHandler
	webAppHandler   WebAppDataHandler
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

	// Set up callback handler for inline keyboard buttons
	p.bot.Handle(telebot.OnCallback, func(c telebot.Context) error {
		if p.callbackHandler == nil {
			return nil
		}

		cb := c.Callback()
		senderName := cb.Sender.FirstName
		if cb.Sender.LastName != "" {
			senderName += " " + cb.Sender.LastName
		}

		callback := &Callback{
			ID:         cb.ID,
			Data:       cb.Data,
			ChatID:     fmt.Sprintf("%d", cb.Message.Chat.ID),
			SenderID:   fmt.Sprintf("%d", cb.Sender.ID),
			SenderName: senderName,
			MessageID:  fmt.Sprintf("%d", cb.Message.ID),
		}

		return p.callbackHandler(ctx, callback)
	})

	// Set up web app data handler
	p.bot.Handle(telebot.OnWebApp, func(c telebot.Context) error {
		if p.webAppHandler == nil {
			return nil
		}

		msg := c.Message()
		senderName := msg.Sender.FirstName
		if msg.Sender.LastName != "" {
			senderName += " " + msg.Sender.LastName
		}

		data := &WebAppData{
			Data:       msg.WebAppData.Data,
			ButtonText: msg.WebAppData.Text,
			ChatID:     fmt.Sprintf("%d", msg.Chat.ID),
			SenderID:   fmt.Sprintf("%d", msg.Sender.ID),
			SenderName: senderName,
		}

		return p.webAppHandler(ctx, data)
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

// Telegram-specific metadata keys for OutgoingMessage.Metadata.
const (
	// MetaInlineKeyboard specifies inline keyboard buttons.
	// Value: [][]InlineButton (rows of buttons)
	MetaInlineKeyboard = "telegram_inline_keyboard"

	// MetaDisablePreview disables link preview in messages.
	// Value: bool (default: false)
	MetaDisablePreview = "telegram_disable_preview"

	// MetaDisableNotification sends the message silently.
	// Value: bool (default: false)
	MetaDisableNotification = "telegram_disable_notification"

	// MetaProtectContent protects the message from forwarding/saving.
	// Value: bool (default: false)
	MetaProtectContent = "telegram_protect_content"
)

// InlineButton represents a Telegram inline keyboard button.
type InlineButton struct {
	// Text is the button label text.
	Text string `json:"text"`

	// URL is an HTTP/HTTPS URL to open when button is pressed.
	URL string `json:"url,omitempty"`

	// CallbackData is data sent to bot when button is pressed.
	CallbackData string `json:"callback_data,omitempty"`

	// WebAppURL is the URL for a Web App to open when button is pressed.
	WebAppURL string `json:"web_app_url,omitempty"`

	// SwitchInlineQuery opens inline mode in another chat.
	SwitchInlineQuery string `json:"switch_inline_query,omitempty"`

	// SwitchInlineQueryCurrentChat opens inline mode in current chat.
	SwitchInlineQueryCurrentChat string `json:"switch_inline_query_current_chat,omitempty"`
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

	// Build send options
	opts := &telebot.SendOptions{}
	switch msg.Format {
	case provider.MessageFormatMarkdown:
		opts.ParseMode = telebot.ModeMarkdown
	case provider.MessageFormatHTML:
		opts.ParseMode = telebot.ModeHTML
	}

	// Apply metadata options
	if val, ok := msg.Metadata[MetaDisablePreview].(bool); ok && val {
		opts.DisableWebPagePreview = true
	}
	if val, ok := msg.Metadata[MetaDisableNotification].(bool); ok && val {
		opts.DisableNotification = true
	}
	if val, ok := msg.Metadata[MetaProtectContent].(bool); ok && val {
		opts.Protected = true
	}

	// Build inline keyboard if provided
	if keyboard := p.buildInlineKeyboard(msg); keyboard != nil {
		opts.ReplyMarkup = keyboard
	}

	_, err = p.bot.Send(chat, msg.Content, opts)
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}

	return nil
}

// buildInlineKeyboard constructs a telebot.ReplyMarkup from metadata.
func (p *Provider) buildInlineKeyboard(msg provider.OutgoingMessage) *telebot.ReplyMarkup {
	keyboardData, ok := msg.Metadata[MetaInlineKeyboard]
	if !ok {
		return nil
	}

	// Handle both [][]InlineButton and [][]map[string]any (from JSON)
	var rows [][]InlineButton

	switch v := keyboardData.(type) {
	case [][]InlineButton:
		rows = v
	case []any:
		// Handle JSON-decoded format
		for _, rowAny := range v {
			rowSlice, ok := rowAny.([]any)
			if !ok {
				continue
			}
			var row []InlineButton
			for _, btnAny := range rowSlice {
				btn := parseInlineButton(btnAny)
				if btn != nil {
					row = append(row, *btn)
				}
			}
			if len(row) > 0 {
				rows = append(rows, row)
			}
		}
	default:
		return nil
	}

	if len(rows) == 0 {
		return nil
	}

	// Convert to telebot format
	markup := &telebot.ReplyMarkup{}
	var telebotRows [][]telebot.InlineButton
	for _, row := range rows {
		var telebotBtns []telebot.InlineButton
		for _, btn := range row {
			telebotBtn := p.convertButton(btn)
			telebotBtns = append(telebotBtns, telebotBtn)
		}
		telebotRows = append(telebotRows, telebotBtns)
	}
	markup.InlineKeyboard = telebotRows
	return markup
}

// convertButton converts an InlineButton to a telebot.InlineButton.
func (p *Provider) convertButton(btn InlineButton) telebot.InlineButton {
	telebotBtn := telebot.InlineButton{Text: btn.Text}

	switch {
	case btn.WebAppURL != "":
		telebotBtn.WebApp = &telebot.WebApp{URL: btn.WebAppURL}
	case btn.URL != "":
		telebotBtn.URL = btn.URL
	case btn.CallbackData != "":
		telebotBtn.Data = btn.CallbackData
	case btn.SwitchInlineQuery != "":
		telebotBtn.InlineQuery = btn.SwitchInlineQuery
	case btn.SwitchInlineQueryCurrentChat != "":
		telebotBtn.InlineQueryChat = btn.SwitchInlineQueryCurrentChat
	}

	return telebotBtn
}

// parseInlineButton parses a map into an InlineButton.
func parseInlineButton(data any) *InlineButton {
	m, ok := data.(map[string]any)
	if !ok {
		return nil
	}

	btn := &InlineButton{}
	if v, ok := m["text"].(string); ok {
		btn.Text = v
	}
	if v, ok := m["url"].(string); ok {
		btn.URL = v
	}
	if v, ok := m["callback_data"].(string); ok {
		btn.CallbackData = v
	}
	if v, ok := m["web_app_url"].(string); ok {
		btn.WebAppURL = v
	}
	if v, ok := m["switch_inline_query"].(string); ok {
		btn.SwitchInlineQuery = v
	}
	if v, ok := m["switch_inline_query_current_chat"].(string); ok {
		btn.SwitchInlineQueryCurrentChat = v
	}

	if btn.Text == "" {
		return nil
	}
	return btn
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

// CallbackHandler handles inline keyboard button callbacks.
type CallbackHandler func(ctx context.Context, callback *Callback) error

// WebAppDataHandler handles web app data responses.
type WebAppDataHandler func(ctx context.Context, data *WebAppData) error

// Callback represents an inline keyboard callback.
type Callback struct {
	// ID is the callback query ID.
	ID string

	// Data is the callback data from the button.
	Data string

	// ChatID is the chat where the callback originated.
	ChatID string

	// SenderID is the user who pressed the button.
	SenderID string

	// SenderName is the user's display name.
	SenderName string

	// MessageID is the message containing the button.
	MessageID string
}

// WebAppData represents data sent from a Telegram Web App.
type WebAppData struct {
	// Data is the data string sent by the web app.
	Data string

	// ButtonText is the text of the button that opened the web app.
	ButtonText string

	// ChatID is the chat where the web app was opened.
	ChatID string

	// SenderID is the user who opened the web app.
	SenderID string

	// SenderName is the user's display name.
	SenderName string
}

// Command represents a bot command for registration.
type Command struct {
	// Command is the command name (without leading slash).
	Command string

	// Description is the command description shown in the menu.
	Description string
}

// LocalizedCommands maps language codes to command lists.
// Use empty string "" for default/fallback commands.
type LocalizedCommands map[string][]Command

// OnCallback registers a callback handler for inline keyboard buttons.
func (p *Provider) OnCallback(handler CallbackHandler) {
	p.callbackHandler = handler
}

// OnWebAppData registers a handler for web app data responses.
func (p *Provider) OnWebAppData(handler WebAppDataHandler) {
	p.webAppHandler = handler
}

// SetCommands sets bot commands for the default language.
func (p *Provider) SetCommands(ctx context.Context, commands []Command) error {
	if p.bot == nil {
		return fmt.Errorf("telegram bot not connected")
	}

	telebotCmds := make([]telebot.Command, len(commands))
	for i, cmd := range commands {
		telebotCmds[i] = telebot.Command{
			Text:        cmd.Command,
			Description: cmd.Description,
		}
	}

	return p.bot.SetCommands(telebotCmds)
}

// SetLocalizedCommands sets bot commands for multiple languages.
// The language code "" sets the default commands.
func (p *Provider) SetLocalizedCommands(ctx context.Context, commands LocalizedCommands) error {
	if p.bot == nil {
		return fmt.Errorf("telegram bot not connected")
	}

	for langCode, cmds := range commands {
		telebotCmds := make([]telebot.Command, len(cmds))
		for i, cmd := range cmds {
			telebotCmds[i] = telebot.Command{
				Text:        cmd.Command,
				Description: cmd.Description,
			}
		}

		params := &telebot.CommandParams{}
		if langCode != "" {
			params.LanguageCode = langCode
		}

		if err := p.bot.SetCommands(telebotCmds, params); err != nil {
			return fmt.Errorf("set commands for language %q: %w", langCode, err)
		}
	}

	return nil
}

// DeleteCommands removes bot commands for the specified language (or default if empty).
func (p *Provider) DeleteCommands(ctx context.Context, languageCode string) error {
	if p.bot == nil {
		return fmt.Errorf("telegram bot not connected")
	}

	params := &telebot.CommandParams{}
	if languageCode != "" {
		params.LanguageCode = languageCode
	}

	return p.bot.DeleteCommands(params)
}

// AnswerCallback responds to a callback query with optional text and alert.
func (p *Provider) AnswerCallback(ctx context.Context, callbackID, text string, showAlert bool) error {
	if p.bot == nil {
		return fmt.Errorf("telegram bot not connected")
	}

	return p.bot.Respond(&telebot.Callback{ID: callbackID}, &telebot.CallbackResponse{
		Text:      text,
		ShowAlert: showAlert,
	})
}

// Ensure Provider implements provider.Provider interface.
var _ provider.Provider = (*Provider)(nil)
