---
layout: default
parent: Sinks
title: datadog
---

# Datadog Sink

{: .warn }
Blip works with Datadog, but Datadog does not support or contribute to Blip.

The datadog sink sends metrics to [Datadog](https://www.datadoghq.com/).

This sink is an optional replacement for the [Datadog Agent for MySQL](https://docs.datadoghq.com/integrations/mysql/).
Consequently, the metrics it sends are [billed as custom metrics](https://docs.datadoghq.com/account_management/billing/custom_metrics/).

Metrics are reported as domain-qualified Blip metric names: `status.global.threads_running`.
All [tags](../config/config-file#tags) are reported as Datadog tags.

Metrics can be sent either to Datadog API directly or through a Datadog agent using DogStatsD protocol.
Only one of the methods can be used at a time.
If `dogstatsd-host` option is set, DogStatsD is used for sending metrics, otherwise `api` keys must be provided.
If the `doststatsd-host` doesn't include the port, `8125` is used as the default port.

## Quick Reference

```yaml
sinks:
  datadog:
    api-compress: "true"
    api-key-auth: ""
    api-key-auth-file: ""
    app-key-auth: ""
    app-key-auth-file: ""
    metric-translator: ""
    metric-prefix: ""
    dogstatsd-host: ""
```
