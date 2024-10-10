---
---

Blip uses pseudo-logging based on internal events that are either "info" or error.
By default, Blip prints only errors to `STDERR`.
Start `blip` with the [`--log`]({{< ref "blip#--log" >}}) option to print info events to `STDOUT`.

<p class="note">
<a href="blip#--debug">Debug</a> info is printed to <code>STDERR</code>.
You can also toggle debug logging by sending a request to the `/debug` [API endpoint]({{< ref "/api" >}}) or by sending the `SIGUSR1` signal to the Blip process.
</p>

This is _pseudo-logging_ because there is no traditional log printing, only events that are printed by default.
See [Develop / Events]({{< ref "/develop/events" >}}) to learn how to change or enhance Blip pseudo-logging by receiving and handling events.
