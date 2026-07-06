# scaffold toolbox litellm

LiteLLM service for scaffold. It starts the OpenAI-compatible proxy, exposes
the base URL and API key environment variables, and can mount a LiteLLM
config file for provider routing.

## Install

```bash
go get github.com/hlfshell/scaffold-toolbox/litellm
```

```go
import "github.com/hlfshell/scaffold-toolbox/litellm"
```

## Example

```go
proxy, err := litellm.NewLiteLLM("llm-proxy", "latest",
	litellm.WithConfigFile("./litellm.yaml"),
	litellm.WithMasterKey("sk-local"),
)
if err != nil {
	return err
}
```
