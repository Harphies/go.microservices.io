package producer

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/sasl/aws"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"net"
	"strings"
	"sync"
	"time"
)

type MSKEventBroker struct {
	client *kgo.Client
	logger *zap.Logger
}

type EventType string

const (
	otelName               = "messaging/kafka"
	CreatedEvent EventType = "Created"
	DeletedEvent EventType = "Deleted"
	UpdatedEvent EventType = "Updated"
)

type event struct {
	Type    EventType
	Payload interface{}
}

// NewKafkaStream instantiates a Stream
func NewKafkaStream(logger *zap.Logger, host string) (*MSKEventBroker, error) {
	sess, err := session.NewSession()
	client, err := kgo.NewClient(
		kgo.SeedBrokers(strings.Split(host, ",")...),
		kgo.SASL(aws.ManagedStreamingIAM(func(ctx context.Context) (aws.Auth, error) {
			val, err := sess.Config.Credentials.GetWithContext(ctx)
			if err != nil {
				logger.Error("failed to create aws session", zap.Error(err))
				return aws.Auth{}, err
			}
			return aws.Auth{
				AccessKey:    val.AccessKeyID,
				SecretKey:    val.SecretAccessKey,
				SessionToken: val.SessionToken,
				UserAgent:    "franz-go/creds_test/v1.0.0",
			}, nil
		})),

		kgo.Dialer((&tls.Dialer{NetDialer: &net.Dialer{Timeout: 10 * time.Second}}).DialContext),
	)

	if err != nil {
		logger.Error("failed to establish connection with AWS MSK", zap.Error(err))
		return nil, err
	}

	logger.Info("Successfully Established Connection with AWS MSK Kafka")
	return &MSKEventBroker{
		client: client,
		logger: logger,
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

	ctx := context.Background()

	var wg sync.WaitGroup
	wg.Add(1)
	record := &kgo.Record{Topic: topicName, Value: b.Bytes()}
	k.client.Produce(ctx, record, func(_ *kgo.Record, err error) {
		defer wg.Done()
		if err != nil {
			k.logger.Info(fmt.Sprintf("failed to publish the event type %s to topic %s with errror [%v]", eventType, topicName, err.Error()))
		}

	})
	wg.Wait()

	k.logger.Info(fmt.Sprintf("event with type %s successfully published to topic %s", eventType, topicName))

	return nil
}

func newOTELSpan(ctx context.Context, name string) trace.Span {
	_, span := otel.Tracer(otelName).Start(ctx, name)

	return span
}
