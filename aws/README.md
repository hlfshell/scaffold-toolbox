# scaffold toolbox aws

MiniStack-backed local AWS service for scaffold. It starts a single local
AWS-compatible endpoint on port 4566, exposes SDK configuration helpers,
and can create common resources such as S3 buckets, SQS queues, and SNS
topics.

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
	aws.WithServices("s3", "sqs"),
	aws.WithS3Bucket("documents"),
	aws.WithSQSQueue("jobs"),
)
if err != nil {
	return err
}

stack := scaffold.NewStack("app", scaffold.WithServices(cloud))
```

If `WithServices` is omitted, the AWS stack enables services implied by
requested resources. For example, `WithS3Bucket` enables `s3`, `WithSQSQueue`
enables `sqs`, and `WithSNSTopic` enables `sns`.

Use `WithDockerSocket` when you want MiniStack features that create real
backing containers, such as RDS, ElastiCache, ECS, or Docker-backed Lambda.
Use `WithDockerNetwork("network-name")` when those backing containers need
to be reachable from other scaffold services on a shared Docker network.

`WithEnv` passes MiniStack configuration flags through directly, such as
`PERSIST_STATE`, service-specific dataplane flags, or logging options.
