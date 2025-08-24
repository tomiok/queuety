package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Contador de mensajes publicados por tema
	MessagesPublishedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "queuety_messages_published_total",
			Help: "Total published messages by topic",
		},
		[]string{"topic"},
	)

	// Contador de mensajes entregados por tema
	MessagesDeliveredTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "queuety_messages_delivered_total",
			Help: "Total delivered messages by topic",
		},
		[]string{"topic"},
	)

	// Contador de mensajes fallidos por tema
	MessagesFailedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "queuety_messages_failed_total",
			Help: "Failed message deliveries by topic",
		},
		[]string{"topic"},
	)

	// Definir buckets para percentiles específicos
	messageProcessingBuckets = []float64{
		0.001,   // Bucket más bajo
		0.01,    // Bucket para latencias muy bajas
		0.05,    // Bucket para latencias bajas
		0.1,     // Bucket para latencias medias-bajas
		0.25,    // Bucket para P50 (50%)
		0.5,     // Bucket para latencias medias
		1,       // Bucket para latencias medias-altas
		2.5,     // Bucket para P95 (95%)
		5,       // Bucket para P99 (99%)
		10,      // Bucket para P99.9 (99.9%)
	}

	// Histograma de latencia de procesamiento de mensajes
	MessageProcessingSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "queuety_message_processing_seconds",
			Help:    "Processing latency histogram by topic and operation",
			Buckets: messageProcessingBuckets,
		},
		[]string{"topic", "operation"},
	)

	// Gauge para el promedio de procesamiento de mensajes
	MessageProcessingAverage = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "queuety_message_processing_average_seconds",
			Help: "Average processing time for messages by topic and operation",
		},
		[]string{"topic", "operation"},
	)

	// Gauge para el número total de temas activos
	TopicsTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "queuety_topics_total",
			Help: "Number of active topics",
		},
	)

	// Gauge para el número de suscriptores por tema
	SubscribersTotal = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "queuety_subscribers_total",
			Help: "Subscribers per topic",
		},
		[]string{"topic"},
	)

	// Gauge para el número de conexiones TCP activas
	ActiveConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "queuety_active_connections",
			Help: "Current TCP connections count",
		},
	)

	// Contador de operaciones de BadgerDB
	BadgerOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "queuety_badger_operations_total",
			Help: "BadgerDB operation metrics",
		},
		[]string{"operation", "status"},
	)

	// Contador de intentos de autenticación
	AuthAttemptsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "queuety_auth_attempts_total",
			Help: "Authentication attempts tracking",
		},
		[]string{"result"},
	)
)

// IncrementPublishedMessages incrementa el contador de mensajes publicados para un tema específico
func IncrementPublishedMessages(topic string) {
	MessagesPublishedTotal.WithLabelValues(topic).Inc()
}

// IncrementDeliveredMessages incrementa el contador de mensajes entregados para un tema específico
func IncrementDeliveredMessages(topic string) {
	MessagesDeliveredTotal.WithLabelValues(topic).Inc()
}

// IncrementFailedMessages incrementa el contador de mensajes fallidos para un tema específico
func IncrementFailedMessages(topic string) {
	MessagesFailedTotal.WithLabelValues(topic).Inc()
}

// ObserveMessageProcessingTime observa el tiempo de procesamiento de un mensaje
func ObserveMessageProcessingTime(topic, operation string, duration float64) {
	MessageProcessingSeconds.WithLabelValues(topic, operation).Observe(duration)
	
	// Actualizar el promedio
	MessageProcessingAverage.WithLabelValues(topic, operation).Set(duration)
}

// UpdateActiveTopicsCount actualiza el número de temas activos
func UpdateActiveTopicsCount(count int) {
	TopicsTotal.Set(float64(count))
}

// UpdateSubscribersCount actualiza el número de suscriptores para un tema específico
func UpdateSubscribersCount(topic string, count int) {
	SubscribersTotal.WithLabelValues(topic).Set(float64(count))
}

// UpdateActiveConnectionsCount actualiza el número de conexiones TCP activas
func UpdateActiveConnectionsCount(count int) {
	ActiveConnections.Set(float64(count))
}

// IncrementBadgerOperation incrementa el contador de operaciones de BadgerDB
func IncrementBadgerOperation(operation, status string) {
	BadgerOperationsTotal.WithLabelValues(operation, status).Inc()
}

// IncrementAuthAttempt incrementa el contador de intentos de autenticación
func IncrementAuthAttempt(result string) {
	AuthAttemptsTotal.WithLabelValues(result).Inc()
} 