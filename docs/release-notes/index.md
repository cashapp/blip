---
layout: default
title: "Release Notes"
nav_order: 900
permalink: /release-notes
---

# Blip Release Notes

## v1.0

### v1.0.2 (03 Jul 2023)

* `datadog` sink:
  * Fixed timestamps: DD expects timestamp as seconds, not milliseconds
  * Send new `event.SINK_ERROR` and debug DD API errors on successful request
* `query.response-time` and `wait.io.table` collectors:
  * Added `truncate-timeout` option and error policy
  * Fixed docs: option `truncate-table` defaults to "yes"
* Fixed GitHub Dependabot alerts

### v1.0.1 (02 Mar 2023)

* `datadog` sink:
  * Fixed intermittent panic
  * Fixed HTTP error 413 (payload too large) by dynamically partitioning metrics
  * Added option `api-compress` (default: yes)
* `repl` collector:
  * Added option `report-not-a-replica`
  * Moved pkg vars `statusQuery` and `newTerms` to `Repl` to handle multiple collectors on different versions
  * Fixed docs (only `repl.running` is currently collected)
* Updated `aws/AuthToken.Password`: pass context to `auth.BuildAuthToken`
* Fixed GitHub Dependabot alerts
* Fixed `blip.VERSION`

### v1.0.0 (22 Dec 2022)

* First GA, production-ready release.
