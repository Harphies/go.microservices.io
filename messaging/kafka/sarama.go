package kafka

import (
	"crypto/tls"
	"encoding/json"
	"github.com/IBM/sarama"
	"go.uber.org/zap"
	"log"
	"strings"
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
	config.Producer.Return.Errors = true
	//config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 5
	//config.Producer.Partitioner = sarama.NewHashPartitioner
	//config.Producer.Compression = sarama.CompressionSnappy

	// Batching for network throughput and latency
	//config.Producer.Flush.Frequency = 500 * time.Millisecond
	//config.Producer.Flush.Messages = 100
	//config.Producer.Flush.MaxMessages = 1000
	//config.Producer.Flush.Bytes = 1048576

	// Network Settings
	//config.Net.MaxOpenRequests = 5
	//config.Net.DialTimeout = 30 * time.Second
	//config.Net.ReadTimeout = 30 * time.Second
	//config.Net.WriteTimeout = 30 * time.Second

	if useAuth {
		log.Println("UseAuth is set")
		config.Net.SASL.Enable = true
		config.Net.SASL.User = saslScramUsername
		config.Net.SASL.Password = saslScramPassword
		config.Net.SASL.Mechanism = sarama.SASLMechanism(securityMechanism)
		config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient { return NewKafkaClient() }
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
	producer, err := sarama.NewSyncProducer(c.brokers, c.saramaConfig)
	if err != nil {
		c.logger.Error("Failed to start Sarama producer: %v", zap.Error(err))
		return err
	}

	defer func() {
		if err = producer.Close(); err != nil {
			c.logger.Error("Error closing producer: %v", zap.Error(err))
		}
	}()

	// publish event
	publishMessage := func(message interface{}) {
		jsonData, err := json.Marshal(message)
		if err != nil {
			c.logger.Error("Error marshalling message: %v", zap.Error(err))
		}

		msg := &sarama.ProducerMessage{
			Topic: topicName,
			Key:   sarama.StringEncoder(eventType),
			Value: sarama.StringEncoder(jsonData),
		}

		// Send the message
		partition, offset, err := producer.SendMessage(msg)
		if err != nil {
			c.logger.Error("Failed to send message", zap.Error(err))
		}
		c.logger.Info("Message sent to partition with offset", zap.Any("partition", partition), zap.Any("offset", offset))
	}
	publishMessage(eventPayload)

	return err
}
