package kafka

import (
	"context"
	"fmt"

	"github.com/segmentio/kafka-go"
)

type Client struct {
	writer *kafka.Writer
}

func Connect(kafkaAddr string, topic string) Client {
	writer := kafka.Writer{
		Addr:     kafka.TCP("localhost:9092"),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}
	client := Client{
		writer: &writer,
	}
	return client
}

func (client *Client) ProduceEvent(eventName string, event []byte) error {
	message := kafka.Message{Key: []byte(eventName), Value: event}
	err := client.writer.WriteMessages(context.Background(), message)
	if err != nil {
		fmt.Println("Error occured on Kafka Log production", err)
		return err
	}
	fmt.Println("Sent to Kafka:", message)
	return nil
	// fmt.Println(events["buckets"])
	// buckets, ok := events["buckets"].([]interface{})
	// fmt.Println(buckets, ok)
	// if !ok {
	// 	err := fmt.Errorf("Failed parsing result buckets %v", events["buckets"])
	// 	fmt.Println(err)
	// 	return err
	// }
	// fmt.Println(buckets)
	// messages := make([]kafka.Message, 0, len(buckets))
	// var byteMessage []byte
	// for _, value := range buckets {
	// 	mapValue := value.(map[string]interface{})
	// 	byteMessage, err := json.Marshal(mapValue)
	// 	if err != nil {
	// 		fmt.Println("Can't marshal value", value, err)
	// 		return err
	// 	}
	//newMessage := kafka.Message{Key: []byte(eventName), Value: byteMessage}

	// }
	// return nil
}
