---
layout: default
title: Prometheus
nav_order: 110
permalink: /v1.0/prometheus
---

# Prometheus

Blip can emulate Prometheus `mysqld_exporter`.
To enable, set [`config.exporter.mode`](config/config-file#mode) to `dual` or `legacy`:

```yaml
exporter:
  mode: dual
```

In dual mode, Blip works normally _and_ it emulates Prometheus.

```yaml
exporter:
  mode: legacy
```

In legacy mode, Blip only emulates Prometheus, and Blip sinks are ignored.

See the [`exporter` config](config/config-file#exporter) for the full configuration.

## Emulation

Blip emulates Prometheus `mysqld_exporter` by running a second API that listens on the Prometheus `mysqld_exporter` port and endpoint: `127.0.0.1:9104/metrics` by default.
It responds to scrape requests in the Exposition format by using built-in [Prometheus domain translators](../develop/domain-translator#prometheus-translator).

```
% curl -s 127.0.0.1:9104/metrics
# TYPE mysql_global_status_buffer_pool_dirty_pages gauge
mysql_global_status_buffer_pool_dirty_pages 0
# HELP mysql_global_status_buffer_pool_page_changes_total Innodb buffer pool page state changes.
# TYPE mysql_global_status_buffer_pool_page_changes_total counter
mysql_global_status_buffer_pool_page_changes_total{operation="flushed"} 3183
# HELP mysql_global_status_buffer_pool_pages Innodb buffer pool pages by state.
# TYPE mysql_global_status_buffer_pool_pages gauge
mysql_global_status_buffer_pool_pages{state="data"} 4993
mysql_global_status_buffer_pool_pages{state="free"} 3122
mysql_global_status_buffer_pool_pages{state="misc"} 77
# HELP mysql_global_status_bytes_received Generic counter metric from SHOW GLOBAL STATUS.
# TYPE mysql_global_status_bytes_received counter
mysql_global_status_bytes_received 1.7407381e+08
# HELP mysql_global_status_bytes_sent Generic counter metric from SHOW GLOBAL STATUS.
# TYPE mysql_global_status_bytes_sent counter
mysql_global_status_bytes_sent 1.379287523e+09
# HELP mysql_global_status_commands_total Total number of executed MySQL commands.
# TYPE mysql_global_status_commands_total counter
mysql_global_status_commands_total{command="admin_commands"} 151
mysql_global_status_commands_total{command="alter_db"} 1
mysql_global_status_commands_total{command="alter_event"} 0
mysql_global_status_commands_total{command="alter_function"} 0
```

## Compatibile Domains

Currently, Prometheus `mysqld_exporter` emulation collects and returns metrics from the following domains:

* [`innodb`](metrics/domains#innodb)
* [`status.global`](metrics/domains#statusglobal)
* [`var.global`](metrics/domains#varglobal)

Additional domains can be enabled if there is demand for this feature.
[File an issue](https://github.com/cashapp/blip/issues) to discuss this.

The default plan for returned by [`blip.PromPlan()`](https://pkg.go.dev/github.com/cashapp/blip#PromPlan).
