# Tasks

Roadmap and future work for OmniChat.

## Providers

### New Providers

- [ ] SMS via Twilio (`providers/sms/twilio`)
- [ ] SMS via Telnyx (`providers/sms/telnyx`)
- [ ] Microsoft Teams (`providers/teams`)
- [ ] Matrix (`providers/matrix`)

### Provider Enhancements

- [ ] Gmail: Add receive/inbox support (currently send-only)
- [ ] Slack: Block Kit support for rich messages
- [ ] Discord: Slash commands support
- [ ] Telegram: Inline mode support

## Features

### Message Handling

- [ ] Message editing support
- [ ] Message deletion support
- [ ] Reactions support across providers
- [ ] Read receipts where supported
- [ ] Typing indicators where supported

### Infrastructure

- [ ] Webhook mode as alternative to polling/Socket Mode
- [ ] Message queue integration (Redis, NATS)
- [ ] Configurable rate limiting per provider
- [ ] Retry logic with exponential backoff

## Testing

- [ ] Integration test suite with live provider connections
- [ ] Provider conformance test expansion
- [ ] Benchmark suite for message throughput

## Documentation

- [ ] Video tutorials for provider setup
- [ ] Architecture decision records (ADRs)
- [ ] Contributing guide
