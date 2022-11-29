---
layout: default
parent: API
title: Blip
---

# Blip
{: .no_toc }

Blip API endpoints return information about Blip: version, configuration, and so forth.

* TOC
{:toc}

---

## GET /config

Return final Blip config including [`config.monitors`](../config/config-file#monitors) but not monitors loaded from other sources.

<div class="code-example" markdown="1">
GET
{: .label .label-green .mt-3 }
`/plugins`
{: .d-inline }

### Query
{: .no_toc }

|Key|Value|Required|Purpose|
|---|-----|--------|-------|
|`json`|(None)|No|Return config JSON-encoded instead of YAML-encoded|

### Response
{: .no_toc }

```yaml
api:
  bind: 127.0.0.1:7522
mysql:
  timeout-connect: 10s
  username: blip
```

Default response encoding is YAML.
Use query key `json` to return with JSON encoding.

</div> <!---------------------------------------------------------------------->

## GET /registered

Returns registered metric [collectors](../develop/collectors) and [sinks](../develop/sinks).

<div class="code-example" markdown="1">
GET
{: .label .label-green .mt-3 }
`/registered`
{: .d-inline }

### Response
{: .no_toc }

```json
{
  "collectors": [
    "status.global",
    "var.global"
  ],
  "sinks": [
    "datadog",
    "log",
    "signalfx"
  ],
}
```

</div> <!---------------------------------------------------------------------->

## GET /version

Returns the Blip version (same as [`--version`](../config/blip#--version)).

<div class="code-example" markdown="1">
GET
{: .label .label-green .mt-3 }
`/version`
{: .d-inline }

### Response
{: .no_toc }

```json
v1.0.75
```

Response encoding is text (string).
