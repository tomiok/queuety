# Queuety Protocol Specification

## Overview

Queuety uses a TCP-based binary protocol for client-server communication. The protocol supports both JSON and binary message formats with a simple framing mechanism.

## Wire Format

All messages sent over the wire follow this structure:

```
┌──────────────┬──────────────────┬────────────────────┐
│ Format Flag  │  Message Length  │  Message Payload   │
│   (1 byte)   │    (4 bytes)     │   (N bytes)        │
└──────────────┴──────────────────┴────────────────────┘
```

### Field Descriptions

1. **Format Flag** (1 byte)
   - `0x01`: JSON format
   - `0x02`: Binary format

2. **Message Length** (4 bytes)
   - Little-endian unsigned 32-bit integer
   - Indicates the length of the payload in bytes
   - Maximum recommended size: 10 MB (10,485,760 bytes)

3. **Message Payload** (N bytes)
   - The actual message content
   - Format depends on the Format Flag (JSON or Binary)

## Message Structure

### JSON Format (0x01)

When using JSON format, the payload is a UTF-8 encoded JSON object with the following structure:

```json
{
  "id": "string",           // Message ID (UUID format: "false-{uuid}" for undelivered)
  "next_id": "string",      // Next message ID (UUID, used for ACK tracking)
  "type": "string",         // Message type (see Message Types section)
  "user": "string",         // Username (for authentication)
  "password": "string",     // Password (for authentication, plaintext)
  "topic": {                // Topic information
    "Name": "string"        // Topic name
  },
  "body": {},               // Message body (arbitrary JSON)
  "body_string": "string",  // String representation of body
  "timestamp": 0,           // Unix timestamp in seconds
  "ack": false,             // ACK status (true if acknowledged)
  "attempts": 0             // Number of delivery attempts
}
```

**Example JSON Message:**
```json
{
  "id": "false-550e8400-e29b-41d4-a716-446655440000",
  "next_id": "550e8400-e29b-41d4-a716-446655440000",
  "type": "NEW_MESSAGE",
  "user": "",
  "password": "",
  "topic": {
    "Name": "orders"
  },
  "body": {"order_id": 12345, "amount": 99.99},
  "body_string": "{\"order_id\": 12345, \"amount\": 99.99}",
  "timestamp": 1704067200,
  "ack": false,
  "attempts": 0
}
```

### Binary Format (0x02)

When using binary format, the payload is a tightly-packed binary structure. All integers use little-endian byte order.

**Binary Message Structure:**

```
┌─────────────────────────────────────────────────────────┐
│ ID Length (2 bytes) + ID (variable)                     │
├─────────────────────────────────────────────────────────┤
│ Next ID Length (2 bytes) + Next ID (variable)           │
├─────────────────────────────────────────────────────────┤
│ Type Length (2 bytes) + Type (variable)                 │
├─────────────────────────────────────────────────────────┤
│ User Length (2 bytes) + User (variable)                 │
├─────────────────────────────────────────────────────────┤
│ Password Length (2 bytes) + Password (variable)         │
├─────────────────────────────────────────────────────────┤
│ Topic Name Length (2 bytes) + Topic Name (variable)     │
├─────────────────────────────────────────────────────────┤
│ Body Length (4 bytes) + Body (variable)                 │
├─────────────────────────────────────────────────────────┤
│ Body String Length (4 bytes) + Body String (variable)   │
├─────────────────────────────────────────────────────────┤
│ Timestamp (8 bytes, int64)                              │
├─────────────────────────────────────────────────────────┤
│ ACK (1 byte, 0x00 = false, 0x01 = true)                 │
├─────────────────────────────────────────────────────────┤
│ Attempts (4 bytes, int32)                               │
└─────────────────────────────────────────────────────────┘
```

**Field Types:**
- String fields: `uint16` length prefix (2 bytes) + UTF-8 string bytes
- Body fields: `uint32` length prefix (4 bytes) + raw bytes
- Timestamp: `int64` (8 bytes)
- ACK: `byte` (1 byte: 0x00 or 0x01)
- Attempts: `int32` (4 bytes)

## Message Types

The `type` field indicates the purpose of the message:

| Type | Description |
|------|-------------|
| `NEW_TOPIC` | Create a new topic |
| `NEW_MESSAGE` | Publish a new message to a topic |
| `NEW_SUB` | Subscribe to a topic (broadcast mode) |
| `ACK` | Acknowledge receipt of a message |
| `AUTH` | Authenticate with username/password |
| `AUTH_SUCCESS` | Authentication succeeded |
| `AUTH_FAILED` | Authentication failed |

### Future Message Types (Consumer Groups)

| Type | Description |
|------|-------------|
| `JOIN_GROUP` | Join a consumer group for a topic |
| `LEAVE_GROUP` | Leave a consumer group |

## Connection Flow

### 1. Establishing Connection

```
Client                                Server
  |                                     |
  |--- TCP Connect (port 9845) ------->|
  |                                     |
  |<---------- Connected --------------|
  |                                     |
```

### 2. Authentication (Optional)

If the server requires authentication (user/password configured), clients must authenticate:

```
Client                                Server
  |                                     |
  |--- AUTH Message ------------------->|
  |    {                                |
  |      "type": "AUTH",                |
  |      "user": "username",            |
  |      "password": "password"         |
  |    }                                |
  |                                     |
  |<--- AUTH_SUCCESS or AUTH_FAILED ---|
  |    {                                |
  |      "type": "AUTH_SUCCESS"         |
  |    }                                |
  |                                     |
```

**AUTH Message Example:**

Wire format: `0x01` (JSON) + `4 bytes length` + JSON payload

```json
{
  "id": "auth-uuid",
  "type": "AUTH",
  "user": "myuser",
  "password": "mypassword",
  "timestamp": 1704067200,
  "ack": false,
  "attempts": 0
}
```

**Response:**
```json
{
  "type": "AUTH_SUCCESS"
}
```
or
```json
{
  "type": "AUTH_FAILED"
}
```

If authentication fails, the server closes the connection.

### 3. Creating a Topic

```
Client                                Server
  |                                     |
  |--- NEW_TOPIC Message -------------->|
  |    {                                |
  |      "type": "NEW_TOPIC",           |
  |      "topic": {"Name": "orders"}    |
  |    }                                |
  |                                     |
```

**NEW_TOPIC Message Example:**

```json
{
  "id": "uuid-123",
  "type": "NEW_TOPIC",
  "topic": {
    "Name": "orders"
  },
  "timestamp": 1704067200,
  "ack": false,
  "attempts": 0
}
```

### 4. Subscribing to a Topic (Broadcast Mode)

```
Client                                Server
  |                                     |
  |--- NEW_SUB Message ---------------->|
  |    {                                |
  |      "type": "NEW_SUB",             |
  |      "topic": {"Name": "orders"}    |
  |    }                                |
  |                                     |
  |    (Client now receives messages)   |
  |                                     |
```

**NEW_SUB Message Example:**

```json
{
  "id": "uuid-456",
  "next_id": "uuid-456",
  "type": "NEW_SUB",
  "topic": {
    "Name": "orders"
  },
  "timestamp": 1704067200,
  "ack": false,
  "attempts": 0
}
```

### 5. Publishing a Message

```
Client (Publisher)                    Server
  |                                     |
  |--- NEW_MESSAGE -------------------->|
  |    {                                |
  |      "type": "NEW_MESSAGE",         |
  |      "topic": {"Name": "orders"},   |
  |      "body": {...},                 |
  |      "body_string": "..."           |
  |    }                                |
  |                                     |
```

**NEW_MESSAGE Example:**

```json
{
  "id": "false-uuid-789",
  "next_id": "uuid-789",
  "type": "NEW_MESSAGE",
  "topic": {
    "Name": "orders"
  },
  "body": {
    "order_id": 12345,
    "customer": "John Doe",
    "total": 150.00
  },
  "body_string": "{\"order_id\":12345,\"customer\":\"John Doe\",\"total\":150.00}",
  "timestamp": 1704067200,
  "ack": false,
  "attempts": 0
}
```

### 6. Receiving Messages

```
Server                                Client (Subscriber)
  |                                     |
  |--- NEW_MESSAGE -------------------->|
  |    (Message delivered to subscriber)|
  |                                     |
  |<--- ACK Message --------------------|
  |    {                                |
  |      "type": "ACK",                 |
  |      "id": "false-uuid-789",        |
  |      "next_id": "uuid-789"          |
  |    }                                |
  |                                     |
```

**ACK Message Example:**

```json
{
  "id": "false-uuid-789",
  "next_id": "uuid-789",
  "type": "ACK",
  "topic": {
    "Name": "orders"
  },
  "body": {
    "order_id": 12345,
    "customer": "John Doe",
    "total": 150.00
  },
  "body_string": "{\"order_id\":12345,\"customer\":\"John Doe\",\"total\":150.00}",
  "timestamp": 1704067200,
  "ack": true,
  "attempts": 1
}
```

## Message Delivery Semantics

### At-Least-Once Delivery

Queuety guarantees at-least-once delivery:

1. When a message is published, it's saved to BadgerDB with ID prefix `false-{uuid}`
2. Server sends message to all subscribers
3. If send fails, message remains in database with `false-` prefix
4. Subscriber receives message and sends ACK
5. Server updates message in database (removes `false-` prefix, updates to `next_id`)
6. Background scheduler periodically checks for messages with `false-` prefix
7. Undelivered messages are retried up to 3 times
8. After 3 attempts, message is dropped

### Message ID Convention

- **Undelivered:** `false-{uuid}` (e.g., `false-550e8400-e29b-41d4-a716-446655440000`)
- **Delivered:** `{uuid}` (e.g., `550e8400-e29b-41d4-a716-446655440000`)

The `next_id` field contains the target ID after successful delivery.

## Rate Limiting

When rate limiting is enabled on the server:

1. Messages that exceed the rate limit are queued
2. Queue size is configurable (default: 1000 messages)
3. If queue is full, new messages are dropped
4. Queued messages are processed as rate limit allows

Rate limiting is transparent to clients - no protocol changes required.

## Error Handling

### Connection Errors

If the TCP connection is lost:
- Server removes client from all topic subscriptions
- Unacknowledged messages will be retried
- Client should reconnect and resubscribe

### Message Parsing Errors

If a message cannot be parsed:
- Server logs error and continues listening
- Invalid message is discarded
- Connection remains open

### Authentication Errors

If authentication fails:
- Server sends `AUTH_FAILED` message
- Server closes the connection
- Client must reconnect with valid credentials

## Wire Format Examples

### Example 1: JSON NEW_TOPIC Message

**Hex dump:**
```
01                          # Format flag (JSON)
3C 00 00 00                 # Length: 60 bytes (little-endian)
7B 22 69 64 22 3A ...       # JSON payload: {"id":"uuid-123", ...}
```

### Example 2: Binary NEW_MESSAGE

**Hex dump:**
```
02                          # Format flag (Binary)
A0 00 00 00                 # Length: 160 bytes (little-endian)

# ID
28 00                       # ID length: 40 bytes
66 61 6C 73 65 2D ...       # "false-550e8400-..."

# Next ID
24 00                       # Next ID length: 36 bytes
35 35 30 65 38 34 ...       # "550e8400-..."

# Type
0B 00                       # Type length: 11 bytes
4E 45 57 5F 4D 45 53 ...    # "NEW_MESSAGE"

# User
00 00                       # User length: 0

# Password
00 00                       # Password length: 0

# Topic Name
06 00                       # Topic length: 6
6F 72 64 65 72 73           # "orders"

# Body (JSON)
1A 00 00 00                 # Body length: 26 bytes
7B 22 6F 72 64 65 72 ...    # {"order_id":12345}

# Body String
1A 00 00 00                 # Body string length: 26 bytes
7B 22 6F 72 64 65 72 ...    # {"order_id":12345}

# Timestamp
00 90 8B 9F 65 00 00 00    # 1704067200 (int64)

# ACK
00                          # false

# Attempts
00 00 00 00                 # 0 attempts
```

## Implementation Notes

### Server-Side

- Server listens on TCP (default port: 9845)
- Separate web server for metrics (different port)
- Each client connection is handled in a goroutine
- Format flag determines parsing strategy
- Messages are broadcast to all topic subscribers
- Failed sends are persisted to BadgerDB

### Client-Side

- Client must read format flag first (1 byte)
- Then read length (4 bytes little-endian)
- Then read exactly `length` bytes for payload
- Parse based on format flag
- Always send ACK after processing message
- Handle connection errors with reconnection logic

### Format Selection

**Use JSON when:**
- Human readability is important
- Debugging/development
- Interoperability with non-Go clients
- Message size is not critical

**Use Binary when:**
- Performance is critical
- Network bandwidth is limited
- Processing large volumes of messages
- Reduced serialization overhead is needed

## Compatibility Notes

- Protocol version is not explicitly sent (implicit in message structure)
- Future versions should maintain backward compatibility
- Unknown message types should be logged and ignored
- Clients should handle both JSON and binary formats
- Format flag in received messages indicates server's format choice

## Security Considerations

1. **Authentication:** Password sent in plaintext - use TLS for production
2. **No message encryption:** Payloads are not encrypted at protocol level
3. **No authorization:** All authenticated clients can access all topics
4. **DoS protection:** 10 MB max message size enforced by clients
5. **Rate limiting:** Server-side rate limiting protects against message floods

**Recommendations for Production:**
- Use TLS/SSL for encrypted connections
- Implement application-level encryption for sensitive payloads
- Add authorization layer for topic access control
- Deploy behind a firewall or VPN
- Monitor for unusual traffic patterns

## References

- **BadgerDB:** Used for persistent storage of undelivered messages
- **UUID:** Used for message IDs (github.com/google/uuid)
- **Little-endian:** All multi-byte integers use little-endian byte order
- **UTF-8:** All strings are UTF-8 encoded

## Appendix: Quick Reference

### Format Flags
- `0x01`: JSON
- `0x02`: Binary

### Message Types
- `NEW_TOPIC`: Create topic
- `NEW_MESSAGE`: Publish message
- `NEW_SUB`: Subscribe (broadcast)
- `ACK`: Acknowledge delivery
- `AUTH`: Authenticate
- `AUTH_SUCCESS`: Auth succeeded
- `AUTH_FAILED`: Auth failed

### Default Values
- Port: 9845 (TCP)
- Max message size: 10 MB
- Max retry attempts: 3
- Scheduler interval: 3600 seconds (1 hour)
- Default format: JSON

### Connection Checklist
1. ✓ Open TCP connection to server:9845
2. ✓ Send AUTH message (if required)
3. ✓ Wait for AUTH_SUCCESS
4. ✓ Send NEW_TOPIC to create topics
5. ✓ Send NEW_SUB to subscribe
6. ✓ Send NEW_MESSAGE to publish
7. ✓ Read messages: format flag + length + payload
8. ✓ Send ACK after processing
9. ✓ Handle reconnection on errors
