# scaffold localstack

Typed LocalStack harness for scaffold. It uses the `localstack/localstack` container image.

```go
local, err := localstack.NewLocalStack("aws", "latest", "s3", "sqs")
err = local.Create(ctx)
defer local.Cleanup(context.WithoutCancel(ctx))

cfg, err := local.AWSConfig(ctx)
```

The harness exposes endpoint URL, region, fake credentials, environment variables, and an AWS SDK v2 config helper.
