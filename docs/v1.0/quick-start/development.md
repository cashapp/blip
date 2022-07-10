---
layout: default
parent: Quick Start
title: Development
---

# Development

This quick start guide is for **development**: running Blip on your laptop or any other development environment where we can take shortcuts to get Blip running as quickly as possible.
The goal is only to get an idea of how Blip works.


```sh
cd bin/blip
go build
```

Presuming a standard MySQL instance is running on your laptop, first create a `blip` user:

## Exercise 1: Indistinguishable from Magic (Blip)

### 1. Create `blip` MySQL user

```sql
CREATE USER IF NOT EXISTS 'blip' IDENTIFIED BY ''; -- no password
GRANT SELECT ON `performance_schema`.* TO 'blip'@'%';
GRANT REPLICATION CLIENT ON *.* TO 'blip'@'%';
```

```sh
$ blip
```

By default, Blip automatically finds local MySQL instances, and tries a few default username-password combinations.

If successful, it will dump metrics to `STDOUT`.
If not successful, run with `--debug`.

## Exercise 2:

## Exercise 3: Like a Tax Form, Only More Fun (Configuration)


## Exercise 4: Right Down The (Sink)

## Exercise 5: The Strong, Silent Type (Logs and API)
