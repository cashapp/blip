---
layout: default
parent: Plans
title: "Scope"
nav_order: 1
---

# Scope

Plans have two scopes:

### Shared

Shared plans are scoped to Blip and can be used by any monitor that references them.
All plans configured in `config.plans` and the [built-in plans](#built-in) plans are shared.
This is the normal case: you define one or more plan (or none, using the built-in Blip plan), and all the monitors use those plans.
Since interpolation works in plan files (see [File > Interpolation](./file.html#interpolation)), shared plans can still be tailored to each monitor, if necessary (although it's usually not necessary).

### Monitor

Monitor plans are scoped to one monitor and can only be used by that monitor.
All plans configured in `config.monitor.plans` are monitor plans.
See [`config.plans`]

## Built-in

### Blip

### Prometheus mysqld_exporter
