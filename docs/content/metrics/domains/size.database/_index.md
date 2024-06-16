---
title: "size.database"
---

The `size.database` domain includes metrics about database sizes.

{{< toc >}}

## Usage

Database size is calculated as:

```sql
SELECT
  table_schema AS db,
  SUM(data_length + index_length) AS bytes
FROM
  information_schema.tables
WHERE
  /* include or exclude list */
GROUP BY 1
```

Database size includes all secondary indexes.

{{< hint type=note >}}
Reported table size and size _on disk_ can vary due to several factors.
{{< /hint >}}

Since database sizes aren't expected to have large and rapid changes, best practice is to collect this domain infrequently: 5, 10, 15, 30, or 60 _minutes_.

## Derived Metrics

### `bytes`

| | |
|---|---|
|**Metric Type**|gauge|
|**Value Units**|bytes|

Database size in bytes.

## Options

### `exclude`

| | |
|---|---|
|**Value Type**|CSV string of db.table|
|**Default**|`mysql,information_schema,performance_schema,sys`|

A comma-separated list of database or table names to exclude (ignored if `include` is set).

### `include`

| | |
|---|---|
|**Value Type**|CSV string of db.table|
|**Default**||

A comma-separated list of database or table names to include (overrides option `exclude`).

### `like`

|Value|Default|Description|
|---|---|---|
|yes||Use LIKE pattern matching for `include` and `exclude`|
|no|&check;|Use literal database names|

If enabled (`like = yes`), the query becomes `(table_schema LIKE 'foo') OR (table_schema LIKE 'bar')`, where "foo" and "bar" are the included or excluded database names and can contain wildcards like `%`.
Else, the query becomes `table_schema IN ('foo', 'bar')`.

### `total`

|Value|Default|Description|
|---|---|---|
|yes|&check;|Report size of all tables combined|
|no| |Only report tables individually|


## Group Keys

|Key|Value|
|---|---|
|`db`|Database name, or empty string for all databases|

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

