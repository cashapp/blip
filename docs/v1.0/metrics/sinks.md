---
layout: default
parent: Metrics
title: Sinks
nav_order: 5
---

# Metric Sinks

## Built-in

### Chronosphere

```yaml
sinks:
  chronosphere:
    url: "http://127.0.0.1:3030/openmetrics/write"
```

Defaults should work presuming a local Chronocollector is running; else, set `url` option to address of Chronocollector.

Reports all [tags](../config/config-file.html#tags) as Prometheus labels.

Reports Prometheus-style metric names: `mysql_status_threads_running` instead of `status.global.threads_running`.

### SignalFx

```yaml
sinks:
  signalfx:
    auth-token: ""
    auth-token-file: ""
```

Must provide `auth-token` or `auth-token-file` in config.

Reports all [tags](../config/config-file.html#tags) as dimensions.

Reports domain-qualified metric names: `status.global.threads_running`.

## Custom
