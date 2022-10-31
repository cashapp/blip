---
layout: default
parent: Sinks
title: chronosophere
---

# Chronosphere Sink

{: .warn }
The chronosphere sink is experimental.
It might be removed in future versions.

The chronosphere sink sends metrics to [Chronosphere](https://chronosphere.io/).

It _pushes_ metrics to a Chronosphere collector.
If you want a Chronosphere collector to scrape metrics, use [Prometheus emulation](../prometheus) instead.

This sink reports Prometheus-style metric names: `mysql_status_threads_running` instead of `status.global.threads_running`.
It reports all [tags](../config/config-file#tags) as Prometheus labels.

## Quick Reference

```yaml
sinks:
  chronosphere:
    debug: "no"
    strict-tr: "no"
    url: "http://127.0.0.1:3030/openmetrics/write"
```

## Options

### `debug`

{: .var-table }
|**Valid values**|`yes` or `no`|
|**Default value**|`no`|

Print debug output to STDERR.

### `strict-tr`

{: .var-table }
|**Valid values**|`yes` or `no`|
|**Default value**|`no`|

Error if metrics cannot be translated to Prometheus/Exposition format.

### `url`

{: .var-table }
|**Valid values**|URL|
|**Default value**|`http://127.0.0.1:3030/openmetrics/write`|

URL of Chronocollector.
