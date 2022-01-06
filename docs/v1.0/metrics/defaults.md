---
layout: default
parent: Metrics
title: Defaults
nav_order: 1
---

# Default Metrics

Every 5 seconds:

From `status.global`:

* `com_begin`
* `com_commit`
* `com_delete`
* `com_delete_multi`
* `com_insert`
* `com_insert_select`
* `com_replace`
* `com_replace_select`
* `com_rollback`
* `com_select`
* `com_update`
* `com_update_multi`
* `innodb_buffer_pool_pages_dirty`
* `innodb_buffer_pool_pages_flushed`
* `innodb_buffer_pool_pages_free`
* `innodb_buffer_pool_pages_total`
* `innodb_buffer_pool_read_requests`
* `innodb_buffer_pool_reads`
* `innodb_buffer_pool_wait_free`
* `innodb_data_read`
* `innodb_data_reads`
* `innodb_data_writes`
* `innodb_data_written`
* `innodb_os_log_written`
* `queries`
* `threads_running`

From `innodb`:

* `buffer_LRU_batch_flush_total_pages`
* `buffer_flush_adaptive_total_pages`
* `buffer_flush_background_total_pages`
* `innodb_log_waits`
* `innodb_os_log_pending_writes`
* `lock_deadlocks`
* `lock_row_lock_current_waits`
* `lock_row_lock_time`
* `lock_row_lock_waits`
* `lock_timeouts`
* `log_lsn_checkpoint_age_total`
* `log_max_modified_age_async`
* `trx_active_transactions`
* `trx_rseg_history_len`

From `query.global`:

* `response_time` (p999) if MySQL 8.0, Percona Server 5.7, or MariaDB 10.x

---

Every 20 seconds:

* `aborted_clients`
* `aborted_connects`
* `binlog_cache_disk_use`
* `bytes_received`
* `bytes_sent`
* `com_admin_commands`
* `com_flush`
* `com_kill`
* `com_purge`
* `com_show_processlist`
* `com_show_slave_status`
* `com_show_status`
* `com_show_variables`
* `com_show_warnings`
* `com_stmt_execute`
* `com_stmt_prepare`
* `connections`
* `created_tmp_disk_tables`
* `created_tmp_files`
* `created_tmp_tables`
* `max_used_connections`
* `prepared_stmt_count`
* `select_full_join`
* `select_full_range_join`
* `select_range_check`
* `select_scan`
* `threads_connected`

---

Every 5 minutes:

* Database sizes (per-db)
* Binlog size

---

Every 15 minutes:

Sysvars:

* `max_connections`
* `max_prepared_stmt_count`
* `innodb_log_file_size`
