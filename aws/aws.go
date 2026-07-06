package aws

import (
	"context"
	"fmt"
	"strings"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/hlfshell/scaffold"
	scaffoldcontainer "github.com/hlfshell/scaffold/container"
	"github.com/hlfshell/scaffold/logs"
)

/*
Stack is a MiniStack-backed AWS development stack. It starts MiniStack,
then creates the requested local AWS resources so application code can
use normal AWS SDK clients without touching real cloud accounts.
*/
type Stack struct {
	container *scaffoldcontainer.Container
	name      string
	region    string
	accessKey string
	secretKey string
	port      string
	docker    bool
	env       map[string]string
	buckets   []string
	queues    []string
	topics    []string
	services  []string
	queueURLs map[string]string
	topicARNs map[string]string
}

// Option configures the AWS stack before it starts.
type Option func(*Stack)

/*
WithRegion sets the AWS region used by generated SDK clients.
*/
func WithRegion(region string) Option {
	return func(stack *Stack) {
		if region != "" {
			stack.region = region
		}
	}
}

/*
WithCredentials sets the fake credentials used by generated SDK clients.
MiniStack accepts dummy credentials, so these are for application config
compatibility.
*/
func WithCredentials(accessKey string, secretKey string) Option {
	return func(stack *Stack) {
		if accessKey != "" {
			stack.accessKey = accessKey
		}
		if secretKey != "" {
			stack.secretKey = secretKey
		}
	}
}

/*
WithEnv adds MiniStack environment variables. Use this for service flags
such as persistence, debug logging, or real sidecar-backed dataplanes.
*/
func WithEnv(env map[string]string) Option {
	return func(stack *Stack) {
		for key, value := range env {
			stack.env[key] = value
		}
	}
}

/*
WithServices limits MiniStack startup to the named AWS services. If this
is not set, Scaffold enables services implied by requested resources,
such as s3 for buckets, sqs for queues, and sns for topics.
*/
func WithServices(services ...string) Option {
	return func(stack *Stack) {
		stack.services = append(stack.services, services...)
	}
}

/*
WithDockerSocket mounts the local Docker socket into MiniStack. This is
needed for MiniStack features that create real backing containers, such
as RDS, ElastiCache, ECS, and Lambda Docker execution.
*/
func WithDockerSocket() Option {
	return func(stack *Stack) {
		stack.docker = true
	}
}

/*
WithDockerNetwork sets the Docker network MiniStack should use for
container-backed AWS services. It also mounts the Docker socket because
MiniStack needs Docker access to create those backing services.
*/
func WithDockerNetwork(network string) Option {
	return func(stack *Stack) {
		stack.docker = true
		if network != "" {
			stack.env["DOCKER_NETWORK"] = network
		}
	}
}

/*
WithS3Bucket registers an S3 bucket to create after MiniStack is ready.
*/
func WithS3Bucket(bucket string) Option {
	return func(stack *Stack) {
		stack.buckets = append(stack.buckets, bucket)
	}
}

/*
WithSQSQueue registers an SQS queue to create after MiniStack is ready.
*/
func WithSQSQueue(queue string) Option {
	return func(stack *Stack) {
		stack.queues = append(stack.queues, queue)
	}
}

/*
WithSNSTopic registers an SNS topic to create after MiniStack is ready.
*/
func WithSNSTopic(topic string) Option {
	return func(stack *Stack) {
		stack.topics = append(stack.topics, topic)
	}
}

/*
NewStack creates a MiniStack-backed AWS service. MiniStack provides a
single local AWS endpoint on port 4566; options define which resources
Scaffold should create after the emulator is ready.
*/
func NewStack(name string, tag string, options ...Option) (*Stack, error) {
	stack := &Stack{
		name:      name,
		region:    "us-east-1",
		accessKey: "test",
		secretKey: "test",
		env:       map[string]string{},
		queueURLs: map[string]string{},
		topicARNs: map[string]string{},
	}
	for _, option := range options {
		option(stack)
	}
	stack.configureServices()

	containerOptions := []scaffoldcontainer.ContainerOption{
		scaffoldcontainer.WithTag(tag),
		scaffoldcontainer.WithPort("4566", ""),
		scaffoldcontainer.WithEnv(stack.env),
	}
	if stack.docker {
		containerOptions = append(containerOptions, scaffoldcontainer.WithBind("/var/run/docker.sock", "/var/run/docker.sock"))
	}

	container, err := scaffoldcontainer.NewContainer(
		name+"-ministack",
		"ministackorg/ministack",
		containerOptions...,
	)
	if err != nil {
		return nil, err
	}
	stack.container = container

	return stack, nil
}

func (s *Stack) Name() string {
	return s.name
}

/*
SetNetwork attaches MiniStack to a shared Docker network.
*/
func (s *Stack) SetNetwork(name string) {
	s.container.SetNetwork(name)
}

/*
SetLabels merges inherited Docker labels onto MiniStack resources.
*/
func (s *Stack) SetLabels(labels map[string]string) {
	s.container.SetLabels(labels)
}

/*
SetNamePrefix prefixes the MiniStack Docker container name.
*/
func (s *Stack) SetNamePrefix(prefix string) {
	s.container.SetNamePrefix(prefix)
}

/*
Create starts MiniStack and creates the configured AWS resources.
*/
func (s *Stack) Create(ctx context.Context) error {
	err := s.container.Start(ctx)
	if err != nil {
		return fmt.Errorf("failed to start ministack container: %w", err)
	}

	ports := s.container.GetPorts()
	s.port = ports["4566"]

	err = scaffold.WaitForHTTP(ctx, s.EndpointURL()+"/_ministack/health", 200, 30*time.Second)
	if err != nil {
		s.container.Cleanup(context.WithoutCancel(ctx))
		return fmt.Errorf("ministack failed to become ready: %w", err)
	}

	err = s.createResources(ctx)
	if err != nil {
		s.container.Cleanup(context.WithoutCancel(ctx))
		return err
	}

	return nil
}

/*
EndpointURL returns the local MiniStack edge endpoint.
*/
func (s *Stack) EndpointURL() string {
	return fmt.Sprintf("http://127.0.0.1:%s", s.port)
}

/*
AWSConfig returns an AWS SDK v2 config routed to MiniStack.
*/
func (s *Stack) AWSConfig(ctx context.Context) (awssdk.Config, error) {
	return config.LoadDefaultConfig(
		ctx,
		config.WithRegion(s.region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(s.accessKey, s.secretKey, "")),
		config.WithEndpointResolverWithOptions(awssdk.EndpointResolverWithOptionsFunc(
			func(service string, region string, options ...any) (awssdk.Endpoint, error) {
				return awssdk.Endpoint{
					URL:           s.EndpointURL(),
					SigningRegion: s.region,
				}, nil
			},
		)),
	)
}

/*
S3Client returns an S3 client configured for MiniStack.
*/
func (s *Stack) S3Client(ctx context.Context) (*s3.Client, error) {
	config, err := s.AWSConfig(ctx)
	if err != nil {
		return nil, err
	}

	return s3.NewFromConfig(config, func(options *s3.Options) {
		options.UsePathStyle = true
	}), nil
}

/*
SQSClient returns an SQS client configured for MiniStack.
*/
func (s *Stack) SQSClient(ctx context.Context) (*sqs.Client, error) {
	config, err := s.AWSConfig(ctx)
	if err != nil {
		return nil, err
	}

	return sqs.NewFromConfig(config), nil
}

/*
SNSClient returns an SNS client configured for MiniStack.
*/
func (s *Stack) SNSClient(ctx context.Context) (*sns.Client, error) {
	config, err := s.AWSConfig(ctx)
	if err != nil {
		return nil, err
	}

	return sns.NewFromConfig(config), nil
}

/*
QueueURL returns the URL for a queue created by the stack.
*/
func (s *Stack) QueueURL(name string) (string, bool) {
	value, ok := s.queueURLs[name]
	return value, ok
}

/*
TopicARN returns the ARN for a topic created by the stack.
*/
func (s *Stack) TopicARN(name string) (string, bool) {
	value, ok := s.topicARNs[name]
	return value, ok
}

func (s *Stack) Env() map[string]string {
	env := map[string]string{
		"AWS_ACCESS_KEY_ID":     s.accessKey,
		"AWS_SECRET_ACCESS_KEY": s.secretKey,
		"AWS_REGION":            s.region,
		"AWS_ENDPOINT_URL":      s.EndpointURL(),
	}
	for name, value := range s.queueURLs {
		env[envKey("SQS", name, "URL")] = value
	}
	for name, value := range s.topicARNs {
		env[envKey("SNS", name, "ARN")] = value
	}

	return env
}

func (s *Stack) Endpoints() map[string]string {
	return map[string]string{
		s.name: s.EndpointURL(),
	}
}

/*
Cleanup removes the MiniStack container.
*/
func (s *Stack) Cleanup(ctx context.Context) error {
	return s.container.Cleanup(ctx)
}

/*
Logs returns MiniStack logs keyed by the AWS stack name.
*/
func (s *Stack) Logs(ctx context.Context) (logs.LogStreams, error) {
	stream, err := s.container.Logs(ctx)
	if err != nil {
		return nil, err
	}

	return logs.LogStreams{s.name: stream}, nil
}

func (s *Stack) createResources(ctx context.Context) error {
	s3Client, err := s.S3Client(ctx)
	if err != nil {
		return err
	}
	for _, bucket := range s.buckets {
		_, err := s3Client.HeadBucket(ctx, &s3.HeadBucketInput{
			Bucket: awssdk.String(bucket),
		})
		if err == nil {
			continue
		}

		_, err = s3Client.CreateBucket(ctx, &s3.CreateBucketInput{
			Bucket: awssdk.String(bucket),
		})
		if err != nil {
			return fmt.Errorf("failed to create s3 bucket %s: %w", bucket, err)
		}
	}

	sqsClient, err := s.SQSClient(ctx)
	if err != nil {
		return err
	}
	for _, queue := range s.queues {
		output, err := sqsClient.CreateQueue(ctx, &sqs.CreateQueueInput{
			QueueName: awssdk.String(queue),
		})
		if err != nil {
			return fmt.Errorf("failed to create sqs queue %s: %w", queue, err)
		}
		if output.QueueUrl != nil {
			s.queueURLs[queue] = *output.QueueUrl
		}
	}

	snsClient, err := s.SNSClient(ctx)
	if err != nil {
		return err
	}
	for _, topic := range s.topics {
		output, err := snsClient.CreateTopic(ctx, &sns.CreateTopicInput{
			Name: awssdk.String(topic),
		})
		if err != nil {
			return fmt.Errorf("failed to create sns topic %s: %w", topic, err)
		}
		if output.TopicArn != nil {
			s.topicARNs[topic] = *output.TopicArn
		}
	}

	return nil
}

func (s *Stack) configureServices() {
	services := append([]string{}, s.services...)
	if len(s.buckets) > 0 {
		services = append(services, "s3")
	}
	if len(s.queues) > 0 {
		services = append(services, "sqs")
	}
	if len(s.topics) > 0 {
		services = append(services, "sns")
	}

	services = uniqueServices(services)
	if len(services) > 0 {
		s.env["SERVICES"] = strings.Join(services, ",")
	}
}

func uniqueServices(services []string) []string {
	seen := map[string]bool{}
	output := []string{}
	for _, service := range services {
		service = strings.TrimSpace(service)
		if service == "" || seen[service] {
			continue
		}
		seen[service] = true
		output = append(output, service)
	}

	return output
}

func envKey(parts ...string) string {
	key := ""
	for _, part := range parts {
		if key != "" {
			key += "_"
		}
		for _, char := range part {
			if char >= 'a' && char <= 'z' {
				key += string(char - 32)
			} else if (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') {
				key += string(char)
			} else {
				key += "_"
			}
		}
	}

	return key
}

var _ scaffold.Service = (*Stack)(nil)
