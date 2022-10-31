---
layout: default
parent: Metrics
title: Collecting
---

# Collecting
{: .no_toc }

[Introduction / Metrics](../intro/metrics) and [Introduction / Plans](../intro/plans) briefly describe how Blip collects metrics.
TL;DR: _you specify metrics (by domain) in a plan_.

This page details how Blip collects metrics and how to customize metrics collection.
After reading this page, be sure to read the next, [Reporting](reporting), to learn how Blip reports metrics.

Understanding Blip metrics collection begins with domains.

* TOC
{:toc}

## Domains

Blip organizes _all_ metrics into [domains](domains).
Domains are a Blip invention; they are not inherent to MySQL or other monitoring tools.
Blip uses domains for organization and consistency because MySQL metrics lack both, which makes it nearly impossible to read or write coherently about MySQL metrics.

Each domain uses one or more _source_: a MySQL command or table from which Blip collects metrics for the domain.
Most domains map very closely to the source of MySQL metrics that they represent.
For example, domain [`var.global`](domains#varglobal) uses source `SHOW GLOBAL VARIABLES`.

To collect a specific metric, find the [domain](domains) that uses the source of the metric, the add the domain and metric to the [plan](../plans/).
For example, to collect global system variable [`max_connections`](https://dev.mysql.com/doc/refman/8.0/en/server-system-variables.html#sysvar_max_connections):

```yaml
sysvars:
  freq: 1h
  collect:
    var.global:
      metrics:
        - max_connections
```

That plan is only a snippet, but it demonstrates the basic concepts that apply to collecting all metrics (by domain).

### Reusing

Recall from [Introduction / Plans](../intro/plans) that Blip automatically levels up: when level collection times overlap, Blip automatically collects all metrics in the overlapping levels.
Leveling up occurs because domains can be reused at different levels (domains are unique per level).
This is important because it's both common and necessary for collecting different metrics at different frequencies&mdash;which is the main benefit of level plans.
For example, [`status.global`](domains#statusglobal) collects metrics from `SHOW GLOBAL STATUS` (the primary source of MySQL metrics), and it's used to collect different metrics at different frequencies:

```yaml
kpi:
  freq: 5s
  collect:
    status.global:
      metrics:
        - queries

standard:
  freq: 20s
  collect:
    status.global:
      metrics:
        - bytes_sent
        - bytes_received
```

The `kpi` level every 1 second collects metric `Queries` to calculate true QPS.
The `standard` level every 20 seconds collects metrics `Bytes_sent` and `Bytes_receive` to calculate network throughput, which doesn't need 1-second resolution.
Every 20 seconds, Blip automatically levels up to collect all metrics in both levels because 20 overlaps 5.
(_Overlap_ means a higher level `H` overlaps a lower level `L` when `H` is a multiple of `L`; or more simply: `H mod L = 0`.)
Therefore, you should not reuse _metrics_; each metric (per domain) should appear in only one level.

<p class="note">
Reuse domains but not metrics.
Blip automatically collects all metrics in overlapping lower levels.
</p>

### Options

Each domain has its own options that are documented in the [domain reference](domains) and printed by running:

```sh
blip --print-domains
```

Since domains are unique per plan level, their options are unique per level, too.
See [Configure / Collectors](../config/collectors).

## Metrics

Metrics to collect are listed under `metrics` for each domain in a plan:

```yaml
standard:
  freq: 20s
  collect:
    status.global:
      metrics:
        - Queries
        - Threads_running
    var.global:
      metrics:
        - Max_connections
```

A few domains have options to collect all metrics from the MySQL source, but most require explicitly listing the metrics to collect.
This seems tedious when writing a plan, but it's necessary because almost every MySQL metric source includes non-metrics.
For example, `SHOW GLOBAL STATUS` includes many strings, like `Innodb_buffer_pool_dump_status = "Dumping of buffer pool not started"`&mdash;clearly not a metric.
Listing metrics to collect results in remarkably better metrics collection&mdash;it's an investment with a very high return.

### MySQL Metrics

MySQL metrics are straight from MySQL and listed in a plan with the exact same name as they appear in MySQL.
If you want to collect these metrics from [`status.global`](domains#statusglobal),

```
mysql> SHOW GLOBAL STATUS;
+--------------------------------+---------+
| Variable_name                  | Value   |
+--------------------------------+---------+
| Aborted_clients                | 44      |
| Aborted_connects               | 38      |
| Binlog_cache_disk_use          | 0       |
| Binlog_cache_use               | 5       |
| Binlog_stmt_cache_disk_use     | 0       |
| Binlog_stmt_cache_use          | 0       |
| Bytes_received                 | 93451   |
| Bytes_sent                     | 2361896 |
```

then specify those metrics in a plan:

```yaml
standard:
  freq: 20s
  collect:
    status.global:
      metrics:
        - Aborted_clients
        - Aborted_connects
        - Binlog_cache_disk_use
        - Binlog_cache_use
        - Binlog_stmt_cache_disk_use
        - Binlog_stmt_cache_use
        - Bytes_received
        - Bytes_sent
```

Blip only collects the metrics listed in the plan.

### Derived Metrics

_Derived metrics_ are metrics that Blip creates (derives) from MySQL metrics.
Domain [`size.database`](domains#sizedatabase) is a canonical example: MySQL does not provide a single database size metric; tools like Blip derive the metric from various MySQL tables and values.
That domain collects a derived metric called `bytes`.

Derived metrics are _not_ automatically collected; they must be explicitly listed in the plan like MySQL metrics.
For example:

```yaml
level:
  collect:
    size.database:
      metrics:
        - bytes  # derived metric
```

The [domain reference](domains) documents derived metrics for each domain.

### Renaming

Blip never renames MySQL metrics on collection.
Even in cases like `trx_rseg_history_len` (a metric from the [`innodb`](domains#innodb) domain), Blip never renames MySQL metrics.
(That metric is a gauge for InnoDB history list length.)

Metrics can be renamed _after_ collection by using the [TransformMetrics plugin](../develop/integration-api#plugins) or writing a [custom sink](../develop/sinks).
See [Reporting](reporting) to understand how metrics are reported post-collection, which provides a consistent and programmatic basis for the plugin or sink.

## Collectors

_Collectors_ (short for _metric collectors_) are low-level components that collect metrics for [domains](domains).
Collectors and domains are one-to-one: one domain uses one collector to collect the metrics from the MySQL source.

Since collectors are low-level, this documentation refers more frequently to domains, which are user-level.
The distinction is made when necessary, like [Configure / Collectors](../config/collectors), but generally collectors and domains are synonymous since they're are one-to-one.

See [Develop / Collectors](../develop/collectors) to learn how to create new collectors (and domains) to make Blip collect new metrics.

## Defaults

Blip uses a [default plan](../plans/defaults) to collect over 70 of the most important MySQL metrics by default.
