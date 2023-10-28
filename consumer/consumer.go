package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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
	brokerConfigMap := os.Getenv("KAFKA_BROKERS")
	if brokerConfigMap == "" {
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

	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": brokerConfigMap,
		"group.id":          "my-consumer-group",
		"auto.offset.reset": "earliest",
	})

	if err != nil {
		fmt.Printf("Error creating Kafka consumer: %v\n", err)
		os.Exit(1)
	}

	defer c.Close()

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

	err = c.SubscribeTopics([]string{topic}, nil)
	if err != nil {
		fmt.Printf("Error subscribing to topic: %v\n", err)
		os.Exit(1)
	}

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	run := true

	interval := time.Second / time.Duration(messagesPerSecond)

	for run == true {
		select {
		case sig := <-sigchan:
			fmt.Printf("Caught signal %v: terminating\n", sig)
			run = false
		default:
			ev := c.Poll(100)
			if ev == nil {
				continue
			}
			switch e := ev.(type) {
			case *kafka.Message:
				fmt.Printf("Message on %s: %s\n", e.TopicPartition, string(e.Value))

				// Instrument Prometheus metrics
				messagesConsumed.Inc()
				start := time.Now()
				duration := time.Since(start)
				messageConsumptionDuration.Observe(duration.Seconds())
			case kafka.Error:
				fmt.Printf("Error: %v\n", e)
			}

			// Sleep to control the consumption rate
			time.Sleep(interval)
		}
	}
}

func isReady() bool {
	// Check if the Kafka broker connection is established
	brokerConfigMap := os.Getenv("KAFKA_BROKERS")
	if brokerConfigMap == "" {
		return false
	}

	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": brokerConfigMap,
		"group.id":          "my-consumer-group",
		"auto.offset.reset": "earliest",
	})

	if err != nil {
		return false
	}

	defer c.Close()

	// Subscribe to a test topic to check Kafka connectivity
	testTopic := os.Getenv("KAFKA_TOPIC_TEST")
	err = c.SubscribeTopics([]string{testTopic}, nil)
	if err != nil {
		return false
	}

	// Attempt to consume a test message
	timeout := 5000 // Set a timeout (in milliseconds) for consuming a message
	ev := c.Poll(timeout)
	if ev == nil {
		return false
	}
	_, isMessage := ev.(*kafka.Message)
	if !isMessage {
		return false
	}

	// If all checks pass, the application is considered ready
	return true
}
