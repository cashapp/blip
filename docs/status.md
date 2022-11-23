---
layout: default
title: Status
nav_order: 800
permalink: /status
---

# Status
{: .no_toc }

Blip has many features in various states of development and maturity.
Since a monitor needs to be more reliable than the thing it monitors, it's important to know the status of Blip features.
The following feature statuses reflect only stability and reliableness, _not_ feature completeness.

Alpha
: Feature is new and not widely tested in the real world. Expect bugs.

Beta
: Feature seems stable in the real world, but it's still considered new. There might be bugs.

<span class="ga">GA</span>
: Feature has been running in the real world for several months. Bugs are not expected.

## v1.0

### Metric Collectors

|Domain|Status|
|-------|------|
|[aws.rds](v1.0/metrics/domains#awsrds)|<span class="ga">GA</span>|
|[innodb](v1.0/metrics/domains#innodb)|<span class="ga">GA</span>|
|[query.global](v1.0/metrics/domains#queryglobal)|<span class="ga">GA</span>|
|[repl](v1.0/metrics/domains#repl)|<span class="ga">GA</span>|
|[repl.lag](v1.0/metrics/domains#repllag)|<span class="ga">GA</span>|
|[size.binlog](v1.0/metrics/domains#sizebinlog)|<span class="ga">GA</span>|
|[size.database](v1.0/metrics/domains#sizedatabase)|<span class="ga">GA</span>|
|[size.table](v1.0/metrics/domains#sizetable)|<span class="ga">GA</span>|
|[status.global](v1.0/metrics/domains#statusglobal)|<span class="ga">GA</span>|
|[stmt.current](v1.0/metrics/domains#stmtcurrent)|<span class="ga">GA</span>|
|[tls](v1.0/metrics/domains#tls)||
|[var.global](v1.0/metrics/domains#varglobal)|<span class="ga">GA</span>|

### Sinks

|Sink|Status|
|-------|------|
|[chronosphere](v1.0/sinks/chronosphere)|Alpha|
|[datadog](v1.0/sinks/datadog)|<span class="ga">GA</span>|
|[log](v1.0/sinks/log)|<span class="ga">GA</span>|
|[retry](v1.0/sinks/retry)|<span class="ga">GA</span>|
|[signalfx](v1.0/sinks/signalfx)|<span class="ga">GA</span>|

### Cloud

|Feature|Status|
|-------|------|
|[AWS IAM auth](v1.0/cloud/aws)|Beta|
|[AWS Secrets Manager](v1.0/cloud/aws)|Alpha|

### General

|Feature|Status|
|-------|------|
|[API](v1.0/api)|Alpha|
|[Heartbeat](v1.0/heartbeat)|Alpha|
|[Monitor Loading Stop-loss](v1.0/monitors/loading#stop-loss)|Alpha|
|[Plan Changing](v1.0/plans/changing)|Alpha|
|[Plan Error Policy](v1.0/plans/error-policy)|<span class="ga">GA</span>|
|[Plan File](v1.0/plans/file)|<span class="ga">GA</span>|
|[Plan Table](v1.0/plans/table)|Alpha|
|[Prometheus emulation](v1.0/prometheus)|Alpha|
