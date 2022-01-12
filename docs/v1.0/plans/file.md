---
layout: default
parent: Plans
title: "File"
nav_order: 3
---

# Plan File

One YAML file specifies one Blip plan.

## Syntax

Plan files use YAML syntax.
The high-level pseudo structure is:

```
---
level-name:
  <level config>
  collect:
    <domain name>:
      <domain config>
```

That structure repeats for each level, which is given a unique name.

Following is a small but realistic example:

```yaml
---
performance:
  freq: 5s
  collect:
    status.global:
      options:
        all: yes
      metrics:
        # ALL because of option ^
    innodb:
      metrics:
        - trx_rseg_history_len # HLL
data-size:
  freq: 60m
  collect:
    data.size:
      # All defaults
```

The example above specifies two levels, "performance" and "data-size".

The "performance" level is collected every 5 seconds.
For this level, Blip collect metrics from two domains: [`status.global`](domains#statusglobal) and [`innodb`](domains#innodb).

The "data-size" level is collected every 60 minutes.
For this level, Blip collects metrics for one domain: [`data.size`](domains#datasize).

When the two levels overlap (every 60 minutes), Blip collects metrics for both levels.

The "data-size" level demonstrates the minimum valid level config: `freq` and at least one domain with all defaults.

### Level Config

|Parameter|Value|Required?|Purpose|
|`collect`|[Domain Config](#domain-config)|YES|Configures which domains and metrics to collect|
|`freq`|[Go duration string](https://pkg.go.dev/time#ParseDuration)|YES|Interval at which level is collected|

### Domain Config

|Parameter|Value|Required?|Purpose|
|`metrics`|list of strings|no|List of metrics to collect; not required but common unless domain has option to collect all metrics, or only collects a fixed list of metrics|
|`options`|key-value pairs (strings)|no|Sets [collector options](../metrics/collectors#options)|

Values for both `metrics` and `options` are domain (and collector) specific.
See [Domains](domains) for the latter, and [Collectors > Options](../metrics/collectors#options) for the latter.

## Naming

Plan names are _exactly_ as written in the [`plans` section](../config/config-file#plans) of the Blip config file.
For example:

```yaml
plans:
  files:
    - "/blip/plan1.yaml"
    - ../plan2.yaml
    - ${HOME}/plan3.yaml
```

The first plan name is `/blip/plan1.yaml` (Blip does not strip base paths).
The second plan name is `../plan2.yaml` (Blip does not replace relative paths).
The third plan name is `/home/user/plan3.yaml` where `HOME=/home/user` because interpolation happens first.

## Interpolation

Blip interpolates option _values_ in the [domain config](#domain-config).

Interpolation in plan files is identical to interpolation in the config file (see [Config File > Interpolation](../config/config-file#interpolation)).

Environment variable
: `${FOO}`

Environment variable with default value
: `${FOO:-default}`

Monitor variable
: `%{monitor.VAR}`

{: .note }
**NOTE**: `${}` and `%{}` are always required.
