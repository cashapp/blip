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

## percona.stats
_User Statistics_

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

* `reponse_time`<br>
Type: gauge<br>
Reponse time for all queries, reported as a percentile (default: P999) in millliseconds.
The true percentile might be slightly more or less depending on how the histogram buckets are configured (see note 1).

### query.id
_Not implemented yet._

The `query.id` domain includes metrics for unique queires identified by digest SHA and set in `meta` as `id`.

## repl
_MySQL Replication_

{: .var-table}
|Blip version|v1.0.0|
|Sources|MySQL, Percona pt-heartbeat, Blip Heartbeat|
|MySQL config|MySQL: no; other sources: yes|
|Group keys||
|Meta||

The `repl` metric domain includes metrics related to classic MySQL (not Group Replication).
The primary metric, which is derived from various sources, is `lag` (in milliseconds).

#### Required Options
{: .no_toc }

* `source`: Source MySQL instance

#### Derived Metrics
{: .no_toc }

* `lag`<br>
Type: gauge

* `running`<br>
Type: gauge (bool)<br>
1 if replica is running, else 0

## rocksdb
_RocksDB Store Engine_

Reserved for future use.

## size
_Data, Index, and File Sizes_

### size.binlog
_Binary Log Storage Size_

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
