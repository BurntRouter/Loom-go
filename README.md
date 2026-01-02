# Loom-go

Go client library for the Loom service - a high-performance message streaming platform supporting QUIC and HTTP/3 protocols.

## Features

- **QUIC Streams**: Low-latency bidirectional streaming using QUIC protocol (`transport=quic`)
- **HTTP/3 Support**: Standard HTTP/3 transport via POST /stream (`transport=h3`)
- **TLS Configuration**: Flexible TLS settings including `InsecureSkipVerify` for development
- **Producer/Consumer Pattern**: Clean API for publishing and consuming messages
- **Streaming Messages**: Efficient chunked message streaming with declarable sizes
- **Room-based Routing**: Organize message flows using named rooms
- **Token Authentication**: Built-in support for authentication tokens

## Installation

```bash
go get github.com/BurntRouter/Loom-go
```

## Quick Start

### Producer Example

```go
package main

import (
    "context"
    "log"
    "strings"
    "time"

    "github.com/BurntRouter/Loom-go/loom"
)

func main() {
    ctx := context.Background()

    // Create a producer
    producer, err := loom.NewProducer(ctx, loom.ClientOptions{
        Server:    "localhost:4242",
        Transport: loom.TransportQUIC,
        Name:      "my-producer",
        Room:      "default",
        Token:     "your-auth-token",
        TLS: loom.TLSOptions{
            InsecureSkipVerify: true, // Only for development!
        },
    })
    if err != nil {
        log.Fatal(err)
    }
    defer producer.Close()

    // Send a message
    message := strings.NewReader("Hello, Loom!")
    err = producer.Produce(
        ctx,
        []byte("my-message-key"), // used for partitioning the room
        uint64(message.Len()),
        0, // msgID (passed through to consumer)
        message,
        64<<10, // 64KB chunk size
    )
    if err != nil {
        log.Fatal(err)
    }

    log.Println("Message sent successfully!")
}
```

### Consumer Example

```go
package main

import (
    "context"
    "io"
    "log"

    "github.com/BurntRouter/Loom-go/loom"
)

func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Create a consumer
    consumer, err := loom.NewConsumer(ctx, loom.ClientOptions{
        Server:    "localhost:8443",
        Transport: loom.TransportQUIC,
        Name:      "my-consumer",
        Room:      "default",
        Token:     "your-auth-token",
        TLS: loom.TLSOptions{
            InsecureSkipVerify: true, // Only for development!
        },
    })
    if err != nil {
        log.Fatal(err)
    }
    defer consumer.Close()

    // Consume messages
    err = consumer.Consume(ctx, func(msg loom.Message) error {
        log.Printf("Received message with key: %s (size: %d bytes)", 
            string(msg.Key), msg.DeclaredSize)

        // Read the message body
        body, err := io.ReadAll(msg.Reader)
        if err != nil {
            return err
        }

        log.Printf("Message content: %s", string(body))
        return nil
    })
    if err != nil && err != context.Canceled {
        log.Fatal(err)
    }
}
```

### Manual Message Processing

For more control over message acknowledgment, use the `Next()` method:

```go
for {
    msg, err := consumer.Next()
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Key: %s, Size: %d", string(msg.Key), msg.DeclaredSize)

    // Process message body in chunks or all at once
    body, err := io.ReadAll(msg.Reader)
    if err != nil {
        log.Fatal(err)
    }

    // Manually acknowledge the message
    if err := consumer.Ack(msg.MsgID); err != nil {
        log.Fatal(err)
    }
}
```

## Configuration Options

### ClientOptions

```go
type ClientOptions struct {
    Server    string         // Server address (e.g., "localhost:8443")
    Transport string         // "quic" or "h3"
    Name      string         // Client identifier
    Room      string         // Room name for message routing
    Token     string         // Authentication token
    TLS       TLSOptions     // TLS configuration
}
```

### Transport Types

- `loom.TransportQUIC` - QUIC-based transport (default, recommended for low latency)
- `loom.TransportH3` - HTTP/3 transport (better firewall compatibility)

### TLS Options

```go
type TLSOptions struct {
    InsecureSkipVerify bool   // Skip certificate verification (dev only)
}
```

## Protocol Details

Loom uses a custom binary protocol over QUIC or HTTP/3:

1. **Handshake**: Magic bytes "LOOM" + version byte + role (Producer/Consumer) + name + room + token
2. **Messages**: Key + declared size + message ID + chunked body
3. **Chunks**: Variable-length encoded size + data
4. **End of Message**: Zero-length chunk
5. **Acknowledgments**: Frame type + message ID

## Error Handling

Always check for errors and properly close connections:

```go
producer, err := loom.NewProducer(ctx, options)
if err != nil {
    return fmt.Errorf("failed to create producer: %w", err)
}
defer producer.Close()
```

## Context Cancellation

All operations respect context cancellation:

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

err := producer.Produce(ctx, key, size, 0, reader, chunkSize)
```

## Requirements

- Go 1.24.0 or later
- github.com/quic-go/quic-go v0.58.0+

## License

Apache 2.0
