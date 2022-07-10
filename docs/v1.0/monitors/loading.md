---
layout: default
parent: Monitors
title: "Loading"
---

# Loading

## Default Load Sequence

1. `LoadMonitors` plugin
2. `monitors` section in Blip config file
3. `--monitors` files
4. AWS instances
5. Local instances

## Stop-loss

Stop-loss is a feature of auto reloading that prevents Blip from dropping too many MySQL instances due to unrelated external issues.
