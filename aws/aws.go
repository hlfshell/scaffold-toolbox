package aws

import (
	"context"
	"fmt"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
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
	container      *scaffoldcontainer.Container
	registry       *scaffoldcontainer.Container
	name           string
	region         string
	accessKey      string
	secretKey      string
	port           string
	registryPort   string
	docker         bool
	networkName    string
	createNetwork  bool
	networkCreated bool
	env            map[string]string
	buckets        []string
	queues         []string
	topics         []string
	tables         []DynamoDBTable
	secrets        []Secret
	params         []Parameter
	streams        []KinesisStream
	buses          []EventBus
	images         []Image
	ecsClusters    []string
	ecsServices    []ECSService
	ecsTasks       []ECSRunTask
	ecsTaskARNs    []ecsStartedTask
	registryConfig registryConfig
	services       []string
	queueURLs      map[string]string
	topicARNs      map[string]string
}

type registryConfig struct {
	enabled  bool
	hostPort string
	image    string
	tag      string
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
			stack.networkName = network
			stack.env["DOCKER_NETWORK"] = network
		}
	}
}

/*
WithS3Bucket registers an S3 bucket to create after MiniStack is ready.
*/
func WithS3Bucket(bucket string) Option {
	return func(stack *Stack) {
		stack.addService("s3")
		stack.buckets = append(stack.buckets, bucket)
	}
}

/*
WithSQSQueue registers an SQS queue to create after MiniStack is ready.
*/
func WithSQSQueue(queue string) Option {
	return func(stack *Stack) {
		stack.addService("sqs")
		stack.queues = append(stack.queues, queue)
	}
}

/*
WithSNSTopic registers an SNS topic to create after MiniStack is ready.
*/
func WithSNSTopic(topic string) Option {
	return func(stack *Stack) {
		stack.addService("sns")
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
		registryConfig: registryConfig{
			image: "registry",
			tag:   "2",
		},
	}
	for _, option := range options {
		option(stack)
	}
	if stack.registryConfig.enabled && stack.networkName == "" {
		stack.networkName = name + "-network"
		stack.createNetwork = true
	}
	if stack.networkName != "" {
		stack.env["DOCKER_NETWORK"] = stack.networkName
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
	if stack.networkName != "" {
		stack.container.SetNetwork(stack.networkName)
	}
	if stack.registryConfig.enabled {
		registry, err := scaffoldcontainer.NewContainer(
			stack.registryContainerName(),
			stack.registryConfig.image,
			scaffoldcontainer.WithTag(stack.registryConfig.tag),
			scaffoldcontainer.WithPort("5000", stack.registryConfig.hostPort),
		)
		if err != nil {
			return nil, err
		}
		stack.registry = registry
		if stack.networkName != "" {
			stack.registry.SetNetwork(stack.networkName)
		}
	}

	return stack, nil
}

func (s *Stack) Name() string {
	return s.name
}

/*
SetNetwork attaches MiniStack to a shared Docker network.
*/
func (s *Stack) SetNetwork(name string) {
	s.networkName = name
	s.createNetwork = false
	s.container.SetNetwork(name)
	if s.registry != nil {
		s.registry.SetNetwork(name)
	}
}

/*
SetLabels merges inherited Docker labels onto MiniStack resources.
*/
func (s *Stack) SetLabels(labels map[string]string) {
	s.container.SetLabels(labels)
	if s.registry != nil {
		s.registry.SetLabels(labels)
	}
}

/*
SetNamePrefix prefixes the MiniStack Docker container name.
*/
func (s *Stack) SetNamePrefix(prefix string) {
	s.container.SetNamePrefix(prefix)
	if s.registry != nil {
		s.registry.SetNamePrefix(prefix)
	}
}

/*
Create starts MiniStack and creates the configured AWS resources.
*/
func (s *Stack) Create(ctx context.Context) error {
	if s.createNetwork {
		created, err := scaffoldcontainer.CreateNetwork(ctx, s.networkName, map[string]string{})
		if err != nil {
			return err
		}
		s.networkCreated = created
	}
	if s.networkName != "" {
		s.container.SetNetwork(s.networkName)
		if s.registry != nil {
			s.registry.SetNetwork(s.networkName)
		}
	}
	if err := s.startRegistry(ctx); err != nil {
		s.cleanupAfterCreateFailure(ctx)
		return err
	}
	if err := s.preloadImages(ctx); err != nil {
		s.cleanupAfterCreateFailure(ctx)
		return err
	}

	err := s.container.Start(ctx)
	if err != nil {
		s.cleanupAfterCreateFailure(ctx)
		return fmt.Errorf("failed to start ministack container: %w", err)
	}

	ports := s.container.GetPorts()
	s.port = ports["4566"]

	err = scaffold.WaitForHTTP(ctx, s.EndpointURL()+"/_ministack/health", 200, 30*time.Second)
	if err != nil {
		s.cleanupAfterCreateFailure(ctx)
		return fmt.Errorf("ministack failed to become ready: %w", err)
	}

	err = s.createResources(ctx)
	if err != nil {
		s.cleanupAfterCreateFailure(ctx)
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
DynamoDBClient returns a DynamoDB client configured for MiniStack.
*/
func (s *Stack) DynamoDBClient(ctx context.Context) (*dynamodb.Client, error) {
	config, err := s.AWSConfig(ctx)
	if err != nil {
		return nil, err
	}

	return dynamodb.NewFromConfig(config), nil
}

/*
SecretsManagerClient returns a Secrets Manager client configured for MiniStack.
*/
func (s *Stack) SecretsManagerClient(ctx context.Context) (*secretsmanager.Client, error) {
	config, err := s.AWSConfig(ctx)
	if err != nil {
		return nil, err
	}

	return secretsmanager.NewFromConfig(config), nil
}

/*
SSMClient returns an SSM client configured for MiniStack.
*/
func (s *Stack) SSMClient(ctx context.Context) (*ssm.Client, error) {
	config, err := s.AWSConfig(ctx)
	if err != nil {
		return nil, err
	}

	return ssm.NewFromConfig(config), nil
}

/*
KinesisClient returns a Kinesis client configured for MiniStack.
*/
func (s *Stack) KinesisClient(ctx context.Context) (*kinesis.Client, error) {
	config, err := s.AWSConfig(ctx)
	if err != nil {
		return nil, err
	}

	return kinesis.NewFromConfig(config), nil
}

/*
EventBridgeClient returns an EventBridge client configured for MiniStack.
*/
func (s *Stack) EventBridgeClient(ctx context.Context) (*eventbridge.Client, error) {
	config, err := s.AWSConfig(ctx)
	if err != nil {
		return nil, err
	}

	return eventbridge.NewFromConfig(config), nil
}

/*
ECSClient returns an ECS client configured for MiniStack.
*/
func (s *Stack) ECSClient(ctx context.Context) (*ecs.Client, error) {
	config, err := s.AWSConfig(ctx)
	if err != nil {
		return nil, err
	}

	return ecs.NewFromConfig(config), nil
}

/*
LambdaClient returns a Lambda client configured for MiniStack.
*/
func (s *Stack) LambdaClient(ctx context.Context) (*lambda.Client, error) {
	config, err := s.AWSConfig(ctx)
	if err != nil {
		return nil, err
	}

	return lambda.NewFromConfig(config), nil
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
	return s.HostEnv()
}

func (s *Stack) Endpoints() map[string]string {
	endpoints := map[string]string{
		s.name: s.EndpointURL(),
	}
	if s.RegistryAddress() != "" {
		endpoints[s.name+"-registry"] = s.RegistryAddress()
	}

	return endpoints
}

/*
Cleanup removes the MiniStack container.
*/
func (s *Stack) Cleanup(ctx context.Context) error {
	var firstErr error
	if err := s.cleanupECSResources(ctx); err != nil {
		firstErr = err
	}
	if err := s.container.Cleanup(ctx); err != nil && firstErr == nil {
		firstErr = err
	}
	if err := s.cleanupRegistry(ctx); err != nil && firstErr == nil {
		firstErr = err
	}
	if s.networkCreated {
		if err := scaffoldcontainer.RemoveNetwork(ctx, s.networkName); err != nil && firstErr == nil {
			firstErr = err
		}
		s.networkCreated = false
	}

	return firstErr
}

/*
Logs returns MiniStack logs keyed by the AWS stack name.
*/
func (s *Stack) Logs(ctx context.Context) (logs.LogStreams, error) {
	stream, err := s.container.Logs(ctx)
	if err != nil {
		return nil, err
	}

	streams := logs.LogStreams{s.name: stream}
	if s.registry != nil {
		registryStream, err := s.registry.Logs(ctx)
		if err != nil {
			_ = streams.Close()
			return nil, err
		}
		streams[s.name+"-registry"] = registryStream
	}

	return streams, nil
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

	if err := s.createDynamoDBTables(ctx); err != nil {
		return err
	}
	if err := s.createSecrets(ctx); err != nil {
		return err
	}
	if err := s.createParameters(ctx); err != nil {
		return err
	}
	if err := s.createKinesisStreams(ctx); err != nil {
		return err
	}
	if err := s.createEventBuses(ctx); err != nil {
		return err
	}
	if err := s.createECSResources(ctx); err != nil {
		return err
	}

	return nil
}

func (s *Stack) configureServices() {
	services := append([]string{}, s.services...)

	services = uniqueServices(services)
	if len(services) > 0 {
		s.env["SERVICES"] = joinServices(services)
	}
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

func (s *Stack) registryContainerName() string {
	return s.name + "-registry"
}

func (s *Stack) cleanupAfterCreateFailure(ctx context.Context) {
	ctx = context.WithoutCancel(ctx)
	_ = s.cleanupECSResources(ctx)
	_ = s.container.Cleanup(ctx)
	_ = s.cleanupRegistry(ctx)
	if s.networkCreated {
		_ = scaffoldcontainer.RemoveNetwork(ctx, s.networkName)
		s.networkCreated = false
	}
}

var _ scaffold.Service = (*Stack)(nil)
