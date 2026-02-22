package provider

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// AgentProcessor processes messages through an AI agent.
type AgentProcessor interface {
	Process(ctx context.Context, sessionID, content string) (string, error)
}

// Router routes messages between providers and handlers.
type Router struct {
	providers map[string]Provider
	handlers  []RouteHandler
	agent     AgentProcessor
	logger    *slog.Logger
	mu        sync.RWMutex
}

// RouteHandler processes routed messages.
type RouteHandler struct {
	Pattern RoutePattern
	Handler MessageHandler
}

// RoutePattern defines which messages to match.
type RoutePattern struct {
	// Providers limits to specific providers (empty = all).
	Providers []string

	// ChatTypes limits to specific chat types (empty = all).
	ChatTypes []ChatType

	// Prefix matches messages starting with a prefix.
	Prefix string
}

// NewRouter creates a new message router.
func NewRouter(logger *slog.Logger) *Router {
	if logger == nil {
		logger = slog.Default()
	}
	return &Router{
		providers: make(map[string]Provider),
		handlers:  []RouteHandler{},
		logger:    logger,
	}
}

// SetAgent sets the agent processor for the router.
func (r *Router) SetAgent(agent AgentProcessor) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.agent = agent
}

// ProcessWithAgent creates a message handler that processes through the agent and sends responses.
func (r *Router) ProcessWithAgent() MessageHandler {
	return func(ctx context.Context, msg IncomingMessage) error {
		r.mu.RLock()
		agent := r.agent
		r.mu.RUnlock()

		if agent == nil {
			r.logger.Warn("no agent configured, message not processed",
				"provider", msg.ProviderName,
				"chat", msg.ChatID)
			return nil
		}

		// Use chatID as session ID for conversation continuity
		sessionID := fmt.Sprintf("%s:%s", msg.ProviderName, msg.ChatID)

		r.logger.Info("processing message",
			"provider", msg.ProviderName,
			"chat", msg.ChatID,
			"from", msg.SenderName)

		response, err := agent.Process(ctx, sessionID, msg.Content)
		if err != nil {
			r.logger.Error("agent processing error",
				"provider", msg.ProviderName,
				"chat", msg.ChatID,
				"error", err)
			return err
		}

		// Send response back to the same provider/chat
		return r.Send(ctx, msg.ProviderName, msg.ChatID, OutgoingMessage{
			Content: response,
			ReplyTo: msg.ID,
		})
	}
}

// Register adds a provider to the router.
func (r *Router) Register(provider Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := provider.Name()
	r.providers[name] = provider

	// Set up message handler
	provider.OnMessage(func(ctx context.Context, msg IncomingMessage) error {
		return r.route(ctx, msg)
	})

	r.logger.Info("provider registered", "name", name)
}

// Unregister removes a provider from the router.
func (r *Router) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.providers, name)
	r.logger.Info("provider unregistered", "name", name)
}

// OnMessage adds a message handler with a pattern.
func (r *Router) OnMessage(pattern RoutePattern, handler MessageHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers = append(r.handlers, RouteHandler{
		Pattern: pattern,
		Handler: handler,
	})
}

// Send sends a message to a specific provider and chat.
func (r *Router) Send(ctx context.Context, providerName, chatID string, msg OutgoingMessage) error {
	r.mu.RLock()
	provider, ok := r.providers[providerName]
	r.mu.RUnlock()

	if !ok {
		return fmt.Errorf("provider not found: %s", providerName)
	}

	return provider.Send(ctx, chatID, msg)
}

// Broadcast sends a message to all registered providers.
func (r *Router) Broadcast(ctx context.Context, chatIDs map[string]string, msg OutgoingMessage) error {
	r.mu.RLock()
	providers := make(map[string]Provider, len(r.providers))
	for k, v := range r.providers {
		providers[k] = v
	}
	r.mu.RUnlock()

	var errs []error
	for name, chatID := range chatIDs {
		if provider, ok := providers[name]; ok {
			if err := provider.Send(ctx, chatID, msg); err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", name, err))
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("broadcast errors: %v", errs)
	}
	return nil
}

// ConnectAll connects all registered providers.
func (r *Router) ConnectAll(ctx context.Context) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for name, provider := range r.providers {
		if err := provider.Connect(ctx); err != nil {
			return fmt.Errorf("connect %s: %w", name, err)
		}
		r.logger.Info("provider connected", "name", name)
	}
	return nil
}

// DisconnectAll disconnects all registered providers.
func (r *Router) DisconnectAll(ctx context.Context) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var errs []error
	for name, provider := range r.providers {
		if err := provider.Disconnect(ctx); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", name, err))
		} else {
			r.logger.Info("provider disconnected", "name", name)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("disconnect errors: %v", errs)
	}
	return nil
}

// GetProvider returns a provider by name.
func (r *Router) GetProvider(name string) (Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[name]
	return p, ok
}

// ListProviders returns all registered provider names.
func (r *Router) ListProviders() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}

// route dispatches a message to matching handlers.
func (r *Router) route(ctx context.Context, msg IncomingMessage) error {
	r.mu.RLock()
	handlers := make([]RouteHandler, len(r.handlers))
	copy(handlers, r.handlers)
	r.mu.RUnlock()

	for _, h := range handlers {
		if matchPattern(h.Pattern, msg) {
			if err := h.Handler(ctx, msg); err != nil {
				r.logger.Error("handler error",
					"provider", msg.ProviderName,
					"chat", msg.ChatID,
					"error", err)
				// Continue to other handlers
			}
		}
	}
	return nil
}

// matchPattern checks if a message matches a route pattern.
func matchPattern(pattern RoutePattern, msg IncomingMessage) bool {
	// Check provider filter
	if len(pattern.Providers) > 0 {
		found := false
		for _, p := range pattern.Providers {
			if p == msg.ProviderName {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check chat type filter
	if len(pattern.ChatTypes) > 0 {
		found := false
		for _, ct := range pattern.ChatTypes {
			if ct == msg.ChatType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check prefix filter
	if pattern.Prefix != "" {
		if len(msg.Content) < len(pattern.Prefix) {
			return false
		}
		if msg.Content[:len(pattern.Prefix)] != pattern.Prefix {
			return false
		}
	}

	return true
}

// All returns a pattern that matches all messages.
func All() RoutePattern {
	return RoutePattern{}
}

// FromProviders returns a pattern that matches messages from specific providers.
func FromProviders(providers ...string) RoutePattern {
	return RoutePattern{Providers: providers}
}

// DMOnly returns a pattern that matches only DM messages.
func DMOnly() RoutePattern {
	return RoutePattern{ChatTypes: []ChatType{ChatTypeDM}}
}

// GroupOnly returns a pattern that matches only group messages.
func GroupOnly() RoutePattern {
	return RoutePattern{ChatTypes: []ChatType{ChatTypeGroup}}
}
