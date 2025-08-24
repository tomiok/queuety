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

## Observabilidad

Queuety proporciona capacidades avanzadas de observabilidad utilizando OpenTelemetry y Prometheus.

### Métricas

#### Métricas de Mensajes
- `queuety_messages_published_total`: Total de mensajes publicados por tema
- `queuety_messages_delivered_total`: Total de mensajes entregados por tema
- `queuety_messages_failed_total`: Total de mensajes fallidos por tema

#### Métricas de Rendimiento
- `queuety_message_processing_seconds`: Histograma de latencia de procesamiento de mensajes
- `queuety_message_processing_average_seconds`: Tiempo promedio de procesamiento de mensajes

#### Métricas de Sistema
- `queuety_topics_total`: Número total de temas activos
- `queuety_subscribers_total`: Número de suscriptores por tema
- `queuety_active_connections`: Número de conexiones TCP activas

#### Métricas de Base de Datos
- `queuety_badger_operations_total`: Métricas de operaciones de BadgerDB

#### Métricas de Autenticación
- `queuety_auth_attempts_total`: Intentos de autenticación (éxito/fallo)

### Trazas (Spans)

Queuety instrumenta múltiples operaciones con spans de OpenTelemetry:

#### Spans de Conexión
- `handle_connections`: Manejo de conexiones de cliente
  - Atributos: `client.remote_addr`, `client.local_addr`

#### Spans de Mensajes
- `send_message`: Envío de nuevos mensajes
  - Atributos: `topic.name`, `message.id`
- `handle_json_message`: Procesamiento de mensajes JSON
  - Atributos: `topic.name`, `operation`

#### Spans de Base de Datos
- `badger_save_message`: Guardar mensaje en BadgerDB
  - Atributos: `message.id`, `topic.name`
- `badger_update_message_ack`: Actualizar ACK de mensaje
  - Atributos: `message.id`, `topic.name`
- `badger_check_not_delivered_messages`: Verificar mensajes no entregados
  - Atributos: `messages.count`, `topics.checked`

#### Spans de Autenticación
- `do_login`: Proceso de inicio de sesión
  - Atributos: `user.attempt`, `client.remote_addr`

#### Spans de Gestión de Temas
- `add_subscriber`: Añadir nuevo suscriptor
  - Atributos: `topic.name`
- `add_topic`: Crear nuevo tema
  - Atributos: `topic.name`

### Configuración

Para habilitar OpenTelemetry, configura las siguientes variables de entorno:

- `QUEUETY_OTEL_ENABLED`: Habilitar OpenTelemetry (`true`/`false`)
- `OTEL_EXPORTER_OTLP_GRPC_ENDPOINT`: Endpoint gRPC de OpenTelemetry
- `OTEL_EXPORTER_OTLP_HTTP_ENDPOINT`: Endpoint HTTP de OpenTelemetry (alternativo)

Para habilitar Prometheus, configura las siguientes variables de entorno:

- `QUEUETY_PROM_METRICS_ENABLED`: Exponer metricas de Prometheus (`true`/`false`)

### Exportadores Soportados

- Prometheus (endpoint `/metrics`)
- OpenTelemetry (gRPC y HTTP)

*Nota: La instrumentación de métricas y trazas está en desarrollo continuo.*

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
