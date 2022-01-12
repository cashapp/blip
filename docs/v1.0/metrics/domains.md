---
layout: default
parent: Metrics
title: Domains
nav_order: 3
---

# Metric Domains
{: .no_toc }

* TOC
{:toc}

---

## access
_Access statistics_

### access.index
_Index access statistics_

Reserved for future use.

### access.table
_Table access statistics_

Reserved for future use.

## aria
_MariaDB Aria Storage Engine_

Reserved for future use.

## autoinc
_Auto-increment Columns_

Not implemented yet but planned.

## aws
_Amazon Web Services_

### aws.rds
_Amazon RDS for MySQL_

### aws.aurora
_Amazon Aurora_

## azure
_Microsoft Azure_

Reserved for future use.

## error
### error.global
### error.query
### error.repl
### error.client

## event
_MySQL Events_

Reserved for future use.

### event.stage
### event.stmt
### event.trx
### event.wait

## file
_Files and Tablespaces_

Reserved for future use.

## galera
_Percona XtraDB Cluster and MariaDB Cluster (wsrep)_

Reserved for future use.

## gcp
_Google Cloud_

Reserved for future use.

## gr
_MySQL Group Replication_

Reserved for future use.

## host
_Host (Client)_

.COUNT_HOST_BLOCKED_ERRORS

## innodb
_InnoDB Metrics_

Metrics from [`INFORMATION_SCHEMA.INNODB_METRICS`](https://dev.mysql.com/doc/refman/en/information-schema-innodb-metrics-table.html).

{: .var-table}
|Blip version|v1.0.0|
|MySQL config|maybe|
|Sources|I_S|
|Group keys||
|Meta|subsystem=`SUBSYSTEM` column|

### innodb.mutex
_InnoDB Mutexes_

Metrics from `SHOW ENGINE INNODB MUTEX`.
Not implement yet.


## mariadb
_MariaDB Enhancements_

Reserved for future use.

## ndb
_MySQL NDB Cluster_

Reserved for future use.

## oracle
_Oracle Cloud and Enterprise Enhancements_

Reserved for future use.

## percona
_Percona Server Enhancements_

Metrics from [Percona User Statistics](https://www.percona.com/doc/percona-server/8.0/diagnostics/user_stats.html).

### percona.userstat.index
_Percona `userstat` Index Statistics_

{: .var-table}
|Blip version|v1.0.0|
|MySQL config|yes|
|Sources|`INFORMATION_SCHEMA.INDEX_STATISTICS`|
|Group keys||
|Meta||


### percona.userstat.table
_Percona `userstat` Table Statistics_

{: .var-table}
|Blip version|v1.0.0|
|MySQL config|yes|
|Sources|`INFORMATION_SCHEMA.TABLE_STATISTICS`|
|Group keys||
|Meta||

## processlist
_Processlist_

SHOW PROCESSLIST; — or — I_S.PROCESSLIST;

## pfs
_Performance Schema_

SHOW ENGINE PERFORMANCE_SCHEMA STATUS;

## pxc
_Percona XtraDB Cluster_

Reserved: use galera.

## query
_Query Metrics_


### query.global
_Global Query Response Time_

{: .var-table}
|Blip version|v1.0.0|
|MySQL config|yes|
|Sources|MySQL 8.0 [p_s.events_statements_histogram_global](https://dev.mysql.com/doc/refman/8.0/en/performance-schema-statement-histogram-summary-tables.html), Percona Server 5.7 [RTD plugin](https://www.percona.com/doc/percona-server/5.7/diagnostics/response_time_distribution.html)|
|Group keys||
|Meta key-values|&bull; `pN=pA`: where `pN` is configured percentile (default: `p999`) and `pA` is actual percentile (see note 1)|

The `query.global` domain includes metrics for all queries, which is currently only response time.
By default, it reports the P999 (99.9th percentile) reponse time using either MySQL 8.0 [performance_schema.events_statements_histogram_global](https://dev.mysql.com/doc/refman/8.0/en/performance-schema-statement-histogram-summary-tables.html) or Percona Server 5.7 [Response Time Distribution plugin](https://www.percona.com/doc/percona-server/5.7/diagnostics/response_time_distribution.html).

Multiple percentiles can be collect&mdash;`p95`, `p99`, and `p999` for example.
The metric for each percentile is denoted by meta key `pN`.

#### Notes

1. MySQL (and Percona Server) use histograms with varible bucket ranges.
Therefore, the P99 might actually be P98.9 or P99.2.
Meta key `pN` indicates the configured percentile, and its value `pA` indicates the actual percentile that was used.

#### Derived Metrics
{: .no_toc }

* `reponse_time`<br>
Type: gauge<br>
Response time for all queries, reported as a percentile (default: P999) in milliseconds.
The true percentile might be slightly more or less depending on how the histogram buckets are configured (see note 1).

### query.id
_Not implemented yet._

The `query.id` domain includes metrics for unique queries identified by digest SHA and set in `meta` as `id`.

## repl
_MySQL Replication_

Not implemented yet.

### repl.lag
_MySQL Replication Lag_

{: .var-table}
|Blip version|v1.0.0|
|Sources|[Blip Heartbeat](../heartbeat), ~~Percona pt-heartbeat, or ~~MySQL~~|
|MySQL config|MySQL: no; other sources: yes|
|Group keys||
|Meta||
|Metrics|&bull; `max` (gauge): Maximum replication lag (milliseconds) observed during collect interval.<br>|

Requires option `source` in the plan; use `%{monitor.meta.repl-source}` like:

```yaml
level:
  collect:
    repl.lag:
      options:
        source: "%{monitor.meta.repl-source}"
```

Then define `config.monitor.meta.repl-source` in the [monitor meta](../config/config-file#meta):
```yaml
monitors:
  - hostname: replica.db
    meta:
      repl-source: source.db
```

## rocksdb
_RocksDB Store Engine_

Reserved for future use.

## size
_Data, Index, and File Sizes_

### size.binlog
_Binary Log Storage Size_

{: .var-table}
|Blip version|v1.0.0|
|Sources|`SHOW BINARY LOGS`|
|MySQL config|no|
|Group keys||
|Meta||

#### Derived Metrics
{: .no_toc }

* `bytes`<br>
Type: gauge<br>
Total size of all binary logs in bytes.

### size.data
_Database and Table Storage Size_

{: .var-table}
|Blip version|v1.0.0|
|MySQL config|no|
|Group keys|`db`, `tbl`|
|Meta||

### size.index
_Index Storage Size_

### size.file
_File Storage Size_

.innodb_undo
.innodb_temp

## status
_MySQL Status Variables_

Classic MySQL status variables.

### status.account
_Status by Account_

Reserved for future use.

### status.global
_Global Status Variables_

{: .var-table}
|Blip version|v1.0.0|
|MySQL config|no|
|Group keys||
|Meta||

Classic `SHOW GLOBAL STATUS`.
Might used `performance_schema.global_status` table.

### status.host
_Status by Host (Client)_

Reserved for future use.

### status.thread
_Status by Thread_

Reserved for future use.

### status.user
_Status by User_

Reserved for future use.

## thd
Threads

Reserved for future use.

## tls
TLS (SSL) Status and Configuration

#### Derived Metrics
{: .no_toc }

* enabled (have_ssl)
* ssl_server_not_before (date-time converted to Unix timestamp)
* ssl_server_not_after	(date-time converted to Unix timestamp)
* current_tls_version

## tokudb
TokuDB Storage Engine

Reserved for future use.

## var.global
_MySQL System Variables_

{: .var-table}
|Blip version|v1.0.0|
|Sources|SHOW, SELECT @@GLOBAL, p_s|
|MySQL config|no|
|Group keys||
|Meta||

Classic MySQL `SHOW GLOBAL VARIBLES`.
