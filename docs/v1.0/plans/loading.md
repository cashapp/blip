---
layout: default
parent: Plans
title: "Loading"
---

# Loading

Plans have two scopes:

### Shared

Shared plans are scoped to Blip and can be used by any monitor that references them.
All plans configured in `config.plans` and the [built-in plans](#built-in) plans are shared.
This is the normal case: you define one or more plan (or none, using the built-in Blip plan), and all the monitors use those plans.
Since interpolation works in plan files (see [File > Interpolation](./file#interpolation)), shared plans can still be tailored to each monitor, if necessary (although it's usually not necessary).

### Monitor

Monitor plans are scoped to one monitor and can only be used by that monitor.
All plans configured in `config.monitor.plans` are monitor plans.
See [`config.plans`]

## Built-in

### Blip

### Prometheus mysqld_exporter

## Load Order

```yaml
---
# Zero config
#plans:
  #default: blip

plans:
  files: first.yaml

plans:
  files: [first.yaml,second.yaml]
  #default: first.yaml

plans:
  files: [first.yaml,second.yaml]
  default: second.yaml

plans:
  files: [first.yaml,second.yaml]
  adjust:
    freq: 1s
    readonly: first.yaml
    active: second.yaml

plans:
  files: first.yaml:second.yaml
  default: first.yaml
  adjust:     # ERROR: default and adjust mutually exclusive
    freq: 1s
    readonly: first.yaml
    active: second.yaml

plans:
  files: first.yaml:second.yaml
  table: blip.plans

plans:
  table: blip.plans # WHERE 1=1 (all rows)
  #default: WHERE monitorId IS NULL ORDER BY name ASC LIMIT 1

plans:
  table: blip.plans
  default: plan1

monitors:
  - id: host1
    plans:
      table: blip.plans # WHERE monitorId=host1
      #default: WHERE monitorId=host1 ORDER BY name ASC LIMIT 1

```
