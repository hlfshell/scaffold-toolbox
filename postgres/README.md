# scaffold postgres

Typed Postgres harness for scaffold. It uses the official `postgres` container image.

## Install

```bash
go get github.com/hlfshell/scaffold-toolbox/postgres
```

```go
import "github.com/hlfshell/scaffold-toolbox/postgres"
```

```go
pg, err := postgres.NewPostgres("app-postgres", "16", "user", "pass", "app")
if err != nil {
	return err
}
pg.WithSQL("create table users (id serial primary key, email text not null);")

err = pg.Create(ctx)
if err != nil {
	return err
}
defer pg.Cleanup(context.WithoutCancel(ctx))

db, err := pg.ConnectWithTimeout(10 * time.Second)
```

Preload helpers can run SQL strings or SQL files. Cleanup closes the database handle and removes the container and anonymous volumes.
