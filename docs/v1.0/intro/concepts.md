---
layout: default
title: "Blip"
parent: Introduction
nav_order: 1
---

# Blip

In the simplest setup, a single Blip instance monitors (collect metrics from) a single MySQL instance:

![Single Blip Instance](/assets/img/blip-single.png)

But a single instnace of Blip&mdash;a single running `blip` binary&mdash;can monitor any number of MySQL instances:

![Multiple Blip Monitors](/assets/img/blip-multi.png)

In Blip lingo (and source code), a _monitor_ collects metrics from a single instance of MySQL, as shown above.
In short, "a monitor monitors MySQL."

Monitors are the central concept in Blip, so let's zoom in on one monitor to see what makes it ticks:

![Blip Monitor Diagram](/assets/img/blip-monitor.png)

A monitor has more parts than shown above, but three parts are most important:

_Sinks_
: Sinks send metrics to some graphing system. A monitor can send metrics to more than one sink. Blip has built-in sinks for [SignalFx](https://docs.signalfx.com/en/latest/) and [Chronosphere](https://chronosphere.io/), and it's easy to write your own sink, which means Blip can send metrics anywhere.

_Plan_
: A plan determines which metrics to collect. Blip has a default plan that collects more than 60 of the most important MySQL metrics. Plans can be customized.

_Engine_
: The engine is the core part of a monitor that collects metrics from MySQL (according to the plan).

There are many more parts to a monitor, but sinks and plans are the two you are most likely to customize&mdash;more on this later.

You can't customize the engine, but you can customize the plan that determines which metrics the engine collects.
Let's zoom in on the engine:

![Blip Engine Diagram](/assets/img/blip-engine.png)

Inside the engine, another part called a _metrics collector_ (or _collector_ for short) collects metrics for one _domain_: a logical group of MySQL metrics.
Above, the engine has <span style="color:magenta;">four collectors</span> that correspond to four domains:

|Domain|Logical Group|
|------|-------------|
|status.global|`SHOW GLOBAL STATUS`|
|var.global|`SHOW GLOBAL VARIABLES`|
|repl|`SHOW REPLICA STATUS`, ...|
|aws|CloudWatch Metrics|

First, you might notice that a collector is not required required to collect metrics from MySQL.
An AWS collector, for example, collects related MySQL metrics from Amazon CloudWatch.
But most collectors collect metrics from various outputs of MySQL.

Second, why a new abstraction&mdash;why "domains"?
Because MySQL metrics are unorganized, and some metrics can be obtained from multiple outpus.
For example, you can obtain the global system variable `max_connections` from three different outputs (or commands):

* `SHOW GLOBAL VARIABLES LIKE 'max_connections';`
* `SELECT @@GLOBAL.max_connections;`
* `SELECT * FROM global_variables WHERE variable_name='max_connections';`

That's a trivial exmaple.
Metric domains become important&mdash;practically necessary&mdash;when you account for the MySQL [Performance Schema](https://dev.mysql.com/doc/refman/8.0/en/performance-schema) (and sometimes the MySQL [sys Schema](https://dev.mysql.com/doc/refman/8.0/en/sys-schema)), various command and output changes from MySQL 5.6 to 5.7 to 8.0, subtle differences between distributions (Oracle vs. Percona vs. MariaDB), and cloud providers (like Amazon RDS).
Metric domains simplify _how_ metrics are collected.
As a user, you shouldn't care how metrics are collected; you should only care _which_ metrics are collected.
Domains and plans make that possible...

A _level plan_ (or _plan_ for short) configures which metrics to collect&mdash;by domain.
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

First of all, don't worry: Blip has a built-in plan that collects every common and important MySQL metric, including meta-metrics like database and table sizes.
You never have to specify a plan, but chances are that you'll eventually write your own custom plans, espcially when you learn what other Blip features plans make possible.

In the plan snippet above (which is YAML syntax), there is a single level called `key-perf-indicators` collected every 5 seconds (more on levels in a moment).
At this level, Blip collects everything configured under `collect:`, which is just two metrics in the `status.global` domain: `Queries` and `Threads_running`.
Real plans a much larger, listing tens (or hunrdeds) of metrics from various domains.

Now the quesiton burning in your mind: what are "levels"?
A _level_ is a group of metrics collected at a unique frequency.
`key-perf-indicators` metrics are collected every 5 seconds: 5s, 10s, 15s, and so forth.
That's not very fancy!
Every metrics collector collects metrics at some frequency.
But consider this:

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
    size.data:
      options:
        exclude: test_db,dba_stuff
      metrics:
        # Automatic
```

Now the plan has two levels: the original `key-perf-indicators` plus `database-sizes` every 60 seconds.
(The domain `size.data` "collects" [calculatues] data size metrics for databases and tables.)
Blip combines levels automatically: at 55 seconds, Blip collects only `key-perf-indicators` metrics; but at 60 seconds, it collects `key-perf-indicators` metrics _and_ `database-sizes` metrics.
In technical terms, it collects every level where `freq % T == 0`, where `T` is the number of seconds elapsed.
Now that's fancy: with level plans, Blip collects different metrics at different frequencies.

Levels aren't just fancy, they're more efficient and (potentially) less expensive.
For example, calculating table sizes when there are thousands of tables can be relatively slow.
Consequently, it's neither wise nor necessary to collect them every few seconds like regular metrics.
By relegating table size collection to a slower frequency, regular metrics collection remains fast and efficient.
And if you pay for a hosted metrics graphhing solution, then table size metrics every few seconds is a waste of money (because table sizes don't usually change that fast).

Speaking of paying for a hosted metrics graphhing solution, you need to send the metrics somewhere, and anywhere is possible with pluggable metric sinks (or _sinks_ for short).
Blip has built-in sinks for [SignalFx](https://docs.signalfx.com/en/latest/) and [Chronosphere](https://chronosphere.io/), but the sink is an interface, which means its trivial to write a new sink for wherever you send your metrics.
Even better: domains make it possible to programmatically transform and rename metrics from Blip to any data format or protocol because Blip metrics have a consistent naming schema: lowercase `domain.metric`
For example, `status.global.threads_running`.
Your sink might strip the domain prefix, or rename it.
It could also add labels, dimensions, and so forth&mdash;there are no limits.
After collecting metrics specified by the plan, Blip passes them (in a simple data structure) to each sink, which are free to do anything (or nothing) with the metrics.

---

Keep learning: [Sinks&nbsp;&darr;](sinks)
