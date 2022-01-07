---
layout: default
parent: Sinks
title: signalfx
---

# SignalFx Sink

```yaml
sinks:
  signalfx:
    auth-token: ""
    auth-token-file: ""
```

Must provide `auth-token` or `auth-token-file` in config.

Reports all [tags](../config/config-file#tags) as dimensions.

Reports domain-qualified metric names: `status.global.threads_running`.
