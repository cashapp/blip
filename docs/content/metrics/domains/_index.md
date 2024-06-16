---
geekdocCollapseSection: true
---

This page documents the metric domains from which Blip currently collects metrics.
Use [`--print-domains`]({{< ref "/config/blip#--print-domains" >}}) to list these domains from the command line:

```sh
$ blip --print-domains | less
```

Each domain begins with a table:

Blip version
: Blip version domain was added or changed.

MySQL config
: If MySQL must be explicitly or specially configured to provide the metrics.

Sources
: MySQL source of metrics.

Derived metrics
: [Derived metrics]({{< ref "collecting#derived-metrics" >}}). Omitted if none.

Group keys
: [Metric groups]({{< ref "reporting#groups" >}}). Omitted if none.

Meta
: [Metric meta]({{< ref "reporting#meta" >}}). Omitted if none.

Options
: [Domain options]({{< ref "collecting#options" >}}). Omitted if none.

Error policy
: MySQL error codes handled by optional [error policy]({{< ref "/plans/error-policy" >}}). Omitted if none.
