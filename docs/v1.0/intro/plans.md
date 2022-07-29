---
layout: default
title: "3. Plans"
parent: Introduction
nav_order: 4
---

# Plans

A _level plan_ (or _plan_ for short) specifies which metrics to collect and how often.

_How often_ is the main problem that plans were invented to solve.
Consider metric `Queries` (used to calculate QPS) versus a metric to report database size.
`Queries` should be collected every few seconds because QPS a key performance indicator (KPI), so higher resolution (more frequent collection) is better.
But database size should be collect every few minutes because it does not change quickly, so lower resolution (less frequent collection) is better.
Plans make it possible and easy to collect different metrics at different times.
Here's how a plan looks for this example:

```yaml
kpi:                # level name
  freq: 1s          # collection frequency
  collect:
    status.global:  # domain
      metrics:
        - queries   # metrics in domain
size:
  freq: 5m
  collect:
    size.database:
      # Defaults
```

Each unique frequency in a plan is a _level_.
The plan snippet above has two levels: `kpi` and `size`.
The metrics listed (by domain) for each level are collect at the specified (`freq`): `kpi` every 1 second, and `data` every 5 minutes.
Through a combination of levels, domains, and metrics Blip can collect any metrics at any frequency, which allows for maximum efficiency when collecting, reporting, and storing metrics.

<div class="note">
Blip has a default plan that collects more than 60 of the most important MySQL metrics.
</div>

When level collection times overlap, Blip automatically collects all metrics in the overlapping levels.
This is called _leveling up_ because levels are sorted ascending by frequency and Blip calculates the highest level to collect.
For example, suppose a plan specifies three levels:

![Three Levels](/blip/assets/img/three-levels.png)

When these level overlap in time (measured in seconds since Blip started collecting metrics for this plan), Blip collects each overlapping level:

![Three Levels](/blip/assets/img/level-times.png)

At 5, 10, and 15 seconds, Blip collects only level 1.
At 20 seconds , Blip collects metrics for both levels 1 and 2 because `20 mod 5 = 0` and `20 mod 20 = 0`, respectively.
At 30 seconds, Blip collects metrics for both levels 1 and 3 because `30 mod 5 = 0` and `30 mod 20 = 0`, respectively.
And at 60 seconds, Blip collects metrics for all three levels because `60 mod freq = 0`.
Then the cycle repeats: at time 65 seconds, Blip collects only level 1 again, and so forth.

Blip plans can do more, but for this introduction it's sufficient to know that they allow you to fine-tune metrics collection, which increases the quality of monitoring while reducing costs.

---

Last page; keep going: [Sinks&nbsp;&darr;](sinks)
