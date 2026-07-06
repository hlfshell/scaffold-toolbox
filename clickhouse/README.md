# scaffold toolbox clickhouse

ClickHouse service for scaffold. It starts `clickhouse/clickhouse-server`,
waits for the HTTP API, exposes HTTP and native endpoints, and can run SQL
setup from strings or files.

## Install

```bash
go get github.com/hlfshell/scaffold-toolbox/clickhouse
```

```go
import "github.com/hlfshell/scaffold-toolbox/clickhouse"
```

## Example

```go
analytics, err := clickhouse.NewClickHouse("analytics", "latest", "default", "secret", "events")
if err != nil {
	return err
}

analytics.WithSQL("CREATE TABLE IF NOT EXISTS events.local (id UInt64) ENGINE = Memory")
```
