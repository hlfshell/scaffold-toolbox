# scaffold mysql

Typed MySQL harness for scaffold. It uses the official `mysql` container image.

## Install

```bash
go get github.com/hlfshell/scaffold-toolbox/mysql
```

```go
import "github.com/hlfshell/scaffold-toolbox/mysql"
```

```go
mysql, err := mysql.NewMysql("app-mysql", "8", "user", "pass", "app")
mysql.WithSQL("create table users (id int primary key);")
err = mysql.Create(ctx)
defer mysql.Cleanup(context.WithoutCancel(ctx))
```

Preload helpers can run SQL strings or SQL files. Cleanup closes the database handle and removes the container and anonymous volumes.
