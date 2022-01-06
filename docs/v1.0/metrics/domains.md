---
layout: default
parent: Metrics
title: Domain Reference
nav_order: 3
---

# Metric Domains
{: .no_toc }

* TOC
{:toc}

---

## aria
MariaDB Aria Storage Engine

Reserved for future use.

## autoinc
Auto-increment Columns

Dimensions:	db, tbl
Metrics:
cur:
.cur	dim: db || tbl
.max

## aws
Amazon Web Services

### aws.rds
Amazon RDS for MySQL

### aws.aurora
Amazon Aurora

## azure
Microsoft Azure

Reserved for future use.

## error
### error.global
### error.query
### error.repl
### error.client

## event

Reserved for future use.
### event.stage
### event.stmt
### event.trx
### event.wait

## file
Files and Tablespaces

Reserved for future use.

##galera
Percona XtraDB Cluster and MariaDB Cluster (wsrep)

Reserved for future use.

## gcp
Google Cloud

Reserved for future use.

## gr
Group Replication

Reserved for future use.

## host
Host (Client)

.COUNT_HOST_BLOCKED_ERRORS

## innodb
InnoDB

INFORMATION_SCHEMA.INNODB_METRICS

.buffer.buffer_flush_adaptive_total_pages

.log.log_lsn_checkpoint_age

.transaction.trx_rseg_history_len

.ahi

### innodb.mutex
InnoDB Mutexes

SHOW ENGINE INNODB MUTEX
.redo_rseg.waits

## mariadb
MariaDB Enhancements

Reserved for future use.

## ndb
MySQL NDB Cluster

Reserved for future use.

## oracle
Oracle Cloud and Enterprise Enhancements

Reserved for future use.

## percona

Percona Server Enhancements

## percona.stats
User Statistics
.percona.stats.client
.percona.stats.idx
.percona.stats.tbl
.percona.stats.thd
.percona.stats.user


## processlist
Processlist

SHOW PROCESSLIST; — or — I_S.PROCESSLIST;

## pfs
Performance Schema

SHOW ENGINE PERFORMANCE_SCHEMA STATUS;

## pxc
Percona XtraDB Cluster

Reserved: use galera.

## query
.response_time	{p999}

### query.global
### query.id
			{id=<SHA>}

## repl
Replication

.running
.lag-ms	Lag in milliseconds

## rocksdb
RocksDB Store Engine

Reserved for future use.

## size

Storage Size (Bytes)

### size.binlog
Binary Log Storage Size

### size.data
Database and Table Storage Size

### size.index
Index Storage Size

### size.file

File Storage Size

.innodb_undo
.innodb_temp

## status
MySQL Server Status

### status.account
Status by Account

Reserved for future use.

### status.global
SHOW GLOBAL STATUS — or — P_S.GLOBAL_STATUS
.com_select
.threads_running
.innodb_log_waits
.queries

### status.host
Status by Host (Client)

Reserved for future use.

### status.thread
Status by Thread

Reserved for future use.

### status.user
Status by User

Reserved for future use.

## thd
Threads

Reserved for future use.

## tls
TLS (SSL) Status and Configuration

enabled (have_ssl)
ssl_server_not_before (date-time converted to Unix timestamp)
ssl_server_not_after	(date-time converted to Unix timestamp)
current_tls_version

## tokudb
TokuDB Storage Engine

Reserved for future use.

## var.global
MySQL System Variables

SELECT @@GLOBAL.var — SHOW GLOBAL VARIABLES — P_S.GLOBAL_VARIABLES
innodb_log_file_size
max_connections
sync_binlog
