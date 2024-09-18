package s3

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"go.uber.org/zap"
)

// ErrInvalidObjectPath is returned when the object path is invalid
var ErrInvalidObjectPath = errors.New("invalid object path")

// ErrObjectNotFound is returned when the requested object does not exist
var ErrObjectNotFound = errors.New("object not found")

// ErrInvalidS3URL is returned when the provided S3 URL is invalid
var ErrInvalidS3URL = errors.New("invalid S3 URL")

// AmazonS3Backend is a storage backend for Amazon S3
type AmazonS3Backend struct {
	Bucket     string
	Client     *s3.Client
	Downloader *manager.Downloader
	Prefix     string
	Uploader   *manager.Uploader
	logger     *zap.Logger
}

// Object represents a generic storage object
type Object struct {
	Meta         Metadata
	Path         string
	Content      []byte
	LastModified time.Time
	logger       *zap.Logger
}

// Metadata represents the meta information of the object
type Metadata struct {
	Name    string
	Version string
}

// ObjectSliceDiff provides information on what has changed since last calling ListObjects
type ObjectSliceDiff struct {
	Change  bool
	Removed []Object
	Added   []Object
	Updated []Object
}

// Backend is a generic interface for storage backends
type Backend interface {
	ListObjects(prefix string) ([]Object, error)
	GetObject(path string) (Object, error)
	PutObject(path string, content []byte) error
	DeleteObject(path string) error
}

// NewAmazonS3Backend creates a new instance of AmazonS3Backend
func NewAmazonS3Backend(logger *zap.Logger, bucket, region, prefix string) (*AmazonS3Backend, error) {
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load IAM Role for Service Account OIDC token credentials: %w", err)
	}

	client := s3.NewFromConfig(cfg)
	return &AmazonS3Backend{
		Bucket:     bucket,
		Client:     client,
		Prefix:     cleanPrefix(prefix),
		logger:     logger,
		Downloader: manager.NewDownloader(client),
		Uploader:   manager.NewUploader(client),
	}, nil
}

// PutObject uploads an object to Amazon S3 bucket, at prefix
func (b *AmazonS3Backend) PutObject(path, contentType string, content []byte) (string, error) {
	if objectPathIsInvalid(path) {
		return "", ErrInvalidObjectPath
	}

	key := b.objectPath(path)
	s3Input := &s3.PutObjectInput{
		Bucket:      aws.String(b.Bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(content),
		ContentType: aws.String(contentType),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := b.Uploader.Upload(ctx, s3Input)
	if err != nil {
		return "", fmt.Errorf("failed to upload object: %w", err)
	}

	return key, nil
}

// GetObject retrieves an object from the S3 bucket
func (b *AmazonS3Backend) GetObject(pathOrURL string) (Object, error) {
	var bucket, key string
	var err error

	// Check if the input is a full S3 URL
	if strings.HasPrefix(strings.ToLower(pathOrURL), "https://") {
		bucket, key, err = parseS3URL(pathOrURL)
		if err != nil {
			return Object{}, err
		}
	} else {
		if objectPathIsInvalid(pathOrURL) {
			return Object{}, ErrInvalidObjectPath
		}
		bucket = b.Bucket
		key = b.objectPath(pathOrURL)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	obj := Object{
		Path:   key,
		logger: b.logger,
	}

	buf := manager.NewWriteAtBuffer([]byte{})
	_, err = b.Downloader.Download(ctx, buf, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		var nsk *types.NoSuchKey
		if errors.As(err, &nsk) {
			return Object{}, ErrObjectNotFound
		}
		return Object{}, fmt.Errorf("failed to download object: %w", err)
	}

	obj.Content = buf.Bytes()

	// Get object metadata
	headOutput, err := b.Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return Object{}, fmt.Errorf("failed to get object metadata: %w", err)
	}

	obj.LastModified = *headOutput.LastModified
	obj.Meta = Metadata{
		Name:    path.Base(key),
		Version: aws.ToString(headOutput.VersionId),
	}

	return obj, nil
}

func (b *AmazonS3Backend) objectPath(path string) string {
	return strings.TrimPrefix(fmt.Sprintf("%s/%s", b.Prefix, path), "/")
}

func cleanPrefix(prefix string) string {
	return strings.Trim(prefix, "/")
}

func objectPathIsInvalid(path string) bool {
	return path == "" || path == "/" || strings.HasPrefix(path, "/") || strings.HasSuffix(path, "/")
}

func removePrefixFromObjectPath(prefix, path string) string {
	if prefix == "" {
		return path
	}
	return strings.TrimPrefix(path, prefix+"/")
}

// parseS3URL parses a full S3 URL and returns the bucket and key
func parseS3URL(s3URL string) (bucket, key string, err error) {
	u, err := url.Parse(s3URL)
	if err != nil {
		return "", "", fmt.Errorf("%w: %v", ErrInvalidS3URL, err)
	}

	if u.Scheme != "https" || !strings.HasSuffix(u.Hostname(), ".amazonaws.com") {
		return "", "", ErrInvalidS3URL
	}

	parts := strings.SplitN(u.Hostname(), ".", 2)
	if len(parts) != 2 {
		return "", "", ErrInvalidS3URL
	}

	bucket = parts[0]
	key = strings.TrimPrefix(u.Path, "/")

	return bucket, key, nil
}
