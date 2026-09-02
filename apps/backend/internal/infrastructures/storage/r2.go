package storage

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"backend/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	smithy "github.com/aws/smithy-go"
)

// r2Region is the region value Cloudflare R2 expects; it is always "auto".
const r2Region = "auto"

type R2Storage struct {
	client  *s3.Client
	presign *s3.PresignClient
	bucket  string
}

func NewR2(cfg *config.R2) (*R2Storage, error) {
	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfg.AccountID)
	awsCfg, err := awsConfig.LoadDefaultConfig(
		context.Background(),
		awsConfig.WithRegion(r2Region),
		awsConfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID, cfg.SecretAccessKey, "",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("load r2 aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(options *s3.Options) {
		options.BaseEndpoint = aws.String(endpoint)
	})
	return &R2Storage{
		client:  client,
		presign: s3.NewPresignClient(client),
		bucket:  cfg.Bucket,
	}, nil
}

func (s *R2Storage) PresignPut(
	ctx context.Context,
	key string,
	contentType string,
	expires time.Duration,
) (string, error) {
	request, err := s.presign.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}, s3.WithPresignExpires(expires))
	if err != nil {
		return "", fmt.Errorf("presign put object %s: %w", key, err)
	}
	return request.URL, nil
}

func (s *R2Storage) Head(ctx context.Context, key string) (*ObjectMetadata, error) {
	output, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			var httpErr interface{ HTTPStatusCode() int }
			if errors.As(err, &httpErr) && httpErr.HTTPStatusCode() == http.StatusNotFound {
				return nil, ErrObjectNotFound
			}
		}
		return nil, fmt.Errorf("head object %s: %w", key, err)
	}

	metadata := &ObjectMetadata{Key: key}
	if output.ContentLength != nil {
		metadata.Size = *output.ContentLength
	}
	if output.ContentType != nil {
		metadata.ContentType = *output.ContentType
	}
	if output.ETag != nil {
		metadata.ETag = *output.ETag
	}
	if output.LastModified != nil {
		metadata.LastModified = *output.LastModified
	}
	return metadata, nil
}

func (s *R2Storage) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("delete object %s: %w", key, err)
	}
	return nil
}
