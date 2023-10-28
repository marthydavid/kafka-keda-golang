package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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

func main() {
	brokerConfigMap := os.Getenv("KAFKA_BROKERS")
	if brokerConfigMap == "" {
		fmt.Println("KAFKA_BROKERS environment variable not set.")
		os.Exit(1)
	}
	topic := os.Getenv("KAFKA_TOPIC")
	numMessagesStr := os.Getenv("MAX_MESSAGES")
	numMessages, err := strconv.Atoi(numMessagesStr)
	messagesPerSecondStr := os.Getenv("MESSAGES_PER_SECOND")
	messagesPerSecond, err := strconv.Atoi(messagesPerSecondStr)
	if err != nil {
		fmt.Println("Invalid MESSAGES_PER_SECOND value.")
		os.Exit(1)
	}

	p, err := kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": brokerConfigMap})
	if err != nil {
		fmt.Printf("Error creating Kafka producer: %v\n", err)
		os.Exit(1)
	}

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

	for i := 1; i <= numMessages; i++ {
		message := fmt.Sprintf("Message %d", i)

		err = p.Produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
			Value:          []byte(message),
		}, nil)

		if err != nil {
			fmt.Printf("Failed to produce message %d: %v\n", i, err)
		} else {
			fmt.Printf("Produced message %d: %s\n", i, message)
		}

		// Instrument Prometheus metrics
		messagesProduced.Inc()
		start := time.Now()
		duration := time.Since(start)
		messageProductionDuration.Observe(duration.Seconds())

		// Sleep to control the rate
		time.Sleep(interval)
	}

	p.Close()
}

// Add a custom function to check application readiness
func isReady() bool {
	// Check if the Kafka broker connection is established
	brokerConfigMap := os.Getenv("KAFKA_BROKERS")
	if brokerConfigMap == "" {
		return false
	}
	testTopic := os.Getenv("KAFKA_TOPIC_TEST")
	p, err := kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": brokerConfigMap})
	if err != nil {
		return false
	}

	defer p.Close()

	// Attempt to produce a test message to check Kafka connectivity
	testMessage := "Test message for readiness check"
	err = p.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &testTopic, Partition: kafka.PartitionAny},
		Value:          []byte(testMessage),
	}, nil)

	if err != nil {
		return false
	}

	// If all checks pass, the application is considered ready
	return true
}
