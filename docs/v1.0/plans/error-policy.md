---
layout: default
parent: Plans
title: "Error Policy"
nav_order: 6
---

# Error Policy

An error policy defines how a metric collector handles a specific MySQL errors.
Error policies are optional, and most metric collectors do not define any.
Instead, they rely on the default error policy, `report,drop,retry`: report the error, drop the metric, and retry.

Since one error policy handles one specific MySQL error, they are _not_ intended for [general error handling](../monitors/error-handling).
Instead, they are intended to handle different MySQL setup without different Blip plans.

```yaml
collect:
  repl:
    metrics:
      - running
    errors:
      access-denied: "ignore,drop,retry"
```

## Format

The value has three parts: `<report>,<metric>,<retry>`

Report:
* `ignore`: Silently ignore the error; report _nothing_ (not even an event)
* `report`: Report the metric (**default**)
* `report-once`: Report the metric only the first time it occurs, then ignore further errors

Metric:
* `drop`: Drop the metric (**default**)
* `zero`: Report zero value

Retry:
* `retry`: Keep trying to collect metric (**default**)
* `stop`: Stop collecting metric
