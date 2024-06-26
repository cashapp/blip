---
title: "var.global"
---

The `var.global` domain includes global MySQL system variables ("sysvars").
These aren't metrics, but values are often needed for calculating or graphing limits.

{{< toc >}}

## Usage

The domain can collect numeric values from MySQL globals sysvars.
For example, it's common to report a percentage of maximum connection used: `Max_used_connections / Max_connections * 100`.

Since sysvars change infrequently, it is recommend to collect them infrequently: 1 hour or more.
However, some graphing solutions have problems handling sparse data points.
(For example, a 15 minute graph window might not look back 45 minutes for the last sysvar collected.)
In this case, increase the collection frequency to 15, 10, or 5 minutes.

These pseudo metrics are reported as gauges.

## Derived Metrics

None.

## Options

### `all`

|Value|Default|Description|
|-----|-------|-----------|
|yes  | |Collect _all_ 600+ metrics (NOT RECOMMENDED)|
|no   |&check;|Collect only sysvars listed in the plan|

### `source`

|Value|Default|Description|
|-----|-------|-----------|
|auto|&check;|Auto-determine best source|
|pfs||`performance_schema.global_variables`|
|select||`@@GLOBAL.metric_name`|
|show||`SHOW GLOBAL VARIABLES`|

## Group Keys

None.

## Meta

None.

## Error Policies

None.

## MySQL Config

None.

## Changelog

|Blip Version|Change|
|------------|------|
|v1.0.0      |Domain added|

