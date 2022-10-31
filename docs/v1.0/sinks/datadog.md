---
layout: default
parent: Sinks
title: datadog
---

# Datadog Sink

The datadog sink sends metrics to [Datadog](https://www.datadoghq.com/).

This sink is an optional replacement for the [Datadog Agent for MySQL](https://docs.datadoghq.com/integrations/mysql/).
Consequently, the metrics it sends are [billed as custom metrics](https://docs.datadoghq.com/account_management/billing/custom_metrics/).

Metrics are reported as domain-qualified Blip metric names: `status.global.threads_running`.
All [tags](../config/config-file#tags) are reported as Datadog tags.

## Quick Reference

```yaml
sinks:
  datadog:
    api-key-auth: ""
    api-key-auth-file: ""
    app-key-auth: ""
    app-key-auth-file: ""
    metric-translator: ""
    metric-prefix: ""
```
