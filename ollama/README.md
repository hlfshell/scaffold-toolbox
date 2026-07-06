# scaffold toolbox ollama

Ollama service for scaffold. It starts the official Ollama container, exposes
the local API endpoint, and can pull models as part of stack startup.

## Install

```bash
go get github.com/hlfshell/scaffold-toolbox/ollama
```

```go
import "github.com/hlfshell/scaffold-toolbox/ollama"
```

## Example

```go
models, err := ollama.NewOllama("models", "latest")
if err != nil {
	return err
}

models.WithModel("nomic-embed-text")
```
