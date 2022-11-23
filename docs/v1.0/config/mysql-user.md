---
layout: default
parent: Configure
title: MySQL User
---

# MySQL User

Minimum privileges:

* `SELECT ON performance_schema.*`

Recommend privileges:

* `SELECT, PROCESS, REPLICATION CLIENT ON *.*`

For database and table sizes, Blip needs `SELECT ON *.*`.
Although sizes are metadata, MySQL requires `SELECT` on an object to read its metadata.

Blip needs the `PROCESS` privilege to query `information_schema.innodb_metrics`.

## Heartbeat

For [heartbeat](../heartbeat):

* `INSERT, UPDATE, DELETE ON blip.heartbeat`

<p class="warn">
<b>Never grant <code>ALL</code> or <code>SUPER</code> privileges to the Blip MySQL user!</b>
</p>

<p class="warn">
<b>Never grant write privileges to the Blip MySQL user except on the <a href="../heartbeat#table">heartbeat table</a>!</b>
</p>
