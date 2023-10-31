package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	kafka "github.com/segmentio/kafka-go"
)

var (
	messagesProduced = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "messages_produced_total",
		Help: "Total number of messages produced",
	})
	messageProductionDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "message_production_duration_seconds",
		Help:    "Time taken to produce a message",
		Buckets: prometheus.ExponentialBuckets(0.01, 10, 5),
	})
)

type Timespan time.Duration

func (t Timespan) Format(format string) string {
	z := time.Unix(0, 0).UTC()
	return z.Add(time.Duration(t)).Format(format)
}

func main() {
	broker := os.Getenv("KAFKA_BROKERS")
	topic := os.Getenv("KAFKA_TOPIC")
	messagesPerSecondStr := os.Getenv("MESSAGES_PER_SECOND")
	messagesPerSecond, err := strconv.Atoi(messagesPerSecondStr)
	if err != nil {
		fmt.Println("Invalid MESSAGES_PER_SECOND value.")
		os.Exit(1)
	}

	config := kafka.WriterConfig{
		Brokers:  []string{broker},
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}

	writer := kafka.NewWriter(config)

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

	// Number of messages to produce
	interval := time.Second / time.Duration(messagesPerSecond)

	for {
		message := fmt.Sprintf("Message %d", time.Now().UnixNano())

		err := writer.WriteMessages(
			context.Background(),
			kafka.Message{
				Key:   nil,
				Value: []byte(message),
			},
		)

		if err != nil {
			fmt.Printf("Failed to produce message: %v\n", err)
		}

		// Instrument Prometheus metrics
		messagesProduced.Inc()
		start := time.Now()
		duration := time.Since(start)
		messageProductionDuration.Observe(duration.Seconds())
		fmt.Printf("interval: %s\n", Timespan(interval).Format("15:04:05.000"))
		// Sleep to control the rate
		time.Sleep(interval)
		fmt.Printf("Message on topic %s: %s\n", topic, string(message))
	}
}

// Add a custom function to check application readiness
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
