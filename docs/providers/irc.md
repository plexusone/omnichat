# IRC

The IRC provider uses [ergochat/irc-go](https://github.com/ergochat/irc-go) for connecting to IRC servers.

## Installation

```bash
go get github.com/plexusone/omnichat/providers/irc
```

## Configuration

```go
import "github.com/plexusone/omnichat/providers/irc"

p, err := irc.New(irc.Config{
    Server:   "irc.libera.chat:6697",
    Nick:     "mybot",
    Password: "nickserv-password",  // Optional
    Channels: []string{"#channel1", "#channel2"},
    UseTLS:   true,
    Logger:   slog.Default(),
})
```

### Config Options

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `Server` | `string` | Yes | IRC server address (e.g., `irc.libera.chat:6697`) |
| `Nick` | `string` | Yes | Bot nickname |
| `User` | `string` | No | Username (defaults to Nick) |
| `RealName` | `string` | No | Real name field (defaults to Nick) |
| `Password` | `string` | No | NickServ password for authentication |
| `Channels` | `[]string` | No | Channels to join on connect |
| `UseTLS` | `bool` | No | Use TLS encryption (default: true) |
| `Logger` | `*slog.Logger` | No | Logger instance |

## IRC Networks

Common public IRC networks:

| Network | Server | TLS Port | Website |
|---------|--------|----------|---------|
| Libera.Chat | `irc.libera.chat` | 6697 | [libera.chat](https://libera.chat) |
| OFTC | `irc.oftc.net` | 6697 | [oftc.net](https://www.oftc.net) |
| EFnet | `irc.efnet.org` | 6697 | [efnet.org](http://www.efnet.org) |
| IRCnet | `open.ircnet.net` | 6697 | [ircnet.org](http://www.ircnet.org) |

## Usage

### Basic Setup

```go
package main

import (
    "context"
    "log/slog"
    "os"

    "github.com/plexusone/omnichat/provider"
    "github.com/plexusone/omnichat/providers/irc"
)

func main() {
    logger := slog.Default()

    p, err := irc.New(irc.Config{
        Server:   "irc.libera.chat:6697",
        Nick:     "mybot",
        Password: os.Getenv("IRC_PASSWORD"),
        Channels: []string{"#mychannel"},
        UseTLS:   true,
        Logger:   logger,
    })
    if err != nil {
        panic(err)
    }

    router := provider.NewRouter(logger)
    router.Register(p)

    router.OnMessage(provider.All(), func(ctx context.Context, msg provider.IncomingMessage) error {
        return router.Send(ctx, "irc", msg.ChatID, provider.OutgoingMessage{
            Content: "Hello from IRC!",
        })
    })

    ctx := context.Background()
    router.ConnectAll(ctx)
    defer router.DisconnectAll(ctx)

    select {}
}
```

### Sending Messages

```go
// Send to channel
router.Send(ctx, "irc", "#mychannel", provider.OutgoingMessage{
    Content: "Hello, channel!",
})

// Send direct message
router.Send(ctx, "irc", "username", provider.OutgoingMessage{
    Content: "Hello via DM!",
})
```

### Joining Channels

Channels specified in config are joined automatically on connect. You can also join dynamically:

```go
// Type assertion to access IRC-specific methods
if ircProvider, ok := p.(*irc.Provider); ok {
    ircProvider.JoinChannel("#newchannel")
    ircProvider.PartChannel("#oldchannel", "")
}
```

## Message Mapping

| IRC | OmniChat |
|-----|----------|
| Target (channel/nick) | `ChatID` |
| Timestamp (nanoseconds) | `ID` |
| Sender nick | `SenderID`, `SenderName` |
| Message text | `Content` |
| Channel (#prefix) | `ChatType = ChatTypeChannel` |
| Private message | `ChatType = ChatTypeDM` |

## Event Handling

The provider handles these IRC events:

### Messages (PRIVMSG)

```go
router.OnMessage(provider.All(), func(ctx context.Context, msg provider.IncomingMessage) error {
    // msg.ChatID is "#channel" for channels, "nick" for DMs
    // msg.ChatType is ChatTypeChannel or ChatTypeDM
    // msg.Content is the message text
    return nil
})
```

### Join/Part Events

```go
router.OnEvent(func(ctx context.Context, event provider.Event) error {
    switch event.Type {
    case provider.EventTypeMemberJoined:
        nick := event.Data["nick"].(string)
        log.Printf("%s joined %s", nick, event.ChatID)
    case provider.EventTypeMemberLeft:
        nick := event.Data["nick"].(string)
        reason := event.Data["reason"].(string)
        log.Printf("%s left %s: %s", nick, event.ChatID, reason)
    }
    return nil
})
```

## NickServ Authentication

Most IRC networks support NickServ for nickname registration:

### Register a Nickname (Manual)

Connect with a regular IRC client and register:

```
/msg NickServ REGISTER password email@example.com
```

### Configure Password

```go
p, err := irc.New(irc.Config{
    Server:   "irc.libera.chat:6697",
    Nick:     "registeredbot",
    Password: os.Getenv("IRC_PASSWORD"),
    // ...
})
```

The provider sends the password during connection for NickServ identification.

## TLS Configuration

TLS is enabled by default. To disable (not recommended):

```go
p, err := irc.New(irc.Config{
    Server: "irc.example.com:6667",  // Non-TLS port
    Nick:   "mybot",
    UseTLS: false,
})
```

Standard ports:

- TLS: 6697
- Unencrypted: 6667

## Message Length Limits

IRC has line length limits (~512 bytes including protocol overhead). The provider automatically splits long messages at word boundaries.

## Environment Variables

```bash
IRC_SERVER=irc.libera.chat:6697
IRC_NICK=mybot
IRC_PASSWORD=nickserv-password
IRC_CHANNELS=#channel1,#channel2
```

## Troubleshooting

### Connection refused

1. Verify server address and port
2. Check if TLS is required (most networks require TLS on port 6697)
3. Ensure no firewall blocking

### Nickname already in use

1. Choose a different nickname
2. If you own the nickname, configure NickServ password

### Cannot join channel

1. Channel may be invite-only (+i)
2. Channel may require registration (+r)
3. Contact channel operators

### Messages not received

1. Verify bot has joined the channel
2. Check channel modes
3. Review logs for errors

## Limitations

- No message editing or deletion (IRC protocol limitation)
- No threading or replies
- No native media support
- No read receipts or typing indicators
- Message length limits (~400 characters per line)

## Next Steps

- [Router](../reference/router.md) - Message routing
- [Testing](../reference/testing.md) - Test your implementation
