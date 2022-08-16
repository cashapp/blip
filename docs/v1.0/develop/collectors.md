---
layout: default
parent: Develop
title: Collectors
---

# Collectors

Collectors are low-level components that collect metrics for [domains](../metrics/domains).
See [metrics/status.global/global.go](https://github.com/cashapp/blip/blob/main/metrics/status.global/global.go) for a reference example with extensive code comments.

## Conventions

### Naming

Blip strives to report MySQL metric names as-is&mdash;no modifications&mdash;so that what you see in MySQL is what you get in Blip.

However, MySQL metric names are very inconsistent:

* `Foo_bar` (most common)
* `Foo_Bar` (replica status)
* `foo_bar` (InnoDB metrics)
* `foo_bar_count` (type suffix)
* `foo_bar_usec` (unit suffix)

For consistency, Blip metric names have three requirements:

1. Only `snake_case`
1. Always lowercase
1. No prefixes or suffixes

A fully-qualified metric name includes a domain: `status.global.threads_running`.
The metric name is always the last field (split on `.`).

### Domains

Blip metric domain names have three requirements:

1. Always lowercase
1. One word: `[a-z]+`
1. Singular noun: "size" not "sizes"; "query" not "queries"

Common abbreviation and acronyms are preferred, especially when they match MySQL usage: "thd" not "thread"; "pfs" not "performanceschema"; and so on.

{: .note }
Currently, domain names fit this convention, but if a need arises to allow hyphenation ("domain-name"), it might be allowed.
Snake case ("domain_name") and camel case ("domainName") are not allowed: the former is used by metrics, and the latter is not Blip style.

Blip uses sub-domains for two purposes: MySQL-grouped metrics, or metrics that are related but different.

The [`error` domain](domains#error) is an example of metrics that are related by different.
[`error.query`](domains#errorquery) and [`error.repl`](domains#errorepl) both comprise error-related metrics, hence the common root domain, but the specific metrics for each are different.

The[`status` domain](domains#status) is an example of MySQL-grouped metrics.
MySQL provides status metrics grouped by account, global, host, thread, and user.
(_Global_ is the most common, as in `SHOW GLOBAL STATUS`.)
Blip has a sub-domain for each group&mdash;`status.account`, `status.global`, and so on&mdash;that makes advanced plans like the following possible:

```yaml
level:
  collect:
    status.global:
      options:
        all: yes
    status.host:
      options:
        host: 10.1.1.1
      metrics:
        - queries
        - threads_running
```

The plan snippet above collects all global status metrics (`status.global`) but only two status metrics for host 10.1.1.1 (`status.host`).

MySQL-grouped metrics are an _explicit group_: `status.host` explicitly groups by `host`.
See [Reporting > Groups](reporting#groups) for more details.

{: .note}
For simplicity, sub-domains are called "domains" in the rest of the docs.
The term is only used here to clarify the distinction and usage.

## Code

### 1. Develop

Implement `blip.Collector` and `blip.CollectorFactory`.

### 2. Register

```go
metrics.Register(myFactory, "mydomain")
```

### 3. Reference

```yaml
level:
  collect:
    mydomain:
```
