---
layout: default
title: "1. Blip"
parent: Introduction
nav_order: 1
---

# Blip

In the simplest setup, a single Blip instance monitors (collect metrics from) a single MySQL instance:

![Single Blip Instance](/blip/assets/img/blip-single.png)

But a single instance of Blip&mdash;a single running `blip` binary&mdash;can monitor any number of MySQL instances:

![Multiple Blip Monitors](/blip/assets/img/blip-multi.png)

In Blip lingo, a _monitor_ collects metrics from a single instance of MySQL, as shown above.
In short, "a monitor monitors MySQL."

Monitors are the central concept in Blip, so let's zoom in on one monitor to see what makes it ticks:

![Blip Monitor Diagram](/blip/assets/img/blip-monitor.png)

A monitor has more parts than shown above, but the three most important parts are:

_Sinks_
: Sinks send metrics to a graphing system. A monitor can send metrics to more than one sink. Blip has built-in sinks for [SignalFx](https://docs.signalfx.com/en/latest/) and [Chronosphere](https://chronosphere.io/), and it's easy to write your own sink, which means Blip can send metrics anywhere.

_Plan_
: A plan determines which metrics to collect.
Blip has a default plan that collects more than 60 of the most important MySQL metrics.
Plans can be customized, which means Blip can collect any MySQL metrics you need (or not collect metrics that you don't need).

_Engine_
: The engine does the real work: it collects metrics from MySQL according to a plan.
You control the engine by customizing a plan (or just use the built-in default plan).

Let's zoom in on the engine:

![Blip Engine Diagram](/blip/assets/img/blip-engine.png)

Inside the engine, another part called a _metrics collector_ (or _collector_ for short) collects metrics for one _domain_: a logical group of MySQL metrics.
Above, the engine has <span style="color:magenta;">four collectors</span> that correspond to four domains:

|Domain|Metrics|
|------|-------------|
|status.global|`SHOW GLOBAL STATUS`|
|var.global|`SHOW GLOBAL VARIABLES`|
|repl|`SHOW REPLICA STATUS`, ...|
|aws|CloudWatch Metrics|

First, notice that a collector is not required required to collect metrics from MySQL.
An AWS collector, for example, collects related MySQL metrics from Amazon CloudWatch.
But most collectors collect metrics from various outputs of MySQL.

Second, why a new abstraction&mdash;why "domains"?
Because MySQL metrics are unorganized, and metrics can be collected from multiple sources.
For example, you can collect the global system variable `max_connections` from three different sources:

* `SHOW GLOBAL VARIABLES LIKE 'max_connections';`
* `SELECT @@GLOBAL.max_connections;`
* `SELECT * FROM global_variables WHERE variable_name='max_connections';`

Domains are important&mdash;practically necessary&mdash;when you account for the MySQL [Performance Schema](https://dev.mysql.com/doc/refman/8.0/en/performance-schema) (and sometimes the MySQL [sys Schema](https://dev.mysql.com/doc/refman/8.0/en/sys-schema)), various command and output changes from MySQL 5.6 to 5.7 to 8.0, subtle differences between distributions (Oracle vs. Percona vs. MariaDB), and cloud providers (like Amazon RDS).
Domains simplify _how_ metrics are collected.
As a user, you shouldn't care how metrics are collected; you should only care _which_ metrics are collected.
Domains and plans make that possible.

A _level plan_ (or _plan_ for short) specifies which metrics to collect by domain.
Here's a snippet of a plan that collects two metrics every 5 seconds:

```yaml
key-perf-indicators:
  freq: 5s
  collect:
    status.global:
      metrics:
        - Queries
        - Threads_running
```

Plans specify at least one _level_: a named group of metrics collected at a unique frequency.
In this example, the level name is `key-perf-indicators`, and its frequency (`freq: 5s`) is every 5 seconds: 5s, 10s, 15s, and so forth.
It collects two metrics from the `status.global` domain: `Queries` and `Threads_running`.

Real plans have several levels and tens (or hundreds) of metrics from various domains.
For example, the default plan has four levels and collects over 60 metrics.
But here's a simpler example with two levels:

```yaml
key-perf-indicators:
  freq: 5s
  collect:
    status.global:
      metrics:
        - Queries
        - Threads_running
database-sizes:
  freq: 60s
  collect:
    size.database:
      options:
        exclude: test_db,dba_stuff
      metrics:
        # Automatic
```

Now the plan has two levels: the original `key-perf-indicators` plus `database-sizes` every 60 seconds.
Blip combines levels automatically: at 55 seconds, it collects only `key-perf-indicators` metrics; but at 60 seconds, it collects `key-perf-indicators` metrics _and_ `database-sizes` metrics.
In technical terms, Blip collects every level where `freq % T == 0`, where `T` is the number of seconds elapsed since the monitor started.

<p class="note">
You don't have to write a plan to use Blip because it has a Default plan that collects the most common and important MySQL metrics, including derived metrics like database sizes.
Writing your own plan allows you to customize which metrics to collect and when (how frequently).
</p>

Plans and levels allow you to collect different metrics at different frequencies.
For example, you can collect and report key performance indicators every 5 seconds, but collect and report table sizes every 60s.
This is more efficient and cost-effective, especially if you use a hosted metrics graphing solution.

When the engine is done collecting metrics for a level, the monitor sends the metrics to one or more metric sink configured for the monitor.
A _sink_ is a plugin that accepts a Blip metrics data structure, then translates and sends the metrics somewhere.
A sink can also rename metrics, add labels/dimensions/tags, and so forth&mdash;whatever is appropriate for the sink destination.
Blip has built-in sinks for [SignalFx](https://docs.signalfx.com/en/latest/) and [Chronosphere](https://chronosphere.io/), but since the sink is a plugin, it's trivial to write new sinks to support new destinations.

---

Keep learning: [Monitors&nbsp;&darr;](monitors)
