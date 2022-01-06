---
layout: default
title: "Plans"
parent: Introduction
nav_order: 4
---

# Plans

A _level plan_ (or _plan_ for short) configures which metrics to collect.
Plans solve three problems:

* Which metrics to collect?
* How to collect those metrics?
* How _often_ to collect those metrics?

The first problem arises because there are over 1,000 MySQL metrics but 0 standards for which metrics to collect.
Some engineers collect nearly all metrics and use what they need in graphs.
Some engineers collect as few metrics as possible due to costs.
Some engineers don't know what to collect, relying on monitor defaults to be useful.

Plans help solve the first problem by not hard-coding which metcis to collect.
Write a plan to collect only the metrics you need.

<div class="note">
Blip has a built-in default plan that collects more than 60 of the most important MySQL metrics, which might be all you need.
</div>

The second problem arises because there are many versions and distributions of MySQL, which makes some metrics a moving target.
For example: where do you collect a MySQL replication lag metric?
The oldest and perhaps still most common source is `Seconds_Behind_Master` in the `SHOW SLAVE STATUS` output.
But those two changed to `Seconds_Behind_Source` and `SHOW REPLICA STATUS`, respectively.
And what if you don't use those and, instead, use [pt-heartbeat](https://www.percona.com/doc/percona-toolkit/LATEST/pt-heartbeat.html) or the Blip built-in heartbeat?
Or what if you're running [MySQL Group Replication](https://dev.mysql.com/doc/refman/8.0/en/group-replication.html)?
Or what if you run MySQL in the cloud and the cloud provider emits its own replication lag metric?

Plans help solve the second problem by using _metric domains_ (or _domains_ for short) to name logically-related group of MySQL metrics.
Probably the most well known group is `SHOW GLOBAL STATUS`, to which Blip gives the domain name `status.global`.
A replication lag metric is scoped within the `repl` domain (short for "replication"), which hides (abstarcts way) the technical details of how it's collected.
When you write a plan that collects replication lag, the plan works everywhere because domains specify _which_ metrics to collect, not necessarily _how_ to collect them.

The third problem arises from cost and storage limits: if everything was fast and free, you would collect all metrics every 1 second.
But this is (almost) never done because it requires signfiicant storage and processing, which lead to significant costs.
Instead, the norm is collecting all metrics every 10, 20, or 30 seconds.
But even 10 seconds is too long for a busy database because, for example, at only 5,000 QPS, that resolution averages out the metrics for 50,000 queries.

Plans help solve the third problem by allowing you to collect different metrics at different frequencies&mdash;which is the "level" in the full term: "level plan".
It helps to remember as: "Higher the level, higher the wait (time between collection)."
For example, imagine three levels as shown below.

![Three Levels](/assets/img/three-levels.png)

Level 1, the base level, is collected frqeuently (shortest wait time): every 5 seconds.
Level 2 is collected less frequently: every 20 seconds.
Level 3, the highest level, is collected the most infrequently (longest wait time): every 30 seconds.

Blip automatically combines levels when they overlap and collects all metrics at that time.

![Three Levels](/assets/img/level-times.png)

At 20 seconds (since Blip started collecting metrics for this plan), Blip collects metrics for both levels 1 and 2 because `20 mod 5 = 0` and `20 mod 20 = 0`, respectively.
At 30 seconds, Blip collects metrics for both levels 1 anbd 3 because `30 mod 5 = 0` and `30 mod 20 = 0`, respectively.
And at 60 seconds, Blip collecgts metrics for all three levels because `60 mod freq = 0`.

Blip plans can do more, but for this introduction it's sufficient to know that they allow you to fine-tune metrics collection, which increases the quality of monitoring while reducing costs.

---

Keep going: [Quick Start&nbsp;&darr;](../quick-start/)
