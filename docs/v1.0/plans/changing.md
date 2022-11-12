---
layout: default
parent: Plans
title: "Changing"
---

# Changing Plans

When `config.plans.change` is set, Blip changes plans while running based on the state of MySQL:

|State|Connected to MySQL|Collecting Metrics|Description|
|-----|------------------|------------------|-----------|
|`offline`|no|no|Completely offline, no connection to MySQL|
|`standby`|**YES**|**YES**|Connected to MySQL but HA passive mode|
|`read-only`|**YES**|**YES**|MySQL is read-only|
|`active`|**YES**|**YES**|MySQL is writable|
