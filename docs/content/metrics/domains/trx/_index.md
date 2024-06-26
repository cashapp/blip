---
title: "trx"
---

The `trx` domain includes metrics about transactions.

{{< toc >}}

## Usage

Currently, the domain reports only one metric derived from Information Schema table [`innodb_trx`](https://dev.mysql.com/doc/refman/en/information-schema-innodb-trx-table.html): `oldest`.
This is useful for monitoring and alerting on long-running transactions that might signal a problem.

{{< hint type=note >}}
This domain does _not_ collect or report column values from `INFORMATION_SCHEMA.INNODB_TRX`.
If these metrics are needed, please create an issue or submit a PR.
{{< /hint >}}

Future version might report other metrics or use Performance Schema tables to obtain transaction metrics.

## Derived Metrics

### `oldest`

| | |
|---|---|
|**Metric Type**|gauge|
|**Value Units**|seconds|

Time of oldest active (still running) transaction in seconds, calculated as:

```sql
SELECT
  COALESCE(UNIX_TIMESTAMP(NOW()) - UNIX_TIMESTAMP(MIN(trx_started)), 0) t
FROM
  information_schema.innodb_trx;
```

## Options

None.

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
