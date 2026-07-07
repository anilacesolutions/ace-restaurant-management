// Package storage wraps object storage for user-uploaded assets (menu images).
// It is deliberately thin: the browser uploads directly to S3 with a presigned
// PUT URL, so the API never streams file bytes through Go.
package storage

import (
	"context"
	"fmt"
	"strings"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3 issues presigned upload URLs and resolves public object URLs. Construct
// with New; a nil *S3 means image upload is disabled (no bucket configured).
type S3 struct {
	client     *s3.Client
	presign    *s3.PresignClient
	bucket     string
	region     string
	publicBase string
}

// New builds an S3 client from the default AWS credential chain (env vars,
// ~/.aws, or an IAM role in prod). publicBase is an optional CDN/base URL in
// front of the bucket; when empty, object URLs are derived from bucket+region.
func New(ctx context.Context, bucket, region, publicBase string) (*S3, error) {
	if bucket == "" {
		return nil, fmt.Errorf("bucket is required")
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}
	client := s3.NewFromConfig(cfg)
	return &S3{
		client:     client,
		presign:    s3.NewPresignClient(client),
		bucket:     bucket,
		region:     region,
		publicBase: strings.TrimRight(publicBase, "/"),
	}, nil
}

// PresignPut returns a URL the browser can PUT the object to for the given
// duration. contentType must match the Content-Type header the browser sends.
func (s *S3) PresignPut(ctx context.Context, key, contentType string, ttl time.Duration) (string, error) {
	out, err := s.presign.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      &s.bucket,
		Key:         &key,
		ContentType: &contentType,
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", fmt.Errorf("presign put: %w", err)
	}
	return out.URL, nil
}

// PublicURL returns the URL the stored object is served from after upload.
func (s *S3) PublicURL(key string) string {
	if s.publicBase != "" {
		return s.publicBase + "/" + key
	}
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucket, s.region, key)
}
