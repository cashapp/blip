---
weight: 1
title: "1. Concepts" 
---

Blip runs one _monitor_ per MySQL instance to collect metrics:

![Blip Monitors](/blip/img/blip_monitors.svg)

Each monitor is completely independent, so a single Blip can collect different metrics from different MySQL instances.
But for easy setup, monitors can inherit common configuration.

Monitors are the central concept in Blip, so let's zoom in on one to see what makes it tick:

![Blip Monitor Internals](/blip/img/blip_monitor_internal.svg)

A monitor has more parts than shown above, but the three most important parts are:

_Sinks_
: Sinks are plugins that send metrics to a graphing system.
A monitor can send metrics to more than one sink.
Blip has several [built-in sinks]({{< ref "/sinks" >}}) for common graphing systems.
Or you can write a custom sink to make Blip send metrics in any format to any graphing system.

_Plan_
: A plan specifies which metrics to collect and how often.
Blip has a [default plan]({{< ref "/plans/defaults" >}}) that collects more than 60 of the most important MySQL metrics.
Or you can write a plan to collect only the MySQL metrics you need.
(Plans are explained in the [fourth part]({{< ref "plans" >}}) of the introduction.)

_Engine_
: The engine collects metrics from MySQL according to the plan, and sends them to the sinks.

Let's zoom in on the engine:

![Blip Monitor Engine](/blip/img/blip_monitor_engine.svg)

The engine uses _metric collectors_ (or _collectors_ for short) to collects MySQL metrics per domain.
(More on domains in the [third part]({{< ref "metrics" >}}) of the introduction.)
Above, the engine has <span style="color:magenta;">four collectors</span> that correspond to four domains:

|Domain|Metrics|
|------|-------------|
|status.global|`SHOW GLOBAL STATUS`|
|var.global|`SHOW GLOBAL VARIABLES`|
|repl|`SHOW REPLICA STATUS`|
|aws|Amazon CloudWatch|

Collectors usually collect metrics from MySQL, but they are not limited to MySQL.
For example, the `aws` collector collects Amazon RDS metrics from Amazon CloudWatch.

Collectors are plugins, so you can write custom collectors that plug into the engine.
But you probably won't need to because Blip has built-in collectors for all the most common metrics.

Each monitor has its own plans, collectors, and sinks.
And since these are customizable, you can make Blip collect and report anything&mdash;without modifying its core code or submitting a PR.

---

Keep learning: [Monitors&nbsp;&darr;]({{< ref "monitors" >}})
