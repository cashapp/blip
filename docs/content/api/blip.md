---
---

Blip API endpoints return information about Blip: version, configuration, and so forth.

{{< toc >}}

## GET /config

Return final Blip config including [`config.monitors`]({{< ref "/config/config-file#monitors" >}}) but not monitors loaded from other sources.

## GET /plugins

*Query*

|Key|Value|Required|Purpose|
|---|-----|--------|-------|
|`json`|(None)|No|Return config JSON-encoded instead of YAML-encoded|

*Response*

```yaml
api:
  bind: 127.0.0.1:7522
mysql:
  timeout-connect: 10s
  username: blip
```

Default response encoding is YAML.
Use query key `json` to return with JSON encoding.

## GET /registered

Returns registered metric [collectors]({{< ref "/develop/collectors" >}}) and [sinks]({{< ref "/develop/sinks" >}}).

*Response*

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

## GET /version

Returns the Blip version (same as [`--version`]({{< ref "/config/blip#--version" >}})).

*Response*

```json
"v1.0.75"
```

Response encoding is text (string).
