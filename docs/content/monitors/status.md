---
title: "Status"
---

Calling API endpoint [`/status/monitors`]({{< ref "api/status#get-statusmonitors" >}}) returns real-time status for every monitor:

```json
{
  "localhost": {
    "dsn": "blip:...@unix(/tmp/mysql.sock)/?parseTime=true",
    "engine-plan": "default-mysql",
    "heartbeat-reader": "25337641430 ms lag from node2 (), next in 1s",
    "level-collect": "last collected and sent metrics for default-mysql/performance at 2022-11-28T20:37:03-05:00 in 2.249204ms",
    "level-collector": "idle; started collecting default-mysql/performance at 2022-11-28T20:37:03-05:00",
    "level-plan": "default-mysql",
    "level-state": "active",
    "monitor": "running since 2022-11-28T20:36:57-05:00"
  }
}
```

The output is keyed on monitor ID.
