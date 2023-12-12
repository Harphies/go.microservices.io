package producer

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

type MSKEventBroker struct {
	client      *kafka.Producer
	logger      *zap.Logger
	serviceName string
	topicName   string
}

type EventType string

const (
	otelName               = "recommendationservice/internal/storage/messaging/kafka"
	CreatedEvent EventType = "Created"
	DeletedEvent EventType = "Deleted"
	UpdatedEvent EventType = "Updated"
)

type event struct {
	Type    EventType
	Payload interface{}
}

// NewKafkaStream instantiates a Stream
func NewKafkaStream(logger *zap.Logger, host, saslScramUsername, saslScramPassword, serviceName string) (*MSKEventBroker, error) {
	config := kafka.ConfigMap{
		"bootstrap.servers": host,
		"sasl.mechanisms":   kafka.ScramMechanismSHA512,
		"security.protocol": "SASL_SSL",
		"sasl.username":     saslScramUsername,
		"sasl.password":     saslScramPassword,
	}

	client, err := kafka.NewProducer(&config)
	if err != nil {
		logger.Error("failed to establish connection with MSK", zap.Error(err))
		return nil, err
	}

	defer client.Close()

	logger.Info("Successfully Established Connection with AWS MSK Kafka")
	return &MSKEventBroker{
		client:      client,
		logger:      logger,
		serviceName: serviceName,
		topicName:   serviceName,
	}, nil
}

// Created publishes a message indicating a record was created.
func (k *MSKEventBroker) Created(eventPayload interface{}, topicName string) error {
	//msgType := k.serviceName + ".event.created"
	return k.publish(CreatedEvent, eventPayload, topicName)
}

// Deleted publishes a message indicating a task was deleted.
func (k *MSKEventBroker) Deleted(id string, topicName string) error {
	//msgType := k.serviceName + ".event.deleted"
	return k.publish(DeletedEvent, id, topicName)
}

// Updated publishes a message indicating a task was updated.
func (k *MSKEventBroker) Updated(eventPayload interface{}, topicName string) error {
	//mstType := k.serviceName + ".event.updated"
	return k.publish(UpdatedEvent, eventPayload, topicName)
}

func (k *MSKEventBroker) publish(eventType EventType, eventPayload interface{}, topicName string) error {

	var b bytes.Buffer

	evt := event{
		Type:    eventType,
		Payload: eventPayload,
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

	return nil
}

func newOTELSpan(ctx context.Context, name string) trace.Span {
	_, span := otel.Tracer(otelName).Start(ctx, name)

	return span
}
