package aws

import "net/url"

/*
ConnectionConfig describes how an application should connect to the
MiniStack AWS endpoint.
*/
type ConnectionConfig struct {
	EndpointURL      string
	Region           string
	AccessKeyID      string
	SecretAccessKey  string
	S3ForcePathStyle bool
}

/*
Env returns common AWS environment variables for SDKs and application
frameworks. Path-style keys are included because S3-compatible local
endpoints usually cannot serve virtual-hosted bucket names.
*/
func (c ConnectionConfig) Env() map[string]string {
	env := map[string]string{
		"AWS_ACCESS_KEY_ID":     c.AccessKeyID,
		"AWS_SECRET_ACCESS_KEY": c.SecretAccessKey,
		"AWS_REGION":            c.Region,
		"AWS_DEFAULT_REGION":    c.Region,
		"AWS_ENDPOINT_URL":      c.EndpointURL,
	}
	if c.S3ForcePathStyle {
		env["AWS_S3_FORCE_PATH_STYLE"] = "true"
		env["S3_FORCE_PATH_STYLE"] = "true"
	}

	return env
}

/*
HostConnection returns connection settings for applications running on
the host machine.
*/
func (s *Stack) HostConnection() ConnectionConfig {
	return s.connection(s.EndpointURL())
}

/*
ContainerConnection returns connection settings for applications running
in another container on the same Docker network as MiniStack.
*/
func (s *Stack) ContainerConnection() ConnectionConfig {
	return s.connection(s.InternalEndpointURL())
}

/*
HostEnv returns environment variables for applications running on the
host machine.
*/
func (s *Stack) HostEnv() map[string]string {
	return s.envForConnection(s.HostConnection())
}

/*
ContainerEnv returns environment variables for applications running in
another container on the same Docker network as MiniStack.
*/
func (s *Stack) ContainerEnv() map[string]string {
	return s.envForConnection(s.ContainerConnection())
}

/*
InternalEndpointURL returns the Docker-network endpoint for MiniStack.
It is reachable by sibling containers only when they share a user-defined
Docker network with this stack.
*/
func (s *Stack) InternalEndpointURL() string {
	return "http://" + s.container.Name() + ":4566"
}

func (s *Stack) connection(endpoint string) ConnectionConfig {
	return ConnectionConfig{
		EndpointURL:      endpoint,
		Region:           s.region,
		AccessKeyID:      s.accessKey,
		SecretAccessKey:  s.secretKey,
		S3ForcePathStyle: true,
	}
}

func (s *Stack) envForConnection(config ConnectionConfig) map[string]string {
	env := config.Env()
	for key, value := range s.resourceEnv(config) {
		env[key] = value
	}
	for key, value := range s.RegistryEnv() {
		env[key] = value
	}

	return env
}

func (s *Stack) resourceEnv(config ConnectionConfig) map[string]string {
	env := map[string]string{}
	for name, value := range s.queueURLs {
		env[envKey("SQS", name, "URL")] = rewriteEndpointURL(value, config.EndpointURL)
	}
	for name, value := range s.topicARNs {
		env[envKey("SNS", name, "ARN")] = value
	}

	return env
}

func rewriteEndpointURL(value string, endpoint string) string {
	parsedValue, err := url.Parse(value)
	if err != nil {
		return value
	}
	parsedEndpoint, err := url.Parse(endpoint)
	if err != nil {
		return value
	}
	if parsedValue.Scheme == "" || parsedValue.Host == "" || parsedEndpoint.Scheme == "" || parsedEndpoint.Host == "" {
		return value
	}

	parsedValue.Scheme = parsedEndpoint.Scheme
	parsedValue.Host = parsedEndpoint.Host
	return parsedValue.String()
}
