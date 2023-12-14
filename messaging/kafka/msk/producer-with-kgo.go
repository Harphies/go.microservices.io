package msk

//import (
//	"bytes"
//	"context"
//	"crypto/tls"
//	"encoding/json"
//	"fmt"
//	"github.com/aws/aws-sdk-go/aws/session"
//	"github.com/twmb/franz-go/pkg/kgo"
//	"github.com/twmb/franz-go/pkg/sasl/aws"
//	"go.opentelemetry.io/otel"
//	"go.opentelemetry.io/otel/trace"
//	"go.uber.org/zap"
//	"net"
//	"strings"
//	"time"
//)
//
//type EventBroker struct {
//	client *kgo.Client
//	logger *zap.Logger
//}
//
//type EventType string
//
//const (
//	otelName               = "messaging/kafka"
//	CreatedEvent EventType = "Created"
//	DeletedEvent EventType = "Deleted"
//	UpdatedEvent EventType = "Updated"
//)
//
//type event struct {
//	Type    EventType
//	Value interface{}
//}
//
//// NewKafkaStream instantiates a Stream
//func NewKafkaStream(logger *zap.Logger, host string) (*EventBroker, error) {
//	sess, err := session.NewSession()
//	if err != nil {
//		logger.Info(fmt.Sprintf("unable to initialize aws session: %v", err))
//	}
//	client, err := kgo.NewClient(
//		kgo.SeedBrokers(strings.Split(host, ",")...),
//		kgo.SASL(aws.ManagedStreamingIAM(func(ctx context.Context) (aws.Auth, error) {
//			val, err := sess.Config.Credentials.GetWithContext(ctx)
//			if err != nil {
//				logger.Error("failed to create aws session", zap.Error(err))
//				return aws.Auth{}, err
//			}
//			return aws.Auth{
//				AccessKey:    val.AccessKeyID,
//				SecretKey:    val.SecretAccessKey,
//				SessionToken: val.SessionToken,
//				UserAgent:    "franz-go/creds_test/v1.0.0",
//			}, nil
//		})),
//
//		kgo.Dialer((&tls.Dialer{NetDialer: &net.Dialer{Timeout: 180 * time.Second}}).DialContext),
//	)
//
//	if err != nil {
//		logger.Error("failed to establish connection with AWS MSK", zap.Error(err))
//		return nil, err
//	}
//
//	logger.Info("Successfully Established Connection with AWS MSK Kafka")
//	return &EventBroker{
//		client: client,
//		logger: logger,
//	}, nil
//}
//
//// Created publishes a message indicating a record was created.
//func (k *EventBroker) Created(eventPayload interface{}, topicName string) error {
//	//msgType := k.serviceName + ".event.created"
//	return k.publish(CreatedEvent, eventPayload, topicName)
//}
//
//// Deleted publishes a message indicating a task was deleted.
//func (k *EventBroker) Deleted(id string, topicName string) error {
//	//msgType := k.serviceName + ".event.deleted"
//	return k.publish(DeletedEvent, id, topicName)
//}
//
//// Updated publishes a message indicating a task was updated.
//func (k *EventBroker) Updated(eventPayload interface{}, topicName string) error {
//	//mstType := k.serviceName + ".event.updated"
//	return k.publish(UpdatedEvent, eventPayload, topicName)
//}
//
//func (k *EventBroker) publish(eventType EventType, eventPayload interface{}, topicName string) error {
//
//	var b bytes.Buffer
//
//	evt := event{
//		Type:    eventType,
//		Value: eventPayload,
//	}
//
//	if err := json.NewEncoder(&b).Encode(evt); err != nil {
//		return err
//	}
//
//	ctx := context.Background()
//
//	record := &kgo.Record{Topic: topicName, Value: b.Bytes()}
//	k.client.Produce(ctx, record, func(_ *kgo.Record, err error) {
//		if err != nil {
//			k.logger.Info(fmt.Sprintf("failed to publish the event type %s to topic %s with errror [%v]", eventType, topicName, err.Error()))
//		}
//	})
//
//	k.logger.Info(fmt.Sprintf("event with type %s successfully published to topic %s", eventType, topicName))
//
//	return nil
//}
//
//func newOTELSpan(ctx context.Context, name string) trace.Span {
//	_, span := otel.Tracer(otelName).Start(ctx, name)
//
//	return span
//}
