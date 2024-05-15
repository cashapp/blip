---
weight: 3
title: "3. Metrics"
---

Blip collects the most important MySQL metrics by default.
If you're not a MySQL expert or DBA, then all you have to do is configure a sink to report them.
(Sink are explained in the [fifth part]({{< ref "/intro/sinks" >}}) of the introduction.)
However, Blip is also designed for experts and DBA who know that MySQL metrics are vast and unorganized.
To manage the complexity of MySQL metrics, Blip organizes metrics into domains.

A Blip _metric domain_ (or _domain_ for short) is a logical group of MySQL metrics.
Most domains map intuitively to the source of MySQL metrics that they represent.
For example, the [`status.global`]({{< ref "/metrics/domains#statusglobal" >}}) domain represents metrics from `SHOW GLOBAL STATUS`.

Blip metric domains are a necessary abstraction because MySQL metrics are unorganized, inconsistent, and different for each MySQL distribution and version.
In addition to that, metrics can be collected from different sources.
For example, there are three sources for InnoDB metrics: `SHOW ENGINE INNODB STATUS`, `SHOW GLOBAL STATUS`, and `information_schema.innodb_metrics`.
And among those sources, the metrics differ in subtle (and sometimes not-so-subtle) ways.
Even MySQL experts struggle with all this complexity.

Blip domains simplify and hide (most of) the complexity.
For example again, the [`innodb`]({{< ref "/metrics/domains#innodb" >}}) domain represents InnoDB metrics regardless of the varying details&mdash;most of which Blip tries to detect automatically and use the best source.

Every metric belongs to a Blip domain.
The first step to customizing Blip metrics collection is to familiarize yourself with the [Blip metric domains]({{< ref "/metrics/domains" >}}).
The second step is specifying which metrics to collect (by domain) in a plan.
(Plans are explained in the [next part]({{< ref "plans" >}}) of the introduction.)
Here's a tiny plan that collects two metrics, `Threads_running` and `Queries`, from the [`status.global`]({{< ref "/metrics/domains#statusglobal" >}}) domain every 5 seconds:

```yaml
level:
  freq: 5s
  collect:
    status.global: # domain
      metrics:
        - threads_running
        - queries
```

In the tiny plan above, the domain is specified on line 4: `status.global`.
The metrics to collect in that domain are specified on lines 5, 6, and 7.
When Blip boots, it loads the metrics collector plugin for the domain, which does the actual metrics collection when called from the monitor engine.
(These concepts are explained in the [first part]({{< ref "concepts" >}}) of the introduction.)

Blip collects only the metrics that you specify, nothing else.
This allows for very precise and efficient metrics collection, which is necessary because the vast majority of MySQL metrics are not generally useful.
For example, `SHOW GLOBAL STATUS` dumps over 400 metrics as of MySQL 8.0, but less than 100 are generally useful&mdash;and some aren't even metrics, like `Rsa_public_key`.
Although MySQL metrics are a mess, Blip helps rein in the chaos with plans that specify which metrics to collect and how often.

---

Don't stop; keep learning: [Plans&nbsp;&darr;]({{< ref "plans" >}})
