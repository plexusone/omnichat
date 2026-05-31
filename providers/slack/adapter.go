// Package slack provides a Slack provider for omnichat using Socket Mode.
package slack

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"

	"github.com/plexusone/omnichat/provider"
)

// Provider implements the Provider interface for Slack.
type Provider struct {
	api            *slack.Client
	socketClient   *socketmode.Client
	botToken       string
	appToken       string
	logger         *slog.Logger
	messageHandler provider.MessageHandler
	eventHandler   provider.EventHandler
	botUserID      string
	cancelFunc     context.CancelFunc
}

// Config configures the Slack provider.
type Config struct {
	// BotToken is the Slack bot token (xoxb-...).
	BotToken string

	// AppToken is the Slack app-level token for Socket Mode (xapp-...).
	AppToken string

	// Logger is the logger instance.
	Logger *slog.Logger

	// Debug enables debug logging for the Slack client.
	Debug bool
}

// New creates a new Slack provider.
func New(config Config) (*Provider, error) {
	if config.BotToken == "" {
		return nil, fmt.Errorf("slack bot token required")
	}
	if config.AppToken == "" {
		return nil, fmt.Errorf("slack app token required for socket mode")
	}
	if config.Logger == nil {
		config.Logger = slog.Default()
	}

	return &Provider{
		botToken: config.BotToken,
		appToken: config.AppToken,
		logger:   config.Logger,
	}, nil
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "slack"
}

// Connect establishes connection to Slack via Socket Mode.
func (p *Provider) Connect(ctx context.Context) error {
	// Create API client
	p.api = slack.New(
		p.botToken,
		slack.OptionAppLevelToken(p.appToken),
	)

	// Get bot user ID
	authTest, err := p.api.AuthTest()
	if err != nil {
		return fmt.Errorf("slack auth test: %w", err)
	}
	p.botUserID = authTest.UserID

	// Create socket mode client
	p.socketClient = socketmode.New(
		p.api,
		socketmode.OptionLog(newSlackLogger(p.logger)),
	)

	// Create a cancellable context for the event loop
	eventCtx, cancel := context.WithCancel(ctx)
	p.cancelFunc = cancel

	// Start event handler goroutine
	go p.handleEvents(eventCtx)

	// Run socket mode client
	go func() {
		if err := p.socketClient.RunContext(eventCtx); err != nil {
			p.logger.Error("slack socket mode error", "error", err)
		}
	}()

	p.logger.Info("slack bot connected", "user_id", p.botUserID, "team", authTest.Team)
	return nil
}

// Disconnect closes the Slack connection.
func (p *Provider) Disconnect(ctx context.Context) error {
	if p.cancelFunc != nil {
		p.cancelFunc()
		p.logger.Info("slack bot disconnected")
	}
	return nil
}

// Slack-specific metadata keys for OutgoingMessage.Metadata.
const (
	// MetaUnfurlLinks controls whether to unfurl text-based links.
	// Value: bool (default: true for markdown format, false otherwise)
	MetaUnfurlLinks = "slack_unfurl_links"

	// MetaUnfurlMedia controls whether to unfurl media-based links.
	// Value: bool (default: true)
	MetaUnfurlMedia = "slack_unfurl_media"

	// MetaReplyBroadcast controls whether to broadcast a thread reply to the channel.
	// Value: bool (default: false)
	// Only applies when ReplyTo is set (i.e., replying in a thread).
	MetaReplyBroadcast = "slack_reply_broadcast"
)

// Send sends a message to a Slack channel.
func (p *Provider) Send(ctx context.Context, channelID string, msg provider.OutgoingMessage) error {
	if p.api == nil {
		return fmt.Errorf("slack client not connected")
	}

	// Build message options
	opts := []slack.MsgOption{
		slack.MsgOptionText(msg.Content, false),
	}

	// Handle thread replies
	if msg.ReplyTo != "" {
		opts = append(opts, slack.MsgOptionTS(msg.ReplyTo))

		// Check for reply broadcast (only applicable for thread replies)
		if broadcast, ok := msg.Metadata[MetaReplyBroadcast].(bool); ok && broadcast {
			opts = append(opts, slack.MsgOptionBroadcast())
		}
	}

	// Handle unfurl controls
	unfurlLinks := p.getUnfurlLinks(msg)
	unfurlMedia := p.getUnfurlMedia(msg)

	if unfurlLinks {
		opts = append(opts, slack.MsgOptionEnableLinkUnfurl())
	} else {
		opts = append(opts, slack.MsgOptionDisableLinkUnfurl())
	}

	if !unfurlMedia {
		opts = append(opts, slack.MsgOptionDisableMediaUnfurl())
	}

	_, _, err := p.api.PostMessageContext(ctx, channelID, opts...)
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}

	return nil
}

// getUnfurlLinks returns whether to unfurl links based on metadata and format.
func (p *Provider) getUnfurlLinks(msg provider.OutgoingMessage) bool {
	// Check explicit metadata setting first
	if val, ok := msg.Metadata[MetaUnfurlLinks].(bool); ok {
		return val
	}
	// Default: enable for markdown format only
	return msg.Format == provider.MessageFormatMarkdown
}

// getUnfurlMedia returns whether to unfurl media based on metadata.
func (p *Provider) getUnfurlMedia(msg provider.OutgoingMessage) bool {
	// Check explicit metadata setting first
	if val, ok := msg.Metadata[MetaUnfurlMedia].(bool); ok {
		return val
	}
	// Default: enable media unfurling
	return true
}

// OnMessage registers a message handler.
func (p *Provider) OnMessage(handler provider.MessageHandler) {
	p.messageHandler = handler
}

// OnEvent registers an event handler.
func (p *Provider) OnEvent(handler provider.EventHandler) {
	p.eventHandler = handler
}

// handleEvents processes incoming Socket Mode events.
func (p *Provider) handleEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case evt := <-p.socketClient.Events:
			p.handleEvent(ctx, evt)
		}
	}
}

// handleEvent processes a single Socket Mode event.
func (p *Provider) handleEvent(ctx context.Context, evt socketmode.Event) {
	switch evt.Type {
	case socketmode.EventTypeEventsAPI:
		eventsAPIEvent, ok := evt.Data.(slackevents.EventsAPIEvent)
		if !ok {
			return
		}

		// Acknowledge the event
		if err := p.socketClient.Ack(*evt.Request); err != nil {
			p.logger.Error("failed to ack events API event", "error", err)
		}

		p.handleEventsAPIEvent(ctx, eventsAPIEvent)

	case socketmode.EventTypeSlashCommand:
		// Could handle slash commands if needed
		if err := p.socketClient.Ack(*evt.Request); err != nil {
			p.logger.Error("failed to ack slash command", "error", err)
		}

	case socketmode.EventTypeInteractive:
		// Could handle interactive components if needed
		if err := p.socketClient.Ack(*evt.Request); err != nil {
			p.logger.Error("failed to ack interactive event", "error", err)
		}
	}
}

// handleEventsAPIEvent processes Events API events.
func (p *Provider) handleEventsAPIEvent(ctx context.Context, event slackevents.EventsAPIEvent) {
	switch event.Type {
	case slackevents.CallbackEvent:
		innerEvent := event.InnerEvent
		switch ev := innerEvent.Data.(type) {
		case *slackevents.MessageEvent:
			p.handleMessageEvent(ctx, ev)
		case *slackevents.ReactionAddedEvent:
			p.handleReactionEvent(ctx, ev, true)
		case *slackevents.ReactionRemovedEvent:
			p.handleReactionRemovedEvent(ctx, ev)
		case *slackevents.MemberJoinedChannelEvent:
			p.handleMemberJoinedEvent(ctx, ev)
		case *slackevents.MemberLeftChannelEvent:
			p.handleMemberLeftEvent(ctx, ev)
		}
	}
}

// handleMessageEvent processes incoming messages.
func (p *Provider) handleMessageEvent(ctx context.Context, ev *slackevents.MessageEvent) {
	// Ignore messages from the bot itself
	if ev.User == p.botUserID {
		return
	}

	// Ignore message subtypes like message_changed, message_deleted, etc.
	// We only want regular messages and bot messages
	if ev.SubType != "" && ev.SubType != "bot_message" {
		// Handle message edits and deletions as events
		if ev.SubType == "message_changed" && p.eventHandler != nil {
			if err := p.eventHandler(ctx, provider.Event{
				Type:         provider.EventTypeMessageEdited,
				ProviderName: "slack",
				ChatID:       ev.Channel,
				Data: map[string]any{
					"message_ts": ev.TimeStamp,
				},
				Timestamp: time.Now(),
			}); err != nil {
				p.logger.Error("event handler error", "event", "message_changed", "error", err)
			}
		} else if ev.SubType == "message_deleted" && p.eventHandler != nil {
			if err := p.eventHandler(ctx, provider.Event{
				Type:         provider.EventTypeMessageDeleted,
				ProviderName: "slack",
				ChatID:       ev.Channel,
				Data: map[string]any{
					"message_ts": ev.TimeStamp,
				},
				Timestamp: time.Now(),
			}); err != nil {
				p.logger.Error("event handler error", "event", "message_deleted", "error", err)
			}
		}
		return
	}

	if p.messageHandler == nil {
		return
	}

	msg := p.convertIncoming(ev)
	if err := p.messageHandler(ctx, msg); err != nil {
		p.logger.Error("message handler error", "error", err)
	}
}

// handleReactionEvent processes reaction added events.
func (p *Provider) handleReactionEvent(ctx context.Context, ev *slackevents.ReactionAddedEvent, added bool) {
	if p.eventHandler == nil {
		return
	}

	if err := p.eventHandler(ctx, provider.Event{
		Type:         provider.EventTypeReaction,
		ProviderName: "slack",
		ChatID:       ev.Item.Channel,
		Data: map[string]any{
			"reaction":   ev.Reaction,
			"user_id":    ev.User,
			"message_ts": ev.Item.Timestamp,
			"added":      added,
		},
		Timestamp: time.Now(),
	}); err != nil {
		p.logger.Error("event handler error", "event", "reaction_added", "error", err)
	}
}

// handleReactionRemovedEvent processes reaction removed events.
func (p *Provider) handleReactionRemovedEvent(ctx context.Context, ev *slackevents.ReactionRemovedEvent) {
	if p.eventHandler == nil {
		return
	}

	if err := p.eventHandler(ctx, provider.Event{
		Type:         provider.EventTypeReaction,
		ProviderName: "slack",
		ChatID:       ev.Item.Channel,
		Data: map[string]any{
			"reaction":   ev.Reaction,
			"user_id":    ev.User,
			"message_ts": ev.Item.Timestamp,
			"added":      false,
		},
		Timestamp: time.Now(),
	}); err != nil {
		p.logger.Error("event handler error", "event", "reaction_removed", "error", err)
	}
}

// handleMemberJoinedEvent processes member joined events.
func (p *Provider) handleMemberJoinedEvent(ctx context.Context, ev *slackevents.MemberJoinedChannelEvent) {
	if p.eventHandler == nil {
		return
	}

	if err := p.eventHandler(ctx, provider.Event{
		Type:         provider.EventTypeMemberJoined,
		ProviderName: "slack",
		ChatID:       ev.Channel,
		Data: map[string]any{
			"user_id":    ev.User,
			"inviter_id": ev.Inviter,
		},
		Timestamp: time.Now(),
	}); err != nil {
		p.logger.Error("event handler error", "event", "member_joined", "error", err)
	}
}

// handleMemberLeftEvent processes member left events.
func (p *Provider) handleMemberLeftEvent(ctx context.Context, ev *slackevents.MemberLeftChannelEvent) {
	if p.eventHandler == nil {
		return
	}

	if err := p.eventHandler(ctx, provider.Event{
		Type:         provider.EventTypeMemberLeft,
		ProviderName: "slack",
		ChatID:       ev.Channel,
		Data: map[string]any{
			"user_id": ev.User,
		},
		Timestamp: time.Now(),
	}); err != nil {
		p.logger.Error("event handler error", "event", "member_left", "error", err)
	}
}

// convertIncoming converts a Slack message event to an IncomingMessage.
func (p *Provider) convertIncoming(ev *slackevents.MessageEvent) provider.IncomingMessage {
	// Determine chat type
	chatType := provider.ChatTypeChannel
	if ev.ChannelType == "im" {
		chatType = provider.ChatTypeDM
	} else if ev.ChannelType == "mpim" {
		chatType = provider.ChatTypeGroup
	} else if ev.ThreadTimeStamp != "" && ev.ThreadTimeStamp != ev.TimeStamp {
		// Message is in a thread
		chatType = provider.ChatTypeThread
	}

	// Parse timestamp
	timestamp := parseSlackTimestamp(ev.TimeStamp)

	return provider.IncomingMessage{
		ID:           ev.TimeStamp, // Slack uses timestamp as message ID
		ProviderName: "slack",
		ChatID:       ev.Channel,
		ChatType:     chatType,
		SenderID:     ev.User,
		SenderName:   ev.Username,
		Content:      ev.Text,
		ReplyTo:      ev.ThreadTimeStamp,
		Timestamp:    timestamp,
		Metadata: map[string]any{
			"channel_type": ev.ChannelType,
			"source_team":  ev.SourceTeam,
			"bot_id":       ev.BotID,
		},
	}
}

// parseSlackTimestamp converts a Slack timestamp to time.Time.
func parseSlackTimestamp(ts string) time.Time {
	// Slack timestamps are in the format "1234567890.123456"
	var sec, usec int64
	_, err := fmt.Sscanf(ts, "%d.%d", &sec, &usec)
	if err != nil {
		return time.Time{}
	}
	return time.Unix(sec, usec*1000)
}

// slackLogger adapts slog.Logger to Slack's logging interface.
type slackLogger struct {
	logger *slog.Logger
}

func newSlackLogger(logger *slog.Logger) *slackLogger {
	return &slackLogger{logger: logger}
}

func (l *slackLogger) Output(calldepth int, s string) error {
	l.logger.Debug(s, "source", "slack-sdk")
	return nil
}

// Ensure Provider implements provider.Provider interface.
var _ provider.Provider = (*Provider)(nil)
