package storage

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/config"
)

type S3Storage struct {
	client        *s3.Client
	presignClient *s3.PresignClient
	bucket        string
	baseURL       string
	presignExpire time.Duration
}

func NewS3(cfg config.AWSConfig) (*S3Storage, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("s3: load config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg)
	return &S3Storage{
		client:        client,
		presignClient: s3.NewPresignClient(client),
		bucket:        cfg.S3.Bucket,
		baseURL:       cfg.S3.BaseURL,
		presignExpire: cfg.S3.PresignExpire,
	}, nil
}

// Upload streams a file to S3 and returns its storage key (not a URL — callers
// persist the key and resolve it via PresignDownload at read time).
func (s *S3Storage) Upload(ctx context.Context, folder, filename string, body io.Reader, contentType string) (string, error) {
	key := buildKey(folder, filename)
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("s3: upload: %w", err)
	}
	return key, nil
}

// PresignUpload generates a pre-signed URL clients can PUT to directly.
func (s *S3Storage) PresignUpload(ctx context.Context, folder, extension string) (uploadURL, key string, err error) {
	key = buildKey(folder, uuid.NewString()+extension)
	req, err := s.presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(s.presignExpire))
	if err != nil {
		return "", "", fmt.Errorf("s3: presign upload: %w", err)
	}
	return req.URL, key, nil
}

// PresignDownload generates a pre-signed GET URL for a private object.
// requestBase is unused: a presigned S3 URL is already absolute and points at
// AWS regardless of how the browser reached our API.
func (s *S3Storage) PresignDownload(ctx context.Context, key string, requestBase string) (string, error) {
	req, err := s.presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(s.presignExpire))
	if err != nil {
		return "", fmt.Errorf("s3: presign download: %w", err)
	}
	return req.URL, nil
}

// Download fetches an object's body from S3. The caller closes the reader.
func (s *S3Storage) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("s3: download %q: %w", key, err)
	}
	return out.Body, nil
}

// Delete removes an object from S3.
func (s *S3Storage) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	return err
}

func buildKey(folder, filename string) string {
	ext := filepath.Ext(filename)
	base := strings.TrimSuffix(filename, ext)
	return fmt.Sprintf("%s/%s%s", folder, base, ext)
}
