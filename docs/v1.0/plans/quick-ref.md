---
layout: default
parent: Plans
title: "Quick Reference"
nav_order: 10
---

# Quick Reference

### Interpolation

```
${ENV_VAR}
%{monitor.hostname}
```

### Plan File

Following is a full Blip config file (YAML syntax).
This is only a reference to show all configuration variables.

```yaml
---
level_1:
  freq: 5s
  collect:
    domain1:
      options:
      	opt_1: value_1
	opt_N: value_N
      metrics:
        - metric_1
	- metric_N
level_N:
  freq: 10s
  collect:
    domain_1:
      options:
      	opt_1: value_1
	opt_N: value_N
      metrics:
        - metric_1
	- metric_N
```
