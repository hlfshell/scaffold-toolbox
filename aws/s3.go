package aws

import (
	"bytes"
	"context"
	"fmt"
	"os"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

/*
UploadObject writes an object to an S3 bucket through the MiniStack S3
endpoint.
*/
func (s *Stack) UploadObject(ctx context.Context, bucket string, key string, body []byte) error {
	client, err := s.S3Client(ctx)
	if err != nil {
		return err
	}
	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: awssdk.String(bucket),
		Key:    awssdk.String(key),
		Body:   bytes.NewReader(body),
	})
	if err != nil {
		return fmt.Errorf("failed to upload s3 object %s/%s: %w", bucket, key, err)
	}

	return nil
}

/*
UploadObjectString writes a string as an S3 object through the MiniStack
S3 endpoint.
*/
func (s *Stack) UploadObjectString(ctx context.Context, bucket string, key string, body string) error {
	return s.UploadObject(ctx, bucket, key, []byte(body))
}

/*
UploadFile reads a host file and writes it to an S3 bucket through the
MiniStack S3 endpoint.
*/
func (s *Stack) UploadFile(ctx context.Context, bucket string, key string, path string) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	return s.UploadObject(ctx, bucket, key, body)
}
