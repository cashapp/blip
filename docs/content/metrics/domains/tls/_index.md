---
title: "tls"
---

The `tls` domain includes metrics about the status and configuration of TLS (SSL).

{{< toc >}}

## Usage

Industry best practice is to always use TLS with MySQL.
This domain reports a single derived metric, `enabled`, that should be monitored to ensure that every MySQL instance has TLS enabled.

## Derived Metrics

### `enabled`

| | |
|---|---|
|**Metric Type**|bool|
|**Value Units**||

True (1) if `have_ssl = YES`, else false (0).
Metrics sinks that don't support bool report this metric as a gauge.

{{< hint type=note >}}
`have_ssl` is deprecated as of MySQL 8.0.26.
This domain does not currently support the [`tls_channel_status` table](https://dev.mysql.com/doc/refman/8.0/en/performance-schema-tls-channel-status-table.html) but there is an [open issue](https://github.com/cashapp/blip/issues/133) to fix this.
{{< /hint >}}

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
