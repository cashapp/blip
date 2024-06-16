---
title: "Prometheus"
---

Blip can emulate Prometheus `mysqld_exporter`.
This is called _exporter mode_.

To enable, set [`config.exporter.mode`]({{< ref "config/config-file#mode" >}}) to `dual` or `legacy`:

```yaml
exporter:
  mode: dual
```

In dual exporter mode, Blip works normally _and_ it emulates `mysqld_exporter`.

```yaml
exporter:
  mode: legacy
```

In legacy exporter mode, Blip only emulates `mysqld_exporter`.
All other parts of Blip are ignored (they aren't even started), including sinks.

See the [`exporter` config]({{< ref "config/config-file#exporter" >}}) for the full configuration.

## Emulation

Blip emulates Prometheus `mysqld_exporter` by running a second API that listens on the Prometheus `mysqld_exporter` port and endpoint: `127.0.0.1:9104/metrics` by default.
It responds to scrape requests in the Exposition format by using built-in [Prometheus domain translators]({{< ref "/develop/domain-translator#prometheus-translator" >}}).

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

Prometheus `mysqld_exporter` emulation and the [`prom-pushgateway`]({{< ref "sinks/prom-pushgateway" >}}) sink are compatible with these Blip domains:

* [`innodb`]({{< ref "metrics/domains/innodb/" >}})
* [`status.global`]({{< ref "metrics/domains/status.global/" >}})
* [`var.global`]({{< ref "metrics/domains/var.global/" >}})

Compatibility requires a [Prometheus domain translator]({{< ref "/develop/domain-translator#prometheus-translator" >}}) .
Additional domains can be enabled if there is demand for this feature.
[File an issue](https://github.com/cashapp/blip/issues) to request and discuss, or submit a PR.

## Plan

In exporter mode, Blip still uses a plan, but the plan must have only 1 level.

The [default plan]({{< ref "plans/defaults" >}}) is `default-exporter`.

Set [`config.exporter.plan`]({{< ref "config/config-file#plan" >}}) to specify a different plan.
