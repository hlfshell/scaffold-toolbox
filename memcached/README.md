# scaffold memcached

Typed Memcached harness for scaffold. It uses the official `memcached` container image.

## Install

```bash
go get github.com/hlfshell/scaffold-toolbox/memcached
```

```go
import "github.com/hlfshell/scaffold-toolbox/memcached"
```

```go
cache, err := memcached.NewMemcached("app-cache", "latest")
cache.WithItem("hello", []byte("world"))
err = cache.Create(ctx)
defer cache.Cleanup(context.WithoutCancel(ctx))
```

Preload helpers can set cache items. Cleanup closes the client and removes the container.
