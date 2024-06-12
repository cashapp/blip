---
weight: 9
geekdocCollapseSection: true
---

The Blip REST API provides runtime information and control of the server and monitors.

By default, Blip listens on `127.0.0.1:7522`.
Change this by setting [`config.api.bind`]({{< ref "/config/config-file#bind" >}}).

The API is optional; disabling has no effect on other parts of Blip.
You can disable the API by setting [`config.api.disable`]({{< ref "/config/config-file#disable" >}}) to true.

The API is organized by resources listed in the table of contents at the end of this page.

## Security

The API currently has no security mechanisms for authentication, OPTIONS, CORS, and so on.

The API does not use TLS.

## Responses

### Encoding

`GET` response bodies are JSON unless documented otherwise.

`POST` response bodies are text (string).

### Status Codes

All endpoints use standard HTTP status codes:

* 200 on success
* 3xx on client errors
* 404 on monitor not found
* 5xx on server errors

Only special status codes are documented explicitly.
