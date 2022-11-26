---
layout: default
parent: Plans
title: "Loading"
---

{: .no_toc }
# Loading

The simplest way to load a plan for all monitors is by specifying a top-level [`config.plans.files`](../config/config-file#files-1) section:

```yaml
plans:
  files:
    - metrics.yaml
```

Blip will use the plan in `metrics.yaml` for all monitors since nothing else is specified.

Beyond this simplest use case, Blip load plans from multiple sources and scopes, following a load order when there are multiple possible plans for a single monitor.
The rest of this page details advanced plan loading.

* TOC
{:toc}

## Sources

Blip loads plans from four sources:

1. [`LoadPlans` plugin](../develop/integration-api#plugins) exclusivity, if defined; else
2. [Files](file) and [tables](table), if any are specified; else
3. [Default](#default) plans

If the `LoadPlans` plugin is defined, Blip ignores the other three sources and calls only the plugin.
Else, Blip loads plans from files and tables, which is the typical case.
If no files or tables are specified, Blip loads a default plan that collect over 70 of the most important MySQL server metrics.

After plans are loaded, the source doesn't matter (although it's recorded for debugging) because plans are saved in a map data structure by name.
In the Blip config, plans are referenced by name or used according to [plan precedence](#precedence).

## Scope

Plans have three scopes, in order of precedence: monitor, shared, and default.
To illustrate, let's use this example config:

```yaml
plans:          #
  files:        # Shared plans
    - foo.yaml  #

monitors:
  - hostname: db1
    plans:         #
      files:       # Monitor plans
        - p1.yaml  #

  - hostname: db2
    plans:             #
      files:           # Monitor plans
        - p2.yaml      #
        - p2-alt.yaml  #

  - hostname: db3
    plan: bar.yaml  # Shared plan reference
```

### Monitor

Monitor plans are scoped to one monitor and can only be used by that monitor.
Monitor plans are specified by a [`config.plans`](../config/config-file#plans) section in a monitor (under `monitors:`):

Only monitor `db1` can access plan `p1.yaml`.
And since monitor plans have first precedence, the monitor will load `p1.yaml`, not the shared plan.

If multiple monitor plans are specified and [plan changing](changing) is not enabled, like monitor `db2`, [plan precedence](../plans/loading#precedence) determines which monitor plan is used.
In this case, it's the first monitor plan file: `p2.yaml`.

### Shared

Shared plans are scoped to Blip and can be used by any monitor that references them.
As shown in the example above, shared plans are specified by a top-level [`config.plans`](../config/config-file#plans) section.

If a monitor has no monitor plans, then it uses a shared plan according to the [load order](#load-order).
Or, a monitor can references a specific shared plan by setting [`config.monitor.plan`](../config/config-file#plan-2) as shown for monitor `db3`.

### Default

Default plans are technically shared, but they're a special scope because they're hard-coded in Blip.
Monitors can references default plans by name using [`config.monitor.plan`](../config/config-file#plan-2).
See [Plans / Defaults](defaults) for the list of default plans.

## Precedence

When multiple plans (monitor or shared) are loaded, Blip uses one plan based on this order of precedence:

1. If table, first plan alphabetical by name (`ORDER BY name ASC LIMIT 1`)
2. If files, first file listed in [`config.plans.files`](../config/config-file#files)
3. Default by auto-detection

In short, Blip chooses the first table plan, or the first file plan, or the best default plan&mdash;in that order.

Plan precedence is scoped: if monitor plans are specified, only monitor plans are used; else shared plans are used.

{: .note }
Precedence is ignored when [`config.monitor.plan`](../config/config-file#plan-2) is set because this variable sets the shared plan to use.
