package consumer

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

func main() {
	brokerConfigMap := os.Getenv("KAFKA_BROKERS")
	if brokerConfigMap == "" {
		fmt.Println("KAFKA_BROKERS environment variable not set.")
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

	topic := "my-kafka-topic"
	err = c.SubscribeTopics([]string{topic}, nil)

	if err != nil {
		fmt.Printf("Error subscribing to topic: %v\n", err)
		os.Exit(1)
	}

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	run := true

	// Messages per second
	messagesPerSecondStr := os.Getenv("MESSAGES_PER_SECOND")
	messagesPerSecond, err := strconv.Atoi(messagesPerSecondStr)
	if err != nil {
		fmt.Println("Invalid MESSAGES_PER_SECOND value.")
		os.Exit(1)
	}

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
			case kafka.Error:
				fmt.Printf("Error: %v\n", e)
			}

			// Sleep to control the consumption rate
			time.Sleep(interval)
		}
	}
}
