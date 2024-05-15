---
title: "MySQL User"
---

Minimum privileges:

* `SELECT ON performance_schema.*`

Recommend privileges:

* `SELECT, PROCESS, REPLICATION CLIENT ON *.*`

For database and table sizes, Blip needs `SELECT ON *.*`.
Although sizes are metadata, MySQL requires `SELECT` on an object to read its metadata.

Blip needs the `PROCESS` privilege to query `information_schema.innodb_metrics`.

{{< hint type=warning >}}
<b>Never grant <code>ALL</code> or <code>SUPER</code> privileges to the Blip MySQL user!</b>
{{< /hint >}}

## Password

The Blip MySQL user typically uses a password, but other authentication methods are supported too.
See [Monitors / MySQL Connection / Authentication]({{< ref "/monitors/mysql-connection#authentication" >}}).

## Heartbeat

For [heartbeat]({{< ref "heartbeat" >}}):

* `INSERT, UPDATE, DELETE ON blip.heartbeat`

{{< hint type=warning >}}
<b>Never grant write privileges to the Blip MySQL user except on the <a href="heartbeat#table">heartbeat table</a>!</b>
{{< /hint >}}

## Plan Table

If using a [plan table]({{< ref "/plans/table" >}}), the recommend privileges work since they grant `SELECT` on all tables.
But if you use the minimum privileges, then the Blip MySQL also requires:

* `SELECT ON blip.plans`
