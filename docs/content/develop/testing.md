---
---

Blip uses standard Go testing. From the repo root, run:

```bash
go test ./...
```

## MySQL

Local Blip testing uses [Docker Compose](https://docs.docker.com/compose/) in [test/docker/](https://github.com/cashapp/blip/tree/main/test/docker) to run a matrix of MySQL servers:

* MySQL 5.7.34
* MySQL 8.0.25
* Percona Server 5.7.35

These are subject to change with various releases.

In Go tests, use [test.Connection()](https://github.com/cashapp/blip/blob/main/test/mysql.go) to make a `*sql.DB` to a specific MySQL server.
