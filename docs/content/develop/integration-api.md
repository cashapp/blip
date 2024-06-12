---
---

Integration allows you to customize every major aspect of Blip without modifying its core code.
That makes it easy and safe to tailor Blip to meet any requirements and work in any environment.
For example, Blip does not collect [MySQL NDB](https://dev.mysql.com/doc/refman/en/mysql-cluster.html) metrics, but if you run NDB, you can write a [custom metrics collector]({{< ref "develop/collectors" >}}) for NDB, register it in Blip, then collect NDB metrics exactly the same as the built-in metric collectors.
In fact, the built-in metric collectors implement the same interface; the only difference is that Blip automatically registers them on startup.

How you integrate with Blip depends on what you're trying to customize:

|Customize|Integration API|
|:--------|:--------------|
|Collecting metrics|Metrics registry|
|Sending metrics|Sink registry|
|Metric names|Domain translator registry|
|Loading Blip config|Plugins|
|Loading monitors|Plugins|
|Loading plans|Plugins|
|AWS configs|Factories|
|Database connections|Factories|
|HTTP clients|Factories|
|Timeouts|Variables|

{{< hint type=important >}}
**All integrations must be set before calling [`Server.Boot`](https://pkg.go.dev/github.com/cashapp/blip/server#Server.Boot).**
{{< /hint >}}

## Registry

A registry maps a resource name to a factory that produces an object for the resource.
Blip has three registries:

|Registry|Resource Name|Factory Produces|
|:-------|:------------|:---------------|
|[Metrics Registry](https://pkg.go.dev/github.com/cashapp/blip/metrics#Register)|metric domain|[Collector](https://pkg.go.dev/github.com/cashapp/blip#Collector)|
|[Sink Registry](https://pkg.go.dev/github.com/cashapp/blip/sink#Register)|sink|[Sink](https://pkg.go.dev/github.com/cashapp/blip#Sink)|
|[Domain Translator Registry](https://pkg.go.dev/github.com/cashapp/blip/sink/tr#Register)|metric domain|[DomainTranslator](https://pkg.go.dev/github.com/cashapp/blip/sink/tr#DomainTranslator)|

The metric and sink registries are the most important: they allow you to make Blip collect any metrics and send metrics anywhere.

Every registry has a corresponding `Make` function that Blip uses to make objects for the named resources.
For example, when a [plan]({{< ref "intro/plans" >}}) collects the `status.global` domain, internally Blip makes a call like:

```go
collector, err := metrics.Make("status.global")
```

That works because Blip registered the built-in factory for the `status.global` metric domain on startup.
This is also how [custom metric collectors]({{< ref "develop/collectors" >}}) work: by registering a custom metric domain name and factory.

## Factories

[Factories](https://pkg.go.dev/github.com/cashapp/blip#Factories) are interfaces that let you override certain object creation of Blip.
Every factory is optional: if specified, it overrides the built-in factory.

## Plugins

[Plugins](https://pkg.go.dev/github.com/cashapp/blip#Plugins) are function callbacks that let you override specific functionality of Blip.
Every plugin is optional: if specified, it overrides the built-in functionality.

## Events

Implement a [Receiver](https://pkg.go.dev/github.com/cashapp/blip/event#Receiver), then call [event.SetReceiver](https://pkg.go.dev/github.com/cashapp/blip/event#SetReceiver) to override the default.
There is only one event receiver; use [Tee](https://pkg.go.dev/github.com/cashapp/blip/event#Tee) to chain receivers.

## Variables

Various packages have public variables that you can modify to fine-tune aspects of Blip.
For example, the [heartbeat package](https://pkg.go.dev/github.com/cashapp/blip/heartbeat#pkg-variables) has several timeout and retry wait durations.
