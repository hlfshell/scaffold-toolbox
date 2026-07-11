# scaffold toolbox aws

MiniStack-backed local AWS service for scaffold. It starts a single local AWS-compatible endpoint on port 4566, exposes SDK configuration helpers, and can create common resources such as S3 buckets, SQS queues, and SNS topics. Services can be picked per your needs and configured to work with your setup.

Copyable app-shaped setups live in `examples/`. Those examples intentionally use this package directly instead of a separate AWS stack wrapper, so MiniStack service configuration, ECS behavior, local image publishing, S3 helpers, and SDK config stay in one place.

## Install

```bash
go get github.com/hlfshell/scaffold-toolbox/aws
```

```go
import "github.com/hlfshell/scaffold-toolbox/aws"
```

## Example

```go
cloud, err := aws.NewStack("cloud", "latest",
	aws.WithS3("documents"),
	aws.WithSQS("jobs"),
	aws.WithSNS("events"),
)
if err != nil {
	return err
}

stack := scaffold.NewStack("app", scaffold.WithServices(cloud))
```

## Services

Use typed `With<Service>` options instead of passing MiniStack service flags
directly:

```go
cloud, err := aws.NewStack("cloud", "latest",
	aws.WithS3("documents", "uploads"),
	aws.WithDynamoDB(aws.DynamoDBTable{Name: "items"}),
	aws.WithSecretsManager(aws.Secret{Name: "api-key", Value: "secret"}),
	aws.WithSSM(aws.Parameter{Name: "/app/url", Value: "http://localhost"}),
	aws.WithKinesis(aws.KinesisStream{Name: "events"}),
	aws.WithEventBridge(aws.EventBus{Name: "app"}),
	aws.WithLambda(),
	aws.WithECS(),
)
```

Resource-aware options both enable the MiniStack service and create the
requested resources after MiniStack is ready. Today that includes:

- `WithS3` for buckets.
- `WithSQS` for queues.
- `WithSNS` for topics.
- `WithDynamoDB` for simple string-key tables.
- `WithSecretsManager` for string secrets.
- `WithSSM` for parameters.
- `WithKinesis` for streams.
- `WithEventBridge` for event buses.

For lower-level or newly added MiniStack services, `WithServices("service")`
is still available as an escape hatch.

To start every service known to this toolbox module:

```go
cloud, err := aws.NewStack("cloud", "latest", aws.WithAll())
```

Use `WithDockerSocket` when MiniStack needs to create real backing
containers through the host Docker daemon, such as RDS, ElastiCache, ECS,
or Docker-backed Lambda. Use `WithDockerNetwork("network-name")` when those
backing containers need to share a Docker network with other scaffold
services.

`WithEnv` passes MiniStack configuration flags through directly, such as
`PERSIST_STATE`, service-specific dataplane flags, or logging options.

## Application connection config

Use `HostConnection` or `HostEnv` for applications running on your machine.
Use `ContainerConnection` or `ContainerEnv` for applications running in
another container on the same Docker network as MiniStack.

```go
hostEnv := cloud.HostEnv()
fmt.Println(hostEnv["AWS_ENDPOINT_URL"])

containerEnv := cloud.ContainerEnv()
fmt.Println(containerEnv["AWS_ENDPOINT_URL"])
```

`HostEnv` points at the published localhost port, such as
`http://127.0.0.1:32781`. `ContainerEnv` points at the Docker-network name,
such as `http://cloud-ministack:4566`. Container access requires a shared
user-defined Docker network; using the AWS stack inside a scaffold stack with
`scaffold.WithSharedNetwork()` handles that for scaffold services.

Both env helpers include:

- `AWS_ENDPOINT_URL`
- `AWS_REGION`
- `AWS_DEFAULT_REGION`
- `AWS_ACCESS_KEY_ID`
- `AWS_SECRET_ACCESS_KEY`
- `AWS_S3_FORCE_PATH_STYLE=true`
- `S3_FORCE_PATH_STYLE=true`

Queue URLs, topic ARNs, and registry values are included after those
resources exist.

## ECS containers

MiniStack's ECS support runs tasks as real Docker containers through the
host Docker daemon. The AWS toolbox builds on that with typed helpers for
clusters, services, one-shot tasks, and local images.

```go
cloud, err := aws.NewStack("cloud", "latest",
	aws.WithECSCluster("app"),
	aws.WithECSService(aws.ECSService{
		Name:       "api",
		Cluster:    "app",
		Family:     "api",
		LaunchType: aws.ECSLaunchTypeFargate,
		Containers: []aws.ECSContainer{{
			Name:       "api",
			Dockerfile: "./Dockerfile",
			Image:      "app/api:dev",
			Ports: []aws.ECSPort{{
				ContainerPort: 8080,
			}},
		}},
	}),
)
```

`Dockerfile` and `LocalImage` containers automatically enable a local Docker
registry. The image is built or tagged, pushed to that registry, and the ECS
task definition uses the host-reachable registry image because MiniStack asks
the host Docker daemon to run the task.

Use `RegistryAddress`, `RegistryImage`, `RegistryDockerConfigJSON`,
`PushImage`, and `BuildAndPushImage` when you want to prepare images from
your own commands after the stack is running.

Fargate is represented through ECS task definition compatibility and launch
type. Locally, MiniStack still runs the container through Docker. EC2 in
MiniStack is useful for VPC, subnet, security group, and instance metadata;
it does not start real virtual machines.

## Common examples

### S3 bucket and objects

`WithS3` creates buckets after MiniStack is ready. Use the returned S3
client exactly like a normal AWS SDK client.

```go
ctx := context.Background()

cloud, err := aws.NewStack("cloud", "latest",
	aws.WithS3("documents"),
)
if err != nil {
	return err
}
defer cloud.Cleanup(ctx)

if err := cloud.Create(ctx); err != nil {
	return err
}

s3Client, err := cloud.S3Client(ctx)
if err != nil {
	return err
}

_, err = s3Client.PutObject(ctx, &s3.PutObjectInput{
	Bucket: awsSDK.String("documents"),
	Key:    awsSDK.String("notes/hello.txt"),
	Body:   strings.NewReader("hello from scaffold"),
})
if err != nil {
	return err
}

object, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
	Bucket: awsSDK.String("documents"),
	Key:    awsSDK.String("notes/hello.txt"),
})
if err != nil {
	return err
}
defer object.Body.Close()

body, err := io.ReadAll(object.Body)
if err != nil {
	return err
}
fmt.Println(string(body))
```

For common seed data, use the stack helpers:

```go
err = cloud.UploadObjectString(ctx, "documents", "notes/hello.txt", "hello")
err = cloud.UploadFile(ctx, "documents", "fixtures/report.json", "./fixtures/report.json")
```

The imports for that example use aliases to keep the toolbox package and
AWS SDK package distinct:

```go
import (
	"context"
	"fmt"
	"io"
	"strings"

	awsSDK "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/hlfshell/scaffold-toolbox/aws"
)
```

### ECS service from a Dockerfile

Use `WithECSService` for an API or worker that should stay running. A
container with `Dockerfile` automatically enables the local registry, builds
the image, pushes it, and registers the pushed image in the task definition.

```go
cloud, err := aws.NewStack("cloud", "latest",
	aws.WithECSCluster("app"),
	aws.WithECSService(aws.ECSService{
		Name:         "api",
		Cluster:      "app",
		Family:       "api",
		DesiredCount: 1,
		LaunchType:   aws.ECSLaunchTypeEC2,
		Containers: []aws.ECSContainer{{
			Name:       "api",
			Dockerfile: "./api/Dockerfile",
			Image:      "app/api:dev",
			Ports: []aws.ECSPort{{
				ContainerPort: 8080,
			}},
		}},
	}),
)
```

If you already built the image locally, use `LocalImage` instead:

```go
aws.WithECSRunTask(aws.ECSRunTask{
	Name:    "job",
	Cluster: "app",
	Family:  "job",
	Containers: []aws.ECSContainer{{
		Name:       "job",
		LocalImage: "app/job:local",
		Image:      "app/job:dev",
	}},
})
```

### Fargate-style task

For Fargate-shaped workflows, set the launch type to
`ECSLaunchTypeFargate`. MiniStack still runs the task as a local Docker
container, but the registered task definition uses Fargate compatibility.

```go
cloud, err := aws.NewStack("cloud", "latest",
	aws.WithECSCluster("jobs"),
	aws.WithECSRunTask(aws.ECSRunTask{
		Name:       "thumbnailer",
		Cluster:    "jobs",
		Family:     "thumbnailer",
		LaunchType: aws.ECSLaunchTypeFargate,
		Containers: []aws.ECSContainer{{
			Name:       "thumbnailer",
			Dockerfile: "./workers/thumbnailer/Dockerfile",
			Image:      "jobs/thumbnailer:dev",
			Env: map[string]string{
				"INPUT_BUCKET":  "uploads",
				"OUTPUT_BUCKET": "thumbs",
			},
		}},
	}),
)
```

### Lambda container image

MiniStack can run Lambda functions from Docker images using
`PackageType: Image`. The image should be a Lambda-compatible container
image, such as one built from an AWS Lambda base image or one that includes
the Lambda runtime interface client.

```go
ctx := context.Background()

cloud, err := aws.NewStack("cloud", "latest",
	aws.WithLambda(),
	aws.WithDockerfileImage("./lambda/Dockerfile", "functions/echo:dev"),
)
if err != nil {
	return err
}
defer cloud.Cleanup(ctx)

if err := cloud.Create(ctx); err != nil {
	return err
}

lambdaClient, err := cloud.LambdaClient(ctx)
if err != nil {
	return err
}

_, err = lambdaClient.CreateFunction(ctx, &lambda.CreateFunctionInput{
	FunctionName: awsSDK.String("echo"),
	Role:         awsSDK.String("arn:aws:iam::000000000000:role/lambda"),
	PackageType:  lambdatypes.PackageTypeImage,
	Code: &lambdatypes.FunctionCode{
		ImageUri: awsSDK.String(cloud.RegistryImage("functions/echo:dev")),
	},
})
if err != nil {
	return err
}

result, err := lambdaClient.Invoke(ctx, &lambda.InvokeInput{
	FunctionName: awsSDK.String("echo"),
	Payload:      []byte(`{"message":"hello"}`),
})
if err != nil {
	return err
}
fmt.Println(string(result.Payload))
```

Add these imports for the Lambda example:

```go
import (
	"context"
	"fmt"

	awsSDK "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	lambdatypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/hlfshell/scaffold-toolbox/aws"
)
```

### SQS and SNS

Resource-aware options store useful identifiers after startup. `QueueURL`
and `TopicARN` make it easy to wire tests or app config.

```go
cloud, err := aws.NewStack("cloud", "latest",
	aws.WithSQS("jobs"),
	aws.WithSNS("events"),
)
if err != nil {
	return err
}
if err := cloud.Create(ctx); err != nil {
	return err
}

sqsClient, err := cloud.SQSClient(ctx)
if err != nil {
	return err
}
queueURL, ok := cloud.QueueURL("jobs")
if !ok {
	return fmt.Errorf("queue was not created")
}

_, err = sqsClient.SendMessage(ctx, &sqs.SendMessageInput{
	QueueUrl:    awsSDK.String(queueURL),
	MessageBody: awsSDK.String("work item"),
})
if err != nil {
	return err
}
```
