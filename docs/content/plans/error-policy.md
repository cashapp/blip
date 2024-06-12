---
title: "Error Policy"
---

An error policy defines how a metric collector handles specific MySQL errors.
Error policies are optional, and most metric collectors do not define any.
To see which collectors have error policies, check the [domain reference]({{< ref "/metrics/domains" >}}) or use [`--print-domains`]({{< ref "/config/blip#--print-domains" >}}):

```sh
$ blip --print-domains

repl
	Replication status

	(No options)

	Errors:
		access-denied: MySQL error 1227: access denied on 'SHOW REPLICA STATUS' (need REPLICATION CLIENT priv)

	Metrics:
		running (gauge): 1=running (no error), 0=not running, -1=not a replica
```

The output above is truncated to highlight the error policy that the [`repl` collector]({{< ref "/metrics/domains#repl" >}}) handles: `access-denied`.
If MySQL returns error 1227 to the collector, it handles the error according to the error policy if defined.
The default error policy is `report,drop,retry`: report the error, drop the metric, and retry.

{{< hint type=note >}}
Default error policy: `report,drop,retry`: report the error, drop the metric, and retry.
{{< /hint >}}

A different error policy can be specified for the collector in the plan:

```yaml
collect:
  repl:
    metrics:
      - running
    errors:
      access-denied: "ignore,drop,stop"  # New error policy; override default
```

In the example above, instead of the default (`report,drop,retry`), the collector will _ignore_ the error, _drop_ the metric, and _stop_ trying to collect it.
The format of the error policy value is defined in the next section.

Since error policies handle specific MySQL errors, they are not intended for [general error handling]({{< ref "/monitors/error-handling" >}}).
Instead, error policies make it possible to write plans that are "best effort" depending on the MySQL instance: if a collector works, then great; but if not, an error policy prevents Blip from spewing errors.

## Format

An error policy value is a string and comma-separated value with three parts: `<report>,<metric>,<retry>`

Report:
* `ignore`: Silently ignore the error; report _nothing_ (not even an event)
* `report`: Report the metric (**default**)
* `report-once`: Report the metric only the first time it occurs, then ignore further errors

Metric:
* `drop`: Drop the metric (**default**)
* `zero`: Report zero value

Retry:
* `retry`: Keep trying to collect metric (**default**)
* `stop`: Stop collecting metric
