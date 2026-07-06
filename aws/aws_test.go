package aws

import (
	"context"
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	scaffoldcontainer "github.com/hlfshell/scaffold/container"
)

func TestAWSStackCreateResourcesCleanup(t *testing.T) {
	if !scaffoldcontainer.DockerAvailable() {
		t.Skip("docker is not available")
	}

	ctx := context.Background()
	stack, err := NewStack("scaffold-test-aws", "latest",
		WithServices("s3", "sqs"),
		WithS3Bucket("scaffold-test-bucket"),
		WithSQSQueue("jobs"),
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := stack.Create(ctx); err != nil {
		t.Fatal(err)
	}
	defer stack.Cleanup(ctx)

	client, err := stack.S3Client(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: awssdk.String("scaffold-test-bucket")}); err != nil {
		t.Fatal(err)
	}
	if _, ok := stack.QueueURL("jobs"); !ok {
		t.Fatal("expected queue URL for jobs")
	}
}
