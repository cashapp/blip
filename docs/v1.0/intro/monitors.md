---
layout: default
title: "Monitors"
parent: Introduction
nav_order: 2
---

# Monitors

Every MySQL instance that Blip monitors is called a monitor.
For simplicity, the terms _monitor_ and _MySQL instance_ are synonymous because a monitor requires and represents only one MySQL instance.
But there is more to a monitor than its MySQL instance.

<div class="note">
<em>Monitor</em> and <em>MySQL instance</em> are synonymous in Blip.
</div>

Monitors are usually specified in the [Blip config file](../config/config-file), but they can be loaded various ways&mdash;more on this later.
For now, it's only necessary to know that monitors are listed in the YAML config file under the aptly named section `moniotrs`.
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
    aws-rds:
      auth-token: true
```

The first instance is local: Blip connects using socket file `/tmp/mysql.sock`.
The second instance is remote: Blip connects to IP `10.1.1.53`.
The third instance is an Amazon RDS for MySQL instance, and Blip uses [IAM authentication](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/UsingWithRDS.IAMDBAuth).

The point of these contrived examples is: Blip can monitor _any_ MySQL instance anywhere it's running.
(If you have a case where this is not true, please file an issue.)

By default, Blip first attempts to load monitors from its config file (which is `blip.yaml` in the current working directory, by default).
But the config file can specify other ways to load monitors:

```yaml
monitor-loader:
  freq: 60s
  files: [monitors1.yaml, monitors2.yaml]
```

In short, that config snippet makes Blip load (read) monitor configuration from files `monitors1.yaml` and `monitors2.yaml` every 60 seconds.
(Blip can dynamically load [add] and unload [remove] monitors while running.)
The `monitor-load` config is optional; by default, Blip loads monitors from the `monitors` section in its config file.

To further ensure that Blip can monitory _any_ MySQL instance, loading monitors is an optional plugin with this callback signature:

```go
LoadMonitors func(Config) ([]ConfigMonitor, error)
```

Hopefully, built-in features cover every use case, but if you have particular requirements (filtering out certain MySQL instances, for example), you can plug in your own code to load monitors.

In addition to basic MySQL configuration&mdash;how to connect to MySQL: hostname, username, and password, and so forth&mdash;monitors have other optional features and configuration, summarized briefly in the following table.

|Monitor Confg|Feature|
|-------------|-------|
|`aws`|Amazon RDS authentication|
|`exporter`|Prometheus mysqld_exporter emulation|
|`ha`|High availability (not implemented yet)|
|`heartbeat`|Heartbeat to measure replication lag|
|`meta`|User-defined key-value data|
|`plans`|Monitor-specific plans for metrics collection|
|`sinks`|Monitors-specific sinks for sending metrics|
|`tags`|Monitor-specific key-value data passed to sinks|
|`tls`|TLS configuration|

That's a lot of information, but the point is a lot simpler: Blip monitors can do almost anything.
For the most part, these features support Blip in large, automated environments.
If you don't need a feature, you can forget about it: Blip is simple (and fully automatic) by default.
When you need a feature, Blip most likely already supports it.

One last helpful tip:

```sh
$ blip --print-moniotrs --run=false
```

The command line above starts (but does not run) Blip so that it loads monitors and prints them, then exits.
This can help debug monitor loading and configuration.

---

Why stop now; keep learning: [Plans&nbsp;&darr;](plans)
