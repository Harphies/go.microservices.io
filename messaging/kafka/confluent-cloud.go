package kafka

//type EventBroker struct {
//	client *kafka.Producer
//	logger *zap.Logger
//}
//
//// NewKafkaStream instantiates a Stream
//func NewKafkaStream(logger *zap.Logger, brokerEndpoints, saslScramUsername, saslScramPassword, securityProtocol, securityMechanism string) (*EventBroker, error) {
//	config := kafka.ConfigMap{
//		"bootstrap.servers": brokerEndpoints,
//		"sasl.mechanisms":   securityMechanism,
//		"security.protocol": securityProtocol,
//		"sasl.username":     saslScramUsername,
//		"sasl.password":     saslScramPassword,
//	}
//
//	client, err := kafka.NewProducer(&config)
//	if err != nil {
//		logger.Error("failed to establish connection with MSK", zap.Error(err))
//		return nil, err
//	}
//
//	//defer client.Close()
//
//	logger.Info("Successfully Established Connection with AWS MSK Kafka")
//	return &EventBroker{
//		client: client,
//		logger: logger,
//	}, nil
//}
//
//// Publish publishes a message indicating a record was created.
//func (k *EventBroker) Publish(eventPayload interface{}, topicName, eventType string) error {
//	return k.publish(eventType, eventPayload, topicName)
//}
//
//func (k *EventBroker) publish(eventType string, eventPayload interface{}, topicName string) error {
//
//	var b bytes.Buffer
//
//	evt := event{
//		Type:  eventType,
//		Value: eventPayload,
//	}
//
//	if err := json.NewEncoder(&b).Encode(evt); err != nil {
//		return err
//	}
//
//	if err := k.client.Produce(&kafka.Message{
//		TopicPartition: kafka.TopicPartition{
//			Topic:     &topicName,
//			Partition: kafka.PartitionAny,
//		},
//		Value: b.Bytes(),
//	}, nil); err != nil {
//		k.logger.Info(fmt.Sprintf("failed to publish the event type %s to topic %s with errror [%v]", eventType, topicName, err.Error()))
//		return err
//	}
//	k.logger.Info(fmt.Sprintf("event with type %s successfully published to [%s] topic", eventType, topicName))
//
//	return nil
//}
