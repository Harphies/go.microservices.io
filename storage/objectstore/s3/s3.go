package s3

import (
	"bytes"
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	s3manager "github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	s3client "github.com/aws/aws-sdk-go-v2/service/s3"
	"go.uber.org/zap"
	pathutil "path"
	"strings"
	"time"
)

/*
Improvements
Use Multi-part upload API for uploads to improve uploads speed or large files: https://docs.aws.amazon.com/AmazonS3/latest/userguide/mpuoverview.html
*/

// AmazonS3Backend is a storage backend for Amazon S3
type AmazonS3Backend struct {
	Bucket     string
	Client     *s3client.Client
	Downloader *s3manager.Downloader
	Prefix     string
	Uploader   *s3manager.Uploader
	SSE        string
	logger     *zap.Logger
}

type (
	// Object is a generic representation of a storage object
	Object struct {
		Meta         Metadata
		Path         string
		Content      []byte
		LastModified time.Time
		logger       *zap.Logger
	}
	// Metadata represents the meta information of the object
	// includes object name , object version , etc...
	Metadata struct {
		Name    string
		Version string
	}

	// ObjectSliceDiff provides information on what has changed since last calling ListObjects
	ObjectSliceDiff struct {
		Change  bool
		Removed []Object
		Added   []Object
		Updated []Object
	}

	// Backend is a generic interface for storage backends
	Backend interface {
		ListObjects(prefix string) ([]Object, error)
		GetObject(path string) (Object, error)
		PutObject(path string, content []byte) error
		DeleteObject(path string) error
	}
)

// NewAmazonS3Backend creates a new instance of AmazonS3Backend
func NewAmazonS3Backend(logger *zap.Logger, bucket string, region string, prefix string) *AmazonS3Backend {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		logger.Error(fmt.Sprintf("failed to load IAM Role for Service Account OIDC token credentials: [%v]", err.Error()))
	}
	service := s3client.NewFromConfig(cfg)
	b := &AmazonS3Backend{
		Bucket:     bucket,
		Client:     service,
		Prefix:     cleanPrefix(prefix),
		logger:     logger,
		Downloader: s3manager.NewDownloader(service),
		Uploader:   s3manager.NewUploader(service),
	}
	return b
}

// PutObject uploads an object to Amazon S3 bucket, at prefix
func (b AmazonS3Backend) PutObject(path, contentType string, content []byte) (error, string) {
	s3Input := &s3client.PutObjectInput{
		Bucket:      aws.String(b.Bucket),
		Key:         aws.String(pathutil.Join(b.Prefix, path)),
		Body:        bytes.NewBuffer(content),
		ContentType: aws.String(contentType),
	}
	resp, err := b.Uploader.Upload(context.TODO(), s3Input)
	return err, *resp.Key
}

func cleanPrefix(prefix string) string {
	return strings.Trim(prefix, "/")
}

func objectPathIsInvalid(path string) bool {
	return strings.Contains(path, "/") || path == ""
}

func removePrefixFromObjectPath(prefix string, path string) string {
	if prefix == "" {
		return path
	}
	path = strings.Replace(path, fmt.Sprintf("%s/", prefix), "", 1)
	return path
}
