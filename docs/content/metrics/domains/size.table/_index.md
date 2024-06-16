---
title: "size.table"
---

The `size.table` domain includes metrics about table sizes.

{{< toc >}}

## Usage

Table size is calculated as:

```sql
SELECT
  table_schema AS db,
  table_name as tbl,
  COALESCE(data_length + index_length, 0) AS tbl_size_bytes
WHERE
  /* include or exclude list */
FROM
  information_schema.tables
```

Table size includes secondary indexes.

{{< hint type=note >}}
Reported table size and size _on disk_ can vary due to several factors.
If the difference is large, rebuild the table by performing a no-op schema change.
{{< /hint >}}

Since table sizes aren't expected to have large and rapid changes, best practice is to collect this domain infrequently: 5, 10, 15, 30, or 60 _minutes_.

## Derived Metrics

### `bytes`

| | |
|---|---|
|**Metric Type**|gauge|
|**Value Units**|bytes|

Table size in bytes.

## Options

### `exclude`

| | |
|---|---|
|**Value Type**|CSV string of db.table|
|**Default**|`mysql.*,information_schema.*,performance_schema.*,sys.*`|

A comma-separated list of database or table names to exclude (ignored if `include` is set).

### `include`

| | |
|---|---|
|**Value Type**|CSV string of db.table|
|**Default**||

A comma-separated list of database or table names to include (overrides option `exclude`).

### `total`

|Value|Default|Description|
|---|---|---|
|yes|&check;|Report size of all tables combined|
|no| |Only report tables individually|

## Group Keys

|Key|Value|
|---|---|
|`db`, `tbl`|Database and table name, or empty string for all tables (`total`)|

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
