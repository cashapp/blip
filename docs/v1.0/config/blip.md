---
layout: default
parent: Configure
title: Blip
nav_order: 1
---

# Blip
{: .no_toc}

This page details how to configure the Blip binary (`blip`) that runs monitors.
To configure (customize) metrics collection, see [Plans](../plans/) and [Metrics](../metrics/).

* TOC
{:toc}

## Config File

Blip configuration is specified in a single YAML file.
See [Configure / Config File](config-file) for the full list of Blip config variables.

There are 3 ways to specify the Blip config file.

By default, Blip reads `blip.yaml` in the current working directory if it exists:

```sh
$ blip
```

You can specify a config file with the [`--config`](#--config-file) command-line option:

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
The file must exist, else Blip will error on boot.

### `--debug`

{: .help-option-default }
Env var: `BLIP_CONFIG`

{: .help-option }
Print debug to stderr.

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

### `--print-plans`

{: .help-option }
Print all [plans](../plans/) after booting.
<br><br>
This option does not stop Blip from running.
Specify [`--run=false`](#--run) to print plans and exit.

### `--run`

{: .help-option-default }
Default: `true`<br>
Env var: `BLIP_RUN`

{: .help-option }
Run Blip and all monitors after successful boot.
If `--run=false`, Blip starts and loads everything, but exits before running monitors.

### `--version`

{: .help-option }
Print version and exit.

## Zero Config

Blip uses built-it defaults and auto-detection to work without specifying any configuration.
This is called the "zero config".

The zero config should work on your laptop (presuming a standard MySQL setup), but it is not intended for production environments.
At the very least, you need to specify which MySQL instances to monitor in the `monitors` section of the Blip config file.
