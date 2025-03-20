---
title: "innodb.buffer-pool"
---

The `innodb.buffer-pool` domain includes InnoDB metrics from [`INFORMATION_SCHEMA.INNODB_BUFFER_POOL_STATS`](https://dev.mysql.com/doc/refman/8.4/en/information-schema-innodb-buffer-pool-stats-table.html). 

{{< toc >}}

## Usage 

The metric values collected are aggregated over all buffer pools.

For example:

```
mysql> SELECT SUM(POOL_SIZE) POOL_SIZE, SUM(FREE_BUFFERS) FREE_BUFFERS FROM INFORMATION_SCHEMA.INNODB_BUFFER_POOL_STATS\G
*************************** 1. row ***************************
   POOL_SIZE: 95110
   FREE_BUFFERS: 9322
```

The exact `NAME` value is used for the Blip metric name.

## Derived Metrics

None.

## Options

### `all`

|Value|Default|Description|
|-----|-------|-----------|
|yes  | |Collect _all_ 30+ metrics (not recommended)|
|no   |&check;|Collect only metrics listed in the plan|

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
|TBD      |Domain added|
