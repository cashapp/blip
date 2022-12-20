---
layout: default
parent: Configure
title: blip Binary
nav_order: 1
---

# Blip
{: .no_toc}

This page details how to configure the Blip binary (`blip`) that runs monitors.
To configure (customize) metrics collection, see [Plans](../plans/) and [Metrics](../metrics/).

* TOC
{:toc}

## Config File

Blip uses a single YAML file for configuration.
See [Configure / Config File](config-file) for the full list of Blip config variables.

👉 By default, Blip reads `blip.yaml` in the current working directory, if it exits.

You can specify a different config file with the [`--config`](#--config-file) command-line option:

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
Specify Blip configuration file.
The specified file must exist, else Blip will error on boot.
<br><br>
Blip will boot successfully if the _default_ file, `blip.yaml`, does not exist in the current working directly.
In this case, Blip tries to auto-detect a local MySQL instance, which is useful for local development.

### `--debug`

{: .help-option-default }
Env var: `BLIP_CONFIG`

{: .help-option }
Print debug to STDERR.

### `--help`

{: .help-option }
Print help and exit.

### `--log`

{: .help-option-default }
Default: `false`<br>
Env var: `BLIP_LOG`

{: .help-option }
Print all "info" events to STDOUT.
By default, Blip prints only errors to `STDERR`.
See [Logging](logging).

### `--print-config`

{: .help-option }
Print the [server config](config-file) after booting.
<br><br>
This option does not stop Blip from running.
Specify [`--run=false`](#--run) to print the final server config and exit.

### `--print-domains`

{: .help-option }
Print domains and collector options, then exit.

### `--print-monitors`

{: .help-option }
Print the [monitors](config-file) after booting.
The monitors are finalized: monitor defaults and [interpolation](interpolation) have been applied.
<br><br>
This option does not stop Blip from running.
Specify [`--run=false`](#--run) to print monitors and exit.

### `--run`

{: .help-option-default }
Default: `true`<br>
Env var: `BLIP_RUN`

{: .help-option }
Run Blip and all monitors after successful boot.
If `--run=false` (or environment varialbe `BLIP_RUN=false`), Blip starts and loads everything, but exits before running monitors.
See [Startup](#startup).

### `--version`

{: .help-option }
Print version and exit.

## Startup

Blip has a two-phase startup sequence: boot, then run.

The _boot_ phase loads and validates everything: Blip config, monitors, plans, and so forth.
Any error on boot causes Blip to exit with a non-zero exit status.

The _run_ phase runs monitors and the Blip [API](../api).
Once running, Blip does not exit until it receives a signal.
It retires on all errors, including panic.


{: .note }
To boot without running (which is useful to validate everything), specify [`--run=false`](#--run) on the commnand line, or environment varialbe `BLIP_RUN=false`.

## Signals

Blip does a controlled shutdown on `SIGTERM`.
Other signals are not caught.
