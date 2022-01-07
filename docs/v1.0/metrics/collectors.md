---
layout: default
parent: Metrics
title: Collectors
nav_order: 4
---

# Collectors

Metric collectors and [domains](domains) are one-to-one: Blip uses one metric collector for each domain.
The `innodb` collector, for example, collects metrics for the [`innodb` domain](domains#innodb).

Collectors are _not_ configured in the [Blip config file](../config/config-file).
In most cases, you don't need to do anything with respect to collectors because they're designed to work automatically with default values.

## Options

Collectors have options that are set in a plan (see [Plans > File > Options](../plans/file#options)).

```sh
$ blip --print-domains | less
```

Run the command above to print all domains and collector help for each, which includes the collector options.
For example, the [`innodb` domain](domains#innodb) collector help output looks like:

```
innodb
	InnoDB metrics (information_schema.innodb_metrics)

	Options:
		all: Collect all metrics
		| enabled = Enabled metrics (ignore metrics list)
		| no      = Specified metrics (default)
		| yes     = All metrics (ignore metrics list)
```

The collector has one option, `all`, with the valid values listed below.
The default option (`no`) is noted by "(default)".

{: .note}
In rare case where documented, a collector has required options.

## List

Since collectors and domains are one-to-one, run the following command to list all collectors:

```sh
$ blip --print-domains | less
```

## Enable/Disable

Collectors are only used when referenced in a [plan](../plans/).
For example, if no plan references the [`size.index`](domains#sizeindex) domain, then its collector is not used&mdash;it's not even instantiated, so unused collectors do not consume memory.
Therefore, you don't need to enable or disable collectors.

If a collector is not listed (see above), then it has not been registered.
Blip automatically registers all built-in collectors, so a missing collector should only occur with custom collectors (see below).

## Custom

All collectors are plugins, even the Blip built-in collectors.

### 1. Develop

Implement `blip.Collector` and `blip.CollectorFactory`.

### 2. Register

```go
metrics.Register(myFactory, "mydomain")
```

### 3. Reference

```yaml
level:
  collect:
    mydomain:
```
