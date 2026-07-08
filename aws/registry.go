package aws

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	imgtypes "github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
	"github.com/hlfshell/scaffold"
	scaffoldcontainer "github.com/hlfshell/scaffold/container"
)

/*
Image describes a container image that should be made available through
the local registry before MiniStack ECS resources are created.
*/
type Image struct {
	LocalImage string
	Dockerfile string
	ECSImage   string
}

/*
PushedImage describes an image pushed to the local registry. HostImage is
the image reference ECS task definitions should use because MiniStack asks
the host Docker daemon to run ECS containers.
*/
type PushedImage struct {
	HostImage string
}

/*
WithRegistry starts a local Docker registry for ECS task images. hostPort
may be blank to let Docker assign a free host port.
*/
func WithRegistry(hostPort string) Option {
	return func(stack *Stack) {
		stack.registryConfig.enabled = true
		stack.registryConfig.hostPort = hostPort
		stack.docker = true
	}
}

/*
WithRegistryImage changes the local registry container image.
*/
func WithRegistryImage(image string, tag string) Option {
	return func(stack *Stack) {
		stack.registryConfig.enabled = true
		stack.docker = true
		if image != "" {
			stack.registryConfig.image = image
		}
		if tag != "" {
			stack.registryConfig.tag = tag
		}
	}
}

/*
WithLocalImage tags and pushes an existing local Docker image into the
local registry before ECS resources are created.
*/
func WithLocalImage(localImage string, ecsImage string) Option {
	return func(stack *Stack) {
		stack.registryConfig.enabled = true
		stack.docker = true
		stack.images = append(stack.images, Image{
			LocalImage: localImage,
			ECSImage:   ecsImage,
		})
	}
}

/*
WithDockerfileImage builds a Dockerfile and pushes the result into the
local registry before ECS resources are created.
*/
func WithDockerfileImage(dockerfile string, ecsImage string) Option {
	return func(stack *Stack) {
		stack.registryConfig.enabled = true
		stack.docker = true
		stack.images = append(stack.images, Image{
			Dockerfile: dockerfile,
			ECSImage:   ecsImage,
		})
	}
}

/*
RegistryAddress returns the host-reachable registry address.
*/
func (s *Stack) RegistryAddress() string {
	if s.registryPort == "" {
		return ""
	}

	return "127.0.0.1:" + s.registryPort
}

/*
RegistryImage returns the image reference ECS task definitions should use.
*/
func (s *Stack) RegistryImage(image string) string {
	if s.RegistryAddress() == "" || image == "" {
		return image
	}

	return s.RegistryAddress() + "/" + stripRegistry(image)
}

/*
RegistryDockerConfigJSON returns a Docker config.json payload for the
host-reachable registry. The default registry has no authentication.
*/
func (s *Stack) RegistryDockerConfigJSON() ([]byte, error) {
	address := s.RegistryAddress()
	if address == "" {
		return nil, fmt.Errorf("registry is not running")
	}

	payload := map[string]any{
		"auths": map[string]any{
			address: map[string]string{
				"auth": "",
			},
		},
	}

	return json.MarshalIndent(payload, "", "  ")
}

/*
RegistryEnv returns environment variables useful for CLI commands that
build, tag, push, or register ECS task images.
*/
func (s *Stack) RegistryEnv() map[string]string {
	if s.RegistryAddress() == "" {
		return map[string]string{}
	}

	return map[string]string{
		"AWS_ECS_REGISTRY": s.RegistryAddress(),
	}
}

/*
PushImage tags a local image, pushes it to the registry, and returns the
image reference ECS task definitions should use.
*/
func (s *Stack) PushImage(ctx context.Context, localImage string, ecsImage string) (PushedImage, error) {
	if s.RegistryAddress() == "" {
		return PushedImage{}, fmt.Errorf("registry is not running")
	}
	if localImage == "" {
		return PushedImage{}, fmt.Errorf("local image is required")
	}

	return s.pushPreparedImage(ctx, localImage, ecsImage)
}

/*
BuildAndPushImage builds a Dockerfile, pushes the result to the registry,
and returns the image reference ECS task definitions should use.
*/
func (s *Stack) BuildAndPushImage(ctx context.Context, dockerfile string, ecsImage string) (PushedImage, string, error) {
	if s.RegistryAddress() == "" {
		return PushedImage{}, "", fmt.Errorf("registry is not running")
	}
	if dockerfile == "" {
		return PushedImage{}, "", fmt.Errorf("dockerfile is required")
	}

	imageID, buildLogs, err := scaffoldcontainer.BuildDockerfile(ctx, dockerfile)
	if err != nil {
		return PushedImage{}, buildLogs, err
	}

	pushed, err := s.pushPreparedImage(ctx, imageID, ecsImage)
	return pushed, buildLogs, err
}

func (s *Stack) startRegistry(ctx context.Context) error {
	if s.registry == nil {
		return nil
	}

	if s.networkName != "" {
		s.registry.SetNetwork(s.networkName)
	}

	if err := s.registry.Start(ctx); err != nil {
		return fmt.Errorf("failed to start registry container: %w", err)
	}

	ports := s.registry.GetPorts()
	s.registryPort = ports["5000"]

	if err := scaffold.WaitForHTTP(ctx, "http://"+s.RegistryAddress()+"/v2/", http.StatusOK, 30*time.Second); err != nil {
		return fmt.Errorf("registry failed to become ready: %w", err)
	}

	return nil
}

func (s *Stack) preloadImages(ctx context.Context) error {
	for _, image := range s.images {
		if image.Dockerfile != "" {
			if _, _, err := s.BuildAndPushImage(ctx, image.Dockerfile, image.ECSImage); err != nil {
				return err
			}
			continue
		}

		if _, err := s.PushImage(ctx, image.LocalImage, image.ECSImage); err != nil {
			return err
		}
	}

	return nil
}

func (s *Stack) cleanupRegistry(ctx context.Context) error {
	if s.registry != nil {
		if err := s.registry.Cleanup(ctx); err != nil {
			return err
		}
	}
	s.registryPort = ""

	return nil
}

func (s *Stack) pushPreparedImage(ctx context.Context, sourceImage string, ecsImage string) (PushedImage, error) {
	if ecsImage == "" {
		ecsImage = sourceImage
	}

	hostImage := s.RegistryImage(ecsImage)

	client, err := scaffoldcontainer.NewClient(ctx)
	if err != nil {
		return PushedImage{}, err
	}
	defer client.Close()

	if err := client.ImageTag(ctx, sourceImage, hostImage); err != nil {
		return PushedImage{}, err
	}

	auth, err := json.Marshal(registry.AuthConfig{})
	if err != nil {
		return PushedImage{}, err
	}

	stream, err := client.ImagePush(ctx, hostImage, imgtypes.PushOptions{
		RegistryAuth: base64.URLEncoding.EncodeToString(auth),
	})
	if err != nil {
		return PushedImage{}, err
	}
	defer stream.Close()

	if err := readDockerPushOutput(stream); err != nil {
		return PushedImage{}, err
	}

	return PushedImage{HostImage: hostImage}, nil
}

type dockerPushMessage struct {
	Status      string `json:"status"`
	Error       string `json:"error"`
	ErrorDetail struct {
		Message string `json:"message"`
	} `json:"errorDetail"`
}

func readDockerPushOutput(reader io.Reader) error {
	decoder := json.NewDecoder(reader)
	for {
		var message dockerPushMessage
		if err := decoder.Decode(&message); err != nil {
			if err == io.EOF {
				return nil
			}

			return fmt.Errorf("failed to decode docker push output: %w", err)
		}
		if message.Error != "" {
			return fmt.Errorf("docker push failed: %s", message.Error)
		}
		if message.ErrorDetail.Message != "" {
			return fmt.Errorf("docker push failed: %s", message.ErrorDetail.Message)
		}
	}
}

func stripRegistry(image string) string {
	image = strings.TrimSpace(image)
	parts := strings.SplitN(image, "/", 2)
	if len(parts) != 2 {
		return image
	}
	if strings.Contains(parts[0], ".") || strings.Contains(parts[0], ":") || parts[0] == "localhost" {
		return parts[1]
	}

	return image
}
