---
---

This page details how to configure the Blip binary (`blip`) that runs monitors.
To configure (customize) metrics collection, see [Plans]({{< ref "/plans/" >}}) and [Metrics]({{< ref "/metrics/" >}}).

{{< toc >}}

## Config File

Blip uses a single YAML file for configuration.
See [Configure / Config File]({{< ref "config-file" >}}) for the full list of Blip config variables.

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

* Default: `blip.yaml`
* Env var: `BLIP_CONFIG`

Specify Blip configuration file.
The specified file must exist, else Blip will error on boot.

Blip will boot successfully if the _default_ file, `blip.yaml`, does not exist in the current working directly.
In this case, Blip tries to auto-detect a local MySQL instance, which is useful for local development.

### `--debug`

* Env var: `BLIP_CONFIG`

Print debug to STDERR.

### `--help`

Print help and exit.

### `--log`

* Default: `false`<br>
* Env var: `BLIP_LOG`

Print all "info" events to STDOUT.
By default, Blip prints only errors to `STDERR`.
See [Logging]({{< ref "logging" >}}).

### `--print-config`

Print the [server config]({{< ref "config-file" >}}) after booting.

This option does not stop Blip from running.
Specify [`--run=false`](#--run) to print the final server config and exit.

### `--print-domains`

Print domains and collector options, then exit.

### `--print-monitors`

Print the [monitors]({{< ref "config-file" >}}) after booting.
The monitors are finalized: monitor defaults and [interpolation]({{< ref "interpolation" >}}) have been applied.

This option does not stop Blip from running.
Specify [`--run=false`](#--run) to print monitors and exit.

### `--run`

* Default: `true`<br>
* Env var: `BLIP_RUN`

Run Blip and all monitors after successful boot.
If `--run=false` (or environment varialbe `BLIP_RUN=false`), Blip starts and loads everything, but exits before running monitors.
See [Startup](#startup).

### `--version`

Print version and exit.

## Startup

Blip has a two-phase startup sequence: boot, then run.

The _boot_ phase loads and validates everything: Blip config, monitors, plans, and so forth.
Any error on boot causes Blip to exit with a non-zero exit status.

The _run_ phase runs monitors and the Blip [API]({{< ref "/api" >}}).
Once running, Blip does not exit until it receives a signal.
It retires on all errors, including panic.

{{< hint type=note >}}
To boot without running (which is useful to validate everything), specify [`--run=false`](#--run) on the commnand line, or environment varialbe `BLIP_RUN=false`.
{{< /hint >}}

## Signals

Blip does a controlled shutdown on `SIGTERM`.
Other signals are not caught.
