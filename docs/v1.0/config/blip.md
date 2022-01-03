---
layout: default
parent: Configure
title: Blip
nav_order: 1
---

# Blip

## Zero Config

Blip uses built-it defaults and auto-detection to run and work without specifing any configuration.
This is called the "zero config".

The zero config should work on your laptop (presuming a standard MySQL setup), but it is not intended for real production environments.
At the very least, you will need to specify which MySQL instances to monitor in the `monitors` section of the Blip config file.

## Specifying a Config File

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

## Command Line Options

Run `blip --help` to list command line options.

### `--config FILE`

{: .help-option-default }
Default: `blip.yaml`<br>
Env var: `BLIP_CONFIG`

{: .help-option }
Specify Blip configuruation file.

### `--debug`

{: .help-option-default }
Env var: `BLIP_CONFIG`

{: .help-option }
Print debug to stderr.

### `--help`

{: .help-option }
Print help and exit.

### `--plans FILE[,FILE...]`

{: .help-option-default }
Env var: `BLIP_PLANS`

{: .help-option }
Specify plan files.

### `--print-config`

{: .help-option }
Print config.

### `--print-monitors`

{: .help-option }
Print monitors.

### `--print-plans`

{: .help-option }
Print level plans.

### `--run`

{: .help-option-default }
Default: `true`<br>
Env var: `BLIP_RUN`

{: .help-option }
Run Blip and all monitors.
If `--run=false`, Blip starts and loads everything, but exists before running monitors.

### `--strict`

{: .help-option-default }
Env var: `BLIP_STRICT`

{: .help-option }
Enable strict mode.

### `--version`

{: .help-option }
Print version and exit.
