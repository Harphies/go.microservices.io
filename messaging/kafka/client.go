package kafka

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type EventBroker struct {
	client *kafka.Producer
	logger *zap.Logger
}

type EventType string

const (
	otelName = "internal/storage/messaging/kafka"
)

type event struct {
	Type  string
	Value interface{}
}

// NewKafkaStream instantiates a Stream
func NewKafkaStream(logger *zap.Logger, brokerEndpoints, saslScramUsername, saslScramPassword, securityProtocol string) (*EventBroker, error) {
	config := kafka.ConfigMap{
		"bootstrap.servers":         brokerEndpoints,
		"sasl.mechanisms":           kafka.ScramMechanismSHA512,
		"security.protocol":         securityProtocol,
		"sasl.username":             saslScramUsername,
		"sasl.password":             saslScramPassword,
		"receive.message.max.bytes": 1073741824, // 1GB
	}

	client, err := kafka.NewProducer(&config)
	if err != nil {
		logger.Error("failed to establish connection with MSK", zap.Error(err))
		return nil, err
	}

	//defer client.Close()

	logger.Info("Successfully Established Connection with AWS MSK Kafka")
	return &EventBroker{
		client: client,
		logger: logger,
	}, nil
}

// Publish publishes a message indicating a record was created.
func (k *EventBroker) Publish(eventPayload interface{}, topicName, eventType string) error {
	return k.publish(eventType, eventPayload, topicName)
}

func (k *EventBroker) publish(eventType string, eventPayload interface{}, topicName string) error {

	var b bytes.Buffer

	evt := event{
		Type:  eventType,
		Value: eventPayload,
	}

	if err := json.NewEncoder(&b).Encode(evt); err != nil {
		return err
	}

	if err := k.client.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &topicName,
			Partition: kafka.PartitionAny,
		},
		Value: b.Bytes(),
	}, nil); err != nil {
		k.logger.Info(fmt.Sprintf("failed to publish the event type %s to topic %s with errror [%v]", eventType, topicName, err.Error()))
		return err
	}
	k.logger.Info(fmt.Sprintf("event with type %s successfully published to [%s] topic", eventType, topicName))

	return nil
}

func newOTELSpan(ctx context.Context, name string) trace.Span {
	_, span := otel.Tracer(otelName).Start(ctx, name)

	return span
}
