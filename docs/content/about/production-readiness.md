---
weight: 2
---

A monitor must be more reliable than what it monitors.
But the paradox is that reliability is only achieved through extensive real-world usage.
Consequently, Blip is released with features at different levels of readiness:

New
: Feature is new and not widely tested in the real world. Expect bugs. Ready for early adopters.

Stable
: Feature seems stable in the real world, but it's still considered new. There might be bugs. Ready for pre-production testing and burn-in.

<span class="ga">Production</span>
: Feature has been running in the real world for several months. Bugs are not expected. Ready for production.

Feature readiness is documented here to help you make informed decisions about monitoring your databases with Blip.

## v1.x

Blip v1.x is <span class="ga">production</span> ready.

### Metric Collectors

|Domain|Readiness|
|-------|------|
|[aws.rds]({{< ref "metrics/domains/aws.rds/" >}})|<span class="ga">Production</span>|
|[innodb]({{< ref "metrics/domains/innodb/" >}})|<span class="ga">Production</span>|
|[repl]({{< ref "metrics/domains/repl" >}})|<span class="ga">Production</span>|
|[repl.lag]({{< ref "metrics/domains/repl.lag/" >}})|<span class="ga">Production</span>|
|[size.binlog]({{< ref "metrics/domains/size.binlog/" >}})|<span class="ga">Production</span>|
|[size.database]({{< ref "metrics/domains/size.database/" >}})|<span class="ga">Production</span>|
|[size.table]({{< ref "metrics/domains/size.table/" >}})|<span class="ga">Production</span>|
|[status.global]({{< ref "metrics/domains/status.global/" >}})|<span class="ga">Production</span>|
|[stmt.current]({{< ref "metrics/domains/stmt.current/" >}})|<span class="ga">Production</span>|
|[tls]({{< ref "metrics/domains/tls/" >}})|New|
|[var.global]({{< ref "metrics/domains/var.global/" >}})|<span class="ga">Production</span>|
|[wait.io.table]({{< ref "metrics/domains/wait.io.table/" >}})|Stable|

### Sinks

|Sink|Readiness|
|-------|------|
|[chronosphere]({{< ref "sinks/chronosphere" >}})|New|
|[datadog]({{< ref "sinks/datadog" >}})|<span class="ga">Production</span>|
|[log]({{< ref "sinks/log" >}})|<span class="ga">Production</span>|
|[prom-pushgateway]({{< ref "sinks/prom-pushgateway" >}})|New|
|[retry]({{< ref "sinks/retry" >}})|Stable|
|[signalfx]({{< ref "sinks/signalfx" >}})|<span class="ga">Production</span>|

### Cloud

|Feature|Readiness|
|-------|------|
|[AWS IAM auth]({{< ref "cloud/aws" >}})|Stable|
|[AWS Secrets Manager]({{< ref "cloud/aws" >}})|New|

### General

|Feature|Readiness|
|-------|------|
|[API]({{< ref "api" >}})|Stable|
|[Heartbeat]({{< ref "config/heartbeat" >}})|Stable|
|[Monitor Loading Stop-loss]({{< ref "monitors/loading#stop-loss" >}})|New|
|[Plan Changing]({{< ref "plans/changing" >}})|New|
|[Plan Error Policy]({{< ref "plans/error-policy" >}})|Stable|
|[Plan File]({{< ref "plans/file" >}})|<span class="ga">Production</span>|
|[Plan Table]({{< ref "plans/table" >}})|New|
|[Prometheus emulation]({{< ref "prometheus" >}})|New|
