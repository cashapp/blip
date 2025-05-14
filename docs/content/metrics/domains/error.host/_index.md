---
title: "error.host"
---

The `error.host` domain includes server error metrics grouped by host from [error summary tables](https://dev.mysql.com/doc/refman/en/performance-schema-error-summary-tables.html).

{{< toc >}}

## Usage 

For example:

```
mysql> SELECT * FROM performance_schema.events_errors_summary_by_host_by_error WHERE error_number = 3024 AND HOST IN ('host1')\G
*************************** 1. row ***************************
             HOST: host1
     ERROR_NUMBER: 3024
       ERROR_NAME: ER_QUERY_TIMEOUT
        SQL_STATE: HY000
 SUM_ERROR_RAISED: 1
SUM_ERROR_HANDLED: 0
       FIRST_SEEN: 2025-04-30 16:30:16
        LAST_SEEN: 2025-04-30 16:30:16
```

The `SUM_ERR_RAISED` value is used for the Blip `raised` metric. `ERROR_NUMBER`, `ERROR_NAME`, and `HOST` are used to group the metric.

## Derived Metrics

None.

## Options

### `all`

|Value|Default|Description|
|-----|-------|-----------|
|yes  |&check; |Collect metrics on _all_ errors (not recommended)|
|no   | |Collect only metrics for the errors listed in the plan|
|exclude| |Collect metrics only for errors that are not listed in the plan|

### `total`

|Value|Default|Description|
|-----|-------|-----------|
|yes  |&check; |Return the total number of errors raised|
|no   | |Do not return the total number of errors raised|
|only| |Only return the total number of errors raised|

If `all` is not set to `yes` the total will only reflect those errors that are specifically included or not excluded from collection.

### `include`

| | |
|---|---|
|**Value Type**|CSV string of hosts|
|**Default**||

A comma-separated list of hosts to include. Overrides option `exclude`. 

### `exclude`

| | |
|---|---|
|**Value Type**|CSV string of hosts|
|**Default**||

A comma-separated list of hosts to exclude. Ignored if `include` is set. 

### `truncate-table`

|Value|Default|Description|
|---|---|---|
|yes| |Truncate table after each successful collection|
|no|&check; |Do not truncate table|

If the table is truncated (default), the metrics are delta counters.
Else, the values are cumulative counters.

### `truncate-timeout`

| | |
|---|---|
|**Value Type**|[Duration string](https://pkg.go.dev/time#ParseDuration)|250ms|
|**Default**|250ms|

Sets `@@session.lock_wait_timeout` to avoid waiting too long when truncating the table.
Normally, truncating a table is nearly instantaneous, but metadata locks can block the operation.

### `truncate-on-startup`

|Value|Default|Description|
|---|---|---|
|yes|&check;|Truncate source table on start of metric collection|
|no| |Do not truncate source table on startup|

Truncates the source table when Blip starts. The timeout use will be the same as specified by `truncate-timeout`.

## Group Keys

|Key|Value|
|---|---|
|`error_nunmber`|The error number or an empty string for a total|
|`error_name`|The short error name or an empty string for a total|
|`error_host`|The host for the error|

## Meta

None.

## Error Policies

|Name|MySQL Error|
|----|-----------|
|`truncate-timeout`|Error truncating table|

## MySQL Config

See
* [29.12.20.11 Error Summary Tables](https://dev.mysql.com/doc/refman/8.4/en/performance-schema-error-summary-tables.html)

and related pages in the MySQL manual.

## Changelog

|Blip Version|Change|
|------------|------|
|TBD      |Domain added|