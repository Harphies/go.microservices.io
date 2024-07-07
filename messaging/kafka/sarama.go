package kafka

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"github.com/IBM/sarama"
	"go.uber.org/zap"
	"log"
	"strings"
	"time"
)

type BrokerClient struct {
	saramaConfig *sarama.Config
	logger       *zap.Logger
	brokers      []string
}

// NewKafkaStream ...
func NewKafkaStream(logger *zap.Logger, brokerEndpoints, saslScramUsername, saslScramPassword, securityProtocol, securityMechanism string, useAuth bool) (*BrokerClient, error) {
	brokerList := strings.Split(brokerEndpoints, ",")
	config := sarama.NewConfig()

	// producer config for reliability, performance, fault tolerance and security
	config.Producer.Return.Successes = true
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 5
	config.Producer.Partitioner = sarama.NewHashPartitioner
	config.Producer.Compression = sarama.CompressionSnappy

	// Batching for network throughput and latency
	config.Producer.Flush.Frequency = 500 * time.Millisecond
	config.Producer.Flush.Messages = 100
	config.Producer.Flush.MaxMessages = 1000
	config.Producer.Flush.Bytes = 1048576

	// Network Settings
	config.Net.MaxOpenRequests = 5
	config.Net.DialTimeout = 30 * time.Second
	config.Net.ReadTimeout = 30 * time.Second
	config.Net.WriteTimeout = 30 * time.Second

	if useAuth {
		log.Println("UseAuth is set")
		config.Net.SASL.Enable = true
		config.Net.SASL.User = saslScramUsername
		config.Net.SASL.Password = saslScramPassword
		config.Net.SASL.Mechanism = sarama.SASLMechanism(securityMechanism)
		config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient { return NewKafkaClient() }
		config.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA512
	}

	if securityProtocol == "SASL_SSL" {
		config.Net.SASL.Handshake = true
		config.Net.TLS.Enable = true
		config.Net.TLS.Config = &tls.Config{
			InsecureSkipVerify: false,
			ClientAuth:         tls.NoClientCert,
		}
	}

	return &BrokerClient{
		logger:       logger,
		saramaConfig: config,
		brokers:      brokerList,
	}, nil
}

// Publish publishes a message indicating a record was created.
func (c *BrokerClient) Publish(eventPayload interface{}, topicName, eventType string) error {
	return c.publish(eventType, eventPayload, topicName)
}

func (c *BrokerClient) publish(eventType string, eventPayload interface{}, topicName string) error {

	//
	producer, err := sarama.NewAsyncProducer(c.brokers, c.saramaConfig)

	if err != nil {
		return err
	}
	defer func() {
		if err := producer.Close(); err != nil {
			log.Printf("Error closing producer: %v", err)
		}
	}()

	// publish event
	publishMessage := func(message interface{}) {
		var b bytes.Buffer

		evt := event{
			Type:  eventType,
			Value: eventPayload,
		}
		err = json.NewEncoder(&b).Encode(evt)
		if err != nil {
			log.Printf("Error marshalling event: %v", err)
		}
		msg := &sarama.ProducerMessage{
			Topic: topicName,
			Value: sarama.ByteEncoder(b.Bytes()),
		}
		producer.Input() <- msg
	}
	publishMessage(eventPayload)

	return err
}
