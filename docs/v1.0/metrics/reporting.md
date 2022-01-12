---
layout: default
parent: Metrics
title: Reporting
nav_order: 3
---

# Reporting
{: .no_toc }

Blip does not have a single format or protocol for reporting metrics.
Instead, it has well-defined structures for collecting metrics that are passed to [metric sinks](../sinks/), which convert and send metrics in various formats and protocols depending on the sink.

In YAML, the basic structure of Blip metrics is:

```
domain:
  - metric:
      value: <float64>
      type: counter|gauge|...
      group:
        key1: val1
      meta:
        key1: val1
```

All metrics belong to a single [domain](domains).
Each domain collector can collect any number of metrics.
Each metric has a type and value (see [Conventions > Metrics](conventions#metrics)).
[Groups](#groups) and [Meta](#meta) are detailed below.

A realistic example of two metrics from the [`status.global` domain](domains#statusglobal):

```
global.status:
  - threads_running:
      value: 16
      type: gauge
  - queries:
      value: 138923
      type: counter
```

The metric sink determines how those 2 metrics are reported.
For example, a Prometheus sink might report them in Exposition format:

```
# TYPE mysql_global_status_threads_running gauge
mysql_global_status_threads_running 16
# TYPE mysql_global_status_queries counter
mysql_global_status_queries 138923
```

Blip [conventions](conventions) terminate at each metric sink.

## Groups

Certain [domains](domains) (as documented) implicitly or explicitly group metrics.
In both cases, group key-value pairs are set for each metric, and metrics are uniquely identified using _all_ group keys.

{: .note }
_Groups_, _labels_, and _dimensions_ serve the same purpose.
Blip uses the term _group_ because it's similar to MySQL `GROUP BY`.

_Implicit grouping_ means the metrics collector groups metrics automatically (or as configured in the plan by [collector options](collectors#options)).
For example, the [`size.data` collector](domains#sizedata), which collects database and table sizes, groups metrics by database name.
As a result, each metric has a group key on `db`, like the following example:

```
size.database:
  - bytes:
      value: 50920482048
      type: gauge
      group:
        db: foo
  - bytes:
      value: 8920482048
      type: gauge
      group:
        db: bar
  - bytes:
      value: 59840964096
      type: gauge
      group:
        db: ""  # all databases
```

The metric is the same&mdash;`bytes`&mdash;but there are 3 instances of the metric for each of the 3 groups: `db=foo`, `db=bar`, and `db=""`.
The last group, `db=""`, is how this collector represents the metric for all databases (the global group).

_Explicit grouping_ refers to MySQL-grouped metrics&mdash;see [Sub-domains](#sub-domains).
For example, the [`status.host` collector](domains#statushost) is explicitly grouped by `host`.
As a result, each metric has a group key on `host`, like `host=10.1.1.1`.

## Meta

Certain [domains](domains) (as documented) set metadata about a metric.
The canonical example is the `response_time` [collector metric](conventions#collector-metrics) reported by the [`query.global` collector](domains#queryglobal):

```
query.global:
  - response_time:
      value: 130493
      type: gauge
      meta:
        p95: p948
  - response_time:
      value: 255001
      type: gauge
      meta:
        p999: p997
```

As shown above, there are 2 `response_time` metrics that differ by `meta=p95` and `meta=p999`: the former is the 95th percentile response time, and the latter is the 99.9th percentile response time.

Blip uses meta rather than prefixing or suffixing the metric name (see [Conventions > Metrics > Naming](conventions#naming)).
This makes metrics (by name) consistent: the metric is always `response_time`, not `pN_response_time` or `response_time_pN` where `N` could be any number.
It also allows for a greater variety of metrics without special case exceptions to naming or structure; for example, average response time could be denoted by meta key `avg`.
