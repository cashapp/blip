---
---

Status endpoints return real-time status of Blip and monitors in key-value maps with string values.

{{< toc >}}

## GET /status

Returns high-level Blip server status.

### Response

```json
{
  "monitor-loader": "1 monitors started at 2022-11-28T20:26:49-05:00",
  "monitors": "1",
  "server": "running since 2022-11-28T20:26:49-05:00",
  "started": "2022-11-28T20:26:49-05:00",
  "uptime": "22",
  "version": "1.0.75"
}
```

## GET /status/monitors

Returns monitor status for all monitors keyed on [monitor ID]({{< ref "/config/config-file#id" >}}).

### Response

```json
{
  "localhost": {
    "dsn": "blip:...@unix(/tmp/mysql.sock)/?parseTime=true",
    "engine-plan": "default-mysql",
    "heartbeat-reader": "22 ms lag from node2 (), next in 1s",
    "level-collect": "last collected and sent metrics for default-mysql/performance at 2022-11-28T20:37:03-05:00 in 2.249204ms",
    "level-collector": "idle; started collecting default-mysql/performance at 2022-11-28T20:37:03-05:00",
    "level-plan": "default-mysql",
    "level-state": "active",
    "monitor": "running since 2022-11-28T20:36:57-05:00"
  }
}
```
