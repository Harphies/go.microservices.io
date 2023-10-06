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

const (
	otelName = "recommendationservice/internal/storage/messaging/kafka"
)

type event struct {
	Type  string
	Value interface{}
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
	}
	logger.Info("Connection Established Successfully with AWS MSK Kafka")
	return &MSKEventBroker{
		client:      client,
		logger:      logger,
		serviceName: serviceName,
		topicName:   serviceName,
	}, nil
}

// Created publishes a message indicating a record was created.
func (k *MSKEventBroker) Created(record interface{}) error {
	msgType := k.serviceName + ".event.created"
	return k.publish(msgType, record)
}

// Deleted publishes a message indicating a task was deleted.
func (k *MSKEventBroker) Deleted(id string) error {
	msgType := k.serviceName + ".event.deleted"
	return k.publish(msgType, id)
}

// Updated publishes a message indicating a task was updated.
func (k *MSKEventBroker) Updated(record interface{}) error {
	mstType := k.serviceName + ".event.updated"
	return k.publish(mstType, record)
}

func (k *MSKEventBroker) publish(msgType string, record interface{}) error {

	var b bytes.Buffer

	evt := event{
		Type:  msgType,
		Value: record,
	}

	if err := json.NewEncoder(&b).Encode(evt); err != nil {
		return err
	}

	if err := k.client.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &k.topicName,
			Partition: kafka.PartitionAny,
		},
		Value: b.Bytes(),
	}, nil); err != nil {
		fmt.Println("")
	}

	return nil
}

func newOTELSpan(ctx context.Context, name string) trace.Span {
	_, span := otel.Tracer(otelName).Start(ctx, name)

	return span
}
