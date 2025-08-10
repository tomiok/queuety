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
wget https://github.com/tomiok/queuety/releases/download/v0.0.1/queuety

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
### Server
Just run the server as the example above, or if you use this repo, go to the examples package.

[Example usage](/example/simple-server-client/server)

### Client
The client, for this particular case is written in the same package (because if you fork you just need to change one
repository) and can be used as a library, inside the manager package.

[Example usage](/example/simple-server-client/client)

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
go install github.com/tomiok/queuety/manager@v0.0.1

## Roadmap

- [x] At-least-once delivery
- [x] BadgerDB persistence
- [x] In-memory storage option
- [x] Docker support
- [ ] gRPC support
- [x] Authentication (user/password)
- [ ] Clustering
- [ ] REST API
- [ ] Metrics and monitoring

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License
