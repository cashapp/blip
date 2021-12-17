---
layout: default
title: Configure
nav_order: 3
has_children: true
permalink: /v1.0/config
---

# Configure

Blip configuration is specified in a single YAML file.
There are 3 ways to specify the Blip config file.

By default, Blip uses `blip.yaml` in the current working directory:

```sh
$ blip
```

You can specify a config file with the `--config` command-line option:

```sh
$ blip --config FILE
```

Or, you can specify a config file with the `BLIP_CONFIG` environment variable:

```sh
$ export BLIP_CONFIG=FILE
$ blip
```

The command-line option takes precedent over the environment variable.
In the following example, Blip uses only `FILE_2`:

```sh
$ export BLIP_CONFIG=FILE_1
$ blip --config FILE_2
```
