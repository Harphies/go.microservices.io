package aws_sqs

// Queue processing functionalities
import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

type Queue struct {
	URL    string
	Name   string
	client *sqs.SQS
}

// Options to provide queue instantiation
type Options struct {
	QueueName string
	AwsRegion string
}

// NewQueue Instantiate a new Queue Object to enable Queue processing
func NewQueue(opts *Options) *Queue {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(opts.AwsRegion),
	})

	client := sqs.New(sess)

	if err != nil {
		fmt.Printf("failed to initialised new session: %v", err)
	}

	url, err := getQueueUrl(sess, opts.QueueName)
	return &Queue{
		URL:    *url.QueueUrl,
		client: client,
	}
}

func getQueueUrl(sess *session.Session, queueName string) (*sqs.GetQueueUrlOutput, error) {
	client := sqs.New(sess)

	result, err := client.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: &queueName,
	})

	if err != nil {
		return nil, err
	}

	return result, err
}

// SendMessage push messages to Queue

func (q *Queue) SendMessage(messageBody string) error {

	_, err := q.client.SendMessage(&sqs.SendMessageInput{
		QueueUrl:    &q.URL,
		MessageBody: aws.String(messageBody),
	})

	return err
}

// DeleteMessage Deletes a message in the queue

func (q *Queue) DeleteMessage(messageHandle *string) {
	_, err := q.client.DeleteMessage(&sqs.DeleteMessageInput{
		QueueUrl:      &q.URL,
		ReceiptHandle: messageHandle,
	})

	if err != nil {
		fmt.Printf("error while deleting the message from the queue: %v", err)
	}
}

// ProcessMessages long polling of messages from the queue
func (q *Queue) ProcessMessages(chn chan<- *sqs.Message) {
	for {
		result, err := q.client.ReceiveMessage(&sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(q.URL),
			MaxNumberOfMessages: aws.Int64(2),
			WaitTimeSeconds:     aws.Int64(15),
		})

		if err != nil {
			fmt.Printf("failed to fetch sqs messages %v", err)
		}

		for _, message := range result.Messages {
			chn <- message
		}
	}
}
