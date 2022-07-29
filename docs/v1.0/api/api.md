---
layout: default
title: API
nav_order: 80
permalink: /v1.0/api
---

# API

## Server

Server endpoints return information about the `blip` instance (the server) and high-level information about monitors.

* TOC
{:toc}

---

## GET /status

Returns high-level Blip server status.

<div class="code-example" markdown="1">
GET
{: .label .label-green .mt-3 }
`/status`
{: .d-inline }

### Response
{: .no_toc }

```json
{
  "Started"      string            // ISO timestamp (UTC)
  "Uptime"       int64             // seconds
  "MonitorCount" uint              // number of monitors loaded
  "Internal"     map[string]string // Blip components
  "Version"      string            // Blip version
}
```

### Response Status Codes
{: .no_toc }

<strong>200</strong>: Successful operation.
{: .good-response .fs-3 .text-green-200 }
</div>

## GET /version

Return Bip version.

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

### Response Status Codes
{: .no_toc }

<strong>200</strong>: Successful operation.
{: .good-response .fs-3 .text-green-200 }
</div>
