---
layout: default
parent: Sinks
title: chronosophere
---

# Chronosphere Sink

```yaml
sinks:
  chronosphere:
    url: "http://127.0.0.1:3030/openmetrics/write"
```

Defaults should work presuming a local Chronocollector is running; else, set `url` option to address of Chronocollector.

Reports all [tags](../config/config-file#tags) as Prometheus labels.

Reports Prometheus-style metric names: `mysql_status_threads_running` instead of `status.global.threads_running`.
