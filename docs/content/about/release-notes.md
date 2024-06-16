---
weight: 0
---

## v1.1

This is a new series (new minor version).
As per the [Blip versioning guidelines](https://github.com/cashapp/blip/blob/main/CONTRIBUTING.md#versioning), this new series is ***not*** entirely backwards-compatible with v1.0 due to these changes:

|# |Component|v1.0|v1.1|
|--|---------|----|----|
|1 |`datadog` sink|Sends cumulative counters|Sends delta counters|
|2 |`repl.lag` collector|Default `writer=blip`|Default `writer=auto` will use Performance Schema on MySQL 8.x|

How to upgrade (by number in the table above):

1. Datadog counters are deltas by default, so the new behavior works better. Use [`.as_rate()`](https://docs.datadoghq.com/metrics/custom_metrics/type_modifiers/?tab=count) instead of the [`rate()` function](https://docs.datadoghq.com/dashboards/functions/rate/). Note that Datadog charts work best when the interval is set for each metric.
2. To continue using Blip heartbeats, explicitly configure `repl.lag` with option `writer=blip`.

### v1.1.0 (17 Jun 2024)

* `datadog` sink:
  * Changed to send delta counters instead of cumulative counters (PR #106)
* `repl.lag` collector:
  * Added MySQL 8.x Performance Schema support (auto-detected or writer=pfs) (PR #118)
* `wait.io.table` collector:
  * Added `count_star` to metrics
* Added sink `prom-pushgateway` ([Prometheus Pushgateway](https://github.com/prometheus/pushgateway))
* Updated built-in AWS RDS CA from rds-ca-2019 to the global bundle (PR #113)
* Made HA manager a configurable plugin (PR #116)
* Changed `max_used_connetions` to gauge (PR #111)
* Fixed GitHub Dependabot alerts

---

## v1.0

### v1.0.2 (03 Jul 2023)

* `datadog` sink:
  * Fixed timestamps: DD expects timestamp as seconds, not milliseconds
  * Send new `event.SINK_ERROR` and debug DD API errors on successful request
* `query.response-time` and `wait.io.table` collectors:
  * Added `truncate-timeout` option and error policy
  * Fixed docs: option `truncate-table` defaults to "yes"
* Fixed GitHub Dependabot alerts

### v1.0.1 (02 Mar 2023)

* `datadog` sink:
  * Fixed intermittent panic
  * Fixed HTTP error 413 (payload too large) by dynamically partitioning metrics
  * Added option `api-compress` (default: yes)
* `repl` collector:
  * Added option `report-not-a-replica`
  * Moved pkg vars `statusQuery` and `newTerms` to `Repl` to handle multiple collectors on different versions
  * Fixed docs (only `repl.running` is currently collected)
* Updated `aws/AuthToken.Password`: pass context to `auth.BuildAuthToken`
* Fixed GitHub Dependabot alerts
* Fixed `blip.VERSION`

### v1.0.0 (22 Dec 2022)

* First GA, production-ready release.
