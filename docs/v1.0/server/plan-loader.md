---
layout: default
parent: Server
title: "Plan Loader"
nav_order: 2
---

# Plan Loader

## Load Order

```yaml
---
# Zero config
#plans:
  #default: blip

plans:
  files: first.yaml

plans:
  files: [first.yaml,second.yaml]
  #default: first.yaml

plans:
  files: [first.yaml,second.yaml]
  default: second.yaml

plans:
  files: [first.yaml,second.yaml]
  adjust:
    freq: 1s
    readonly: first.yaml
    active: second.yaml

plans:
  files: first.yaml:second.yaml
  default: first.yaml
  adjust:     # ERROR: default and adjust mutually exclusive
    freq: 1s
    readonly: first.yaml
    active: second.yaml

plans:
  files: first.yaml:second.yaml
  table: blip.plans

plans:
  table: blip.plans # WHERE 1=1 (all rows)
  #default: WHERE monitorId IS NULL ORDER BY name ASC LIMIT 1

plans:
  table: blip.plans
  default: plan1

monitors:
  - id: host1
    plans:
      table: blip.plans # WHERE monitorId=host1
      #default: WHERE monitorId=host1 ORDER BY name ASC LIMIT 1

```

## Combining Levels

Levels are combined by the LPC (call stack):

```
sortedLevels()
changePlan()
```
