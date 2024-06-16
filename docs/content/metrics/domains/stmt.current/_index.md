---
title: "stmt.current"
---

The `stmt.current` domain includes metrics about statements (queries) from Performance Schemna table [`events_statements_current`](https://dev.mysql.com/doc/refman/en/performance-schema-events-statements-current-table.html).

{{< toc >}}

## Usage

Currently, this domain reports only two derived metrics: `slow` and `slowest`.
These metrics are used to monitor slow queries that could signal a problem.

## Derived Metrics

### `slow`

| | |
|---|---|
|**Metric Type**|gauge|
|**Value Units**|queries, [0, inf.)|

The number of queries running longer than [`slow-threshold`](#slow-threshold).

### `slowest`

| | |
|---|---|
|**Metric Type**|gauge|
|**Value Units**|microseconds|

The duration of the oldest active query in microseconds.

## Options

### `slow-threshold`

| | |
|---|---|
|**Value Type**|[Duration string](https://pkg.go.dev/time#ParseDuration)|250ms|
|**Default**|1.0s|

How long before an actively running query is counted as slow.

## Group Keys

None.

## Meta

None.

## Error Policies

None.

## MySQL Config

See [29.1 Performance Schema Quick Start](https://dev.mysql.com/doc/refman/en/performance-schema-quick-start.html) and related pages in the MySQL manual.

## Changelog

|Blip Version|Change|
|------------|------|
|v1.0.0      |Domain added|
