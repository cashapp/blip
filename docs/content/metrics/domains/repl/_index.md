---
title: "repl"
---

The `repl` domain includes metrics from multiple sources related to MySQL replication status.

{{< hint type=note >}}
This domain does _not_ collect replication lag.
Use the [`repl.lag`]({{< ref "metrics/domains/repl.lag/" >}}) to collection replication lag.
{{< /hint >}}

{{< toc >}}

## Usage

Currently, the domain reports only one derived metric: [`running`](#running).
It uses `SHOW REPLICA STATUS` (or `SHOW SLAVE STATUS` prior to 8.0.22) to determine if a replica is running.

## Derived Metrics

### `running`

|Value|Meaning|
|-----|-------|
|1|&nbsp;&nbsp;&#9745;MySQL is a replica<br>&nbsp;&nbsp;&#9745;`Slave_IO_Running=Yes`<br>&nbsp;&nbsp;&#9745;`Slave_SQL_Running=Yes`<br>&nbsp;&nbsp;&#9745;`Last_Errno=0`<br>|
|0|MySQL is a replica, but IO and SQL threads are not running or a replication error occurred|
|-1|MySQL is [not a replica](#report-not-a-replica): `SHOW REPLICA STATUS` returns no output|

This metric is intended for alerting: alert if `running` is zero for too long.

Replication lag does _not_ affect this metric: replication can be running but lagging.
Monitor and alert on replication lag separately.

## Options

### `report-not-a-replica`

|Value|Default|Description|
|---|---|---|
|yes| |Report `running = -1` if not a replica.|
|no|&check;|Drop the metric if not a replica.|

## Group Keys

None.

## Meta

|Key|Value|
|---|---|
|`source`|`Source_Host` or `Master_Host`|

## Error Policies

|Name|MySQL Error|
|---|---|
|`access-denied`|1227: access denied on 'SHOW REPLICA STATUS' (need REPLICATION CLIENT priv)"|

## MySQL Config

MySQL must be configured as a replica.

## Changelog

|Blip Version|Change|
|------------|------|
|v1.0.1      |Add [`report-not-a-replica`](#report-not-a-replica)|
|v1.0.0      |Domain added|
