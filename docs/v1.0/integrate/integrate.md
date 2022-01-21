---
layout: default
title: Integrate
nav_order: 140
permalink: /v1.0/integrate
---

# Integrate

Blip was designed from the ground up to integrate with your MySQL environment.
To accomplish that, Blip has two mains points of integration:

_Plugins_
: Plugins are function callbacks that let you override specific functionality of Blip.
Every plugin is optional: if specified, it overrides the built-in functionality.

_Factories_
:  Factories are interfaces that let you override certain object creation of Blip.
Every factory is optional: if specified, it overrides the built-in factory.


## Plugins

### LoadConfig

```go
LoadConfig func(Config) (Config, error)
```

### LoadMonitors

```go
LoadMonitors func(Config) ([]ConfigMonitor, error)
```

### LoadPlans

```go
LoadPlans func(ConfigPlans) ([]Plan, error)
```

### ModifyDB

```go
ModifyDB func(*sql.DB)
```

### StartMonitor

```go
StartMonitor func(ConfigMonitor) bool
```

### TransformMetrics

```go
TransformMetrics func(*Metrics) error
```
