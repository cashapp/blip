---
layout: default
parent: Plans
title: "Defaults"
nav_order: 1
---

# Default Plans

## Blip

```yaml
---
performance:
  freq: 5s
  collect:
    innodb:
      metrics:
      - trx_active_transactions
      - lock_timeouts
      - lock_row_lock_current_waits
      - lock_row_lock_waits
      - lock_row_lock_time
      - buffer_flush_adaptive_total_pages
      - buffer_LRU_batch_flush_total_pages
      - buffer_flush_background_total_pages
      - log_lsn_checkpoint_age_total
      - log_max_modified_age_async
      - innodb_os_log_pending_writes
      - innodb_log_waits
      - trx_rseg_history_len
      - lock_deadlocks
    status.global:
      metrics:
      - queries
      - threads_running
      - com_begin
      - com_commit
      - com_rollback
      - com_select
      - com_delete
      - com_delete_multi
      - com_insert
      - com_insert_select
      - com_replace
      - com_replace_select
      - com_update
      - com_update_multi
      - innodb_data_reads
      - innodb_data_writes
      - innodb_data_written
      - innodb_data_read
      - innodb_buffer_pool_read_requests
      - innodb_buffer_pool_reads
      - Innodb_buffer_pool_wait_free
      - innodb_buffer_pool_pages_dirty
      - innodb_buffer_pool_pages_free
      - innodb_buffer_pool_pages_total
      - innodb_buffer_pool_pages_flushed
      - innodb_os_log_written
additional:
  freq: 20s
  collect:
    status.global:
      metrics:
      - created_tmp_disk_tables
      - created_tmp_tables
      - created_tmp_files
      - connections
      - threads_connected
      - max_used_connections
      - bytes_sent
      - bytes_received
      - binlog_cache_disk_use
      - prepared_stmt_count
      - com_stmt_execute
      - com_stmt_prepare
      - aborted_clients
      - aborted_connects
      - select_full_join
      - select_full_range_join
      - select_range_check
      - select_scan
      - com_flush
      - com_kill
      - com_purge
      - com_admin_commands
      - com_show_processlist
      - com_show_slave_status
      - com_show_status
      - com_show_variables
      - com_show_warnings
data-size:
  freq: 5m
  collect:
    size.binlogs: {}
    size.data: {}
sysvars:
  freq: 15m
  collect:
    var.global:
      metrics:
      - max_connections
      - max_prepared_stmt_count
      - innodb_log_file_size
```

## Prometheus
