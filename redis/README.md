# scaffold redis

Typed Redis harness for scaffold. It uses the official `redis` container image.

## Install

```bash
go get github.com/hlfshell/scaffold-toolbox/redis
```

```go
import "github.com/hlfshell/scaffold-toolbox/redis"
```

```go
redis, err := redis.NewRedis("app-redis", "latest")
redis.WithKey("feature:test", "enabled")
err = redis.Create(ctx)
defer redis.Cleanup(context.WithoutCancel(ctx))
```

Preload helpers can set keys or run a seed function with the Go Redis client. Cleanup closes the client and removes the container.
