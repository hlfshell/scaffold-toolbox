package aws

import (
	"context"
	"fmt"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

const (
	// ECSLaunchTypeEC2 runs the task through MiniStack's Docker-backed EC2 mode.
	ECSLaunchTypeEC2 = "EC2"
	// ECSLaunchTypeFargate marks the task as Fargate-compatible. MiniStack
	// still runs the container through the host Docker daemon.
	ECSLaunchTypeFargate = "FARGATE"
)

/*
ECSPort describes a container port mapping for an ECS container.
*/
type ECSPort struct {
	ContainerPort int32
	HostPort      int32
	Protocol      string
}

/*
ECSContainer describes one container in an ECS task definition. Image may
point at a public image. LocalImage or Dockerfile pushes an image into the
local registry and uses that pushed image in the task definition.
*/
type ECSContainer struct {
	Name       string
	Image      string
	LocalImage string
	Dockerfile string
	Command    []string
	Env        map[string]string
	Ports      []ECSPort
	CPU        int32
	Memory     int32
	Essential  *bool
}

/*
ECSService describes a long-running ECS service backed by MiniStack's
Docker execution.
*/
type ECSService struct {
	Name         string
	Cluster      string
	Family       string
	LaunchType   string
	DesiredCount int32
	CPU          string
	Memory       string
	Containers   []ECSContainer
}

/*
ECSRunTask describes an ECS task to register and run once during stack
startup.
*/
type ECSRunTask struct {
	Name       string
	Cluster    string
	Family     string
	LaunchType string
	Count      int32
	CPU        string
	Memory     string
	Containers []ECSContainer
}

type ecsStartedTask struct {
	Cluster string
	ARN     string
}

/*
WithECSCluster creates an ECS cluster after MiniStack starts.
*/
func WithECSCluster(name string) Option {
	return func(stack *Stack) {
		stack.addService("ecs")
		stack.docker = true
		if name != "" {
			stack.ecsClusters = append(stack.ecsClusters, name)
		}
	}
}

/*
WithECSService registers a task definition and creates a long-running ECS
service. Set LaunchType to ECSLaunchTypeFargate to model a Fargate task.
*/
func WithECSService(service ECSService) Option {
	return func(stack *Stack) {
		stack.addService("ecs")
		stack.docker = true
		if needsRegistry(service.Containers) {
			stack.registryConfig.enabled = true
		}
		stack.ecsServices = append(stack.ecsServices, service)
		if service.Cluster != "" {
			stack.ecsClusters = append(stack.ecsClusters, service.Cluster)
		}
	}
}

/*
WithECSRunTask registers a task definition and runs it once during stack
startup. Set LaunchType to ECSLaunchTypeFargate to model a Fargate task.
*/
func WithECSRunTask(task ECSRunTask) Option {
	return func(stack *Stack) {
		stack.addService("ecs")
		stack.docker = true
		if needsRegistry(task.Containers) {
			stack.registryConfig.enabled = true
		}
		stack.ecsTasks = append(stack.ecsTasks, task)
		if task.Cluster != "" {
			stack.ecsClusters = append(stack.ecsClusters, task.Cluster)
		}
	}
}

func (s *Stack) createECSResources(ctx context.Context) error {
	if len(s.ecsClusters) == 0 && len(s.ecsServices) == 0 && len(s.ecsTasks) == 0 {
		return nil
	}

	client, err := s.ECSClient(ctx)
	if err != nil {
		return err
	}

	for _, cluster := range uniqueServices(s.ecsClusters) {
		if cluster == "" {
			continue
		}
		if _, err := client.CreateCluster(ctx, &ecs.CreateClusterInput{
			ClusterName: awssdk.String(cluster),
		}); err != nil {
			return fmt.Errorf("failed to create ecs cluster %s: %w", cluster, err)
		}
	}

	for _, service := range s.ecsServices {
		if err := s.createECSService(ctx, client, service); err != nil {
			return err
		}
	}
	for _, task := range s.ecsTasks {
		if err := s.runECSTask(ctx, client, task); err != nil {
			return err
		}
	}

	return nil
}

func (s *Stack) createECSService(ctx context.Context, client *ecs.Client, service ECSService) error {
	if service.Name == "" {
		return fmt.Errorf("ecs service name is required")
	}

	cluster := defaultECSCluster(service.Cluster)
	taskDefinitionARN, err := s.registerTaskDefinition(ctx, client, service.Family, service.CPU, service.Memory, service.LaunchType, service.Containers)
	if err != nil {
		return err
	}

	desired := service.DesiredCount
	if desired == 0 {
		desired = 1
	}
	launchType := ecsLaunchType(service.LaunchType)

	_, err = client.CreateService(ctx, &ecs.CreateServiceInput{
		Cluster:        awssdk.String(cluster),
		ServiceName:    awssdk.String(service.Name),
		TaskDefinition: awssdk.String(taskDefinitionARN),
		DesiredCount:   awssdk.Int32(desired),
		LaunchType:     launchType,
	})
	if err != nil {
		return fmt.Errorf("failed to create ecs service %s: %w", service.Name, err)
	}

	return nil
}

func (s *Stack) runECSTask(ctx context.Context, client *ecs.Client, task ECSRunTask) error {
	cluster := defaultECSCluster(task.Cluster)
	taskDefinitionARN, err := s.registerTaskDefinition(ctx, client, task.Family, task.CPU, task.Memory, task.LaunchType, task.Containers)
	if err != nil {
		return err
	}

	count := task.Count
	if count == 0 {
		count = 1
	}
	launchType := ecsLaunchType(task.LaunchType)

	output, err := client.RunTask(ctx, &ecs.RunTaskInput{
		Cluster:        awssdk.String(cluster),
		TaskDefinition: awssdk.String(taskDefinitionARN),
		Count:          awssdk.Int32(count),
		LaunchType:     launchType,
	})
	if err != nil {
		return fmt.Errorf("failed to run ecs task %s: %w", firstNonEmpty(task.Name, task.Family), err)
	}
	for _, ranTask := range output.Tasks {
		if ranTask.TaskArn != nil {
			s.ecsTaskARNs = append(s.ecsTaskARNs, ecsStartedTask{
				Cluster: cluster,
				ARN:     *ranTask.TaskArn,
			})
		}
	}

	return nil
}

func (s *Stack) registerTaskDefinition(ctx context.Context, client *ecs.Client, family string, cpu string, memory string, launchType string, containers []ECSContainer) (string, error) {
	if family == "" {
		family = s.name
	}
	if len(containers) == 0 {
		return "", fmt.Errorf("ecs task definition %s requires at least one container", family)
	}

	containerDefinitions := make([]ecstypes.ContainerDefinition, 0, len(containers))
	for _, container := range containers {
		definition, err := s.ecsContainerDefinition(ctx, container)
		if err != nil {
			return "", err
		}
		containerDefinitions = append(containerDefinitions, definition)
	}

	compatibility := ecsCompatibility(launchType)
	output, err := client.RegisterTaskDefinition(ctx, &ecs.RegisterTaskDefinitionInput{
		Family:                  awssdk.String(family),
		Cpu:                     awssdk.String(defaultString(cpu, "256")),
		Memory:                  awssdk.String(defaultString(memory, "512")),
		RequiresCompatibilities: []ecstypes.Compatibility{compatibility},
		ContainerDefinitions:    containerDefinitions,
	})
	if err != nil {
		return "", fmt.Errorf("failed to register ecs task definition %s: %w", family, err)
	}
	if output.TaskDefinition == nil || output.TaskDefinition.TaskDefinitionArn == nil {
		return "", fmt.Errorf("ecs task definition %s did not return an ARN", family)
	}

	return *output.TaskDefinition.TaskDefinitionArn, nil
}

func (s *Stack) ecsContainerDefinition(ctx context.Context, container ECSContainer) (ecstypes.ContainerDefinition, error) {
	if container.Name == "" {
		return ecstypes.ContainerDefinition{}, fmt.Errorf("ecs container name is required")
	}

	image, err := s.resolveECSImage(ctx, container)
	if err != nil {
		return ecstypes.ContainerDefinition{}, err
	}
	if image == "" {
		return ecstypes.ContainerDefinition{}, fmt.Errorf("ecs container %s requires an image, local image, or dockerfile", container.Name)
	}

	essential := true
	if container.Essential != nil {
		essential = *container.Essential
	}

	definition := ecstypes.ContainerDefinition{
		Name:      awssdk.String(container.Name),
		Image:     awssdk.String(image),
		Command:   container.Command,
		Cpu:       container.CPU,
		Essential: awssdk.Bool(essential),
	}
	if container.Memory != 0 {
		definition.Memory = awssdk.Int32(container.Memory)
	}
	for key, value := range container.Env {
		definition.Environment = append(definition.Environment, ecstypes.KeyValuePair{
			Name:  awssdk.String(key),
			Value: awssdk.String(value),
		})
	}
	for _, port := range container.Ports {
		if port.ContainerPort == 0 {
			return ecstypes.ContainerDefinition{}, fmt.Errorf("ecs container %s has a port mapping without a container port", container.Name)
		}
		protocol := ecstypes.TransportProtocolTcp
		if port.Protocol != "" {
			protocol = ecstypes.TransportProtocol(port.Protocol)
		}
		mapping := ecstypes.PortMapping{
			ContainerPort: awssdk.Int32(port.ContainerPort),
			Protocol:      protocol,
		}
		if port.HostPort != 0 {
			mapping.HostPort = awssdk.Int32(port.HostPort)
		}
		definition.PortMappings = append(definition.PortMappings, mapping)
	}

	return definition, nil
}

func (s *Stack) resolveECSImage(ctx context.Context, container ECSContainer) (string, error) {
	if container.Dockerfile != "" {
		pushed, _, err := s.BuildAndPushImage(ctx, container.Dockerfile, firstNonEmpty(container.Image, container.Name+":latest"))
		if err != nil {
			return "", err
		}
		return pushed.HostImage, nil
	}
	if container.LocalImage != "" {
		pushed, err := s.PushImage(ctx, container.LocalImage, firstNonEmpty(container.Image, container.LocalImage))
		if err != nil {
			return "", err
		}
		return pushed.HostImage, nil
	}

	return container.Image, nil
}

func (s *Stack) cleanupECSResources(ctx context.Context) error {
	if s.port == "" || (len(s.ecsServices) == 0 && len(s.ecsTasks) == 0 && len(s.ecsClusters) == 0) {
		return nil
	}

	client, err := s.ECSClient(ctx)
	if err != nil {
		return err
	}

	var firstErr error
	for i := len(s.ecsServices) - 1; i >= 0; i-- {
		service := s.ecsServices[i]
		if service.Name == "" {
			continue
		}
		_, err := client.DeleteService(ctx, &ecs.DeleteServiceInput{
			Cluster: awssdk.String(defaultECSCluster(service.Cluster)),
			Service: awssdk.String(service.Name),
			Force:   awssdk.Bool(true),
		})
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	for _, taskARN := range s.ecsTaskARNs {
		_, err := client.StopTask(ctx, &ecs.StopTaskInput{
			Cluster: awssdk.String(taskARN.Cluster),
			Task:    awssdk.String(taskARN.ARN),
			Reason:  awssdk.String("scaffold cleanup"),
		})
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	for i := len(s.ecsClusters) - 1; i >= 0; i-- {
		cluster := s.ecsClusters[i]
		if cluster == "" {
			continue
		}
		_, err := client.DeleteCluster(ctx, &ecs.DeleteClusterInput{
			Cluster: awssdk.String(cluster),
		})
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

func needsRegistry(containers []ECSContainer) bool {
	for _, container := range containers {
		if container.Dockerfile != "" || container.LocalImage != "" {
			return true
		}
	}

	return false
}

func defaultECSCluster(cluster string) string {
	if cluster == "" {
		return "default"
	}

	return cluster
}

func ecsLaunchType(launchType string) ecstypes.LaunchType {
	if launchType == ECSLaunchTypeFargate {
		return ecstypes.LaunchTypeFargate
	}

	return ecstypes.LaunchTypeEc2
}

func ecsCompatibility(launchType string) ecstypes.Compatibility {
	if launchType == ECSLaunchTypeFargate {
		return ecstypes.CompatibilityFargate
	}

	return ecstypes.CompatibilityEc2
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}

	return ""
}

func defaultString(value string, fallback string) string {
	if value == "" {
		return fallback
	}

	return value
}
