---
title: retry
---

The retry sink is a pseudo-sink that provides buffering, serialization, and retry for real sinks.
The built-in sinks, except [`log`]({{< ref "log" >}}), use a retry sink to handle those three complexities.

The retry sink uses a LIFO queue (a stack) to prioritize sending the latest metrics.
During a long outage of the real sink, the retry sink drops the oldest metrics and keeps the latest metrics, up to its buffer size, which is configurable.

## Quick Reference

```yaml
sinks:
  retry:
    buffer-size: 60
    send-timeout: 5s
    send-retry-wait: 200ms
```
