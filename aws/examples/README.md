# AWS examples

These examples show app-shaped MiniStack setups using the base `aws` module directly. There is no separate AWS stack package because the AWS toolbox module is already the right abstraction: it starts MiniStack, enables services, creates resources, prepares local ECS images, exposes SDK configuration, and provides clients for follow-up setup.

Keeping these patterns as examples avoids a second AWS API that would have to mirror every MiniStack option. If a helper is generally useful, it should live in `github.com/hlfshell/scaffold-toolbox/aws`; if it is an application shape, it belongs here as a copyable example.

## App stack shape

`app-stack` shows a common local cloud setup:

- S3 bucket for files and seed objects.
- SQS queue for background work.
- ECS/Fargate-style HTTP service from a Dockerfile.
- ECS/Fargate-style one-shot worker task from a Dockerfile.
- Host/container environment helpers for SDK clients.

The important design choice is that the example still uses `aws.NewStack`:

```go
cloud, err := aws.NewStack("cloud", "latest",
	aws.WithS3("documents"),
	aws.WithSQS("jobs"),
	aws.WithECSCluster("app"),
	aws.WithECSService(...),
	aws.WithECSRunTask(...),
)
```

That keeps MiniStack configuration, ECS behavior, image publishing, and SDK helpers in one package.
