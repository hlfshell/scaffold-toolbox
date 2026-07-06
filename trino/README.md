# scaffold toolbox trino

Trino service for scaffold. It starts the official Trino coordinator image,
mounts generated catalog property files, exposes the HTTP endpoint, and can
submit SQL through Trino's REST API.

## Install

```bash
go get github.com/hlfshell/scaffold-toolbox/trino
```

```go
import "github.com/hlfshell/scaffold-toolbox/trino"
```

## Example

```go
query, err := trino.NewTrino("query", "latest",
	trino.WithMemoryCatalog("memory"),
)
if err != nil {
	return err
}
```
