// Copyright 2025 John Wang. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package discord

import (
	"log/slog"
	"testing"
	"time"
)

func TestVoiceConfig_Defaults(t *testing.T) {
	p := &Provider{logger: slog.Default()}
	config := VoiceConfig{}

	vm := NewVoiceManager(p, config)

	if vm.config.AutoLeaveTimeout != 5*time.Minute {
		t.Errorf("AutoLeaveTimeout = %v, want %v", vm.config.AutoLeaveTimeout, 5*time.Minute)
	}
}

func TestVoiceManager_IsChannelAllowed(t *testing.T) {
	p := &Provider{logger: slog.Default()}

	tests := []struct {
		name      string
		config    VoiceConfig
		channelID string
		want      bool
	}{
		{
			name:      "empty lists allow all",
			config:    VoiceConfig{},
			channelID: "123",
			want:      true,
		},
		{
			name: "allowlist allows specified",
			config: VoiceConfig{
				ChannelAllowlist: []string{"123", "456"},
			},
			channelID: "123",
			want:      true,
		},
		{
			name: "allowlist blocks unspecified",
			config: VoiceConfig{
				ChannelAllowlist: []string{"123", "456"},
			},
			channelID: "789",
			want:      false,
		},
		{
			name: "blocklist blocks specified",
			config: VoiceConfig{
				ChannelBlocklist: []string{"123"},
			},
			channelID: "123",
			want:      false,
		},
		{
			name: "blocklist allows unspecified",
			config: VoiceConfig{
				ChannelBlocklist: []string{"123"},
			},
			channelID: "456",
			want:      true,
		},
		{
			name: "blocklist overrides allowlist",
			config: VoiceConfig{
				ChannelAllowlist: []string{"123", "456"},
				ChannelBlocklist: []string{"123"},
			},
			channelID: "123",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vm := NewVoiceManager(p, tt.config)
			got := vm.isChannelAllowed(tt.channelID)
			if got != tt.want {
				t.Errorf("isChannelAllowed(%q) = %v, want %v", tt.channelID, got, tt.want)
			}
		})
	}
}

func TestVoiceManager_ShouldFollowUser(t *testing.T) {
	p := &Provider{logger: slog.Default()}

	config := VoiceConfig{
		FollowUsers: []string{"user1", "user2"},
	}

	vm := NewVoiceManager(p, config)

	if !vm.shouldFollowUser("user1") {
		t.Error("shouldFollowUser(user1) = false, want true")
	}

	if !vm.shouldFollowUser("user2") {
		t.Error("shouldFollowUser(user2) = false, want true")
	}

	if vm.shouldFollowUser("user3") {
		t.Error("shouldFollowUser(user3) = true, want false")
	}
}

func TestVoiceManager_GetChannelUsers(t *testing.T) {
	p := &Provider{logger: slog.Default()}
	vm := NewVoiceManager(p, VoiceConfig{})

	// Add some voice states
	vm.voiceStates["user1"] = &VoiceState{UserID: "user1", ChannelID: "chan1"}
	vm.voiceStates["user2"] = &VoiceState{UserID: "user2", ChannelID: "chan1"}
	vm.voiceStates["user3"] = &VoiceState{UserID: "user3", ChannelID: "chan2"}

	users := vm.GetChannelUsers("chan1")
	if len(users) != 2 {
		t.Errorf("GetChannelUsers(chan1) returned %d users, want 2", len(users))
	}

	users = vm.GetChannelUsers("chan2")
	if len(users) != 1 {
		t.Errorf("GetChannelUsers(chan2) returned %d users, want 1", len(users))
	}

	users = vm.GetChannelUsers("chan3")
	if len(users) != 0 {
		t.Errorf("GetChannelUsers(chan3) returned %d users, want 0", len(users))
	}
}

func TestVoiceManager_GetVoiceState(t *testing.T) {
	p := &Provider{logger: slog.Default()}
	vm := NewVoiceManager(p, VoiceConfig{})

	vm.voiceStates["user1"] = &VoiceState{
		UserID:    "user1",
		ChannelID: "chan1",
		GuildID:   "guild1",
		Muted:     true,
	}

	state := vm.GetVoiceState("user1")
	if state == nil {
		t.Fatal("GetVoiceState(user1) returned nil")
	}

	if state.Muted != true {
		t.Error("state.Muted = false, want true")
	}

	state = vm.GetVoiceState("nonexistent")
	if state != nil {
		t.Error("GetVoiceState(nonexistent) should return nil")
	}
}

func TestVoiceManager_IsConnected(t *testing.T) {
	p := &Provider{logger: slog.Default()}
	vm := NewVoiceManager(p, VoiceConfig{})

	if vm.IsConnected("guild1") {
		t.Error("IsConnected should return false when not connected")
	}

	// Simulate connection
	vm.connections["guild1"] = &VoiceConnection{
		guildID: "guild1",
		chanID:  "chan1",
	}

	if !vm.IsConnected("guild1") {
		t.Error("IsConnected should return true when connected")
	}
}

func TestVoiceConnection_ChannelID(t *testing.T) {
	vc := &VoiceConnection{
		guildID: "guild1",
		chanID:  "chan1",
	}

	if vc.GuildID() != "guild1" {
		t.Errorf("GuildID() = %q, want %q", vc.GuildID(), "guild1")
	}

	if vc.ChannelID() != "chan1" {
		t.Errorf("ChannelID() = %q, want %q", vc.ChannelID(), "chan1")
	}
}

func TestProvider_Voice(t *testing.T) {
	p := &Provider{logger: slog.Default()}

	// Without voice config
	if p.Voice() != nil {
		t.Error("Voice() should return nil when not configured")
	}

	// With voice config
	p.voiceManager = NewVoiceManager(p, VoiceConfig{})
	if p.Voice() == nil {
		t.Error("Voice() should return manager when configured")
	}
}
