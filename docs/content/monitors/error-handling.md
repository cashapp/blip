---
title: "Error Handling"
---

Blip is designed to recover and retry on _all_ errors.
This design reflects the fact that monitoring requires higher reliability and availability than what is monitored.
As a result, even if MySQL is offline, or some transient failure prevents Blip from collecting metrics, it will keep trying.

## Metrics Collection

Monitors do not stop on error, they keep retrying.

When a collector encounters an error:

1. Error policy
2. Engine
3. LPC

If no error policy is defined, the collector returns the error immediately to the engine.

The engine tries to collect all metrics.
If one collector returns an error, it does not affect other collectors.
The engine saves errors and reports them per-collector in its status.
It returns a variable success to the LPC: no errors (all metrics collected), some collected, none collected.

The LPC reports an error unless the engine reports no errors.
It keeps trying all metric collect as usual at the next interval.

## MySQL Connection

On startup, if Blip cannot connect to MySQL, it tries forever with an exponential backoff.
Once it's able to connect, it starts collecting metrics.
Once it has started collecting metrics, any connection errors are handled as defined above.

## Panic

All goroutines in Blip recover from panic with a few exceptions:

* Prom API
