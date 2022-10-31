---
layout: default
title: "Quick Reference"
parent: Configure
---

# Quick Reference

The following are _quick references_, not complete or valid examples.
See [Config File](config-file) for details.

### Interpolation

```
${ENV_VAR}
%{monitor.hostname}
%{monitor.meta.region}
```

### Config File

```yaml
---
# ---------------------------------------------------------------------------
# Server config
# ---------------------------------------------------------------------------

api:
  bind: 127.1:7090
  disable: false

http:
  proxy: "http://proxy.internal"

monitor-loader:
  files: [one.yaml, two.yaml]
  stop-loss: 50%
  aws:
    regions: ["auto","us-east-1"]
  local:
    disable-auto: true
    disable-auto-root: true

# ---------------------------------------------------------------------------
# Monitor defaults
# ---------------------------------------------------------------------------

aws:
  auth-token: true
  disable-auto-region: false
  disable-auto-tls: false
  password-secret: "arn::::"
  region: "us-east-1"

exporter:
  flags:
    web.listen-address: ":9001"
    web.telemetry-path: "/metrics"
  mode: "dual" # or "legacy"

heartbeat:
  freq: 2s
  source-id: "source-host.local"
  role: "west-side"
  table: "blip.heartbeat"

mysql:
  mycnf: "/app/my.cnf"
  username: "blip"
  password: "..."
  password-file: "/var/shm/blip-passwd"
  socket: "/var/lib/mysql.sock"
  timeout-connect: 5s

plans:
  disable-auto: false
  files:
    - none.yaml
    - ro-plan.yaml
    - active-plan.yaml
  monitor: <monitor>
  table: "blip.plans"
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
  chronosphere:
    debug: "yes"
    strict-tr: false
    url: "http://127.0.0.1:3030/openmetrics/write"
  log:
    # No options
  noop:
    # No options
  retry:
    buffer-size: 60
    send-timeout: 5s
    send-retry-wait: 200ms
  signalfx:
    auth-token: ""
    auth-token-file: ""
    metric-prefix: ""
    metric-translator: ""

tags:
  env: ${ENVIRONMENT:-dev}
  dc: ${DATACENTER:-local}
  hostname: "%{monitor.hostname}"

tls:
  ca: "local.ca"
  cert: "/secrets/%{monitor.hostname}.crt"
  key: "/secrets/%{monitor.hostname}.key"

# ---------------------------------------------------------------------------
# Monitors (MySQL instances)
# ---------------------------------------------------------------------------

monitors:
  - id: host1 # Optional; Blip auto-sets based on MySQL config

    # -----------------------------------------------
    # mysql section variables are specified directly:
    hostname: host1.local
    mycnf: my.cnf
    username: metrics
    password: foo
    password-file: /dev/shm/mypasswd
    socket: /tmp/mysql.sock
    timeout-connect: 5s

    # ---------------------------------------------------------------------
    # Override monitor defaults by specifying any of the top-level sections
    tls:
      ca: new.ca # overrides monitor default 'tls.ca: local.ca'

    # ----------------------------------------------
    # Tags override monitor defaults or set new tags
    tags:
      hostname: "host1"  # overrides monitor default
      foo:      "bar"    # new tag

    # ---------------------------------------------------
    # Meta values unique to monitor (no monitor defaults)
    meta:
      repl-source: source-db.local
      canary: yes
```
