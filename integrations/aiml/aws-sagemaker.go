package aiml

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sagemakerruntime"
	"go.uber.org/zap"
)

type SageMakerInference struct {
	client       *sagemakerruntime.SageMakerRuntime
	logger       *zap.Logger
	ctx          context.Context
	endpointName string
}

func NewSageMakerInference(logger *zap.Logger, endpointName string) *SageMakerInference {
	client := sagemakerruntime.New(session.Must(session.NewSession()))
	ctx := context.Background()
	return &SageMakerInference{
		client:       client,
		ctx:          ctx,
		logger:       logger,
		endpointName: endpointName,
	}
}

// Invocation makes inference call to the hosted sagemaker endpoint.
func (s *SageMakerInference) Invocation(inferenceBody map[string]interface{}) []byte {
	body, err := json.Marshal(inferenceBody)
	if err != nil {
		s.logger.Error(fmt.Sprintf("failed to convert inference body to byte array: %v", err))
	}
	s.logger.Info(fmt.Sprintf("making an AI inference with the body data: %v", string(body)))

	resp, err := s.client.InvokeEndpoint(&sagemakerruntime.InvokeEndpointInput{
		EndpointName: aws.String(s.endpointName),
		Body:         body,
		ContentType:  aws.String("application/json"),
	})

	if err != nil {
		s.logger.Error(fmt.Sprintf("failed to make invocation to this sagemaker endpoint %s with error [%+v]", s.endpointName, err.Error()))
	}

	return resp.Body
}
