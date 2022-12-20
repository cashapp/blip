---
layout: default
parent: Plans
title: "Quick Reference"
---

# Quick Reference

The following is a _quick references_, not a complete or valid example.
See [File](file) for details.

### Interpolation

```
${ENV_VAR}
%{monitor.hostname}
%{monitor.meta.region}
```

### Plan File

```yaml
# Level "kpi" every 5 seconds
kpi:
  freq: 5s
  collect:
    status.global:
      metrics:
        - Queries
        - Threads_running
    repl:
      errors:
        access-denied: "report-once,drop,stop"
      metrics:
        - running

# Level "standard" every 20 seconds
standard:
  freq: 20s
  status.global:
    metrics:
     - Select_full_join
     - Select_full_range_join
     - Select_scan
  collect:
    innodb:
      options:
        all: "yes"

# Level "data-size" every 1 minute
standard:
  freq: 1m
  collect:
    size.database:
      # All databases by default; no options or metrics
```
