package producer

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

func main() {
	brokerConfigMap := os.Getenv("KAFKA_BROKERS")
	if brokerConfigMap == "" {
		fmt.Println("KAFKA_BROKERS environment variable not set.")
		os.Exit(1)
	}

	p, err := kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": brokerConfigMap})
	if err != nil {
		fmt.Printf("Error creating Kafka producer: %v\n", err)
		os.Exit(1)
	}

	// Number of messages to produce
	numMessagesStr := os.Getenv("NUM_MESSAGES")
	numMessages, err := strconv.Atoi(numMessagesStr)
	if err != nil {
		fmt.Println("Invalid NUM_MESSAGES value.")
		os.Exit(1)
	}

	// Messages per second
	messagesPerSecondStr := os.Getenv("MESSAGES_PER_SECOND")
	messagesPerSecond, err := strconv.Atoi(messagesPerSecondStr)
	if err != nil {
		fmt.Println("Invalid MESSAGES_PER_SECOND value.")
		os.Exit(1)
	}

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

		// Sleep to control the rate
		time.Sleep(interval)
	}

	p.Close()
}
