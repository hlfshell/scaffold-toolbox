# AWS app stack example

This example uses the base AWS toolbox module to create a local app-shaped cloud environment. It intentionally does not use a `stacks/aws` wrapper.

Run it from this directory after adding real `api.Dockerfile` and `worker.Dockerfile` files:

```bash
go run .
```

The example starts MiniStack, creates S3/SQS resources, builds the Dockerfiles into the local ECS registry, and registers ECS/Fargate-style containers through the MiniStack ECS API.
