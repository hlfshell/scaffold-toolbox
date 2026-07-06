# scaffold toolbox weaviate

Weaviate service for scaffold. It starts a local Weaviate instance with
anonymous access and no default vectorizer, then can create schema classes
and objects through the REST API.

## Install

```bash
go get github.com/hlfshell/scaffold-toolbox/weaviate
```

```go
import "github.com/hlfshell/scaffold-toolbox/weaviate"
```

## Example

```go
vectors, err := weaviate.NewWeaviate("vectors", "latest")
if err != nil {
	return err
}

vectors.WithClass(weaviate.Class{
	Class: "Document",
	Properties: []weaviate.Property{{Name: "title", DataType: []string{"text"}}},
})
```
