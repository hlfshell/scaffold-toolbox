package kubernetes

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	imgtypes "github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
	"github.com/hlfshell/scaffold"
	scaffoldcontainer "github.com/hlfshell/scaffold/container"
)

/*
Image describes a container image that should be made available through
the cluster registry.
*/
type Image struct {
	LocalImage   string
	Dockerfile   string
	ClusterImage string
}

/*
PushedImage describes an image pushed to the local registry. HostImage is
the image reference used by the host Docker daemon. ClusterImage is the
image reference Kubernetes manifests should use.
*/
type PushedImage struct {
	HostImage    string
	ClusterImage string
}

/*
RegistryAddress returns the host-reachable registry address.
*/
func (c *Cluster) RegistryAddress() string {
	if c.registryPort == "" {
		return ""
	}

	return "127.0.0.1:" + c.registryPort
}

/*
RegistryInternalAddress returns the registry address reachable from the
cluster's Docker network.
*/
func (c *Cluster) RegistryInternalAddress() string {
	if !c.registryConfig.enabled {
		return ""
	}

	return c.registryContainerName() + ":5000"
}

/*
RegistryImage returns the image reference Kubernetes should use for an
image stored in the cluster registry.
*/
func (c *Cluster) RegistryImage(image string) string {
	if c.RegistryInternalAddress() == "" || image == "" {
		return image
	}

	return c.RegistryInternalAddress() + "/" + stripRegistry(image)
}

/*
RegistryDockerConfigJSON returns a Docker config.json payload for the
host-reachable registry. The default registry has no authentication, so
the auth entry is intentionally empty.
*/
func (c *Cluster) RegistryDockerConfigJSON() ([]byte, error) {
	address := c.RegistryAddress()
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
build, tag, push, or patch manifests with registry images.
*/
func (c *Cluster) RegistryEnv() map[string]string {
	if c.RegistryAddress() == "" {
		return map[string]string{}
	}

	return map[string]string{
		"KUBE_REGISTRY":          c.RegistryAddress(),
		"KUBE_REGISTRY_INTERNAL": c.RegistryInternalAddress(),
	}
}

/*
PushImage tags a local image, pushes it to the host registry endpoint,
and returns the cluster image reference manifests should use.
*/
func (c *Cluster) PushImage(ctx context.Context, localImage string, clusterImage string) (PushedImage, error) {
	if c.RegistryAddress() == "" {
		return PushedImage{}, fmt.Errorf("registry is not running")
	}
	if localImage == "" {
		return PushedImage{}, fmt.Errorf("local image is required")
	}

	return c.pushPreparedImage(ctx, localImage, clusterImage)
}

/*
BuildAndPushImage builds a Dockerfile, pushes the result to the registry,
and returns the cluster image reference manifests should use.
*/
func (c *Cluster) BuildAndPushImage(ctx context.Context, dockerfile string, clusterImage string) (PushedImage, string, error) {
	if c.RegistryAddress() == "" {
		return PushedImage{}, "", fmt.Errorf("registry is not running")
	}
	if dockerfile == "" {
		return PushedImage{}, "", fmt.Errorf("dockerfile is required")
	}

	imageID, logs, err := scaffoldcontainer.BuildDockerfile(ctx, dockerfile)
	if err != nil {
		return PushedImage{}, logs, err
	}

	pushed, err := c.pushPreparedImage(ctx, imageID, clusterImage)
	return pushed, logs, err
}

func (c *Cluster) startRegistry(ctx context.Context) error {
	if c.registry == nil {
		return nil
	}

	if c.networkName != "" {
		c.registry.SetNetwork(c.networkName)
	}
	c.registry.SetLabels(c.labels)

	if err := c.registry.Start(ctx); err != nil {
		return fmt.Errorf("failed to start registry container: %w", err)
	}

	ports := c.registry.GetPorts()
	c.registryPort = ports["5000"]

	if err := scaffold.WaitForHTTP(ctx, "http://"+c.RegistryAddress()+"/v2/", http.StatusOK, 30*time.Second); err != nil {
		return fmt.Errorf("registry failed to become ready: %w", err)
	}

	return nil
}

func (c *Cluster) preloadImages(ctx context.Context) error {
	for _, image := range c.images {
		if image.Dockerfile != "" {
			if _, _, err := c.BuildAndPushImage(ctx, image.Dockerfile, image.ClusterImage); err != nil {
				return err
			}
			continue
		}

		if _, err := c.PushImage(ctx, image.LocalImage, image.ClusterImage); err != nil {
			return err
		}
	}

	return nil
}

func (c *Cluster) cleanupRegistry(ctx context.Context) error {
	var firstErr error
	if c.registry != nil {
		if err := c.registry.Cleanup(ctx); err != nil {
			firstErr = err
		}
	}
	if c.registryTempDir != "" {
		if err := os.RemoveAll(c.registryTempDir); err != nil && firstErr == nil {
			firstErr = err
		}
		c.registryTempDir = ""
	}
	c.registryPort = ""

	return firstErr
}

func (c *Cluster) prepareRegistryConfig() error {
	tempDir, err := os.MkdirTemp("", c.name+"-registry-*")
	if err != nil {
		return err
	}

	config := fmt.Sprintf(`mirrors:
  "%s":
    endpoint:
      - "http://%s"
`, c.RegistryInternalAddress(), c.RegistryInternalAddress())

	if err := os.WriteFile(filepath.Join(tempDir, "registries.yaml"), []byte(config), 0o600); err != nil {
		_ = os.RemoveAll(tempDir)
		return err
	}

	c.registryTempDir = tempDir
	return nil
}

func (c *Cluster) pushPreparedImage(ctx context.Context, sourceImage string, clusterImage string) (PushedImage, error) {
	if clusterImage == "" {
		clusterImage = sourceImage
	}

	repository := stripRegistry(clusterImage)
	hostImage := c.RegistryAddress() + "/" + repository
	internalImage := c.RegistryImage(repository)

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

	return PushedImage{
		HostImage:    hostImage,
		ClusterImage: internalImage,
	}, nil
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
