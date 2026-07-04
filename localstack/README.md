# scaffold localstack

Typed LocalStack harness for scaffold. It uses the `localstack/localstack` container image.

## Install

```bash
go get github.com/hlfshell/scaffold-toolbox/localstack
```

```go
import "github.com/hlfshell/scaffold-toolbox/localstack"
```

```go
local, err := localstack.NewLocalStack("aws", "3.8.1", "s3", "sqs")
err = local.Create(ctx)
defer local.Cleanup(context.WithoutCancel(ctx))

cfg, err := local.AWSConfig(ctx)
```

Current `latest` LocalStack images may require a LocalStack auth token. Pin a community-compatible tag unless your environment provides `LOCALSTACK_AUTH_TOKEN`.

The harness exposes endpoint URL, region, fake credentials, environment variables, and an AWS SDK v2 config helper.
