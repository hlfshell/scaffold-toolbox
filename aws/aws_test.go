package aws

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/hlfshell/scaffold"
	scaffoldcontainer "github.com/hlfshell/scaffold/container"
)

func TestAWSStackCreateResourcesCleanup(t *testing.T) {
	if !scaffoldcontainer.DockerAvailable() {
		t.Skip("docker is not available")
	}

	ctx := context.Background()
	stack, err := NewStack("scaffold-test-aws", "latest",
		WithS3("scaffold-test-bucket"),
		WithSQS("jobs"),
		WithSNS("events"),
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
	if _, ok := stack.TopicARN("events"); !ok {
		t.Fatal("expected topic ARN for events")
	}
}

func TestAWSServiceOptionsConfigureMiniStackServices(t *testing.T) {
	stack, err := NewStack("scaffold-test-aws-options", "latest",
		WithS3("documents"),
		WithSQS("jobs"),
		WithSNS("events"),
		WithDynamoDB(DynamoDBTable{Name: "items"}),
		WithSecretsManager(Secret{Name: "api-key", Value: "secret"}),
		WithSSM(Parameter{Name: "/app/url", Value: "http://localhost"}),
		WithKinesis(KinesisStream{Name: "events"}),
		WithEventBridge(EventBus{Name: "app"}),
		WithLambda(),
		WithECS(),
		WithServices("s3", "lambda"),
	)
	if err != nil {
		t.Fatal(err)
	}

	services := stack.env["SERVICES"]
	for _, service := range []string{
		"s3",
		"sqs",
		"sns",
		"dynamodb",
		"secretsmanager",
		"ssm",
		"kinesis",
		"events",
		"lambda",
		"ecs",
	} {
		if !containsService(services, service) {
			t.Fatalf("expected SERVICES to contain %s, got %s", service, services)
		}
	}
	if countService(services, "s3") != 1 {
		t.Fatalf("expected s3 once in SERVICES, got %s", services)
	}
}

func TestAWSWithAllEnablesKnownMiniStackServices(t *testing.T) {
	stack, err := NewStack("scaffold-test-aws-all", "latest", WithAll())
	if err != nil {
		t.Fatal(err)
	}

	services := stack.env["SERVICES"]
	for _, service := range allServices {
		if !containsService(services, service) {
			t.Fatalf("expected SERVICES to contain %s, got %s", service, services)
		}
	}
}

func TestAWSECSOptionsEnableDockerRegistryAndServices(t *testing.T) {
	stack, err := NewStack("scaffold-test-aws-ecs", "latest",
		WithECSService(ECSService{
			Name:   "api",
			Family: "api",
			Containers: []ECSContainer{{
				Name:       "api",
				Dockerfile: "./Dockerfile",
				Image:      "app/api:dev",
			}},
		}),
	)
	if err != nil {
		t.Fatal(err)
	}

	if !stack.docker {
		t.Fatal("expected docker socket to be enabled for ECS service")
	}
	if !stack.registryConfig.enabled {
		t.Fatal("expected registry to be enabled for Dockerfile-backed ECS image")
	}
	if stack.env["DOCKER_NETWORK"] == "" {
		t.Fatal("expected generated docker network to be passed to MiniStack")
	}
	if !containsService(stack.env["SERVICES"], "ecs") {
		t.Fatalf("expected SERVICES to contain ecs, got %s", stack.env["SERVICES"])
	}
}

func TestAWSECSServiceDoesNotDisableExplicitRegistry(t *testing.T) {
	stack, err := NewStack("scaffold-test-aws-ecs-registry", "latest",
		WithRegistry("5005"),
		WithECSService(ECSService{
			Name: "api",
			Containers: []ECSContainer{{
				Name:  "api",
				Image: "nginx:alpine",
			}},
		}),
	)
	if err != nil {
		t.Fatal(err)
	}

	if !stack.registryConfig.enabled {
		t.Fatal("expected explicit registry to remain enabled")
	}
	if stack.networkName == "" || !stack.createNetwork {
		t.Fatalf("expected generated registry network, got name=%q create=%v", stack.networkName, stack.createNetwork)
	}
}

func TestAWSRegistryImageUsesHostReachableAddress(t *testing.T) {
	stack, err := NewStack("scaffold-test-aws-registry", "latest", WithRegistry("5005"))
	if err != nil {
		t.Fatal(err)
	}
	stack.registryPort = "5005"

	if got := stack.RegistryImage("docker.io/library/nginx:alpine"); got != "127.0.0.1:5005/library/nginx:alpine" {
		t.Fatalf("unexpected registry image: %s", got)
	}
}

func TestAWSConnectionConfig(t *testing.T) {
	stack, err := NewStack("cloud", "latest", WithSQS("jobs"), WithSNS("events"))
	if err != nil {
		t.Fatal(err)
	}
	stack.port = "4566"
	stack.queueURLs["jobs"] = "http://127.0.0.1:4566/000000000000/jobs"
	stack.topicARNs["events"] = "arn:aws:sns:us-east-1:000000000000:events"

	host := stack.HostConnection()
	if host.EndpointURL != "http://127.0.0.1:4566" {
		t.Fatalf("unexpected host endpoint: %s", host.EndpointURL)
	}
	if !host.S3ForcePathStyle {
		t.Fatal("expected path-style S3 by default")
	}

	container := stack.ContainerConnection()
	if container.EndpointURL != "http://cloud-ministack:4566" {
		t.Fatalf("unexpected container endpoint: %s", container.EndpointURL)
	}

	hostEnv := stack.HostEnv()
	if hostEnv["SQS_JOBS_URL"] != "http://127.0.0.1:4566/000000000000/jobs" {
		t.Fatalf("expected host queue URL to use host endpoint, got %s", hostEnv["SQS_JOBS_URL"])
	}

	env := stack.ContainerEnv()
	expected := map[string]string{
		"AWS_ENDPOINT_URL":        "http://cloud-ministack:4566",
		"AWS_REGION":              "us-east-1",
		"AWS_DEFAULT_REGION":      "us-east-1",
		"AWS_ACCESS_KEY_ID":       "test",
		"AWS_SECRET_ACCESS_KEY":   "test",
		"AWS_S3_FORCE_PATH_STYLE": "true",
		"S3_FORCE_PATH_STYLE":     "true",
		"SQS_JOBS_URL":            "http://cloud-ministack:4566/000000000000/jobs",
		"SNS_EVENTS_ARN":          "arn:aws:sns:us-east-1:000000000000:events",
	}
	for key, value := range expected {
		if env[key] != value {
			t.Fatalf("expected %s=%s, got %s", key, value, env[key])
		}
	}
}

func TestAWSConnectionConfigUsesPrefixedContainerName(t *testing.T) {
	stack, err := NewStack("cloud", "latest", WithRegistry(""))
	if err != nil {
		t.Fatal(err)
	}

	stack.SetNamePrefix("dev-app")

	if stack.InternalEndpointURL() != "http://dev-app-cloud-ministack:4566" {
		t.Fatalf("unexpected internal endpoint: %s", stack.InternalEndpointURL())
	}
	if stack.registry.Name() != "dev-app-cloud-registry" {
		t.Fatalf("unexpected registry name: %s", stack.registry.Name())
	}
}

func TestAWSECSContainerDefinition(t *testing.T) {
	stack, err := NewStack("scaffold-test-aws-container-definition", "latest")
	if err != nil {
		t.Fatal(err)
	}

	definition, err := stack.ecsContainerDefinition(context.Background(), ECSContainer{
		Name:    "api",
		Image:   "nginx:alpine",
		Command: []string{"nginx", "-g", "daemon off;"},
		Env:     map[string]string{"ENV": "local"},
		Ports: []ECSPort{{
			ContainerPort: 8080,
			HostPort:      18080,
			Protocol:      string(ecstypes.TransportProtocolTcp),
		}},
		Memory: 128,
	})
	if err != nil {
		t.Fatal(err)
	}

	if *definition.Name != "api" || *definition.Image != "nginx:alpine" {
		t.Fatalf("unexpected container definition: %#v", definition)
	}
	if len(definition.Environment) != 1 || *definition.Environment[0].Name != "ENV" || *definition.Environment[0].Value != "local" {
		t.Fatalf("unexpected environment: %#v", definition.Environment)
	}
	if len(definition.PortMappings) != 1 || *definition.PortMappings[0].ContainerPort != 8080 || *definition.PortMappings[0].HostPort != 18080 {
		t.Fatalf("unexpected ports: %#v", definition.PortMappings)
	}
	if definition.Memory == nil || *definition.Memory != 128 {
		t.Fatalf("unexpected memory: %#v", definition.Memory)
	}
}

func TestAWSECSRunTaskIntegration(t *testing.T) {
	if !scaffoldcontainer.DockerAvailable() {
		t.Skip("docker is not available")
	}

	ctx := context.Background()
	dir := t.TempDir()
	dockerfile := filepath.Join(dir, "Dockerfile")
	if err := os.WriteFile(dockerfile, []byte(`FROM busybox:1.36
CMD ["sh", "-c", "sleep 60"]
`), 0o600); err != nil {
		t.Fatal(err)
	}

	stack, err := NewStack("scaffold-test-aws-ecs-run", "latest",
		WithECSCluster("app"),
		WithECSRunTask(ECSRunTask{
			Name:    "worker",
			Cluster: "app",
			Family:  "worker",
			Containers: []ECSContainer{{
				Name:       "worker",
				Dockerfile: dockerfile,
				Image:      "app/worker:dev",
			}},
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := stack.Create(ctx); err != nil {
		t.Fatal(err)
	}
	defer stack.Cleanup(context.WithoutCancel(ctx))

	client, err := stack.ECSClient(ctx)
	if err != nil {
		t.Fatal(err)
	}

	err = scaffold.WaitFunc(ctx, time.Minute, time.Second, func(ctx context.Context) error {
		output, err := client.ListTasks(ctx, &ecs.ListTasksInput{
			Cluster: awssdk.String("app"),
		})
		if err != nil {
			return err
		}
		if len(output.TaskArns) == 0 {
			return errNoECSTask
		}

		describe, err := client.DescribeTasks(ctx, &ecs.DescribeTasksInput{
			Cluster: awssdk.String("app"),
			Tasks:   output.TaskArns,
		})
		if err != nil {
			return err
		}
		for _, task := range describe.Tasks {
			for _, container := range task.Containers {
				if container.Image != nil && strings.HasPrefix(*container.Image, stack.RegistryAddress()+"/") {
					return nil
				}
			}
		}

		return errNoRegistryTaskImage
	})
	if err != nil {
		t.Fatal(err)
	}
}

var (
	errNoECSTask           = errString("no ecs task found")
	errNoRegistryTaskImage = errString("no ecs task is using the local registry image")
)

type errString string

func (e errString) Error() string {
	return string(e)
}

func containsService(services string, service string) bool {
	return countService(services, service) > 0
}

func countService(services string, service string) int {
	count := 0
	current := ""
	for _, char := range services {
		if char == ',' {
			if current == service {
				count++
			}
			current = ""
			continue
		}
		current += string(char)
	}
	if current == service {
		count++
	}

	return count
}
