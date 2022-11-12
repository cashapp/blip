---
layout: default
parent: Metrics
title: Reporting
---

# Reporting
{: .no_toc }

Blip does not have a single format or protocol for reporting metrics.
Instead, after [collecting metrics](collecting) Blip uses metric [sinks](../sinks/) to translate the Blip [metric data structure](#metric-data-structure) to a sink-specific protocol.

Normally, a sink sends metrics somewhere, but a sink can do anything with the metrics&mdash;Blip does not impose any restrictions on sinks.
For example, the default [log sink](../sinks/log) dumps metrics to `STDOUT`.

Blip has several [built-in sinks](../sinks/), but unless you happen to use one of them, you will need to write a [custom sink](../develop/sinks) to make Blip report metrics in your environment.
Start by understanding how Blip reports metrics internally (as detailed on this page), then read [Develop / Sinks](../develop/sinks).

* TOC
{:toc}

## Metrics

### Types

Blip metric types are standard:

* `COUNTER`
* `GAUGE`
* `BOOL`  (reserved for future use)
* `EVENT` (reserved for future use)

Blip automatically uses the correct metric type for all metrics.

Blip, like MySQL, does not distinguish between "counter" and "cumulative counter".
Blip counter metrics can reset to zero if MySQL is restarted; otherwise, the value only increases.

### Values

All values, regardless of type, are `float64`.

Negative values are allowed.
Some [derived metrics](collecting#derived-metrics) use negative values as documented in the [domain reference](domains).
MySQL metrics are not supposed to be negative, but there are MySQL bugs that cause negative values.

### Units

MySQL metrics use a variety of units&mdash;from picoseconds to seconds.
When the MySQL metric unit is documented and consistent, Blip reports the value as-is.
For example, `innodb.buffer_flush_avg_time` is documented as "Avg time (ms) spent for flushing recently.", therefore Blip reports the value as-is: as milliseconds.

When the MySQL metric unit is variable, Blip uses the following units:

|Metric Type|Unit|
|-----------|----|
|Query time|microseconds (μs)
|Lock time|microseconds (μs)
|Wait time|microseconds (μs)
|Replication (lag)|milliseconds (ms)
|Data size|bytes

For example, query response time ranges from nanoseconds to seconds (with microsecond precision) depending on the source.
But regardless of the source, Blip reports `query.*.response_time` as _microseconds (μs)_.

{: .note}
To convert units, use the [TransformMetrics plugin](../develop/integration-api#plugins) or write a [custom sink](../develop/sinks)

Blip does _not_ suffix metric names with units, and it does not strip the few MySQL metrics that have unit suffixes.

### Renaming

Blip never renames MySQL metrics on collection or within its [metric data structure](#metric-data-structure).
Metrics can be renamed _after_ collection by using the [TransformMetrics plugin](../develop/integration-api#plugins) or writing a [custom sink](../develop/sinks).

## Metric Data Structure

Internally, Blip stores metrics in a [`blip.Metrics` data structure](https://pkg.go.dev/github.com/cashapp/blip#Metrics):

```
type Metrics struct {
	Begin     time.Time                // when collection started
	End       time.Time                // when collection completed
	MonitorId string                   // ID of monitor (MySQL)
	Plan      string                   // plan name
	Level     string                   // level name
	State     string                   // state of monitor
	Values    map[string][]MetricValue // keyed on domain
}
```

[Metric values](https://pkg.go.dev/github.com/cashapp/blip#MetricValue) (the last field in the struct) are reported per-domain.

To visual the data structure more easily, in YAML it would be:

```yaml
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
Each metric has a type and value.
[Groups](#groups) and [Meta](#meta) are detailed below.

{: .note }
Blip metrics are not stored or report in YAML; these examples are only for illustration.

A realistic example of two metrics from the [`status.global` domain](domains#statusglobal):

```
status.global:
  - threads_running:
      value: 16
      type: gauge
  - queries:
      value: 138923
      type: counter
```

The metric sink determines how those two metrics are reported.
For example, a Prometheus sink might report them in [Exposition format](https://github.com/prometheus/docs/blob/main/content/docs/instrumenting/exposition_formats.md):

```
# TYPE mysql_global_status_threads_running gauge
mysql_global_status_threads_running 16
# TYPE mysql_global_status_queries counter
mysql_global_status_queries 138923
```

Once Blip reports metrics to a sink, the sink fully owns the metrics; Blip no longer references it.
A sink can translate Blip metrics into any format or protocol.

### Groups

Certain [domains](domains) group metrics as documented.
Group key-value pairs are set for each metric, and metrics are uniquely identified using _all_ group keys.

{: .note }
_Groups_, _labels_, and _dimensions_ serve the same purpose.
Blip uses the term _group_ because it's similar to MySQL `GROUP BY`.

For example, the [`size.database` domain](domains#sizedata) groups metrics by database name.
Therefore, each metric has a group keyed on `db`:

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
The last group, `db=""`, is how this domain represents the metric for all databases (the global group).

### Meta

Certain [domains](domains) set metadata about metrics as documented.
The canonical example is derived metric `response_time` reported by the [`query.global` domain](domains#queryglobal):

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

There are 2 instances of metric `response_time` that differ by `meta=p95` and `meta=p999`: the former is the 95th percentile response time, and the latter is the 99.9th percentile response time.

Blip uses meta rather than prefixing or suffixing the metric name to ensure that metric names are stable.
For example, in this case, the metric is always `response_time`, not `pN_response_time` or `response_time_pN` where `N` could be any number.
It also allows a greater variety of metrics without special case exceptions to naming or structure.
For example again, average response time could be denoted by meta key `avg`.
