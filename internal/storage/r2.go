package storage

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/Hisaalu/rogele/internal/config"
)

type R2Client struct {
	bucket    string
	publicURL string
	presigner *s3.PresignClient
	client    *s3.Client
}

func NewR2Client(ctx context.Context, cfg config.R2Config) (*R2Client, error) {
	if !cfg.Enabled() {
		return nil, fmt.Errorf("r2 is not configured")
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion("auto"),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				cfg.AccessKeyID,
				cfg.SecretAccessKey,
				"",
			),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(cfg.Endpoint)
		o.UsePathStyle = true
	})

	return &R2Client{
		bucket:    cfg.Bucket,
		publicURL: strings.TrimSuffix(cfg.PublicURL, "/"),
		presigner: s3.NewPresignClient(client),
		client:    client,
	}, nil
}

func (c *R2Client) PublicURL(key string) string {
	return fmt.Sprintf("%s/%s", c.publicURL, strings.TrimPrefix(key, "/"))
}

func (c *R2Client) BaseURL() string {
	return c.publicURL
}

func (c *R2Client) DeleteObject(ctx context.Context, key string) error {
	if key == "" {
		return fmt.Errorf("delete r2 object: empty key")
	}
	_, err := c.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("delete r2 object %q: %w", key, err)
	}
	return nil
}

func (c *R2Client) ExtractKey(filePath string) string {
	if filePath == "" {
		return ""
	}

	if strings.HasPrefix(filePath, "http://") || strings.HasPrefix(filePath, "https://") {
		parts := strings.SplitN(filePath, "/", 4)
		if len(parts) == 4 {
			path := parts[3]
			path = strings.TrimPrefix(path, "rogele-platform/")
			return path
		}
	}

	path := strings.TrimPrefix(filePath, "/")
	path = strings.TrimPrefix(path, "rogele-platform/")
	return path
}
