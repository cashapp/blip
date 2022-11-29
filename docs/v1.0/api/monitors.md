---
layout: default
parent: API
title: Monitors
---

# Monitors
{: .no_toc }

Monitor endpoints return information about monitors and provide some monitor controls through `POST` methods.

An endpoint might reference all monitors or a single monitor as documented.
Single-monitor endpoints require query key `id` to identify the monitor, which is a [`config.monitor.id`](../config/config-file#id) value.
Use [`GET /monitors`](#get-monitors) to discover monitor IDs if you don't set them explicitly in the Blip config file.

* TOC
{:toc}

---

## GET /monitors

Returns a list of all loaded monitors keyed on [`config.monitor.id`](../config/config-file#id).
The value for each is its redacted DSN (no password).

<div class="code-example" markdown="1">
GET
{: .label .label-green .mt-3 }
`/monitors`
{: .d-inline }

### Response
{: .no_toc }

```json
{
  "localhost": "blip:...@unix(/tmp/mysql.sock)/?parseTime=true"
}
```
</div> <!---------------------------------------------------------------------->

## POST /monitors/reload

Reloads all monitors.
See [Monitors / Loading / Reloading](../monitors/loading#reloading).

Reloading only affects new and removed monitors.
Monitors that did not change are not affected, even on error.

<div class="code-example" markdown="1">
GET
{: .label .label-red .mt-3 }
`/monitors/reload`
{: .d-inline }

### Response
{: .no_toc }

None on success (200 status code).

Error message on 4xx or 5xx status code.

### Status Codes
{: .no_toc }

<strong>409</strong>: Error reloading monitors
{: .bad-response .fs-3 }

<strong>412</strong>: [Stop-loss](../monitors/loading#stop-loss) prevented reloading
{: .bad-response .fs-3 }

</div> <!---------------------------------------------------------------------->

## POST /monitors/start?id=ID

Starts one monitor.
`id` query key is required.

<div class="code-example" markdown="1">
GET
{: .label .label-red .mt-3 }
`/monitors/start?id=ID`
{: .d-inline }

### Query
{: .no_toc }

|Key|Value|Required|Purpose|
|---|-----|--------|-------|
|`id`|[`config.monitor.id`](../config/config-file#id)|Yes|Monitor to start|

### Response
{: .no_toc }

None on success (200 status code).

Error message on 4xx or 5xx status code.

### Status Codes
{: .no_toc }

<strong>409</strong>: Error starting monitor
{: .bad-response .fs-3 }

</div> <!---------------------------------------------------------------------->

## POST /monitors/stop?id=ID

Stops one monitor.
`id` query key is required.

The monitor is stopped but not unloaded, which means it reported by status endpoints.

<div class="code-example" markdown="1">
POST
{: .label .label-red .mt-3 }
`/monitors/stop?id=ID`
{: .d-inline }

### Query
{: .no_toc }

|Key|Value|Required|Purpose|
|---|-----|--------|-------|
|`id`|[`config.monitor.id`](../config/config-file#id)|Yes|Monitor to stop|

### Response
{: .no_toc }

None on success (200 status code).

Error message on 4xx or 5xx status code.
</div>
