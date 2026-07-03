# scaffold memcached

Typed Memcached harness for scaffold. It uses the official `memcached` container image.

```go
cache, err := memcached.NewMemcached("app-cache")
cache.WithItem("hello", []byte("world"))
err = cache.Create(ctx)
defer cache.Cleanup(context.WithoutCancel(ctx))
```

Preload helpers can set cache items. Cleanup closes the client and removes the container.
