---
layout: default
title: "4. Plans"
parent: Introduction
nav_order: 4
---

# Plans

Blip _plans_ specify which metrics to collect and how often.

Plans solve two problems:
* MySQL metrics are unorganized
* Metrics should be collected at different frequencies

The [previous part](metrics) of the introduction addresses the first problem: domains organize MySQL metrics.
This part addresses the second problem.

Consider metric `Queries` (used to calculate QPS) versus a metric to report database size.
`Queries` should be collected every few seconds because QPS a key performance indicator (KPI), so higher resolution (collected more frequently) is better.
But database size should be collect every few minutes because it does not change quickly, so lower resolution (collected less frequently) is better.
Plans make this possible; here's a snippet for this example:

```yaml
kpi:                # level name
  freq: 1s          # collection frequency
  collect:
    status.global:  # domain
      metrics:
        - Queries   # metrics in domain
size:
  freq: 5m
  collect:
    size.database:
      # Defaults
```

The plan snippet above has two levels: `kpi` and `size`.
A _level_ is defined by a unique collection frequency: `1s` and `5m`, respectively.
Metrics listed for each level are collected at the specified frequency.

<div class="note">
Writing a plan is optional.
Blip has a default plan that collects more than 60 of the most important MySQL metrics.
</div>

When frequencies overlap, Blip automatically collects all metrics in the overlapping levels.
This is called _leveling up_ because levels are sorted ascending by frequency and Blip calculates the highest level to collect.
For example, suppose a plan specifies three levels:

![Three Levels](/blip/assets/img/three-levels.png)

When these frequencies overlap (measured in seconds), Blip collects each overlapping level:

![Three Levels](/blip/assets/img/level-times.png)

At 5, 10, and 15 seconds, Blip collects only level 1.
At 20 seconds, Blip collects metrics for both levels 1 and 2 because `20 mod 5 = 0` and `20 mod 20 = 0`, respectively.
At 30 seconds, Blip collects metrics for both levels 1 and 3 because `30 mod 5 = 0` and `30 mod 20 = 0`, respectively.
And at 60 seconds, Blip collects metrics for all three levels because `60 mod freq = 0`.
Then the cycle repeats: at time 65 seconds, Blip collects only level 1 again, and so forth.

Each monitor has its own plan&mdash;or copy of a shared plan.
Although the common use case is a single plan for all monitors, Blip can collect different metrics&mdash;at different frequencies&mdash;for each monitor.

Blip plans can do more, but for this introduction it's sufficient to know that they allow you to fine-tune metrics collection, which increases the quality of monitoring while reducing costs.

---

Last page; keep going: [Sinks&nbsp;&darr;](sinks)
