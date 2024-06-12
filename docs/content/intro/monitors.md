---
weight: 2
title: "2. Monitors"
---

Every MySQL instance that Blip monitors is called a monitor.
For simplicity, the terms _monitor_ and _MySQL instance_ are synonymous because a monitor requires and represents only one MySQL instance.
But there is more to a monitor than its MySQL instance.

<div class="note">
<em>Monitor</em> and <em>MySQL instance</em> are synonymous in Blip.
</div>

Monitors are usually specified in the [Blip config file]({{< ref "config/config-file" >}}), but they can be loaded various ways&mdash;more on this later.
For now, it's only necessary to know that monitors are listed in the YAML config file under the aptly named section `monitors`.
The most basic Blip monitor is a simple hostname, username, and password:

```yaml
monitors:
  - hostname: 127.0.0.1
    username: blip
    password: aStrongRandomPassword
```

Or, if you want to use a `my.cnf` file:

```yaml
monitors:
  -  mycnf: ${HOME}/.my.cnf
```

A single Blip instances can monitor any number of MySQL instances.
Here is a snippet of config that specifies three different MySQL instances:

```yaml
monitors:
  - socket: /tmp/mysql.sock
    username: blip
    password-file: /dev/shm/metrics-password

  - hostname: 10.1.1.53
    username: metrics
    password: foo

  - hostname: db3.us-east-1.amazonaws.com
    aws:
      iam-auth: true
```

The first monitor is a local MySQL instance: Blip connects using socket file `/tmp/mysql.sock`.
The second monitor is a remove MySQL instance: Blip connects to IP `10.1.1.53`.
The third monitor is an Amazon RDS for MySQL instance, and Blip uses [IAM authentication](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/UsingWithRDS.IAMDBAuth).

The point of this contrived example is that _Blip supports all types of MySQL instances_.
If you have a case where Blip does not work, please file an issue.

By default, Blip first attempts to load monitors from its config file (which is `blip.yaml` in the current working directory, by default).
But the config file can specify other ways to load monitors:

```yaml
monitor-loader:
  files: [monitors1.yaml, monitors2.yaml]
```

That config snippet makes Blip load (read) monitor configuration from files `monitors1.yaml` and `monitors2.yaml`.
(Blip can also dynamically load and unload monitors while running through [API]({{< ref "/api" >}}) calls.)
The `monitor-load` config is optional; by default, Blip loads monitors from the `monitors` section in its config file.

To further ensure that Blip can monitory _all_ types of MySQL instance, loading monitors is an optional plugin with this callback signature:

```go
LoadMonitors func(Config) ([]ConfigMonitor, error)
```

The default (built-in) monitor load covers most cases, but if you have a very particular environment, you can completely override the default monitor loader with plugin code.

In addition to basic MySQL configuration&mdash;how to connect to MySQL: hostname, username, and password, and so forth&mdash;monitors have other optional features and configuration, summarized in the following table.

|Monitor Config|Feature|
|:-------------|:------|
|`aws`|Amazon RDS authentication|
|`exporter`|Emulate Prometheus mysqld_exporter|
|`heartbeat`|Heartbeat to measure replication lag|
|`meta`|User-defined key-value data|
|`plans`|Monitor-scoped plans for metrics collection|
|`sinks`|Monitor-specific sinks for sending metrics|
|`tags`|Monitor-specific key-value data passed to sinks|
|`tls`|TLS configuration|

For the most part, these features support Blip in large, automated environments.
If you don't need a feature, you can forget about it: Blip is simple by default.
When you need a feature, Blip most likely already supports it.

One last helpful tip:

```sh
blip --print-monitors --run=false
```

The command line above starts (but does not run) Blip so that it loads monitors and prints them, then exits.
This helps debug monitor loading and configuration.

---

Now the fun part: [Metrics&nbsp;&darr;]({{< ref "metrics" >}})
