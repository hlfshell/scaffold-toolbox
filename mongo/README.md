# scaffold toolbox mongo

MongoDB service for scaffold. It starts the official `mongo` container,
returns an official Go MongoDB client, and can insert seed documents after
the database is ready.

## Install

```bash
go get github.com/hlfshell/scaffold-toolbox/mongo
```

```go
import "github.com/hlfshell/scaffold-toolbox/mongo"
```

## Example

```go
documents, err := mongo.NewMongo("documents", "latest", "root", "secret", "app")
if err != nil {
	return err
}

documents.WithDocuments("users", map[string]any{"name": "Ada"})
```
