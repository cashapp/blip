---
layout: default
parent: Configure
title: Logging
nav_order: 4
---

# Logging

Blip uses pseudo-logging based on internal events that are either "info" or error.
By default, Blip prints only errors to `STDERR`.
Start `blip` with the [`--log`](blip#--log) option to print info events to `STDOUT`.

<p class="note">
<a href="blip#--debug">Debug</a> info is printed to <code>STDERR</code>.
</p>

This is _pseudo-logging_ because there is no traditional log printing, only events that are printed by default.
See [Develop / Events](../develop/events) to learn how to change or enhance Blip pseudo-logging by receiving and handling events.
