# Queuety

#### A simple queue, delivering at-least-once done in pure Go.

## Features

- **At-least-once delivery** guarantee
- **Multiple persistence options**: BadgerDB or in-memory storage
- **TCP-based** communication protocol
- **Automatic retry** mechanism for failed messages
- **Topic-based** message routing
- **Docker support** for easy deployment

## Quick Start

### Using Docker (Recommended)

#### Direct Docker Commands
```bash
git clone https://github.com/tomiok/queuety.git
cd queuety

# Build the image
docker build -t queuety:latest .

# Run with BadgerDB persistence (default and only one by now) 
docker run -d --name queuety queuety:latest
```

### Using Pre-compiled Binary

```bash
# Download the binary (replace with actual release URL)
wget https://github.com/tomiok/queuety/releases/download/{version}/queuety

# Make it executable
chmod +x queuety

# Run with BadgerDB (creates ./badger directory)
./queuety
```

## Storage Options

### BadgerDB (Persistent) - the only one available by now.
- **Pros**: Persistent storage, crash recovery, high performance
- **Cons**: Requires disk space
- **Use case**: Production environments, when message durability is critical

## Protocol options

### TCP (only available now)

---
## Examples
### Server without authentication (lookup the client too)
Just run the server as the example above, or if you use this repo, go to the examples package.

[Example usage](/_example/simple-server-client/server)

### Server with  user and pass authentication (lookup the client too)
The client, for this particular case is written in the same package (because if you fork you just need to change one
repository) and can be used as a library, inside the manager package.

[Example usage](/_example/auth-server-client/server)

## Development

### Building from Source (server)

```bash
git clone https://github.com/tomiok/queuety.git
cd queuety

go build -o queuety ./server/main/main.go
```

### Using Makefile

```bash
make build

make run

make logs

make stop

make clean
```

## Client
The client is only in GitHub now, you can use go get in order to use the manager.
go install github.com/tomiok/queuety/manager@v0.0.4

## Observability

Queuety provides advanced observability capabilities using OpenTelemetry and Prometheus.

### Metrics

#### Message Metrics
- `queuety_messages_published_total`: Total messages published by topic
- `queuety_messages_delivered_total`: Total messages delivered by topic
- `queuety_messages_failed_total`: Total failed messages by topic

#### Performance Metrics
- `queuety_message_processing_seconds`: Message processing latency histogram
- `queuety_message_processing_average_seconds`: Average message processing time

#### System Metrics
- `queuety_topics_total`: Total number of active topics
- `queuety_subscribers_total`: Number of subscribers per topic
- `queuety_active_connections`: Number of active TCP connections

#### Database Metrics
- `queuety_badger_operations_total`: BadgerDB operation metrics

#### Authentication Metrics
- `queuety_auth_attempts_total`: Authentication attempts (success/failure)

### Traces (Spans)

Queuety instruments multiple operations with OpenTelemetry spans:

#### Connection Spans
- `handle_connections`: Client connection handling
  - Attributes: `client.remote_addr`, `client.local_addr`

#### Message Spans
- `send_message`: Sending new messages
  - Attributes: `topic.name`, `message.id`
- `handle_json_message`: JSON message processing
  - Attributes: `topic.name`, `operation`

#### Database Spans
- `badger_save_message`: Save message in BadgerDB
  - Attributes: `message.id`, `topic.name`
- `badger_update_message_ack`: Update message ACK
  - Attributes: `message.id`, `topic.name`
- `badger_check_not_delivered_messages`: Check undelivered messages
  - Attributes: `messages.count`, `topics.checked`

#### Authentication Spans
- `do_login`: Login process
  - Attributes: `user.attempt`, `client.remote_addr`

#### Topic Management Spans
- `add_subscriber`: Add new subscriber
  - Attributes: `topic.name`
- `add_topic`: Create new topic
  - Attributes: `topic.name`

### Configuration

To enable OpenTelemetry, configure the following environment variables:

- `QUEUETY_OTEL_ENABLED`: Enable OpenTelemetry (`true`/`false`)
- `OTEL_EXPORTER_OTLP_GRPC_ENDPOINT`: OpenTelemetry gRPC endpoint
- `OTEL_EXPORTER_OTLP_HTTP_ENDPOINT`: OpenTelemetry HTTP endpoint (alternative)

To enable Prometheus, configure the following environment variables:

- `QUEUETY_PROM_METRICS_ENABLED`: Expose Prometheus metrics (`true`/`false`)

### Supported Exporters

- Prometheus (endpoint `/metrics`)
- OpenTelemetry (gRPC and HTTP)

*Note: Metrics and tracing instrumentation is under continuous development.*

## Roadmap

- [x] At-least-once delivery
- [x] BadgerDB persistence
- [x] In-memory storage option
- [x] Docker support
- [ ] gRPC support
- [x] Authentication (user/password)
- [ ] Clustering
- [ ] REST API
- [x] Metrics and monitoring

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License
GPL 3.0
