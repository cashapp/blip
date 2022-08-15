---
layout: default
parent: Metrics
title: Quick Reference
nav_order: 100
---

# Quick Reference

{: .no_toc }

Following are _all_ Blip domains and the metrics collected in each.
Only domains with a Blip version are collected.
The rest are reserved for future use.

|Domain|Metrics|Blip Version|
|:-----|:------|:-----------|
|accesss|Access statistics||
|access.index|Index access statistics (`sys.schema_index_statistics`)||
|access.table|Table access statistics (`sys.schema_table_statistics`)||
|aria|MariaDB Aria storage engine||
|autoinc|Auto-increment column limits||
|aws|Amazon Web Services||
|[`aws.rds`](collectors#awsrds)|[Amazon RDS metrics](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/monitoring-cloudwatch.html#rds-metrics)|v1.0.0|
|aws.aurora|Amazon Aurora||
|azure|Microsoft Azure||
|error|MySQL, client, and query errors||
|error.client|Client errors||
|error.global|Global error counts and rates||
|error.query|Query errors||
|error.repl|Replication errors||
|event|[MySQL Event Scheduler](https://dev.mysql.com/doc/refman/8.0/en/event-scheduler.html)||
|file|Files and tablespaces||
|galera|Percona XtraDB Cluster and MariaDB Cluster (wsrep)||
|gcp|Google Cloud||
|gr|MySQL Group Replication||
|host|Host (client)||
|[`innodb`](collectors#innodb)|InnoDB metrics [`INFORMATION_SCHEMA.INNODB_METRICS`](https://dev.mysql.com/doc/refman/en/information-schema-innodb-metrics-table.html)|v1.0.0|
|innodb.mutex|InnoDB mutexes `SHOW ENGINE INNODB MUTEX`||
|mariadb|MariaDB enhancements||
|ndb|MySQL NDB Cluster||
|oracle|Oracle enhancements||
|percona|Percona Server enhancements||
|perconca.userstat|[Percona User Statistics](https://www.percona.com/doc/percona-server/8.0/diagnostics/user_stats.html)||
|percona.userstat.index|Percona `userstat` index statistics (`INFORMATION_SCHEMA.INDEX_STATISTICS`)|
|percona.userstat.table|Percona `userstat` table statistics||
|processlist|Processlist `SHOW PROCESSLIST` or `INFORMATION_SCHEMA.PROCESSLIST`||
|pfs|Performance Schema `SHOW ENGINE PERFORMANCE_SCHEMA STATUS`||
|pxc|Percona XtraDB Cluster||
|query|Query metrics||
|[`query.global`](collectors#queryglobal)|Global query metrics (including response time)|v1.0.0|
|query.id|Query metrics||
|repl|MySQL replication `SHOW SLAVE|REPLICA STATUS`|v1.0.0|
|[`repl.lag`](collectors#repllag)|MySQL replication lag (including heartbeats)|v1.0.0|
|rocksdb|RocksDB store engine||
|size|Storage sizes (in bytes)||
|[`size.binlog`](collectors#sizebinlog)|Binary log size|v1.0.0|
|[`size.database`](collectors#sizedatabase)|Database sizes|v1.0.0|
|size.file|File sizes (`innodb_undo` and `innodb_temp`)||
|size.index|Index sizes||
|[`size.table`](collectors#sizetable)|Table sizes|v1.0.0|
|stage|Statement execution stages||
|status.account|Status by account||
|[`status.global`](collectors#statusglobal)|Global status variables `SHOW GLOBAL STATUS`|v1.0.0|
|status.host|Status by host||
|status.thread|Status by thread||
|status.user|Status by user||
|stmt|Statements||
|[`stmt.current`](collectors#stmtcurrent)|Current statments|v1.0.0|
|stmt.history|Historical statements||
|thd|Threads||
|tls|TLS (SSL) status and configuration||
|tokudb|TokuDB storage engine||
|trx|Transactions||
|[`var.global`](collectors#varglobal)|MySQL global system variables (sysvars) `SHOW GLOBAL VARIABLES`|v1.0.0|
|wait|Stage waits||
|wait.current|Current waits||
|wait.history|Historical waits||