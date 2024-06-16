---
title: "percona.response-time"
---

{{< hint type=warning title=Deprecated >}}
MySQL 5.7 was end of life (EOL) October 2023.
This Blip domain/collector is deprecated, no longer supported or developed, and will be removed in a future version.

Use the [`query.response-time`]({{< ref "metrics/domains/query.response-time" >}}) domain for MySQL or Percona Server 8.x.
{{< /hint >}}

The `percona.response-time` domain includes query response time percentiles from the Percona Server 5.7 [Response Time Distribution plugin](https://docs.percona.com/percona-server/5.7/diagnostics/response_time_distribution.html).

## Usage

This domain is deprecated, but `blip --print-domains` will print its usage.

The [`query.response-time`]({{< ref "metrics/domains/query.response-time" >}}) domain for MySQL or Percona Server 8.x supersedes this domain.

## Changelog

|Blip Version|Change|
|------------|------|
|v1.1.0      |Domain deprecated|
|v1.0.0      |Domain added|
