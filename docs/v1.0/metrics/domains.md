---
layout: default
parent: Metrics
title: Domains
---

# Domains
{: .no_toc }

This page documents the metric domains from which Blip currently collects metrics.
Use [`--print-domains`](../config/blip#--print-domains) to list these domains from the command line:

```sh
$ blip --print-domains | less
```

Each domain begins with a table:

Blip version
: Blip version domain was added or changed.

MySQL config
: * _no_ = No configuration required; all metrics available with default MySQL configuration
* _required_ = Metrics require MySQL configuration as documented
* _optional_ = Limited metrics unless MySQL configured as documented

Sources
: MySQL source of metrics.

Group keys
: [Metric groups](reporting#groups). Omitted if none.

Meta
: [Metric meta](reporting#meta). Omitted if none.

Error policy
: MySQL error codes handled by optional [error policy](../plans/error-policy). Omitted if none.

Derived metrics
: [Derived metrics](collecting#derived-metrics). Omitted if none.

---

* TOC
{:toc}

---

{: .config-section-title }
## aws.rds
_Amazon RDS for MySQL_

{: .var-table}
|Blip version|v1.0.0|
|MySQL config|no|
|Sources|Amazon RDS API|

Collects [Amazon RDS metrics](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/monitoring-cloudwatch.html#rds-metrics).

{: .config-section-title }
## innodb
_InnoDB Metrics_

{: .var-table}
|Blip version|v1.0.0|
|MySQL config|maybe|
|Sources|`information_schema.innodb_metrics`|
|Meta|&bull; `subsystem=<SUBSYSTEM column>`|

Metrics from [`INFORMATION_SCHEMA.INNODB_METRICS`](https://dev.mysql.com/doc/refman/en/information-schema-innodb-metrics-table.html).

{: .config-section-title }
## query.global
_Global Query Response Time_

{: .var-table}
|Blip version|v1.0.0|
|MySQL config|yes|
|Sources|MySQL 8.0 [p_s.events_statements_histogram_global](https://dev.mysql.com/doc/refman/8.0/en/performance-schema-statement-histogram-summary-tables.html), Percona Server 5.7 [RTD plugin](https://www.percona.com/doc/percona-server/5.7/diagnostics/response_time_distribution.html)|
|Meta key-values|&bull; `pN=pA`: where `pN` is configured percentile (default: `p999`) and `pA` is actual percentile (see note 1)|
|Derived metrics|&bull; `response_time` (gauge)<br>|

The `query.global` domain includes metrics for all queries, which is currently only response time.
By default, it reports the P999 (99.9th percentile) response time using either MySQL 8.0 [performance_schema.events_statements_histogram_global](https://dev.mysql.com/doc/refman/8.0/en/performance-schema-statement-histogram-summary-tables.html) or Percona Server 5.7 [Response Time Distribution plugin](https://www.percona.com/doc/percona-server/5.7/diagnostics/response_time_distribution.html).

Multiple percentiles can be collected&mdash;`p95`, `p99`, and `p999` for example.
The metric for each percentile is denoted by meta key `pN`.

{: .note}
To convert units, use the [TransformMetrics plugin](../integrate#transformmetrics) or write a [custom sink](../sinks/custom).

#### Derived metrics
{: .no_toc }

* `reponse_time`<br>
Type: gauge<br>
Response time for all queries, reported as a percentile (default: P999) in microseconds.
The true percentile might be slightly more or less depending on how the histogram buckets are configured (see note 1).

#### Notes
{: .no_toc }

1. MySQL (and Percona Server) use histograms with variable bucket ranges.
Therefore, the P99 might actually be P98.9 or P99.2.
Meta key `pN` indicates the configured percentile, and its value `pA` indicates the actual percentile that was used.

{: .config-section-title}
## repl
_MySQL Replication_

{: .var-table}
|Blip version|v1.0.0|
|Sources|&#8805;&nbsp;MySQL 8.0.22: `SHOW REPLICA STATUS`<br>&#8804;&nbsp;MySQL 8.0.21: `SHOW SLAVE STATUS`|
|MySQL config|no|
|Meta||
|Derived metrics|&bull; `running` (gauge)|

The `repl` domain reports a few gauges metrics from the output of `SHOW SLAVE STATUS` (or `SHOW REPLICA STATUS` as of MySQL 8.0.22):

|Replica Status Variable|Collected|
|-----------------------|---------|
|Slave_IO_Running       |&#10003;|
|Slave_SQL_Running      |&#10003;|
|Relay_Log_Space        |&#10003;|
|Seconds_Behind_Master  |&#10003;|
|Auto_Position          |&#10003;|

Although the output has many more fields, most fields are not metric counters or gauges, which is why Blip does not collect them.

#### Derived metrics
{: .no_toc }

* `running`<br>
Type: gauge<br>

  |Value|Meaning|
  |-----|-------|
  |1|&nbsp;&nbsp;&#9745;	 MySQL is a replica<br>&nbsp;&nbsp;&#9745;	 `Slave_IO_Running=Yes`<br>&nbsp;&nbsp;&#9745;	 `Slave_SQL_Running=Yes`<br>&nbsp;&nbsp;&#9745;	 `Last_Errno=0`<br>|
  |0|MySQL is a replica, but IO and SQL threads are not running or a replication error occurred|
  |-1|MySQL is **not a replica**: `SHOW SLAVE|REPLICA STATUS` returns no output|

  Replication lag does not affect the `running` metric: replication can be running but lagging.

{: .config-section-title}
## repl.lag
_MySQL Replication Lag_

{: .var-table}
|Blip version|v1.0.0|
|Sources|[Blip Heartbeat](../heartbeat)|
|MySQL config|yes|
|Meta|&bull; `source=<src_id column>`|
|Derived metrics|&bull; `current` (gauge): Current replication lag (milliseconds).<br>|
|Options|&bull; `network-latency`<br>&bull; `repl-check`<br>&bull; `report-no-heartbeat`<br>&bull; `source-id`<br>&bull; `source-role`<br>&bull; `table`<br>&bull; `writer`|

The `repl.lag` collector measures and reports MySQL replication lag from a source using the [Blip heartbeat](../heartbeat).
By default, it reports replication lag from the latest timestamp (heartbeat), which presumes there is only one writable node in the replication topology at all times.
See [Heartbeat](../heartbeat) to learn more.

#### Derived metrics
{: .no_toc }

* `current`<br>
Type: gauge<br>
The current replication lag in milliseconds.
This is an instantaneous measurement: replication lag at one moment.
As such, it might not detect if lag is "flapping": oscillating between near-zero and a higher value.
But will always detect if replication is steadily lagged and if the lag increases.
A future feature might measure and record lag between report intervals.

#### Options
{: .no_toc }

* `network-latency`<br>
Default: 50<br>
Network latency (in milliseconds) between source and replicas.
The value must be an integer >= 0.
(Do not suffix with "ms".)
See [Heartbeat > Accuracy](../heartbeat#accuracy).
* `repl-check`<br>
MySQL global system variable, like `server_id`.
(Do not prefix with "@".)
If the value is zero, replica lag is not collected.
See [Heartbeat > Repl Check](../heartbeat#repl-check).
* `report-no-heartbeat`<br>
Default: no<br>
If yes, no heartbeat from the source is reported as value -1.
If no, the metric is dropped if no heartbeat from the source.
* `source-id`<br>
Source ID to report lag from.
The default (no value) reports lag from the latest (most recent) timestamp.
See [Heartbeat > Source Following](../heartbeat#source-following).
* `source-role`<br>
Source role to report lag from.
If set, the most recent timestamp is used.
See [Heartbeat > Source Following](../heartbeat#source-following).
* `table`<br>
Default: `blip.heartbeat`<br>
Blip [heartbeat table](../heartbeat#table).
* `writer`<br>
Default: `blip`<br>
Type of heartbeat writer.
Only `blip` is currently supported.

{: .config-section-title }
## size.binlog
_Binary Log Storage Size_

{: .var-table}
|Blip version|v1.0.0|
|Sources|`SHOW BINARY LOGS`|
|MySQL config|no|
|Derived metrics|&bull; `bytes`: Total size of all binary logs in bytes.|
|Error policy|&bull; `access-denied`<br>&bull; `binlog-not-enabled`|

#### Derived metrics
{: .no_toc }

* `bytes`<br>
Type: gauge<br>
Total size of all binary logs in bytes.

#### Error Policy
{: .no_toc }

* `access-denied`
MySQL error 1227: access denied on `SHOW BINARY LOGS`.

* `binlog-not-enabled`
MySQL error 1381: binary logging not enabled.

{: .config-section-title }
## size.database
_Database Storage Sizes_

{: .var-table}
|Blip version|v1.0.0|
|MySQL config|no|
|Group keys|`db`|
|Derived metrics|&bull; `bytes`: Database size in bytes.|

#### Derived metrics
{: .no_toc }

* `bytes`<br>
Type: gauge<br>
Database size in bytes.

{: .config-section-title }
## size.table
_Table Storage Sizes_

{: .var-table}
|Blip version|v1.0.0|
|MySQL config|no|
|Group keys|`db`, `tbl`|
|Derived metrics|&bull; `bytes`: Table size in bytes.|

#### Derived metrics
{: .no_toc }

* `bytes`<br>
Type: gauge<br>
Table size in bytes.

{: .config-section-title }
## status.global
_Global Status Variables_

{: .var-table}
|Blip version|v1.0.0|
|MySQL config|no|

`status.global` collects the primary source of MySQL server metrics: `SHOW GLOBAL STATUS`.

{: .config-section-title }
## stmt.current
_Statement Metrics_

Statements are the second level of the event hierarchy:

```
transactions
└── statements
    └── stages
        └── waits
```

All queries are statements, but not all statements are queries.
For example, "dump binary log" is a statement used by replicas, but it is not a query in the typical sense.
As a result, this domain is much more low-level than the [`query`](#query) domain even though the metrics are nearly identical.

Statement metrics are reported as summary statistics: average, maximum, and so forth.

`stmt.current` reports summary statistics for currently running statements.

{: .config-section-title }
## tls
_TLS (SSL) Status and Configuration_

Not implemented yet but planned.

#### Derived metrics
{: .no_toc }

* enabled (have_ssl)
* ssl_server_not_before (date-time converted to Unix timestamp)
* ssl_server_not_after	(date-time converted to Unix timestamp)
* current_tls_version

{: .config-section-title }
## var.global
_MySQL System Variables_

{: .var-table}
|Blip version|v1.0.0|
|Sources|`SHOW GLOBAL VARIABLES`, `SELECT @@GLOBAL.<var>`, Performance Schema|
|MySQL config|no|

`var.global` collects global MySQL system variables ("sysvars").

These are not technically metrics, but some are required to calculate utilization percentages.
For example, it's common to report `max_connections` to gauge the percentage of max connections used: `Max_used_connections / max_connections * 100`, which would be `status.global.max_used_connections / var.global.max_connections * 100` in Blip metric naming convention.
