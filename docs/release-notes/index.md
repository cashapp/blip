---
layout: default
title: "Release Notes"
nav_order: 900
permalink: /release-notes
---

# Blip Release Notes

## v1.0

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
