package dynamodb

import (
	"context"
	"errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"go.uber.org/zap"
	"strings"
)

const (
	// maxGetOps: DynamoDB API limit, 100 operations per request
	maxGetOps = 100
)

// Interface Implementation of AmazonDynamoDB Table Operations.
type (
	// DynamoDBRepository Repository Interface Implementation
	DynamoDBRepository struct {
		TableName *string
		Client    dynamodbiface.DynamoDBAPI
		Logger    *zap.Logger
	}

	// Student Records
	Student struct {
		PK   string `dynamodbav:"pk"`
		SK   string `dynamodbav:"sk"`
		ID   string `dynamodbav:"id"`
		Name string `dynamodbav:"name"`
		Age  string `dynamodbav:"age"`
	}
)

// NewRepository ...
func NewRepository(logger *zap.Logger, client dynamodbiface.DynamoDBAPI, table string) *DynamoDBRepository {
	return &DynamoDBRepository{
		Client:    client,
		TableName: aws.String(table),
		Logger:    logger,
	}
}

// AddItem put a record Item in the DB
// It takes in a record and add the partition key and sort key before putting it in the table.
func (d *DynamoDBRepository) AddItem(ctx context.Context, item Student) {
	studentItem := NewStudent(item)
	putItem, err := dynamodbattribute.MarshalMap(studentItem)
	if err != nil {
		d.Logger.Info("Error converting the student Item to dynamodb compatible format")
	}

	// build up the input with PutItem or Put Method
	input := &dynamodb.PutItemInput{
		TableName: d.TableName,
		Item:      putItem,
	}

	// client perform the put operation
	_, err = d.Client.PutItemWithContext(ctx, input)
}

// NewStudent Create an Instance of DynamoDB item from a Student struct by adding a
// Partition Key and Sort key to the Incoming Data and return the modified Struct.

func NewStudent(item Student) Student {
	// Build up the partition key
	pk := itemPartitionKey(item.ID)
	//
	// Build up the sort key
	sk := itemSortKey(item.ID)

	// return a data suitable for dynamoDB record.
	return Student{
		PK:   pk,
		SK:   sk,
		ID:   item.ID,
		Age:  item.Age,
		Name: item.Name,
	}
}

// Create a record in a DynamoDB Table
func (d *DynamoDBRepository) Create(ctx context.Context, item interface{}) error {
	// instantiate a new record.
	// Marshall the instantiated record into dynamodb Map
	//putItem, err := dynamodbattribute.Marshal(record)
	//if err != nil {
	//	// TODO: proper logger with uber zap implementation
	//	return errors.New("error marshalling the record to dynamodb compatible type")
	//}
	//
	//// call the appropriate dynamoDB Methods for the Operation
	//input := &dynamodb.PutItemInput{
	//	TableName: d.TableName,
	//	Item: putItem,
	//}
	//
	//// use the client to insert the record.
	//_, err = d.Client.PutItemWithContext(ctx, input)
	//return err

	return errors.New("WIP")
}

// Delete a record in a DynamoDB Table
//func (d *DynamoDBRepository) Delete(ctx, pk )

// --
const (
	itemPKPrefix = "STUDENT"
	itemSKPrefix = "ITEM"
	keySeparator = "#"
)

// --
func itemSortKey(itemID string) string {
	elems := []string{itemSKPrefix, itemID}
	return strings.Join(elems, keySeparator)
}

// --
func itemPartitionKey(itemID string) string {
	elems := []string{itemPKPrefix, itemID}
	return strings.Join(elems, keySeparator)
}
