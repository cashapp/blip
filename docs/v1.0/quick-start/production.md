---
layout: default
parent: Quick Start
title: Production
---

# Production

Order:

1. Build
2. Configure
3. Deploy

Presuming a standard MySQL instance runs on your laptop, first create a `blip` user:

```sql
CREATE USER IF NOT EXISTS 'blip' IDENTIFIED BY '';    -- no password
GRANT SELECT ON `performance_schema`.* TO 'blip'@'%'; -- no privileges
```

Then run `blip` (after compiling it in `bin/blip/`, of course):

```sh
$ blip
```

By default, Blip automatically finds local MySQL instances, and tries a few default username-password combinations.

If successful, it will dump metrics to `STDOUT`.
If not successful, run with `--debug`.
