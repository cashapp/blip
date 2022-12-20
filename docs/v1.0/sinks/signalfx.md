---
layout: default
parent: Sinks
title: signalfx
---

# SignalFx Sink

{: .warn }
Blip works with Splunk, but Splunk does not support or contribute to Blip.

The signalfx sink sends metrics to [Splunk](https://www.splunk.com/), which [acquired SignalFx](https://www.splunk.com/en_us/newsroom/press-releases/2019/splunk-to-acquire-cloud-monitoring-leader-signalfx.html).

It reports all [tags](../config/config-file#tags) as dimensions.

## Quick Reference

```yaml
sinks:
  signalfx:
    auth-token: ""
    auth-token-file: ""
    metric-prefix: ""
    metric-translator: ""
```

## Options

### `auth-token`

{: .var-table }
|**Valid values**|SignalFx auth token|
|**Default value**||

SignalFx auth token.
Either `auth-token` or `auth-token-file` is required.

### `auth-token-file`

{: .var-table }
|**Valid values**|File name|
|**Default value**||

File containing SignalFx auth token.
Either `auth-token` or `auth-token-file` is required.

### `metric-prefix`

{: .var-table }
|**Valid values**|String|
|**Default value**||

A string prepended to every metric name before sending.
For example, `metric-prefix: "mysql."` adds "mysql." to the beginning of every metric name.
The string value is literal; Blip does _not_ add a trailing dot.

### `metric-translator`

{: .var-table }
|**Valid values**|Registered translator name|
|**Default value**||

Pass metrics through registered metrics translator.
This occurs before `metric-prefix`.
