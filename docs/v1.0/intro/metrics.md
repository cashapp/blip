---
layout: default
title: "4. Metrics"
parent: Introduction
nav_order: 3
---

# Metrics

Blip collects the most important MySQL metrics by default.
If you're not a MySQL expert or DBA, then all you have to do is configure a sink to report them.
([Sinks](sinks) are explained later.)
However, Blip is also designed for experts and DBA who know that MySQL metrics are vast and unorganized.
To manage the complexity of MySQL metrics, Blip organizes metrics into domains.

A Blip _metric domain_ (or _domain_ for short) is a logical group of MySQL metrics.
Most domains map very closely to the source of MySQL metrics that they represent.
For example, the [`status.global`](../metrics/domains#statusglobal) domain represents metrics from `SHOW GLOBAL STATUS`.

What source of MySQL metrics do you think the [`repl`](../metrics/domains#repl) domain represents?
You're probably thinking `SHOW SLAVE STATUS`, and you're probably right.
However, that source was renamed as of MySQL 8.0.22; it's now `SHOW REPLICA STATUS`, and some of its metrics were renamed, too.
But wait, there's more...

The [`innodb`](../metrics/domains#innodb) domain represents InnoDB metrics, but MySQL has three difference sources for InnoDB metrics: `SHOW ENGINE INNODB STATUS`, `SHOW GLOBAL STATUS`, and `information_schema.innodb_metrics`.
The [domain documentation](../metrics/domains) specifies which source is used because some domains use multiple sources. (Whenever possible, Blip automatically choose the best source.)

Familiarity with Blip domains is the first step to customizing metrics collection.
The second step is specifying which metrics in the domain to collect.
Here's a tiny [plan](plans) that collects `Threads_running` and `Queries` from the [`status.global`](../metrics/domains#statusglobal) domain every 5 seconds:

```yaml
level:
  freq: 5s
  collect:
    status.global:
      metrics:
        - threads_running
        - queries
```

Ignore the plan syntax for now; it's covered next.

_To customize metrics collection, specify the domains and metrics in those domains, as shown on lines 4 through 7._

Blip collects only the metrics that you specify, nothing else.
This allows for very precise and efficient metrics collection, which is necessary because the vast majority of MySQL metrics are not generally useful (or, they're esoteric such that few people use them except deep MySQL experts.)
For example, `SHOW GLOBAL STATUS` dumps over 400 metrics as of MySQL 8.0, but less than 100 are generally useful&mdash;and some aren't even metrics, like `Rsa_public_key`.
Alas, although MySQL metrics are a mess, Blip helps reign in the chaos with domains and explicit metric collection.

---

Don't stop; keep learning: [Plans&nbsp;&darr;](plans)
