// Package irc provides an IRC provider for omnichat.
package irc

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/ergochat/irc-go/ircevent"
	"github.com/ergochat/irc-go/ircmsg"

	"github.com/plexusone/omnichat/provider"
)

// Provider implements the Provider interface for IRC.
type Provider struct {
	conn           *ircevent.Connection
	config         Config
	logger         *slog.Logger
	messageHandler provider.MessageHandler
	eventHandler   provider.EventHandler
	mu             sync.RWMutex
	connected      bool
	cancelFunc     context.CancelFunc
}

// Config configures the IRC provider.
type Config struct {
	// Server is the IRC server address (e.g., "irc.libera.chat:6697").
	Server string

	// Nick is the bot's nickname.
	Nick string

	// User is the username (defaults to Nick if empty).
	User string

	// RealName is the real name field (defaults to Nick if empty).
	RealName string

	// Password is the NickServ password (optional).
	Password string

	// Channels is the list of channels to join on connect.
	Channels []string

	// UseTLS enables TLS for the connection.
	UseTLS bool

	// Logger is the logger instance.
	Logger *slog.Logger
}

// New creates a new IRC provider.
func New(config Config) (*Provider, error) {
	if config.Server == "" {
		return nil, fmt.Errorf("IRC server address required")
	}
	if config.Nick == "" {
		return nil, fmt.Errorf("IRC nickname required")
	}
	if config.Logger == nil {
		config.Logger = slog.Default()
	}
	if config.User == "" {
		config.User = config.Nick
	}
	if config.RealName == "" {
		config.RealName = config.Nick
	}

	return &Provider{
		config: config,
		logger: config.Logger,
	}, nil
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "irc"
}

// Connect establishes connection to the IRC server.
func (p *Provider) Connect(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.connected {
		return nil
	}

	// Create a cancellable context for the event loop
	ctx, cancel := context.WithCancel(ctx)
	p.cancelFunc = cancel

	// Configure IRC connection
	conn := &ircevent.Connection{
		Server:       p.config.Server,
		Nick:         p.config.Nick,
		User:         p.config.User,
		RealName:     p.config.RealName,
		Password:     p.config.Password,
		UseTLS:       p.config.UseTLS,
		QuitMessage:  "Goodbye",
		Debug:        false, // Set to true for debugging
		RequestCaps:  []string{"message-tags", "server-time"},
		SASLLogin:    "",
		SASLPassword: "",
	}

	if p.config.UseTLS {
		conn.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	// Set up event handlers
	conn.AddConnectCallback(func(e ircmsg.Message) {
		p.logger.Info("connected to IRC server",
			"server", p.config.Server,
			"nick", p.config.Nick,
		)

		// Join configured channels
		for _, channel := range p.config.Channels {
			channel = strings.TrimSpace(channel)
			if channel == "" {
				continue
			}
			// Ensure channel has # prefix
			if !strings.HasPrefix(channel, "#") {
				channel = "#" + channel
			}
			if err := conn.Join(channel); err != nil {
				p.logger.Error("failed to join channel", "channel", channel, "error", err)
				continue
			}
			p.logger.Info("joining channel", "channel", channel)
		}
	})

	// Handle PRIVMSG (messages)
	conn.AddCallback("PRIVMSG", func(e ircmsg.Message) {
		p.handlePrivmsg(ctx, e)
	})

	// Handle JOIN events
	conn.AddCallback("JOIN", func(e ircmsg.Message) {
		p.handleJoin(ctx, e)
	})

	// Handle PART events
	conn.AddCallback("PART", func(e ircmsg.Message) {
		p.handlePart(ctx, e)
	})

	// Handle QUIT events
	conn.AddCallback("QUIT", func(e ircmsg.Message) {
		p.handleQuit(ctx, e)
	})

	// Handle disconnection
	conn.AddCallback("ERROR", func(e ircmsg.Message) {
		p.logger.Warn("IRC error", "message", strings.Join(e.Params, " "))
	})

	p.conn = conn

	// Connect to IRC server
	if err := conn.Connect(); err != nil {
		return fmt.Errorf("connect to IRC server: %w", err)
	}

	p.connected = true

	// Start event loop in background
	go func() {
		p.conn.Loop()
	}()

	return nil
}

// Disconnect closes the IRC connection.
func (p *Provider) Disconnect(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.connected {
		return nil
	}

	if p.cancelFunc != nil {
		p.cancelFunc()
	}

	if p.conn != nil {
		p.conn.Quit()
		p.logger.Info("disconnected from IRC server")
	}

	p.connected = false
	return nil
}

// Send sends a message to an IRC channel or user.
func (p *Provider) Send(ctx context.Context, chatID string, msg provider.OutgoingMessage) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.connected || p.conn == nil {
		return fmt.Errorf("IRC not connected")
	}

	// chatID is either "#channel" for channels or "nickname" for DMs
	target := chatID

	// Split message into lines to handle multi-line messages
	lines := strings.Split(msg.Content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// IRC messages have a max length; split if necessary
		// Max IRC message is ~512 bytes including CRLF and command
		// Safe limit for message content is around 400 characters
		const maxLen = 400
		for len(line) > 0 {
			chunk := line
			if len(chunk) > maxLen {
				// Find a good break point
				breakPoint := maxLen
				if idx := strings.LastIndex(chunk[:maxLen], " "); idx > maxLen/2 {
					breakPoint = idx
				}
				chunk = line[:breakPoint]
				line = strings.TrimSpace(line[breakPoint:])
			} else {
				line = ""
			}

			if err := p.conn.Privmsg(target, chunk); err != nil {
				return fmt.Errorf("send message: %w", err)
			}
		}
	}

	return nil
}

// OnMessage registers a message handler.
func (p *Provider) OnMessage(handler provider.MessageHandler) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.messageHandler = handler
}

// OnEvent registers an event handler.
func (p *Provider) OnEvent(handler provider.EventHandler) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.eventHandler = handler
}

// handlePrivmsg handles incoming PRIVMSG events.
func (p *Provider) handlePrivmsg(ctx context.Context, e ircmsg.Message) {
	p.mu.RLock()
	handler := p.messageHandler
	p.mu.RUnlock()

	if handler == nil {
		return
	}

	// Parse PRIVMSG: :nick!user@host PRIVMSG target :message
	if len(e.Params) < 2 {
		return
	}

	target := e.Params[0]  // Channel or our nick (for DMs)
	content := e.Params[1] // Message content

	// Extract sender info from prefix (nick!user@host)
	senderNick := e.Nick()
	if senderNick == "" {
		// Fallback: parse from source
		if idx := strings.Index(e.Source, "!"); idx > 0 {
			senderNick = e.Source[:idx]
		} else {
			senderNick = e.Source
		}
	}

	// Ignore messages from ourselves
	if senderNick == p.config.Nick {
		return
	}

	// Determine chat type and ID
	var chatType provider.ChatType
	var chatID string

	if strings.HasPrefix(target, "#") {
		// Channel message
		chatType = provider.ChatTypeChannel
		chatID = target
	} else {
		// Direct message (target is our nick)
		chatType = provider.ChatTypeDM
		chatID = senderNick // For DMs, use sender as chatID
	}

	// Parse timestamp from message tags if available
	timestamp := time.Now()
	if present, timeTag := e.GetTag("time"); present {
		if t, err := time.Parse(time.RFC3339, timeTag); err == nil {
			timestamp = t
		}
	}

	// Extract user and host from source
	user := ""
	host := ""
	if idx := strings.Index(e.Source, "!"); idx > 0 {
		rest := e.Source[idx+1:]
		if hostIdx := strings.Index(rest, "@"); hostIdx > 0 {
			user = rest[:hostIdx]
			host = rest[hostIdx+1:]
		}
	}

	msg := provider.IncomingMessage{
		ID:           fmt.Sprintf("%d", timestamp.UnixNano()), // IRC doesn't have message IDs
		ProviderName: "irc",
		ChatID:       chatID,
		ChatType:     chatType,
		SenderID:     senderNick, // IRC uses nick as identifier
		SenderName:   senderNick,
		Content:      content,
		Timestamp:    timestamp,
		Metadata: map[string]any{
			"user":   user,
			"host":   host,
			"source": e.Source,
			"target": target,
		},
	}

	if err := handler(ctx, msg); err != nil {
		p.logger.Error("message handler error",
			"error", err,
			"from", senderNick,
			"channel", chatID,
		)
	}
}

// handleJoin handles JOIN events.
func (p *Provider) handleJoin(ctx context.Context, e ircmsg.Message) {
	p.mu.RLock()
	handler := p.eventHandler
	p.mu.RUnlock()

	if handler == nil {
		return
	}

	nick := e.Nick()
	if nick == "" && len(e.Source) > 0 {
		if idx := strings.Index(e.Source, "!"); idx > 0 {
			nick = e.Source[:idx]
		}
	}

	// Skip our own join
	if nick == p.config.Nick {
		return
	}

	channel := ""
	if len(e.Params) > 0 {
		channel = e.Params[0]
	}

	event := provider.Event{
		Type:         provider.EventTypeMemberJoined,
		ProviderName: "irc",
		ChatID:       channel,
		Data: map[string]any{
			"nick":   nick,
			"source": e.Source,
		},
		Timestamp: time.Now(),
	}

	if err := handler(ctx, event); err != nil {
		p.logger.Error("event handler error",
			"error", err,
			"event", "join",
			"nick", nick,
		)
	}
}

// handlePart handles PART events.
func (p *Provider) handlePart(ctx context.Context, e ircmsg.Message) {
	p.mu.RLock()
	handler := p.eventHandler
	p.mu.RUnlock()

	if handler == nil {
		return
	}

	nick := e.Nick()
	if nick == "" && len(e.Source) > 0 {
		if idx := strings.Index(e.Source, "!"); idx > 0 {
			nick = e.Source[:idx]
		}
	}

	channel := ""
	reason := ""
	if len(e.Params) > 0 {
		channel = e.Params[0]
	}
	if len(e.Params) > 1 {
		reason = e.Params[1]
	}

	event := provider.Event{
		Type:         provider.EventTypeMemberLeft,
		ProviderName: "irc",
		ChatID:       channel,
		Data: map[string]any{
			"nick":   nick,
			"reason": reason,
			"source": e.Source,
		},
		Timestamp: time.Now(),
	}

	if err := handler(ctx, event); err != nil {
		p.logger.Error("event handler error",
			"error", err,
			"event", "part",
			"nick", nick,
		)
	}
}

// handleQuit handles QUIT events.
func (p *Provider) handleQuit(ctx context.Context, e ircmsg.Message) {
	p.mu.RLock()
	handler := p.eventHandler
	p.mu.RUnlock()

	if handler == nil {
		return
	}

	nick := e.Nick()
	if nick == "" && len(e.Source) > 0 {
		if idx := strings.Index(e.Source, "!"); idx > 0 {
			nick = e.Source[:idx]
		}
	}

	reason := ""
	if len(e.Params) > 0 {
		reason = e.Params[0]
	}

	event := provider.Event{
		Type:         provider.EventTypeMemberLeft,
		ProviderName: "irc",
		ChatID:       "", // QUIT affects all channels
		Data: map[string]any{
			"nick":   nick,
			"reason": reason,
			"quit":   true,
		},
		Timestamp: time.Now(),
	}

	if err := handler(ctx, event); err != nil {
		p.logger.Error("event handler error",
			"error", err,
			"event", "quit",
			"nick", nick,
		)
	}
}

// IsConnected returns whether the provider is connected.
func (p *Provider) IsConnected() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.connected
}

// JoinChannel joins an IRC channel.
func (p *Provider) JoinChannel(channel string) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.connected || p.conn == nil {
		return fmt.Errorf("IRC not connected")
	}

	if !strings.HasPrefix(channel, "#") {
		channel = "#" + channel
	}

	if err := p.conn.Join(channel); err != nil {
		return fmt.Errorf("join channel %s: %w", channel, err)
	}
	p.logger.Info("joining channel", "channel", channel)
	return nil
}

// PartChannel leaves an IRC channel.
func (p *Provider) PartChannel(channel string, reason string) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.connected || p.conn == nil {
		return fmt.Errorf("IRC not connected")
	}

	if !strings.HasPrefix(channel, "#") {
		channel = "#" + channel
	}

	if err := p.conn.Part(channel); err != nil {
		return fmt.Errorf("part channel %s: %w", channel, err)
	}
	p.logger.Info("leaving channel", "channel", channel)
	return nil
}

// Ensure Provider implements provider.Provider interface.
var _ provider.Provider = (*Provider)(nil)
