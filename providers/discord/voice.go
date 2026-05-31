// Copyright 2025 John Wang. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package discord

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/grokify/mogo/log/slogutil"

	"github.com/plexusone/omnichat/provider"
)

// VoiceConfig configures Discord voice features.
type VoiceConfig struct {
	// Allowlist of channel IDs the bot can join. Empty means all allowed.
	ChannelAllowlist []string

	// Blocklist of channel IDs the bot cannot join.
	ChannelBlocklist []string

	// FollowUsers is a list of user IDs to auto-follow into voice channels.
	FollowUsers []string

	// AutoLeaveEmpty automatically leaves when channel becomes empty.
	AutoLeaveEmpty bool

	// AutoLeaveTimeout is how long to wait before leaving empty channel.
	AutoLeaveTimeout time.Duration
}

// VoiceState represents a user's voice state.
type VoiceState struct {
	UserID    string
	ChannelID string
	GuildID   string
	Muted     bool
	Deafened  bool
	Speaking  bool
}

// VoiceConnection wraps a Discord voice connection.
type VoiceConnection struct {
	conn     *discordgo.VoiceConnection
	guildID  string
	chanID   string
	mu       sync.RWMutex
	speaking bool
}

// VoiceManager handles voice channel operations.
type VoiceManager struct {
	provider     *Provider
	config       VoiceConfig
	connections  map[string]*VoiceConnection // guildID -> connection
	voiceStates  map[string]*VoiceState      // userID -> state
	audioHandler VoiceAudioHandler
	mu           sync.RWMutex
}

// VoiceAudioHandler handles incoming voice audio.
type VoiceAudioHandler func(guildID, channelID, userID string, audio []byte)

// NewVoiceManager creates a new voice manager for the provider.
func NewVoiceManager(p *Provider, config VoiceConfig) *VoiceManager {
	if config.AutoLeaveTimeout == 0 {
		config.AutoLeaveTimeout = 5 * time.Minute
	}

	return &VoiceManager{
		provider:    p,
		config:      config,
		connections: make(map[string]*VoiceConnection),
		voiceStates: make(map[string]*VoiceState),
	}
}

// SetAudioHandler sets the handler for incoming voice audio.
func (vm *VoiceManager) SetAudioHandler(handler VoiceAudioHandler) {
	vm.audioHandler = handler
}

// RegisterHandlers registers voice-related event handlers with the session.
func (vm *VoiceManager) RegisterHandlers(session *discordgo.Session) {
	session.AddHandler(vm.handleVoiceStateUpdate)
	session.AddHandler(vm.handleVoiceSpeakingUpdate)
}

// JoinChannel joins a voice channel.
func (vm *VoiceManager) JoinChannel(ctx context.Context, guildID, channelID string) error {
	// Check allowlist/blocklist
	if !vm.isChannelAllowed(channelID) {
		return fmt.Errorf("channel %s is not allowed", channelID)
	}

	vm.mu.Lock()
	defer vm.mu.Unlock()

	// Check if already connected to this guild
	if conn, ok := vm.connections[guildID]; ok {
		if conn.chanID == channelID {
			return nil // Already in this channel
		}
		// Move to new channel
		return vm.moveChannel(guildID, channelID)
	}

	// Join new channel
	vc, err := vm.provider.session.ChannelVoiceJoin(guildID, channelID, false, false)
	if err != nil {
		return fmt.Errorf("join voice channel: %w", err)
	}

	vm.connections[guildID] = &VoiceConnection{
		conn:    vc,
		guildID: guildID,
		chanID:  channelID,
	}

	vm.provider.logger.Info("joined voice channel",
		"guild_id", guildID,
		"channel_id", channelID)

	return nil
}

// LeaveChannel leaves a voice channel.
func (vm *VoiceManager) LeaveChannel(guildID string) error {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	conn, ok := vm.connections[guildID]
	if !ok {
		return nil // Not connected
	}

	if err := conn.conn.Disconnect(); err != nil {
		return fmt.Errorf("disconnect voice: %w", err)
	}

	delete(vm.connections, guildID)

	vm.provider.logger.Info("left voice channel",
		"guild_id", guildID,
		"channel_id", conn.chanID)

	return nil
}

// moveChannel moves to a different voice channel in the same guild.
func (vm *VoiceManager) moveChannel(guildID, channelID string) error {
	conn, ok := vm.connections[guildID]
	if !ok {
		return fmt.Errorf("not connected to guild %s", guildID)
	}

	if err := conn.conn.ChangeChannel(channelID, false, false); err != nil {
		return fmt.Errorf("change channel: %w", err)
	}

	conn.chanID = channelID
	return nil
}

// GetConnection returns the voice connection for a guild.
func (vm *VoiceManager) GetConnection(guildID string) *VoiceConnection {
	vm.mu.RLock()
	defer vm.mu.RUnlock()
	return vm.connections[guildID]
}

// IsConnected returns true if connected to a voice channel in the guild.
func (vm *VoiceManager) IsConnected(guildID string) bool {
	vm.mu.RLock()
	defer vm.mu.RUnlock()
	_, ok := vm.connections[guildID]
	return ok
}

// GetVoiceState returns a user's voice state.
func (vm *VoiceManager) GetVoiceState(userID string) *VoiceState {
	vm.mu.RLock()
	defer vm.mu.RUnlock()
	return vm.voiceStates[userID]
}

// GetChannelUsers returns all users in a voice channel.
func (vm *VoiceManager) GetChannelUsers(channelID string) []string {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	var users []string
	for userID, state := range vm.voiceStates {
		if state.ChannelID == channelID {
			users = append(users, userID)
		}
	}
	return users
}

// isChannelAllowed checks if a channel is allowed based on config.
func (vm *VoiceManager) isChannelAllowed(channelID string) bool {
	// Check blocklist first
	for _, blocked := range vm.config.ChannelBlocklist {
		if blocked == channelID {
			return false
		}
	}

	// If allowlist is empty, all non-blocked channels are allowed
	if len(vm.config.ChannelAllowlist) == 0 {
		return true
	}

	// Check allowlist
	for _, allowed := range vm.config.ChannelAllowlist {
		if allowed == channelID {
			return true
		}
	}

	return false
}

// shouldFollowUser checks if the bot should follow a user.
func (vm *VoiceManager) shouldFollowUser(userID string) bool {
	for _, id := range vm.config.FollowUsers {
		if id == userID {
			return true
		}
	}
	return false
}

// handleVoiceStateUpdate handles voice state change events.
func (vm *VoiceManager) handleVoiceStateUpdate(s *discordgo.Session, v *discordgo.VoiceStateUpdate) {
	vm.mu.Lock()

	oldState := vm.voiceStates[v.UserID]
	var oldChannelID string
	if oldState != nil {
		oldChannelID = oldState.ChannelID
	}

	// Update voice state
	if v.ChannelID == "" {
		// User left voice
		delete(vm.voiceStates, v.UserID)
	} else {
		vm.voiceStates[v.UserID] = &VoiceState{
			UserID:    v.UserID,
			ChannelID: v.ChannelID,
			GuildID:   v.GuildID,
			Muted:     v.Mute || v.SelfMute,
			Deafened:  v.Deaf || v.SelfDeaf,
		}
	}

	vm.mu.Unlock()

	// Determine event type
	var eventType provider.EventType
	if oldChannelID == "" && v.ChannelID != "" {
		eventType = provider.EventTypeVoiceJoin
	} else if oldChannelID != "" && v.ChannelID == "" {
		eventType = provider.EventTypeVoiceLeave
	} else if oldChannelID != v.ChannelID {
		eventType = provider.EventTypeVoiceMove
	} else {
		return // No channel change
	}

	// Emit event
	if vm.provider.eventHandler != nil {
		event := provider.Event{
			Type:         eventType,
			ProviderName: "discord",
			ChatID:       v.ChannelID,
			Data: map[string]any{
				"user_id":        v.UserID,
				"guild_id":       v.GuildID,
				"old_channel_id": oldChannelID,
				"new_channel_id": v.ChannelID,
				"muted":          v.Mute || v.SelfMute,
				"deafened":       v.Deaf || v.SelfDeaf,
			},
			Timestamp: time.Now(),
		}
		if err := vm.provider.eventHandler(context.Background(), event); err != nil {
			vm.provider.logger.Error("voice event handler error", "error", err)
		}
	}

	// Auto-follow configured users
	if eventType == provider.EventTypeVoiceJoin && vm.shouldFollowUser(v.UserID) {
		if vm.isChannelAllowed(v.ChannelID) {
			go func() {
				if err := vm.JoinChannel(context.Background(), v.GuildID, v.ChannelID); err != nil {
					vm.provider.logger.Error("auto-follow failed",
						"user_id", v.UserID,
						"channel_id", v.ChannelID,
						"error", err)
				}
			}()
		}
	}

	// Auto-leave empty channels
	if vm.config.AutoLeaveEmpty && eventType == provider.EventTypeVoiceLeave {
		vm.checkAutoLeave(v.GuildID, oldChannelID)
	}
}

// handleVoiceSpeakingUpdate handles voice speaking state changes.
func (vm *VoiceManager) handleVoiceSpeakingUpdate(s *discordgo.Session, v *discordgo.VoiceSpeakingUpdate) {
	vm.mu.Lock()
	if state, ok := vm.voiceStates[v.UserID]; ok {
		state.Speaking = v.Speaking
	}
	vm.mu.Unlock()

	// Emit speaking event
	if vm.provider.eventHandler != nil {
		event := provider.Event{
			Type:         provider.EventTypeVoiceSpeaker,
			ProviderName: "discord",
			Data: map[string]any{
				"user_id":  v.UserID,
				"speaking": v.Speaking,
				"ssrc":     v.SSRC,
			},
			Timestamp: time.Now(),
		}
		if err := vm.provider.eventHandler(context.Background(), event); err != nil {
			vm.provider.logger.Error("voice speaking event handler error", "error", err)
		}
	}
}

// checkAutoLeave checks if the bot should auto-leave an empty channel.
func (vm *VoiceManager) checkAutoLeave(guildID, channelID string) {
	vm.mu.RLock()
	conn, ok := vm.connections[guildID]
	if !ok || conn.chanID != channelID {
		vm.mu.RUnlock()
		return
	}

	// Count users in channel (excluding the bot)
	count := 0
	botID := vm.provider.session.State.User.ID
	for _, state := range vm.voiceStates {
		if state.ChannelID == channelID && state.UserID != botID {
			count++
		}
	}
	vm.mu.RUnlock()

	if count == 0 {
		// Schedule auto-leave
		go func() {
			time.Sleep(vm.config.AutoLeaveTimeout)

			// Re-check before leaving
			vm.mu.RLock()
			count := 0
			for _, state := range vm.voiceStates {
				if state.ChannelID == channelID && state.UserID != botID {
					count++
				}
			}
			vm.mu.RUnlock()

			if count == 0 {
				if err := vm.LeaveChannel(guildID); err != nil {
					vm.provider.logger.Error("auto-leave failed",
						"guild_id", guildID,
						"error", err)
				}
			}
		}()
	}
}

// SendAudio sends audio data to a voice channel.
func (vc *VoiceConnection) SendAudio(data []byte) error {
	if vc.conn == nil || vc.conn.OpusSend == nil {
		return fmt.Errorf("voice connection not ready")
	}

	vc.conn.OpusSend <- data
	return nil
}

// SetSpeaking sets the speaking state.
func (vc *VoiceConnection) SetSpeaking(speaking bool) error {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	if vc.speaking == speaking {
		return nil
	}

	if err := vc.conn.Speaking(speaking); err != nil {
		return fmt.Errorf("set speaking: %w", err)
	}

	vc.speaking = speaking
	return nil
}

// ReceiveAudio starts receiving audio from the voice channel.
// Returns a channel that receives audio packets.
func (vc *VoiceConnection) ReceiveAudio() <-chan *discordgo.Packet {
	return vc.conn.OpusRecv
}

// StreamAudio streams audio from a reader to the voice channel.
// The reader should provide Opus-encoded audio at 48kHz stereo.
func (vc *VoiceConnection) StreamAudio(ctx context.Context, reader io.Reader, frameSize int) error {
	if err := vc.SetSpeaking(true); err != nil {
		return err
	}
	defer func() {
		if err := vc.SetSpeaking(false); err != nil {
			slogutil.LoggerFromContext(ctx, slog.Default()).Error("failed to set speaking state", "error", err)
		}
	}()

	buf := make([]byte, frameSize)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			n, err := reader.Read(buf)
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return fmt.Errorf("read audio: %w", err)
			}

			if err := vc.SendAudio(buf[:n]); err != nil {
				return err
			}
		}
	}
}

// GuildID returns the guild ID of the connection.
func (vc *VoiceConnection) GuildID() string {
	return vc.guildID
}

// ChannelID returns the channel ID of the connection.
func (vc *VoiceConnection) ChannelID() string {
	vc.mu.RLock()
	defer vc.mu.RUnlock()
	return vc.chanID
}
