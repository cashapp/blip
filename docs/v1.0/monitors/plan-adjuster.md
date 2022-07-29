---
layout: default
parent: Monitors
title: "Plan Adjuster"
---

# Plan Adjuster

The plan adjuster changes the plan based on the state of MySQL.

|State|Connected to MySQL|Collecting Metrics|Description|
|-----|------------------|------------------|-----------|
|`offline`|no|no|Completely offline, no connection to MySQL|
|`standby`|**YES**|**YES**|Connected to MySQL but HA passive mode|
|`read-only`|**YES**|**YES**|MySQL is read-only|
|`active`|**YES**|**YES**|MySQL is writable|

When HA is disabled, `standby` state is not used.
