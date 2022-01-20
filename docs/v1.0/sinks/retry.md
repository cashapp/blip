---
layout: default
parent: Sinks
title: retry
---

# Retry Sink

```yaml
sinks:
  retry:
    buffer-size: 60
    send-timeout: 5s
    send-retry-wait: 200ms
```

Retry is a pseudo-sink that provides buffering, serialization, and retry for a real sink.
The built-in sinks, except [`log`](./log), use Retry to handle those three complexities.

Retry uses a LIFO queue (a stack) to prioritize sending the latest metrics.
This means that, during a long outage of the real sink, Retry drops the oldest metrics and keeps the latest metrics, up to its buffer size, which is configurable.
