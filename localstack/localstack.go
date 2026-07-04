package localstack

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/hlfshell/scaffold"
	scaffoldcontainer "github.com/hlfshell/scaffold/container"
	"github.com/hlfshell/scaffold/logs"
)

/*
LocalStack is a typed harness around the localstack/localstack container.
It exposes the edge endpoint and AWS SDK configuration helpers.
*/
type LocalStack struct {
	container *scaffoldcontainer.Container
	name      string
	region    string
	accessKey string
	secretKey string
	port      string
	services  []string
	timeout   time.Duration
}

/*
NewLocalStack creates a LocalStack harness. Optional service names are
passed to LocalStack through the SERVICES environment variable.
*/
func NewLocalStack(name string, tag string, services ...string) (*LocalStack, error) {
	env := map[string]string{
		"DEBUG":          "0",
		"GATEWAY_LISTEN": "0.0.0.0:4566",
	}
	if len(services) > 0 {
		env["SERVICES"] = strings.Join(services, ",")
	}

	container, err := scaffoldcontainer.NewContainer(
		name,
		"localstack/localstack",
		scaffoldcontainer.WithTag(tag),
		scaffoldcontainer.WithPort("4566", ""),
		scaffoldcontainer.WithEnv(env),
	)
	if err != nil {
		return nil, err
	}

	return &LocalStack{
		container: container,
		name:      name,
		region:    "us-east-1",
		accessKey: "test",
		secretKey: "test",
		services:  services,
		timeout:   3 * time.Minute,
	}, nil
}

/*
Name returns the service name used by Scaffold stacks.
*/
func (l *LocalStack) Name() string {
	return l.name
}

/*
SetNetwork attaches the underlying container to a Docker network when it
is created.
*/
func (l *LocalStack) SetNetwork(name string) {
	l.container.SetNetwork(name)
}

/*
SetLabels merges Docker labels onto the underlying container.
*/
func (l *LocalStack) SetLabels(labels map[string]string) {
	l.container.SetLabels(labels)
}

/*
SetNamePrefix prefixes the underlying Docker container name before it is
created.
*/
func (l *LocalStack) SetNamePrefix(prefix string) {
	l.container.SetNamePrefix(prefix)
}

/*
WithReadyTimeout changes how long Create waits for the LocalStack health
endpoint.
*/
func (l *LocalStack) WithReadyTimeout(timeout time.Duration) *LocalStack {
	l.timeout = timeout
	return l
}

/*
Create starts LocalStack with ctx and waits for its health
endpoint to respond.
*/
func (l *LocalStack) Create(ctx context.Context) error {
	err := l.container.Start(ctx)
	if err != nil {
		return fmt.Errorf("failed to start localstack container: %w", err)
	}

	ports := l.container.GetPorts()
	l.port = ports["4566"]

	err = scaffold.WaitForHTTP(ctx, l.EndpointURL()+"/_localstack/health", 200, l.timeout)
	if err != nil {
		l.container.Cleanup(context.WithoutCancel(ctx))
		return fmt.Errorf("localstack failed to become ready: %w", err)
	}

	return nil
}

/*
EndpointURL returns the local LocalStack edge endpoint.
*/
func (l *LocalStack) EndpointURL() string {
	return fmt.Sprintf("http://127.0.0.1:%s", l.port)
}

/*
Region returns the AWS region used by the harness.
*/
func (l *LocalStack) Region() string {
	return l.region
}

/*
AccessKey returns the fake AWS access key used by LocalStack.
*/
func (l *LocalStack) AccessKey() string {
	return l.accessKey
}

/*
SecretKey returns the fake AWS secret key used by LocalStack.
*/
func (l *LocalStack) SecretKey() string {
	return l.secretKey
}

/*
Env returns environment variables that point AWS clients at LocalStack.
*/
func (l *LocalStack) Env() map[string]string {
	return map[string]string{
		"AWS_ACCESS_KEY_ID":     l.accessKey,
		"AWS_SECRET_ACCESS_KEY": l.secretKey,
		"AWS_REGION":            l.region,
		"AWS_ENDPOINT_URL":      l.EndpointURL(),
	}
}

/*
Endpoints returns named LocalStack endpoints.
*/
func (l *LocalStack) Endpoints() map[string]string {
	return map[string]string{
		l.name: l.EndpointURL(),
	}
}

/*
AWSConfig returns an AWS SDK v2 config that resolves all service
endpoints to LocalStack.
*/
func (l *LocalStack) AWSConfig(ctx context.Context) (aws.Config, error) {
	return config.LoadDefaultConfig(
		ctx,
		config.WithRegion(l.region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(l.accessKey, l.secretKey, "")),
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service string, region string, options ...any) (aws.Endpoint, error) {
				return aws.Endpoint{
					URL:           l.EndpointURL(),
					SigningRegion: l.region,
				}, nil
			},
		)),
	)
}

/*
Cleanup removes the LocalStack container.
*/
func (l *LocalStack) Cleanup(ctx context.Context) error {
	return l.container.Cleanup(ctx)
}

/*
Logs returns the LocalStack container logs keyed by service name.
*/
func (l *LocalStack) Logs(ctx context.Context) (logs.LogStreams, error) {
	stream, err := l.container.Logs(ctx)
	if err != nil {
		return nil, err
	}

	return logs.LogStreams{l.name: stream}, nil
}
