package kafka

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/segmentio/kafka-go"
)

func EnsureTopicExists(brokers []string, topic string, partitions int, log *slog.Logger) error{
	conn, err := kafka.Dial("tcp", brokers[0])
	if err != nil{
		return fmt.Errorf("failed to ccnect to kafka: %w", err)
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil{
		return fmt.Errorf("failed to get controller: %w", err)
	}
	controllerConn, err := kafka.Dial("tcp", fmt.Sprintf("%s:%d", controller.Host, controller.Port))

	if err != nil{
		return fmt.Errorf("failed to connect to controller: %w", err)
	}
	defer controllerConn.Close()

	err = controllerConn.CreateTopics(kafka.TopicConfig{
		Topic:topic,
		NumPartitions: partitions,
		ReplicationFactor: 1,
	})
	if err != nil && errors.Is(err, kafka.TopicAlreadyExists){
		return fmt.Errorf("failed to create topic: %w", err)
	}
	
	if errors.Is(err, kafka.TopicAlreadyExists){
		log.Info("kafka topic already exists", "topic", topic)
		return nil
	}
	log.Info("kafka topic created successfully", "topic", topic)
	return nil
}