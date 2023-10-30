package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	kafka "github.com/segmentio/kafka-go"
)

var (
	messagesConsumed = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "messages_consumed_total",
		Help: "Total number of messages consumed",
	})
	messageConsumptionDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "message_consumption_duration_seconds",
		Help:    "Time taken to consume a message",
		Buckets: prometheus.ExponentialBuckets(0.01, 10, 5),
	})
)

func main() {
	broker := os.Getenv("KAFKA_BROKERS")
	if broker == "" {
		fmt.Println("KAFKA_BROKERS environment variable not set.")
		os.Exit(1)
	}
	topic := os.Getenv("KAFKA_TOPIC")
	messagesPerSecondStr := os.Getenv("MESSAGES_PER_SECOND")
	messagesPerSecond, err := strconv.Atoi(messagesPerSecondStr)
	if err != nil {
		fmt.Println("Invalid MESSAGES_PER_SECOND value.")
		os.Exit(1)
	}

	config := kafka.ReaderConfig{
		Brokers:  []string{broker},
		Topic:    topic,
		GroupID:  "my-consumer-group",
		MinBytes: 10,   // Minimum number of bytes to fetch from the broker
		MaxBytes: 10e6, // Maximum number of bytes to fetch from the broker
	}
	reader := kafka.NewReader(config)
	defer reader.Close()

	// Create an HTTP server for metrics and health check endpoints
	http.Handle("/metrics", promhttp.Handler())

	// Health check endpoints
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if isReady() {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	})

	go func() {
		http.ListenAndServe(":8080", nil)
	}()

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	interval := time.Second / time.Duration(messagesPerSecond)

	for {
		select {
		case sig := <-sigchan:
			fmt.Printf("Caught signal %v: terminating\n", sig)
			return
		default:
			m, err := reader.ReadMessage(context.Background())
			if err != nil {
				fmt.Printf("Error reading message: %v\n", err)
				continue
			}

			fmt.Printf("Message on topic %s: %s\n", m.Topic, string(m.Value))

			// Instrument Prometheus metrics
			messagesConsumed.Inc()
			start := time.Now()
			duration := time.Since(start)
			messageConsumptionDuration.Observe(duration.Seconds())

			// Sleep to control the consumption rate
			time.Sleep(interval)
		}
	}
}

func isReady() bool {
	// Check if the Kafka broker is specified
	broker := os.Getenv("KAFKA_BROKERS")
	if broker == "" {
		return false
	}

	// Attempt to create a Kafka reader to check Kafka connectivity
	testTopic := os.Getenv("KAFKA_TOPIC_TEST")
	partition := 0

	conn, err := kafka.DialLeader(context.Background(), "tcp", broker, testTopic, partition)
	if err != nil {
		log.Fatal("failed to dial leader:", err)
	}
	if err := conn.Close(); err != nil {
		log.Fatal("failed to close connection:", err)
	}
	// Attempt to read a test message
	if err != nil {
		return false
	}

	return true
}
