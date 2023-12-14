package dynamodb

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"go.uber.org/zap"
	"strconv"
)

var retryAttempt int64 = 1

type AWSDynamoDbDataStore struct {
	client *dynamodb.Client
	logger *zap.Logger
}

func NewAWSDynamoDbDataStore(logger *zap.Logger, dbRetryAttempt, region string) *AWSDynamoDbDataStore {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		logger.Error(fmt.Sprintf("failed to load  credentials: %v", err.Error()))
	}
	retryAttempt, err = strconv.ParseInt(dbRetryAttempt, 10, 64)
	dynamoDBClient := dynamodb.New(dynamodb.Options{Credentials: cfg.Credentials, Region: cfg.Region, RetryMaxAttempts: int(retryAttempt), RetryMode: aws.RetryModeAdaptive})
	return &AWSDynamoDbDataStore{
		client: dynamoDBClient,
		logger: logger,
	}
}

func (db *AWSDynamoDbDataStore) WriteItem(tableName string, item interface{}) error {
	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		db.logger.Info(fmt.Sprintf("failed to convert item into dynamodb accepted type: [%v]", err.Error()))
		return err
	}
	_, err = db.client.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item:      av,
	})

	return err // the err value will be nil if there's no error, so same as "return nil"
}
