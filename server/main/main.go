package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/tomiok/queuety/server"
	"github.com/tomiok/queuety/server/observability"
)

func main() {
	// Verificar si OpenTelemetry está habilitado
	otelEnabled := os.Getenv("QUEUETY_OTEL_ENABLED")

	// Inicializar OpenTelemetry solo si está explícitamente habilitado
	if otelEnabled == "true" {
		ctx := context.Background()
		err := observability.InitOpenTelemetry(ctx)
		if err != nil {
			log.Printf("Error inicializando OpenTelemetry: %v", err)
		}

		// Configurar cierre de OpenTelemetry al finalizar
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := observability.ShutdownOpenTelemetry(shutdownCtx); err != nil {
				log.Printf("Error cerrando OpenTelemetry: %v", err)
			}
		}()
	}

	badgerPath := os.Getenv("BADGER_PATH")
	if badgerPath == "" {
		badgerPath = "/tmp/badger" //for local and NOT using docker, use tmp. Otherwise, go through Dockerfile env variable.
	}

	s, err := server.NewServer(server.Config{
		Protocol:      "tcp4",
		Port:          ":9845",
		BadgerPath:    badgerPath,
		Duration:      10,
		Auth:          nil,
		WebServerPort: "9846", // Corregir sintaxis del puerto
	})

	if err != nil {
		panic(err)
	}

	log.Fatal(s.Start())
}
