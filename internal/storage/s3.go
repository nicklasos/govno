// Package storage implements backup storage backends.
package storage

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/nicklasos/govno/v2/internal/config"
)

// Object is a stored backup.
type Object struct {
	Key          string
	LastModified time.Time
}

// S3 stores backups in an S3 (or S3-compatible) bucket.
type S3 struct {
	client   *s3.Client
	uploader *transfermanager.Client
	bucket   string
}

// NewS3 builds an S3 backend. Static credentials are used when configured,
// otherwise the AWS default credential chain.
func NewS3(ctx context.Context, cfg config.S3) (*S3, error) {
	var opts []func(*awsconfig.LoadOptions) error
	if cfg.Region != "" {
		opts = append(opts, awsconfig.WithRegion(cfg.Region))
	}
	if cfg.AccessKeyID != "" {
		opts = append(opts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		))
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
			o.UsePathStyle = true
		}
	})
	return &S3{
		client:   client,
		uploader: transfermanager.New(client),
		bucket:   cfg.Bucket,
	}, nil
}

// Put streams r to key. If r returns an error the upload is aborted and no
// object is created.
func (s *S3) Put(ctx context.Context, key string, r io.Reader) error {
	_, err := s.uploader.UploadObject(ctx, &transfermanager.UploadObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        r,
		ContentType: aws.String("application/gzip"),
	})
	if err != nil {
		return fmt.Errorf("upload s3://%s/%s: %w", s.bucket, key, err)
	}
	return nil
}

// List returns all objects under prefix.
func (s *S3) List(ctx context.Context, prefix string) ([]Object, error) {
	var objects []Object
	p := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(prefix),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("list s3://%s/%s: %w", s.bucket, prefix, err)
		}
		for _, o := range page.Contents {
			objects = append(objects, Object{
				Key:          aws.ToString(o.Key),
				LastModified: aws.ToTime(o.LastModified),
			})
		}
	}
	return objects, nil
}

// Delete removes keys, in batches of up to 1000.
func (s *S3) Delete(ctx context.Context, keys []string) error {
	const batch = 1000
	for start := 0; start < len(keys); start += batch {
		end := min(start+batch, len(keys))
		ids := make([]types.ObjectIdentifier, 0, end-start)
		for _, k := range keys[start:end] {
			ids = append(ids, types.ObjectIdentifier{Key: aws.String(k)})
		}
		out, err := s.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: aws.String(s.bucket),
			Delete: &types.Delete{Objects: ids, Quiet: aws.Bool(true)},
		})
		if err != nil {
			return fmt.Errorf("delete from s3://%s: %w", s.bucket, err)
		}
		if len(out.Errors) > 0 {
			e := out.Errors[0]
			return fmt.Errorf("delete s3://%s/%s: %s (and %d more)",
				s.bucket, aws.ToString(e.Key), aws.ToString(e.Message), len(out.Errors)-1)
		}
	}
	return nil
}
