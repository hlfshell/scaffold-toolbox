# scaffold qdrant

Typed Qdrant harness for scaffold. It uses the `qdrant/qdrant` container image and exposes REST and gRPC ports.

## Install

```bash
go get github.com/hlfshell/scaffold-toolbox/qdrant
```

```go
import "github.com/hlfshell/scaffold-toolbox/qdrant"
```

```go
q, err := qdrant.NewQdrant("vectors", "latest")
q.WithCollection(qdrant.CollectionConfig{Name: "docs", Size: 3, Distance: "Cosine"})
q.WithPoints("docs", []qdrant.Point{{ID: 1, Vector: []float64{0.1, 0.2, 0.3}}})

err = q.Create(ctx)
defer q.Cleanup(context.WithoutCancel(ctx))
```

Preload helpers can create collections and insert points, including points loaded from JSON.
