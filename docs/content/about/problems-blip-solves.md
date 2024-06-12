---
weight: 3
---

Blip was created to solve six real-world problems:

## Problem 1: There are no open source monitoring solutions purpose-built for MySQL

You're probably thinking "Yes there is: [`mysqld_exporter`](https://github.com/prometheus/mysqld_exporter)", but that was purpose-built for Prometheus, not MySQL.
For Prometheus to be successful (and it was), it needed exporters for all the major servers, like MySQL.

The same is true for other solutions like the Datadog agent for MySQL and old classics like [`mysql-statsd`](https://github.com/db-art/mysql-statsd): they were all built to support _another_ product, not built solely for MySQL regardless of other products.

The telltale sign of this problem is that others solutions stops working when you change the other product.
For example, the Datadog agent only works with Datadog.
If you change to SignalFx, or Chronosphere, or other products, the Datadog agent won't works.
Likewise, if you're using `mysqld_exporter` and change to Datadog, the former won't work.

### _Solution_

Blip was purpose-built for MySQL by MySQL experts.
It does one thing and it does it extremely well: collect MySQL metrics.
Moreover, it was built to be agnostic (but smart) about the MySQL distribution and version, where MySQL is running, and where the metrics are sent.
For example, Blip works with [Percona Server](https://www.percona.com/software/mysql-database/percona-server) running on bare metal on-premise sending metrics to [Splunk](https://www.splunk.com/), or [MariaDB](https://mariadb.com/) running in the cloud sending metrics to [Datadog](https://www.datadoghq.com/), or [MySQL Community Edition](https://www.mysql.com/) running on your laptop dumping metrics to STDOUT.

## Problem 2: Not cloud-native or cloud-aware

Let's pretend for the moment that the term "cloud-native" is not overhyped.
A cloud-native (or cloud-aware) MySQL monitor would make it easy to use IAM authentication with [Amazon RDS](https://aws.amazon.com/rds/), for example.
Or automatically use the Amazon RDS CA to secure the connection with TLS when connecting to an Amazon RDS instances&mdash;that's cloud-aware.
Or more simply: the monitor should be able to collect out-of-band (OOB) MySQL metrics from the cloud provider, like Amazon CloudWatch.
Since MySQL in the cloud is increasingly common, being cloud-native or cloud-aware is part of being purpose-built for MySQL.

### _Solution_

Blip was built from the ground up to be cloud-native and cloud-aware.
For example, it has built-in support for collecting OOB MySQL metrics for Amazon RDS from Amazon CloudWatch.
It also has stubs and placeholders for the other major cloud providers: Google Cloud, Microsoft Azure, and Oracle OCI.

## Problem 3: Cannot collect new metrics without modifying the core code

Long ago, `SHOW GLOBAL STATUS` was essentially the only source of MySQL metrics.
But over the years, MySQL metrics scattered into many sources and hundreds of new metrics were added.
The pace of change with respect to metrics has slowed a bit with MySQL 8.0, but it certainly has not stopped.
Consequently, a MySQL monitor has to make it easy to collect new metrics, but none do.
Point in case: [Percona `mysqld_exporter`](https://github.com/prometheus/mysqld_exporter) was created&mdash;and Percona made a lot of changes&mdash;because the upstream `mysqld_exporter` was not designed to make collecting new metrics easy.
As further evidence, look at the pileup of long-standing issues requesting new metrics.

### _Solution_

Blip was built from the ground up to solve this problem: every metric collector is a plugin.
Even the built-in collectors are simply plugins that are auto-registered on startup.
If Oracle adds a whole new set of metrics, Blip can collect these with zero core code changes: it only requires a new collector plugin to collect the new metrics.

The Blip collector plugin also means companies can write and use their own plugins with a private clone of the Blip source code&mdash;no open source contribution required.

## Problem 4: Cannot extend or change functionality without modifying the core code

Collecting new metrics isn't the only problem with most MySQL monitors: general functionality is also limited or hard-coded.
For example, you probably don't use IAM authentication with [Amazon RDS](https://aws.amazon.com/rds/) for your current monitoring solution because none support it natively.
Or, what if your monitoring solution is behind a proxy and the HTTP client needs to authentication with that before sending metrics to a hosted provider?
It's quite difficult to anticipate or support the wide array of environments in which people run MySQL monitoring, so most products don't try; you just have to work around the product.

### _Solution_

Blip was built from the ground up to solve this problem, too: nearly every major aspect of its functionality can be modified and customized with a plugin or factory override.
See the [Develop](../v1.0/develop/) section for the many ways you can make Blip work for you, not the other way around.

## Problem 5: Collects too much or too little

MySQL metrics have become extremely polluted: `SHOW GLOBAL STATUS` outputs about 400 metrics, but only ~100 are useful&mdash;and some aren't even metrics.
Any monitor that naively collects the full list of _any_ metric source&mdash;not just `SHOW GLOBAL STATUS`&mdash;will cause two problem.
First, it will spam your metrics storage and graph system with noise and fake metrics.
Second, if you pay for storage and system, it will waste your money.

It's unlikely MySQL will clean up its metrics since it hasn't done so in the last 20 years.
Consequently, a MySQL monitor must be careful and explicit about which metrics it collects, but most are not.

### _Solution_

Blip collects only the metrics you specify in a plan.
Or, use the built-in plan that collects over 60 of the most useful MySQL metrics from various sources.
On the one hand, plans are tedious but MySQL has made them necessary.
On the other hand, plans are nice because different DBA teams like to collect different metrics, which is either not possible or difficult (requiring post-collection filter/dropping) with other monitors.
But with Blip, you can collect exactly the metrics you need.

## Problem 6: Cannot collect metrics at different frequencies

The industry has long recognized and taught that certain metrics are _key performance indicators (KPI)_ that should be collected and reported as frequently as possible: 1 to 5 seconds.
But other metrics can (and should) be collected less frequently, like database and table sizes.
Yet no MySQL monitors make this easy, and most don't even make it possible.
As a result, DBAs usually make a trade-off: they collect all metrics every 10, 20, or 30 seconds&mdash;or worse: every 60 seconds.

### _Solution_

Blip can collect different metrics at different frequencies.
For example: KPI every 5s, standard metrics every 20s, and data sizes every 5 minutes.
This is accomplished with Blip _plans_ that allow you to specific what to collect and how frequently.
To learn more, read through the quick [Introduction](../v1.0/intro/concepts).
