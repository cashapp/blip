---
layout: default
title: "Config File"
parent: Configure
nav_order: 2
---

{: .floating-toc }
* TOC
{:toc}

# Config File

Blip configuration is specified in a single YAML file (see [Specifying a Config File](blip.html#specifying-a-config-file)).
Concetpually, the Blip config file has three parts as shown and defined below.

```yaml
---
# Server config

# Monitor defaults

# Monitors
monitors: []
```

_Server config_
: Top-level sections that configure the `blip` instance, which is called the "server": API, monitor loading, and so forth

_Monitor defaults_
: Top-level sections that provide default values for each monitor: hostname, username, and so on

_Monitors_
: List items under the `monitors` section, one for each MySQL instance to monitor: hostname, username, and so on

A simple Blip config file for monitoring a single MySQL instance looks like:

```yaml
monitors:
  - hostname: db.local
    username: metrics
    password: "myVerySecurePassword"
```

Since no server config is specified, `blip` uses  built-in defaults (see [Zero Config](blip.html#zero-config)), which is probably fine for the server.

{: .src }
{ [config.go](https://github.com/cashapp/blip/blob/main/config.go) }

# Conventions

To reference sections, subsections, and specific user-configurable variables within those sections:

`SECTION`
: Any top-level section. For example: `api`, `plans`, and `monitors`.

`SECTION[.SUB...].VAR`
: A user-configurable variable in the `SECTION` with optional subections (`SUB`). For example: `mysql.username` (section = `mysql`, variable = `username`), or `monitors.heartbeat.freq` (section = 'monitors', subsection = 'heartbeat', variable = 'freq`).

Note the difference between `heartbeat` and `monitors.heartbeat`: the former is the top-level [monitor default](#monitor-defaults); the latter is a [monitor config](#monitors) that inherts the monitor defaults, if any.

Outside this section of the docs, we prefix all config references with `config.`.
For example, elsewhere in the docs, we write `config.api` to make it clear that we're refering to the `api` section of the Blip config file.

All section and variable names are `lowercase-and-hyphenated`.
(But string values that you specifiy can by anything you want.)

`disable` and `disable-auto-...` are used to disable features.
There are no "enable" prefixes or variables.
Instead, some features of either off or auto by default.
If off by default, the feature is enabled by specifying a variable noted in the docs.
For example, `heartbeat` is off by default and enabled when `heartbeat.freq` is specififed.
If auto by default, the feature is disabled by specifying `disable-auto-FEATURE: true`, where `FEATURE` is the feature name.
For example, `aws-rds.disable-auto-region: true` to disable auto-detecting the AWS region.

<br><br><br>

# Interpolation

Blip automatically interploates environment variables and monitor variables in the config file _and_ plans.

Environment variable
: `${FOO}`

Environment variable with default value
: `${FOO:-default}`

Monitor variable
: `%{monitor.VAR}`

{: .note }
**NOTE**: `${}` and `%{}` are always required.

Environment variable interpolation is a simple implementation of the shell standard.
In Blip, only the two cases shown above are supported, and `default` must be a literal value (it cannot be another `${}`).

Monitor variables are scoped to (only work within) a single monitor.
For example:

```yaml
monitors:
  - hostname: db.local
    username: metrics
    tags:
      hostname: %{monitor.hostname}
```

The result is `monitors.tags.hostname = "db.local"` because `%{monitor.hostname}` refers to the local `monitors.hostname` variable.
Blip is remarkably flexible, so this works the other way, too:

```yaml
monitors:
  - hostname: %{monitor.tags.hostname}
    username: metrics
    tags:
      hostname: db.local
```

The result is `monitors.hostname = "db.local"` because `%{monitor.tags.hostname}` refers to the local `monitrs.tags.hostname` variable.

{: .note }
Singular "monitor" in `%{monitor.VAR}`, not plural, to emphasize that the reference is only to the single monitor in which it appears

`%{monitor.VAR}` references outside [monitors](#monitors) or [monitor defaults](#monitor-defaults) are ignored and reuslt in the literal string: "%{monitor.VAR}".

You can use both in a single value, like:

```yaml
tls:
  ca: "${SECRETS_DIR}/%{monitor.hostname}"

monitors:
  - hostname: db1
  - hostname: db2
```

Top-level `tls.ca` specifies a monitor default that applies to all monitors that don't explicily set the varaible.
If `SECRETS_DIR = /secrets`, the result is:

```yaml
monitors:
  - hostname: db1
    tls:
      ca: /secrets/db1
  - hostname: db2
    tls:
      ca: /secrets/db2
```

# Server Config

{: .config-section-title}
## api

The `api` section configures the [Blip API](../api/).

```yaml
api:
  bind: "127.0.0.1:9070"
  disable: false
```

### `bind`

{: .var-table }
|**Type**|string|
|**Valid values**|`addr:port`, `:port`|
|**Default value**|`127.0.0.1:9070`|

The `bind` variable sets the interface address and port that the API listens on.

### `disable`

{: .var-table }
|**Type**|bool|
|**Valid values**|`true`, `false`|
|**Default value**|`false`|

The `disable` variable disables the Blip API.

{: .config-section-title}
## monitor-loader

The `monitor-loader` section configures how Blip finds and loads MySQL instances.

```yaml
monitor-loader:
  freq: ""
  files: []
  stop-loss: ""
  aws:
    regions: []
  local:
    disable-auto: false
    disable-auto-root: false
```

### `freq`

{: .var-table }
|**Type**|string|
|**Valid values**|[Go duration string](https://pkg.go.dev/time#ParseDuration)|
|**Default value**||

The `freq` variable enables automatic monitor reloading.
It's off by default, which means moniitors are loaded only once at startup.

### `files`

{: .var-table }
|**Type**|list of strings|
|**Valid values**|file names|
|**Default value**||

The `files` variable specifies YAML files to load monitors from. Each file must have a `monitors` section.

### `stop-loss`

{: .var-table }
|**Type**|string|
|**Valid values**|&bull;&nbsp;"N%" (percentage) where N is an integer btween 0 and 100 (exclusive)<br>&bull;&nbsp;"N" where N is an integer greater than 0|
|**Default value**||

The `stop-loss` variable enables the [stop-lost feature](../server/monitor-loader.html#stop-loss).

### aws

### `regions`

{: .var-table }
|**Type**|list of strings|
|**Valid values**|AWS region names|
|**Default value**||

The `regions` variable sets which AWS regions to query for RDS intances.

### local

The `local` subsection has only two variables:

`disable-auto: true`
`disable-auto-root: true`

## `strict`

{: .var-table }
|**Type**|bool|
|**Valid values**|`true`, `false`|
|**Default value**|`false`|

The `strict` variable enables strict mode, which is disabled by default.
In strict mode, Blip returns certains errors rather than ignoring them.

# Monitor Defaults

Monitor defaults are top-level sections that set default values for monitors that do not set an explicit value.
Monitor defaults are useful when you have several MySQL instances to monitor and the configuration only differs by basic connection parameters, like hostname or socket.

For example, imagine that you have 10 monitors all with the same username and password.
Instead of setting `username` and `password` in all 10 monitors, you can set these variables once in the top-level `mysql` section:

```yaml
mysql:
  username: "defaultUser"
  password: "defaultPass"
monitors:
  - hostname: db1
  # ...
  - hostname: db10
```

The default `username` and `password` are applied to the 10 monitors because none of them explicitly set these variables.
If a monitor explicitly sets one of the variables, then its explicit value is used instead of the default value.

{: .note }
Monitor defaults are convenient, but explicit monitor configuraiton is more clear, so use monitor defatuls sparingly.
The intended use case is for variables that _must_ be consistent for all monitors.
For example, if Blip monitors Amazon RDS instances in region `us-east-1`, then setting monitor default `aws-rds.region: "us-east-1"` makes sense.

{: .config-section-title}
## aws-rds

The `aws-rds` section configures Amazon RDS for MySQL.

```yaml
aws-rds:
  iam-auth-token: false
  password-secret: ""
  region: ""
  disable-auto-region: false
  disable-auto-tls: false
```

### `iam-auth-token`

{: .var-table }
|**Type**|bool|
|**Valid values**|`true` or `false`|
|**Default value**|`false`|

The `iam-auth-token` variable enables [IAM database authentication](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/UsingWithRDS.IAMDBAuth.html).

When enabled, IAM authentication is the prefered authentication method

### `password-secret`

{: .var-table }
|**Type**|string|
|**Valid values**|AWS Secrets Manager ARN|
|**Default value**||

The `password-secret` variables sets the AWS Secrets Manager ARN that contains the MySQL user password.

### `region`

{: .var-table }
|**Type**|string|
|**Valid values**||
|**Default value**||

The `region` variable sets the AWS region.

### `disable-auto-region`

{: .var-table }
|**Type**|string|
|**Valid values**|`true` or `false`|
|**Default value**|`false`|

The `disable-auto-region` variable enables/disables automatic detection of the AWS region.

### `disable-auto-tls`

{: .var-table }
|**Type**|string|
|**Valid values**|`true` or `false`|
|**Default value**|`false`|

The `disable-auto-tls` variables enables/disables automatic use of the Amazon RDS certifcate authority (CA).
By default, Blip uses the 2019 AWS RDS CA, which is built-in (you don't need to configure anything).
See [AWS](../cloud/aws.html) for details.

{: .config-section-title}
## exporter

The `exporter` section configure Blip to emulate Prometheus `mysqld_exporter`.

```yaml
exporter:
  mode: ""
  flags:
    web.listen-address: "127.0.0.1:9104"
    web.telemetry-path: "/metrics"
```

### `mode`

{: .var-table }
|**Type**|string|
|**Valid values**|`dual` or `legacy`|
|**Default value**||

The `mode` variables enables the [Prometheus emualation feature](../prometheus/).
When set to `dual`, Blip runs normally _and_ emulates Prometheus.
When set to `legacy`, Blip runs _only_ emulates Prometheus.
The feature is disabled by default.

### `flags`

{: .var-table }
|**Type**|key-value map (string: string)|
|**Valid values**|(see list below)|
|**Default value**|(see list below)|

The `flag` variable is a key-value map of strings for certain Prometheus mysqld_exporter flags:

* `web.listen-address` (default: `127.0.0.1:9104`)
* `web.telemetry-path` (default: `/metrics`)

{: .config-section-title}
## heartbeat

The `heartbeat` section configures the [Blip heartbeat feature](../hearbeat/).

```yaml
heartbeat:
  freq: ""
  table: blip.heartbeat
```

### `freq`

{: .var-table }
|**Type**|string|
|**Valid values**|[Go duration string](https://pkg.go.dev/time#ParseDuration)|
|**Default value**||

The `freq` variables sets how frequently heartbeats are written.
See [Hearbeat](../hearbeat/) for details.

### `table`

{: .var-table }
|**Type**|string|
|**Valid values**|valid MySQL table name|
|**Default value**||

The `table` variables sets the Blip heartbeat table.
The default database is `blip` if the table name is not database-qualified like `db.heartbeat`.

{: .config-section-title}
## mysql

The `mysql` section configures how to connect to MySQL.

```yaml
mysql:
  hostname: ""
  mycnf: ""
  password: ""
  password-file: ""
  socket: ""
  timeout-connect: "5s"
  username: "blip"
```

This is the most important and common seciton since it configures how Blip connects to MySQL.
It's also the only section that becomes top-level in each [monitor config](#monitors): in a monitor config, omit `mysql:` and configure these variables at the top level.

### `hostname`

{: .var-table }
|**Type**|string|
|**Valid values**|`hostname` or `hostname:port`|
|**Default value**||

The `hostname` variable sets the MySQL hostname.

### `mycnf`

{: .var-table }
|**Type**|string|
|**Valid values**|my.cnf file name|
|**Default value**||

The `mycnf` variable sets a my.cnf file to read.

Blip reads the `[client]` section of the my.cnf file:

|my.cnf File|Blip Variable|
|------|----|
|host|[`hostname`](#hostname)|
|password|[`password`](#password)|
|port|Appended to [`hostname`](#hostname)|
|socket|[`socket`](#socket)|
|ssl-ca|[`tls.ca`](#ca)|
|ssl-cert|[`tls.cert`](#cert)|
|ssl-key|[`tls.key`](#key)|
|user|[`username`](#username)|

### `username`

{: .var-table }
|**Type**|string|
|**Valid values**||
|**Default value**||

The `username` variable sets the MySQL username.

### `password`

{: .var-table }
|**Type**|string|
|**Valid values**||
|**Default value**||

The `password` variable sets the MySQL password.

### `password-file`

{: .var-table }
|**Type**|string|
|**Valid values**||
|**Default value**||

The `password-file` variable sets a file from which Blip reads the MySQL password.

### `socket`

{: .var-table }
|**Type**|string|
|**Valid values**||
|**Default value**||

The `socket` variable sets the MySQL socket.

### `timeout-connect`

{: .var-table }
|**Type**|string|
|**Valid values**|[Go duration string](https://pkg.go.dev/time#ParseDuration)|
|**Default value**||

The `timeout-connect` variable sets the connection timeout.

{: .config-section-title}
## plans

The `plans` section configures the source of [plans](../plans/).

```yaml
plans:
  files: ["plan.yaml"]
  table: "blip.plans"
  monitor: {}
  adjust:
    # See below
```

### `files`

{: .var-table }
|**Type**|list of strings|
|**Valid values**|file names|
|**Default value**|`plans.yaml`|

The `files` variable is a list of file names from which to load plans.
Blip attempts to load the default, `plans.yaml`, but it is not required and does not cause an error if the file does not exist.
Instead, in this case, Blip uses a default built-in plan.
If plan files are explicitly configured, Blip only reads those plan files.

### `monitor`

{: .var-table }
|**Type**|dictonary|
|**Valid values**|[Monitor](#monitors)|
|**Default value**||

The `monitor` variable configures the MySQL instance from which the [`table`](#table-1) is loaded.

### `table`

{: .var-table }
|**Type**|string|
|**Valid values**|valid MySQL table name|
|**Default value**||

The `table` variable configures the MySQL table name from which plans are loaded.
See

### adjust

The `adjust` subection of the `plan` section configures the [Level Plan Adjuster (LPA) feature](../monitor/level-adjuster.hmtml).

```yaml
plans:
  adjust:
    offline:
      after: ""
      plan: ""
    standby:
      after: ""
      plan: ""
    read-only:
      after: ""
      plan: ""
    active:
      after: ""
      plan: ""
```

Each of the four sections (corresponding to the four [connection states](../monitor/level-adjuster.html#connection-states)) have the same two variables:

#### `after`

{: .var-table }
|**Type**|string|
|**Valid values**|[Go duration string](https://pkg.go.dev/time#ParseDuration)|
|**Default value**||

The `after` variable sets how long before the state takes effect.

#### `plan`

{: .var-table }
|**Type**|string|
|**Valid values**|plan name|
|**Default value**||

The `plan` variable sets the plan to load when the state takes effect.

{: .config-section-title}
## sinks

The `sinks` section configures [built-in metrics sinks](../metrics/sinks.html#built-in) and [custom metrics sinks](../metrics/sinks.html#custom).
This section is a map of maps:

```yaml
sinks:
  sinkName1:
    option1: value1
  sinkName2:
    option1: value1
```

Blip has three built-in sinks named `log`, `singalfx`, and `chronosphere`.
The options for each are listed below.

### chronosphere

|Key|Value|Default|
|---|-----|-------|
|`url`|Remote write URL|`http://127.0.0.1:3030/openmetrics/write`|

### log

The Blip built-in `log` sink has no options.

### signalfix

|Key|Value|Default|
|---|-----|-------|
|`auth-token`|API authentication token||
|`auth-token-file`|File to read API auth token from||

{: .config-section-title}
##  tags

The `tags` section sets user-defined key-value pairs (as strings) that are passed to each sink.
For example (using [interpolation](#interpolation)):

```yaml
tags:
  env: ${ENVIRONMENT:-dev}
  dc: ${DATACENTER:-local}
  hostname: %{monitor.hostname}
```

Blip calls these "tags", but each sink might have a different term for the same concept.
For example, with SignalFx these are called "dimensions".
But the concept is the same: metadata (usually string key-value pairs) attached to metrics that describe or annotate the metrics for grouping, aggregation, or filtering when display in graphs/charts.

The [built-in metrics sinks](../metrics/sinks.html#built-in) automatically send all tags with metrics.
For example, the `signalfx` sink sends the tags as SingalFx dimensions.

{: .config-section-title}
## tls

The `tls` section configures TLS certificates.

```yaml
tls:
  ca: ""
  cert: ""
  key: ""
```

You can specify only `tls.ca`, or `tls.cert` and `tls.key`, or all three; any other combination is invalid.

{: .note}
By default, Blip does not use TLS for MySQL connections _except_ when using AWS; see section [`aws-rds`](#aws-rds) or [AWS](../cloud/aws.html).

### `ca`

{: .var-table }
|**Type**|string|
|**Valid values**|file name|
|**Default value**||

The `ca` variables sets the certificate authority file.

### `cert`

{: .var-table }
|**Type**|string|
|**Valid values**|file name|
|**Default value**||

The `cert` variables sets the public certificate file.

### `key`

{: .var-table }
|**Type**|string|
|**Valid values**|file name|
|**Default value**||

The `key` variables sets the private key file.

# Monitors

The `monitors` section is a list of MySQL instances to monitor.
Each instance is a YAML dictionary containing any of the [monitor default sections](#monitor-defaults) with one exception: `mysql` variables are top-level in a monitor.
The example below shows two different MySQL instances to monitor.

```yaml
monitors:

  - hostname: db1.local
    username: metrics
    password-file: "/secret/db-password"
    heartbeat:
      freq: 1s

  - mycnf: "/secret/my.cnf"
    exporter:
      mode: legacy
```

The first MySQL instance is configured in lines 3-7.
(Note the single, leading hyphen on line 3 that denotes an item in a YAML list.)
The first three variables&mdash;`hostname`, `username`, and `password-file`&mdash;are [`mysql`](#mysql) variables but in a monitor they are top-level.
But all other sections, like [`heartbeat`](#heartbeat) and its variable `freq`, are exactly the same in a monitor.

The second MySQL instance is configured in lines 9-11.
Variable `mycnf` belongs to section `mysql`, but again: in a monitor, [`mysql`](#mysql) variables are top-level.
Section [`exporter`](#exporter) is exactly the same in a monitor.

<b>Refer to [Monitor Defaults](#monitor-defaults) for configuring MySQL instances, and remember: [`mysql`](#mysql) variables are top-level in a monitor (omit `mysql:` and include the variables directly).<b>

Monitors have two variables that only appear in monitors: `id` and `meta`.

### `id`

{: .var-table }
|**Type**|string|
|**Valid values**|any string|
|**Default value**|(automatic)|

The `id` variable uniquely identifies the MySQL instance in Blip.

Every monitor has a unique ID that, by default, Blip sets automatically.
You can set monitor IDs manually, but it's better to let Blip set them automatically to avoid duplicates (which causes a fatal error).

Blip uses monitor IDs to track and report each MySQL instance in its own output and API.

Blip does _not_ use monitor IDs to identify MySQL instances for reporting metrics, but you can use them if you want.
For example:
```yaml
monitors:
  - id: db1
    hostname: db1.local
    tags:
      monitorId: %{monitor.id}
```
Since tags are passed to sinks (which report metrics), all sinks will receive the monitor ID.
(Sinks receive the monitor ID at the code-level too, so technically this example is not necessary.)

Monitor IDs are not garuanteed to be stable&mdash;they might change between Blip versions.
Therefore, do not rely on them outside of Blip for truly stable, unique MySQL instance identification.

### `meta`

{: .var-table }
|**Type**|key-value map (string: string)|
|**Valid values**|any strings|
|**Default value**||

The `meta` variable is a map of key-value strings extrensic to Blip.

"Extrensic to Blip" is a fancy but succinct way of saying that `meta` data is not used by Blip, but it can be used by you.
For example by contrast, all other variables are used by Blip in some way, for some purpose.
But not `meta`.
However, you can still reference `meta` in the config file and [plans](../plans/): `%{monitor.meta.KEY}`.

`meta` solves at least one problem: passing the source DSN for a replica from monitor config to plan.
Imagine you have two MySQL instances: `source.db` and `replica.db`, where the latter replicates from the former.
Problem is: there is no configuration section to define the source MySQL instance.

{: .note}
Configuring replication sources in Blip is not an easy problem to solve because replication topologies change when replicas are used for high availability, and [MySQL Group Replication](https://dev.mysql.com/doc/refman/8.0/en/group-replication.html) has different requirements.

To solve this problem, you configure the source DSN in `meta`:

```yaml
monitors:
  - hostname: replica.db
    meta:
      source-host: source.db
      source-user: repl
      source-pass: pleaseDontLag
```

You could configure it in [`tags`](#tags), too, but tags are copied to sinks whereas metadata is not.
Therefore, this configuration is better placed in `meta` than `tags`.

Then in the plan, reference the metadata:

```yaml
replication:
  freq: 5s
  collect:
    repl:
      options:
        source-host: %{monitor.meta.source-host}
        source-user: %{monitor.meta.source-user}
        source-pass: %{monitor.meta.source-pass}
      metrics:
        - lag
```
