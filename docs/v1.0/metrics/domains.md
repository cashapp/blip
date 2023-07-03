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
: If MySQL must be explicitly or specially configured to provide the metrics.

Sources
: MySQL source of metrics.

Derived metrics
: [Derived metrics](collecting#derived-metrics). Omitted if none.

Group keys
: [Metric groups](reporting#groups). Omitted if none.

Meta
: [Metric meta](reporting#meta). Omitted if none.

Options
: [Domain options](collecting#options). Omitted if none.

Error policy
: MySQL error codes handled by optional [error policy](../plans/error-policy). Omitted if none.

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

<!-------------------------------------------------------------------------->

{: .config-section-title }
## innodb
_InnoDB Metrics_

{: .var-table}
|Blip version|v1.0.0|
|MySQL config|maybe|
|Sources|`information_schema.innodb_metrics`|
|Meta|&bull; `subsystem` = `SUBSYSTEM` column|
|Options|&bull; `all`|

Metrics from [`INFORMATION_SCHEMA.INNODB_METRICS`](https://dev.mysql.com/doc/refman/en/information-schema-innodb-metrics-table.html).

#### Options
{: .no_toc }

* `all`<br>
Default: `no`<br>
If `yes`, all InnoDB metrics are collect&mdash;the whole table.
If `no` (the default), only the explicitly listed InnoDB metrics are collected.
If `enabled`, only InnoDB metrics enabled by the MySQL configuration are collected (`WHERE status='enabled'` in the table).

<!-------------------------------------------------------------------------->

{: .config-section-title }
## percona.response-time
_Percona Server Query Response Time_

{: .var-table}
|Blip version|v1.0.0|
|MySQL config|yes|
|Sources|Percona Server 5.7 [RTD plugin](https://www.percona.com/doc/percona-server/5.7/diagnostics/response_time_distribution.html)|
|Derived metrics|&bull; `pN` (gauge) for each value in the `percentiles` option|
|Meta|&bull; `pN=pA`: where `pN` is collected percentile and `pA` is actual percentile|
|Options|&bull; `flush`<br>&bull; `real-percentiles`|
|Error policy|&bull; `unknown-table`|

The `percona.response-time` domain collects query response time percentiles from the Percona Server 5.7 [Response Time Distribution plugin](https://www.percona.com/doc/percona-server/5.7/diagnostics/response_time_distribution.html).

This domain is functionally identical to [`query.response-time`](#queryresponse-time); only one option name is different:

|`percona.response-time`|`query.response-time`|
|-----------------------|---------------------|
|`flush`|`truncate-table`|

See [`query.response-time`](#queryresponse-time) for details.

#### Error Policy
{: .no_toc }

* `unknown-table`<br>
MySQL error 1109: Unknown table 'query_response_time' in information_schema

<!-------------------------------------------------------------------------->

{: .config-section-title }
## query.response-time
_MySQL Query Response Time_

{: .var-table}
|Blip version|v1.0.0|
|MySQL config|yes|
|Sources|MySQL 8.0 [p_s.events_statements_histogram_global](https://dev.mysql.com/doc/refman/8.0/en/performance-schema-statement-histogram-summary-tables.html)|
|Derived metrics|&bull; `pN` (gauge)<br>|
|Meta|&bull; `pN=pA`: where `pN` is collected percentile and `pA` is actual percentile|
|Options|&bull; `real-percentiles`<br>&bull; `truncate-table`<br>&bull; `truncate-timeout`|
|Error policy|&bull; `table-not-exist`<br>&bull; `truncate-timeout`|

The `query.response-time` domain collect query response time percentiles.
By default, it reports the P999 (99.9th percentile) response time in microseconds.

{: .note}
To convert units, use the [TransformMetrics plugin](../integrate#transformmetrics) or write a [custom sink](../sinks/custom).

#### Derived metrics
{: .no_toc }

* `pN`<br>
Type: gauge<br>
Response time percentile to collect where `N` between 1 and 999.
(The "p" prefix is required.)
`p95` collects the 95th percentile.
`p999` collects the 99.9th percentile.
The response time value is reported in microseconds.
The true percentile might be slightly greater depending on how the histogram buckets are configured.
For example, if collecting `p95`, the real percentile might be `p95.8`.

#### Options
{: .no_toc }

* `real-percentiles`<br>
Default: yes<br>
If yes (default), reports the real percentile in meta for each percentile in options.
MySQL (and Percona Server) use histograms with variable bucket ranges.
Therefore, the P99 might actually be P98.9 or P99.2.
Meta key `pN` indicates the configured percentile, and its value `pA` indicates the actual percentile that was used.

* `truncate-table`<br>
Default: yes<br>
Truncate [performance_schema.events_statements_histogram_global](https://dev.mysql.com/doc/refman/8.0/en/performance-schema-statement-histogram-summary-tables.html) after each collection.
This resets percentile values so that each collection represents the global query response time during the collection interval rather than during the entire uptime of the MySQL.
However, truncating the table interferes with other tools reading (or truncating) the table.

* `truncate-timeout`<br>
Default: 250ms<br>
The amount of time to wait while attempting to truncate [performance_schema.events_statements_histogram_global](https://dev.mysql.com/doc/refman/8.0/en/performance-schema-statement-histogram-summary-tables.html).
Normally, truncating a table is nearly instantaneous, but metadata locks can block the operation.

#### Error Policy
{: .no_toc }

* `table-not-exist`<br>
MySQL error 1146: Table 'performance_schema.events_statements_histogram_global' doesn't exist

* `truncate-timeout`<br>
Truncation failures on table `performance_schema.events_statements_histogram_global`

<!-------------------------------------------------------------------------->

{: .config-section-title}
## repl
_MySQL Replication_

{: .var-table}
|Blip version|v1.0.1|
|Sources|&#8805;&nbsp;MySQL 8.0.22: `SHOW REPLICA STATUS`<br>&#8804;&nbsp;MySQL 8.0.21: `SHOW SLAVE STATUS`|
|MySQL config|no|
|Derived metrics|&bull; `running` (gauge)|
|Meta|&bull; `source` = `Source_Host` or `Master_Host`|
|Options|&bull; `report-not-a-replica`<br>|

The `repl` collects replication metrics.   Currently, it collects a single derived metric: `running` (described below).

A future release will collect these MySQL metrics:

|Replica Status Variable|Collected|
|-----------------------|---------|
|Slave_IO_Running       |&#10003;|
|Slave_SQL_Running      |&#10003;|
|Relay_Log_Space        |&#10003;|
|Seconds_Behind_Master  |&#10003;|
|Auto_Position          |&#10003;|

#### Derived metrics
{: .no_toc }

* `running`<br>
Type: gauge<br>

  |Value|Meaning|
  |-----|-------|
  |1|&nbsp;&nbsp;&#9745;MySQL is a replica<br>&nbsp;&nbsp;&#9745;`Slave_IO_Running=Yes`<br>&nbsp;&nbsp;&#9745;`Slave_SQL_Running=Yes`<br>&nbsp;&nbsp;&#9745;`Last_Errno=0`<br>|
  |0|MySQL is a replica, but IO and SQL threads are not running or a replication error occurred|
  |-1|MySQL is **not a replica**: `SHOW SLAVE|REPLICA STATUS` returns no output|

  Replication lag does not affect the `running` metric: replication can be running but lagging.

#### Options
{: .no_toc }

* `report-not-a-replica`<br>
Default: no<br>
If yes, report `repl.running = -1` if not a replica.
If no, drop the metric if not a replica.

<!-------------------------------------------------------------------------->

{: .config-section-title}
## repl.lag
_MySQL Replication Lag_

{: .var-table}
|Blip version|v1.0.0|
|Sources|[Blip Heartbeat](../heartbeat)|
|MySQL config|yes|
|Derived metrics|&bull; `current` (gauge): Current replication lag (milliseconds)<br>|
|Meta|&bull; `source` = Option `source-id`|
|Options|&bull; `network-latency`<br>&bull; `repl-check`<br>&bull; `report-no-heartbeat`<br>&bull; `report-not-a-replica`<br>&bull; `source-id`<br>&bull; `source-role`<br>&bull; `table`<br>&bull; `writer`|

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

* `report-not-a-replica`<br>
Default: no<br>
If yes, report `repl.running = -1` if not a replica.
If no, drop the metric if not a replica.

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

<!-------------------------------------------------------------------------->

{: .config-section-title }
## size.binlog
_Binary Log Storage Size_

{: .var-table}
|Blip version|v1.0.0|
|Sources|`SHOW BINARY LOGS`|
|MySQL config|no|
|Derived metrics|&bull; `bytes`: Total size of all binary logs in bytes|
|Error policy|&bull; `access-denied`<br>&bull; `binlog-not-enabled`|

#### Error Policy
{: .no_toc }

* `access-denied`
MySQL error 1227: access denied on `SHOW BINARY LOGS`.

* `binlog-not-enabled`
MySQL error 1381: binary logging not enabled.

#### Derived metrics
{: .no_toc }

* `bytes`<br>
Type: gauge<br>
Total size of all binary logs in bytes.

<!-------------------------------------------------------------------------->

{: .config-section-title }
## size.database
_Database Storage Sizes_

{: .var-table}
|Blip version|v1.0.0|
|MySQL config|no|
|Derived metrics|&bull; `bytes`: Database size in bytes|
|Group keys|`db`|

#### Derived metrics
{: .no_toc }

* `bytes`<br>
Type: gauge<br>
Database size in bytes.

<!-------------------------------------------------------------------------->

{: .config-section-title }
## size.table
_Table Storage Sizes_

{: .var-table}
|Blip version|v1.0.0|
|MySQL config|no|
|Derived metrics|&bull; `bytes`: Table size in bytes|
|Group keys|`db`, `tbl`|

#### Derived metrics
{: .no_toc }

* `bytes`<br>
Type: gauge<br>
Table size in bytes.

<!-------------------------------------------------------------------------->

{: .config-section-title }
## status.global
_Global Status Variables_

{: .var-table}
|Blip version|v1.0.0|
|MySQL config|no|
|Sources|`SHOW GLOBAL STATUS`|

`status.global` collects the primary source of MySQL server metrics: `SHOW GLOBAL STATUS`.

<!-------------------------------------------------------------------------->

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

<!-------------------------------------------------------------------------->

{: .config-section-title }
## tls
_TLS (SSL) Status and Configuration_

{: .var-table}
|Blip version|v1.0.0|
|MySQL config|no|
|Sources|Global variables|
|Derived metrics|&bull; `enabled`: True (1) if have_ssl=YES, else false (0)|

#### Derived metrics
{: .no_toc }

* `enabled`<br>
Type: bool<br>
True (1) if `have_ssl = YES`, else false (0).

{: .note }
`have_ssl` is deprecated as of MySQL 8.0.26.
This domain does not currently support the [`tls_channel_status` table](https://dev.mysql.com/doc/refman/8.0/en/performance-schema-tls-channel-status-table.html).

<!-------------------------------------------------------------------------->

{: .config-section-title }
## trx
_Transactions_

{: .var-table}
|Blip version|v1.0.0|
|MySQL config|no|
|Sources|`information_schema.innodb_trx`|
|Derived metrics|&bull; `oldest`: Time of oldest active trx in seconds|

#### Derived metrics
{: .no_toc }

* `oldest`<br>
Type: gauge<br>
Time of oldest active (still running) transaction in seconds.

<!-------------------------------------------------------------------------->

{: .config-section-title }
## var.global
_MySQL System Variables_

{: .var-table}
|Blip version|v1.0.0|
|MySQL config|no|
|Sources|`SHOW GLOBAL VARIABLES`, `SELECT @@GLOBAL.<var>`, Performance Schema|

`var.global` collects global MySQL system variables ("sysvars").

These are not technically metrics, but some are required to calculate utilization percentages.
For example, it's common to report `max_connections` to gauge the percentage of max connections used: `Max_used_connections / max_connections * 100`, which would be `status.global.max_used_connections / var.global.max_connections * 100` in Blip metric naming convention.

<!-------------------------------------------------------------------------->

{: .config-section-title }
## wait.io.table
_Table I/O Wait Metrics_

{: .var-table}
|Blip version|v1.0.0|
|MySQL config|yes|
|Sources|`performance_schema.table_io_waits_summary_by_table`|
|Options|&bull; `exclude`<br>&bull; `include`<br>&bull; `truncate`<br>&bull; `truncate-timeout`<br>&bull; `all`|
|Error policy|&bull; `truncate-timeout`|
|Group keys|`db`, `tbl`|

Summarized table I/O wait metrics from `performance_schema.table_io_waits_summary_by_table`.
All columns in that table can be specified, or use option `all` to collect all columns.

#### Options
{: .no_toc }

* `include`<br>
A comma-separated list of database or table names to include (overrides option `exclude`).

* `exclude`<br>
Default: `mysql.*,information_schema.*,performance_schema.*,sys.*`<br>
A comma-separated list of database or table names to exclude (ignored if `include` is set).

* `truncate-table`<br>
Default: `yes`<br>
If the source table should be truncated to reset data after each retrieval.

* `truncate-timeout`<br>
Default: 250ms<br>
The amount of time to wait while attempting to truncate [performance_schema.events_statements_histogram_global](https://dev.mysql.com/doc/refman/8.0/en/performance-schema-statement-histogram-summary-tables.html).
Normally, truncating a table is nearly instantaneous, but metadata locks can block the operation.

* `all`<br>
Default: `no`<br>
If `yes`, all `performance_schema.table_io_waits_summary_by_table` metrics are collected&mdash;all columns.
If `no` (the default), only the explicitly listed `performance_schema.table_io_waits_summary_by_table` metrics are collected.

#### Error Policy
{: .no_toc }

* `truncate-timeout`<br>
Truncation failures on table `performance_schema.events_statements_histogram_global`
