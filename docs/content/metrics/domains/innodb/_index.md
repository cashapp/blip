---
title: "innodb"
---

The `innodb` domain includes InnoDB metrics from [`INFORMATION_SCHEMA.INNODB_METRICS`](https://dev.mysql.com/doc/refman/en/information-schema-innodb-metrics-table.html).

{{< toc >}}

## Usage 

For example:

```
mysql> SELECT * FROM innodb_metrics WHERE name='trx_rseg_history_len' LIMIT 1\G
*************************** 1. row ***************************
           NAME: trx_rseg_history_len
      SUBSYSTEM: transaction
          COUNT: 0
      MAX_COUNT: 0
      MIN_COUNT: 0
      AVG_COUNT: NULL
    COUNT_RESET: 0
MAX_COUNT_RESET: 0
MIN_COUNT_RESET: 0
AVG_COUNT_RESET: NULL
   TIME_ENABLED: 2021-08-17 08:24:14
  TIME_DISABLED: NULL
   TIME_ELAPSED: 1905927
     TIME_RESET: NULL
         STATUS: enabled
           TYPE: value
        COMMENT: Length of the TRX_RSEG_HISTORY list
```

The exact `NAME` value is used for the Blip metric name.
In the example above, the Blip metric name is `trx_rseg_history_len`, even though this metric means history list length (HLL).
Metric names are unique by `SUBSYSTEM`.

The [`status.global`]({{< ref "metrics/domains/status.global/" >}}) domain includes many of the same metrics because, historically, only `SHOW GLOBAL STATUS` existed.
It's a best practice to collect InnoDB metrics with this domain and exclude them from [`status.global`]({{< ref "metrics/domains/status.global/" >}}).

As a starting point, these are good InnoDB metrics to collect:

```yaml
plan:
  collect:
    innodb:
      metrics:
        # Transactions
        - "trx_active_transactions"
        
        # Row locking
        - "lock_timeouts"
        - "lock_row_lock_current_waits"
        - "lock_row_lock_waits"
        - "lock_row_lock_time"
        
        # Page flushing
        - "buffer_flush_adaptive_total_pages"   #  adaptive flushing
        - "buffer_LRU_batch_flush_total_pages"  #  LRU flushing
        - "buffer_flush_background_total_pages" #  legacy flushing
        
        # Transaction log utilization (%)
        - "log_lsn_checkpoint_age"     # checkpoint age
        - "log_max_modified_age_async" # async flush point
        
        # Transaction log -> storage waits
        - "innodb_os_log_pending_writes"
        - "innodb_log_waits"
        
        # History List Length (HLL)
        - "trx_rseg_history_len"
        
        # Deadlocks
        - "lock_deadlocks"
``` 

## Derived Metrics

None.

## Options

### `all`

|Value|Default|Description|
|-----|-------|-----------|
|yes  | |Collect _all_ 300+ metrics (not recommended)|
|no   |&check;|Collect only metrics listed in the plan|
|enabled| |Collect metrics that are enabled by MySQL (`WHERE status='enabled'`)|

## Group Keys

None.

## Meta

|Key|Value|
|---|-----|
|`subsystem`|`SUBSYSTEM` column|

Technically InnoDB metric names are unique by subsystem, but currently they're unique for the whole table.
It's probably safe to graph them by metric name alone, but if there's a name collision in the future, one metrics will be lost when reporting.

## Error Policies

None.

## MySQL Config

See [17.15.6 InnoDB INFORMATION_SCHEMA Metrics Table](https://dev.mysql.com/doc/refman/en/innodb-information-schema-metrics-table.html).

## Changelog

|Blip Version|Change|
|------------|------|
|v1.0.0      |Domain added|
