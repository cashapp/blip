---
layout: default
title: Readiness
nav_order: 800
permalink: /ready
---

# Readiness
{: .no_toc }

A monitor must be more reliable than what it monitors.
But the paradox is that reliablity is only achieved through extensive real-world usage.
Consequently, Blip is released with features at different levels of readiness:

New
: Feature is new and not widely tested in the real world. Expect bugs. Ready for early adopters.

Stable
: Feature seems stable in the real world, but it's still considered new. There might be bugs. Ready for pre-production testing and burn-in.

<span class="ga">Production</span>
: Feature has been running in the real world for several months. Bugs are not expected. Ready for production.

Feature readiness is documented here to help you make informmed decisions about monitoring your databases with Blip.

## v1.0

Blip v1.0 is <span class="ga">production</span> ready.

### Metric Collectors

|Domain|Readiness|
|-------|------|
|[aws.rds](v1.0/metrics/domains#awsrds)|<span class="ga">Production</span>|
|[innodb](v1.0/metrics/domains#innodb)|<span class="ga">Production</span>|
|[query.global](v1.0/metrics/domains#queryglobal)|<span class="ga">Production</span>|
|[repl](v1.0/metrics/domains#repl)|<span class="ga">Production</span>|
|[repl.lag](v1.0/metrics/domains#repllag)|<span class="ga">Production</span>|
|[size.binlog](v1.0/metrics/domains#sizebinlog)|<span class="ga">Production</span>|
|[size.database](v1.0/metrics/domains#sizedatabase)|<span class="ga">Production</span>|
|[size.table](v1.0/metrics/domains#sizetable)|<span class="ga">Production</span>|
|[status.global](v1.0/metrics/domains#statusglobal)|<span class="ga">Production</span>|
|[stmt.current](v1.0/metrics/domains#stmtcurrent)|<span class="ga">Production</span>|
|[tls](v1.0/metrics/domains#tls)|New|
|[var.global](v1.0/metrics/domains#varglobal)|<span class="ga">Production</span>|

### Sinks

|Sink|Readiness|
|-------|------|
|[chronosphere](v1.0/sinks/chronosphere)|New|
|[datadog](v1.0/sinks/datadog)|<span class="ga">Production</span>|
|[log](v1.0/sinks/log)|<span class="ga">Production</span>|
|[retry](v1.0/sinks/retry)|Stable|
|[signalfx](v1.0/sinks/signalfx)|<span class="ga">Production</span>|

### Cloud

|Feature|Readiness|
|-------|------|
|[AWS IAM auth](v1.0/cloud/aws)|Stable|
|[AWS Secrets Manager](v1.0/cloud/aws)|New|

### General

|Feature|Readiness|
|-------|------|
|[API](v1.0/api)|Stable|
|[Heartbeat](v1.0/heartbeat)|New|
|[Monitor Loading Stop-loss](v1.0/monitors/loading#stop-loss)|New|
|[Plan Changing](v1.0/plans/changing)|New|
|[Plan Error Policy](v1.0/plans/error-policy)|Stable|
|[Plan File](v1.0/plans/file)|<span class="ga">Production</span>|
|[Plan Table](v1.0/plans/table)|New|
|[Prometheus emulation](v1.0/prometheus)|New|
