package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"go.uber.org/zap"
)

var (
	ErrInvalidObjectPath = errors.New("invalid object path")
	ErrObjectNotFound    = errors.New("object not found")
	ErrInvalidS3URL      = errors.New("invalid S3 URL")
)

type AmazonS3Backend struct {
	bucket     string
	client     *s3.Client
	downloader *manager.Downloader
	prefix     string
	uploader   *manager.Uploader
	logger     *zap.Logger
}

type Object struct {
	Meta         Metadata
	Path         string
	Content      []byte
	ContentType  string
	LastModified time.Time
	Size         int64
}

type Metadata struct {
	Name    string
	Version string
}

// NewAmazonS3Backend creates a new optimized S3 backend instance
func NewAmazonS3Backend(logger *zap.Logger, bucket, region, prefix string) (*AmazonS3Backend, error) {
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(region),
		config.WithRetryMode(aws.RetryModeStandard),
		config.WithRetryMaxAttempts(3),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true // Use path-style addressing
	})

	// Configure uploader for optimal performance
	uploader := manager.NewUploader(client, func(u *manager.Uploader) {
		u.PartSize = 5 * 1024 * 1024 // 5MB per part
		u.Concurrency = 3            // Concurrent part uploads
		u.LeavePartsOnError = false  // Cleanup on failures
	})

	// Configure downloader for optimal performance
	downloader := manager.NewDownloader(client, func(d *manager.Downloader) {
		d.PartSize = 5 * 1024 * 1024 // 5MB per part
		d.Concurrency = 3            // Concurrent part downloads
	})

	return &AmazonS3Backend{
		bucket:     bucket,
		client:     client,
		prefix:     cleanPrefix(prefix),
		logger:     logger,
		downloader: downloader,
		uploader:   uploader,
	}, nil
}

// PutObject uploads an object to S3 with optimized settings for both small and large files
func (b *AmazonS3Backend) PutObject(ctx context.Context, path string, contentType string, reader io.Reader, size int64) (string, error) {
	if objectPathIsInvalid(path) {
		return "", ErrInvalidObjectPath
	}

	key := b.objectPath(path)

	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	input := &s3.PutObjectInput{
		Bucket:      aws.String(b.bucket),
		Key:         aws.String(key),
		Body:        reader,
		ContentType: aws.String(contentType),
	}

	_, err := b.uploader.Upload(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to upload object: %w", err)
	}

	return key, nil
}

// GetObject downloads an object from S3 with optimized settings
func (b *AmazonS3Backend) GetObject(ctx context.Context, pathOrURL string) (*Object, error) {
	bucket, key, err := b.resolveBucketAndKey(pathOrURL)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	// Get object metadata first
	headOutput, err := b.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var nsk *types.NoSuchKey
		if errors.As(err, &nsk) {
			return nil, ErrObjectNotFound
		}
		return nil, fmt.Errorf("failed to get object metadata: %w", err)
	}

	// Allocate buffer based on object size
	buf := manager.NewWriteAtBuffer(make([]byte, 0, *headOutput.ContentLength))

	_, err = b.downloader.Download(ctx, buf, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to download object: %w", err)
	}

	return &Object{
		Path:    key,
		Content: buf.Bytes(),
		Meta: Metadata{
			Name:    path.Base(key),
			Version: aws.ToString(headOutput.VersionId),
		},
		ContentType:  aws.ToString(headOutput.ContentType),
		LastModified: *headOutput.LastModified,
		Size:         *headOutput.ContentLength,
	}, nil
}

// GetObjectStream returns a reader for streaming large objects
func (b *AmazonS3Backend) GetObjectStream(ctx context.Context, pathOrURL string) (io.ReadCloser, *Object, error) {
	bucket, key, err := b.resolveBucketAndKey(pathOrURL)
	if err != nil {
		return nil, nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)

	output, err := b.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		cancel()
		var nsk *types.NoSuchKey
		if errors.As(err, &nsk) {
			return nil, nil, ErrObjectNotFound
		}
		return nil, nil, fmt.Errorf("failed to get object: %w", err)
	}

	obj := &Object{
		Path: key,
		Meta: Metadata{
			Name:    path.Base(key),
			Version: aws.ToString(output.VersionId),
		},
		ContentType:  aws.ToString(output.ContentType),
		LastModified: *output.LastModified,
		Size:         *output.ContentLength,
	}

	// Return the reader and a cleanup function
	return &readCloserWithCancel{
		ReadCloser: output.Body,
		cancel:     cancel,
	}, obj, nil
}

type readCloserWithCancel struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (r *readCloserWithCancel) Close() error {
	defer r.cancel()
	return r.ReadCloser.Close()
}

// Helper functions
func (b *AmazonS3Backend) resolveBucketAndKey(pathOrURL string) (bucket, key string, err error) {
	if strings.HasPrefix(strings.ToLower(pathOrURL), "https://") {
		return parseS3URL(pathOrURL)
	}

	if objectPathIsInvalid(pathOrURL) {
		return "", "", ErrInvalidObjectPath
	}

	return b.bucket, b.objectPath(pathOrURL), nil
}

func (b *AmazonS3Backend) objectPath(path string) string {
	return strings.TrimPrefix(fmt.Sprintf("%s/%s", b.prefix, path), "/")
}

func cleanPrefix(prefix string) string {
	return strings.Trim(prefix, "/")
}

func objectPathIsInvalid(path string) bool {
	return path == "" || path == "/" || strings.HasPrefix(path, "/") || strings.HasSuffix(path, "/")
}

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

	return parts[0], strings.TrimPrefix(u.Path, "/"), nil
}
