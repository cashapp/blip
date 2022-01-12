---
layout: default
parent: Metrics
title: Conventions
nav_order: 2
---

# Conventions
{: .no_toc }

Blip conventions provide consistency and structure to make writing plans and reporting metrics easier.
Although the [built-in sinks](../sinks/) report fully-qualified metric names (`status.global.threads_running`), your [custom sink](../sinks/custom) can rename and report metrics however you want.
For example, your sink could ignore Blip domains completely and report only metric names (`threads_running`), or report a simpler custom prefix (`mysql.threads_running`).

* TOC
{:toc}

---

## Metrics

### Naming

Blip strives to report MySQL metric names as-is&mdash;no modifications&mdash;so that what you see in MySQL is what you get in Blip.

However, MySQL metric names are very inconsistent:

* `Foo_bar` (most common)
* `Foo_Bar` (replica status)
* `foo_bar` (InnoDB metrics)
* `foo_bar_count` (type suffix)
* `foo_bar_usec` (unit suffix)

For consistency, Blip metric names have three requirements:

1. Only `snake_case`
1. Always lowercase
1. No prefixes or suffixes

A fully-qualified metric name includes a domain: `status.global.threads_running`.
The metric name is always the last field (split on `.`).

### Types

Blip metric types are standard:

* UNKNOWN
* COUNTER
* GAUGE
* BOOL
* EVENT

Most metrics are counters; a few are gauges.
Blip, like MySQL, does not distinguish between "counter" and "cumulative counter".
Blip counter metrics can reset to zero if MySQL is restarted; otherwise, the value only increases.

Currently, Blip does not report bool or event metrics, but these are reserved for future use.

The unknown type is used only for error detection.

### Values

All values, regardless of type, are `float64`.

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
To convert units, use the [TransformMetrics plugin](../integrate#transformmetrics) or write a [custom sink](../sinks/custom).

Blip does _not_ suffix metric names with units, and it does not strip the few MySQL metrics that have unit suffixes.

## Collector Metrics

Most metrics are MySQL metrics: Blip simply reports the MySQL metric (after making it lowercase).
For example, MySQL metric `Threads_running` is simply `threads_running`, and InnoDB metric `trx_rseg_history_len` is simply `trx_rseg_history_len`.
Even in cases like `trx_rseg_history_len`, Blip never renames MySQL metrics to something more friendly, like `history_list_length`, which the MySQL metric really represents.

{: .note}
Blip never renames MySQL metrics.

**MySQL metrics are _not_ listed in [Domains](domains); see the listed MySQL sources for metrics.**

_Collector metrics_ are metrics created by the collector, usually derived from MySQL metrics.
For example, the [`repl.lag` collector](domains#repllag) reports a metric named `lag`.
There is no MySQL metric by that name; it's created by the collector.

To rename metrics, use the [TransformMetrics plugin](../integrate#transformmetrics) or write a [custom sink](../sinks/custom).

## Domain Naming

Blip metric domain names have three requirements:

1. Always lowercase
1. One word: `[a-z]+`
1. Singular noun: "size" not "sizes"; "query" not "queries"

Common abbreviation and acrynomym are prefered, especially when they match MySQL usage: "thd" not "thread"; "pfs" not "performanceschema"; and so on.

{: .note }
Currently, domain names fit this convention, but if a need arises to allow hyphenation ("domain-name"), it might be allowed.
Snake case ("domain_name") and camel case ("domainName") are not allowed: the former is used by metrics, and the latter is not Blip style.

### Sub-domains

Blip uses sub-domains for two purposes: MySQL-grouped metrics, or metrics that are related but different.

The [`error` domain](domains#error) is an exmaple of metrics that are related by different.
[`error.query`](domains#errorquery) and [`error.repl`](domains#errorepl) both comprise error-related metrics, hence the common root domain, but the specific metrics for each are different.

The[`status` domain](domains#status) is an example of MySQL-grouped metrics.
MySQL provides status metrics grouped by account, global, host, thread, and user.
(_Global_ is the most common, as in `SHOW GLOBAL STATUS`.)
Blip has a sub-domain for each group&mdash;`status.account`, `status.global`, and so on&mdash;that makes advacned plans like the following possible:

```yaml
level:
  collect:
    status.global:
      options:
        all: yes
    status.host:
      options:
        host: 10.1.1.1
      metrics:
        - queries
        - threads_running
```

The plan snippet above collects all global status metrics (`status.global`) but only two status metrics for host 10.1.1.1 (`status.host`).

MySQL-grouped metrics are an _explicit group_: `status.host` explicitly groups by `host`.
See [Reporting > Groups](reporting#groups) for more details.

{: .note}
For simplicitly, sub-domains are called "domains" in the rest of the docs.
The term is only used here to clarify the distinction and usage.
