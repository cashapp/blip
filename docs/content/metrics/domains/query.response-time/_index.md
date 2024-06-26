---
title: "query.response-time"
---

The `query.response-time` domain includes query response time percentiles from MySQL 8.x Performance Schema table [`events_statements_histogram_global`](https://dev.mysql.com/doc/refman/8.0/en/performance-schema-statement-histogram-summary-tables.html).

{{< toc >}}

## Usage

By default, this domain collects and reports the P999 (99.9th percentile) response time in microseconds.

{{< hint type=tip >}}
To convert units, use the [TransformMetrics plugin]({{< ref "develop/integration-api#transformmetrics" >}}) or write a [custom sink]({{< ref "develop/sinks" >}}).
{{< /hint >}}

Unlike other domains, all metrics are derived and specified as percentiles to collect, like:

```yaml
level:
  freq: 5s
  collect:
    query.response-time:
      metrics:
        - p50
        - p95
        - p99
```

The example plan above collects the P50 (median), P95, and P99 query response time percentiles.
The "p" prefix is required.
The value after "p" is an integer between 1 and 999, inclusive.
Blip creates and reports a derived metric for each, like:

```
p50 = 500
p95 = 7123
p99 = 9098
```

The true percentile might be slightly greater depending on how the histogram buckets are configured.
For example, if collecting `p95`, the real percentile might be `p95.8`.
See option [`real-percentiles`](#real-percentiles).

## Derived Metrics

### `pN`

| | |
|---|---|
|**Metric Type**|gauge|
|**Value Units**|microseconds|

For each configured percentile in the plan, Blip reports a corresponding `pN` metric where "N" is the collect (not real) percentile.
See [Usage](#usage) above.

## Options

### `real-percentiles`

|Value|Default|Description|
|---|---|---|
|yes|&check;|Report real percentile in [meta](#meta) for each percentile in options|
|no| |Don't report real percentiles|

MySQL (and Percona Server) use histograms with variable bucket ranges.
Therefore, the P99 might actually be P98.9 or P99.2.
Meta key `pN` indicates the configured percentile, and its value `pA` indicates the actual percentile that was used.

### `truncate-table`

|Value|Default|Description|
|---|---|---|
|yes|&check;|Truncate table each interval|
|no| |Never truncate table|

Truncate [performance_schema.events_statements_histogram_global](https://dev.mysql.com/doc/refman/8.0/en/performance-schema-statement-histogram-summary-tables.html) after each collection.
This resets percentile values so that each collection represents the global query response time during the collection interval rather than during the entire uptime of the MySQL.
However, truncating the table interferes with other tools reading (or truncating) the table.

### `truncate-timeout`

| | |
|---|---|
|**Value Type**|[Duration string](https://pkg.go.dev/time#ParseDuration)|250ms|
|**Default**|250ms|

Sets `@@session.lock_wait_timeout` to avoid waiting too long when truncating [performance_schema.events_statements_histogram_global](https://dev.mysql.com/doc/refman/8.0/en/performance-schema-statement-histogram-summary-tables.html).
Has no effect when [`truncate-table`](#truncate-table) = no.

Normally, truncating a table is nearly instantaneous, but metadata locks can block the operation.

## Group Keys

None.

## Meta

|Key|Value|Description|
|---|-----|-----------|
`pN`|`pA`|`pN` is collected percentile and `pA` is real percentile|

For example, if collecting P95, meta might contain `p95 = p95.8`: the real percentile is P95.8 because MySQL uses a fixed number of buckets to collect and calculate percentiles, and the bucket boundaires don't line up pefectly.

## Error Policies

|Name|MySQL Error|
|----|-----------|
|`table-not-exist`|1146: Table 'performance_schema.events_statements_histogram_global' doesn't exist|
|`truncate-timeout`|Truncation failures on table `performance_schema.events_statements_histogram_global`|

## MySQL Config

Requires Performance Schema.
See [29.1 Performance Schema Quick Start](https://dev.mysql.com/doc/refman/en/performance-schema-quick-start.html) and related pages in the MySQL manual.

## Changelog

|Blip Version|Change|
|------------|------|
|v1.0.2      |Added [`truncate-timeout`](#truncate-timeout)|
|v1.0.0      |Domain added|
