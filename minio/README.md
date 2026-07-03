# scaffold minio

Typed MinIO harness for scaffold. It uses the `minio/minio` container image and exposes the S3-compatible API plus the web console.

```go
store, err := minio.NewMinIO("objects", "latest", "minioadmin", "minioadmin")
store.WithBucket("uploads")
store.WithObject("uploads", "hello.txt", []byte("hello"), "text/plain")

err = store.Create(ctx)
defer store.Cleanup(context.WithoutCancel(ctx))
```

Preload helpers can create buckets, upload byte slices, and upload files. Cleanup removes the container and anonymous volumes.
