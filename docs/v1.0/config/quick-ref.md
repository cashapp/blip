---
layout: default
title: "Quick Reference"
parent: Configure
nav_order: 10
---

# Quick Reference

The following are _quick references_, not complete or valid examples.
See [Config File](config-file) for details.

### Interpolation

```
${ENV_VAR}
%{monitor.hostname}
```

### Config File

```yaml
---
# ---------------------------------------------------------------------------
# Blip server
# ---------------------------------------------------------------------------

api:
  bind: 127.1:7090
  disable: false

monitor-loader:
  freq: 60s
  files: [one.yaml, two.yaml]
  stop-loss: 50%
  aws:
    regions: ["auto","us-east-1"]
  local:
    disable-auto: true
    disable-auto-root: true

strict: true

# ---------------------------------------------------------------------------
# Defaults for monitors
# ---------------------------------------------------------------------------

aws-rds:
  iam-auth-token: true
  password-secret: "arn::::"
  region: "us-east-1"
  disable-auto-region: false
  disable-auto-tls: false

exporter:
  mode: dual|legacy
  flags:
    web.listen-address: :9001

heartbeat:
  freq: 1s
  table: blip.heartbeat

mysql:
  mycnf: my.cnf
  username: blip
  password: blip
  password-file: ""
  timeout-connect: 5s

plans:
  files:
    - none.yaml
    - ro-plan.yaml
    - active-plan.yaml
  table: blip.plans
  monitor: <monitor>
  adjust:
    offline:
      after: 1s
      plan: none.yaml
    standby:
      after: 1s
      plan: none.yaml
    read-only:
      after: 1s
      plan: ro-plan.yaml
    active:
      after: 1s
      plan: active-plan.yaml

sinks:
  retry:
    buffer-size: 60
    send-timeout: 5s
    send-retry-wait: 200ms
  chronosphere:
    url: "http://127.0.0.1:3030/openmetrics/write"
  signalfx:
    auth-token: ""
    auth-token-file: ""
  log:
    # No options

tags:
  env: ${ENVIRONMENT:-dev}
  dc: ${DATACENTER:-local}
  hostname: %{monitor.hostname}

tls:
  ca: local.ca
  cert: /secrets/$%{monitor.hostname}.crt
  key: /secrets/%{monitor.hostname}.key

# ---------------------------------------------------------------------------
# MySQL instances to monitor
# ---------------------------------------------------------------------------

monitors:
  - id: host1
    # mysql:
    hostname: host1.local
    socket: /tmp/mysql.sock
    mycnf: my.cnf
    username: metrics
    password: foo
    password-file: /dev/shm/mypasswd
    timeout-connect: 5s
    aws-rds:
      password-secret: "arn::::"
      iam-auth-token: true
    exporter:
      mode: dual|legacy
      flags:
        "web.listen-address": 127.0.0.1:9104
        "web.telemetry-path": /metrics
    heartbeat:
      freq: 1s
      table: blip.heartbeat
    ha:
      # Reserved
    plans:
      table: "blip.plans"
      #monitor: <monitor>
      adjust:
        readonly:
          after: 2s
          plan: ro.yaml
        active:
          after: 1s
          plan: rw.yaml
    sinks:
      signalfx:
        auth-token: ""
        auth-token-file: ""
      log:
        # No options
      chronosphere:
        url: http://127.0.0.1:3030/openmetrics/write
    tags:
      env: staging
      monitor-id: %{monitor.id}
    tls:
      ca: my-ca
      cert: ${SECRETS}/%{monitor.hostname}.cert
      key:  ${SECRETS}/%{monitor.hostname}.key
    meta:
      source: host2.local
      canary: no
```
