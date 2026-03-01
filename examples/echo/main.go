// Package main demonstrates a simple echo bot using omnichat providers.
package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/mdp/qrterminal/v3"

	"github.com/plexusone/omnichat/provider"
	"github.com/plexusone/omnichat/providers/discord"
	"github.com/plexusone/omnichat/providers/telegram"
	"github.com/plexusone/omnichat/providers/whatsapp"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	// Create a router for managing providers
	router := provider.NewRouter(logger)

	// Register providers based on available tokens
	if token := os.Getenv("DISCORD_TOKEN"); token != "" {
		discordProvider, err := discord.New(discord.Config{
			Token:  token,
			Logger: logger,
		})
		if err != nil {
			log.Fatalf("create discord provider: %v", err)
		}
		router.Register(discordProvider)
		logger.Info("discord provider registered")
	}

	if token := os.Getenv("TELEGRAM_TOKEN"); token != "" {
		telegramProvider, err := telegram.New(telegram.Config{
			Token:  token,
			Logger: logger,
		})
		if err != nil {
			log.Fatalf("create telegram provider: %v", err)
		}
		router.Register(telegramProvider)
		logger.Info("telegram provider registered")
	}

	if os.Getenv("WHATSAPP_ENABLED") == "true" {
		waProvider, err := whatsapp.New(whatsapp.Config{
			DBPath: os.Getenv("WHATSAPP_DB_PATH"),
			Logger: logger,
			QRCallback: func(qr string) {
				// Render QR code in terminal for scanning with WhatsApp
				fmt.Println("\n Scan this QR code with WhatsApp:")
				fmt.Println()
				qrterminal.GenerateWithConfig(qr, qrterminal.Config{
					Level:     qrterminal.L,
					Writer:    os.Stdout,
					BlackChar: qrterminal.WHITE,
					WhiteChar: qrterminal.BLACK,
					QuietZone: 1,
				})
				fmt.Println()
			},
		})
		if err != nil {
			log.Fatalf("create whatsapp provider: %v", err)
		}
		router.Register(waProvider)
		logger.Info("whatsapp provider registered")
	}

	// Check if any providers were registered
	providers := router.ListProviders()
	if len(providers) == 0 {
		log.Fatal("no providers configured. Set DISCORD_TOKEN, TELEGRAM_TOKEN, or WHATSAPP_ENABLED=true")
	}

	// Register an echo handler for all messages
	router.OnMessage(provider.All(), func(ctx context.Context, msg provider.IncomingMessage) error {
		logger.Info("received message",
			"provider", msg.ProviderName,
			"chat", msg.ChatID,
			"from", msg.SenderName,
			"content", msg.Content)

		// Echo the message back
		return router.Send(ctx, msg.ProviderName, msg.ChatID, provider.OutgoingMessage{
			Content: "Echo: " + msg.Content,
			ReplyTo: msg.ID,
		})
	})

	// Connect all providers
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := router.ConnectAll(ctx); err != nil {
		log.Fatalf("connect providers: %v", err)
	}
	logger.Info("all providers connected", "count", len(providers))

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	logger.Info("echo bot running, press Ctrl+C to stop")
	<-sigChan

	// Disconnect all providers
	logger.Info("shutting down...")
	if err := router.DisconnectAll(ctx); err != nil {
		logger.Error("disconnect error", "error", err)
	}
	logger.Info("goodbye!")
}
