module github.com/hlfshell/scaffold-toolbox/stacks/analytics

go 1.25.0

require (
	github.com/hlfshell/scaffold v0.0.1
	github.com/hlfshell/scaffold-toolbox/clickhouse v0.0.0
	github.com/hlfshell/scaffold-toolbox/trino v0.0.1
)

require (
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/containerd/errdefs v1.0.0 // indirect
	github.com/containerd/errdefs/pkg v0.3.0 // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/docker/docker v28.5.2+incompatible // indirect
	github.com/docker/go-connections v0.7.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.29.0 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.69.0 // indirect
	go.opentelemetry.io/otel v1.44.0 // indirect
	go.opentelemetry.io/otel/metric v1.44.0 // indirect
	go.opentelemetry.io/otel/trace v1.44.0 // indirect
	golang.org/x/sys v0.46.0 // indirect
	google.golang.org/grpc v1.80.0 // indirect
)

replace github.com/hlfshell/scaffold => ../../../scaffold

replace github.com/hlfshell/scaffold-toolbox/clickhouse => ../../clickhouse

replace github.com/hlfshell/scaffold-toolbox/trino => ../../trino
