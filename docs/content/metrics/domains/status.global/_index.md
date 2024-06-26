---
title: "status.global"
---

The `status.global` domain includes the primary MySQL metrics from `SHOW GLOBAL STATUS`.

{{< toc >}}

## Usage

This domain needs little explanation because `SHOW GLOBAL STATUS` has been part of MySQL for decades.
What's important to know is that some values aren't metrics, they're status strings or other values.
Therefore, it's necessary to list in the plan which metrics to collect.
Here's a good starting point with two levels to collect more important metrics more frequently:

```yaml
kpi:
  freq: 5s
  collect:
    status.global:
      metrics:
        # Key performance indicators (KPIs)
        - "queries",
        - "threads_running", # gauge
        # Transactions per second (TPS)
        - "com_begin",
        - "com_commit",
        - "com_rollback",
        # Read-write access
        - "com_select", # reads; the rest are writes
        - "com_delete",
        - "com_delete_multi",
        - "com_insert",
        - "com_insert_select",
        - "com_replace",
        - "com_replace_select",
        - "com_update",
        - "com_update_multi",
        # Storage IOPS
        - "innodb_data_reads",
        - "innodb_data_writes",
        # Storage throughput (bytes/s)
        - "innodb_data_written",
        - "innodb_data_read",
        # Buffer pool efficiency
        - "innodb_buffer_pool_read_requests", # logical reads
        - "innodb_buffer_pool_reads",         # disk reads (data not in buffer pool)
        - "innodb_buffer_pool_wait_free",     # free page waits
        # Buffer pool usage
        - "innodb_buffer_pool_pages_dirty", # gauge
        - "innodb_buffer_pool_pages_free",  # gauge
        - "innodb_buffer_pool_pages_total", # gauge
        # Page flushing
        - "innodb_buffer_pool_pages_flushed", # total pages
        # Transaction log throughput (bytes/s)
        - "innodb_os_log_written",
extra:
  freq: 20s
  collect:
    status.global:
        # Temp objects
        - "created_tmp_disk_tables",
        - "created_tmp_tables",
        - "created_tmp_files",
        # Threads and connections
        - "connections",
        - "threads_connected",    # gauge
        - "max_used_connections", # gauge
        # Network throughput
        - "bytes_sent",
        - "bytes_received",
        # Large data changes cached to disk before binlog
        - "binlog_cache_disk_use",
        # Prepared statements
        - "prepared_stmt_count", # gauge
        - "com_stmt_execute",
        - "com_stmt_prepare",
        # Client connection errors
        - "aborted_clients",
        - "aborted_connects",
        # Bad SELECT: should be zero
        - "select_full_join",
        - "select_full_range_join",
        - "select_range_check",
        - "select_scan",
        # Admin and SHOW
        - "com_flush",
        - "com_kill",
        - "com_purge",
        - "com_admin_commands",
        - "com_show_processlist",
        - "com_show_slave_status",
        - "com_show_status",
        - "com_show_variables",
        - "com_show_warnings",
```

{{< hint type=note >}}
Use the [`innodb`]({{< ref "metrics/domains/innodb/" >}}) domain to collect most InnoDB metrics.
{{< /hint >}}

Most metrics are cumulative counters. These metrics are gauges:

* innodb_buffer_pool_pages_dirty
* innodb_buffer_pool_pages_free
* innodb_buffer_pool_pages_total
* innodb_os_log_pending_writes
* innodb_row_lock_current_waits
* max_used_connections
* prepared_stmt_count
* threads_connected
* threads_running

## Derived Metrics

None.

## Options

### `all`

|Value|Default|Description|
|-----|-------|-----------|
|yes  | |Collect _all_ 490+ metrics (not recommended)|
|no   |&check;|Collect only metrics listed in the plan.|

## Group Keys

None.

## Meta

None.

## Error Policies

None.

## MySQL Config

None.

## Changelog

|Blip Version|Change|
|------------|------|
|v1.0.0      |Domain added|
