---
layout: default
parent: Configure
title: Collectors
---

# Collectors

Collectors are low-level components that collect metrics for [domains](../metrics/domains).
Collectors have options documented in the [domain reference](../metrics/domains) and specified as [plan options](../plans/file#options).

Options are used to fine-tune _how_ metrics are collected, not which metrics are collected.
See [Metrics / Collection](../metrics/collecting) for customizing which metrics are collected.

<p class="note">
All collector options are optional.
Collectors are designed to work automatically.
</p>

To print domains and collector options from the command line:

```sh
blip --print-domains
```

For example, the output for the [`innodb` domain](../metrics/domains#innodb) looks like:

```
innodb
  InnoDB metrics (information_schema.innodb_metrics)

  Options:
          all: Collect all metrics
          | enabled = Enabled metrics (ignore metrics list)
          | no      = Specified metrics (default)
          | yes     = All metrics (ignore metrics list)
```

The collector has one option, `all`, with the valid values listed below it.
The default option (`no`) is noted by "(default)".
This option is specified in a plan like:

```yaml
level:
  freq: 5s
    collect:
      innodb:
        options:
          all: "yes"
```

## Enable/Disable

Collectors are only used when a plan uses the domain.
For example, if a plan does not use the [`innodb` domain](../metrics/domains#innodb), then the collector is not created.
(It's not even instantiated, so unused collectors do not consume memory.)
Therefore, you don't need to enable or disable collectors.
