---
layout: default
parent: Plans
title: "File"
nav_order: 2
---

# Plan File

## Interpolation

Interpolation in plan files is idential to interpolation in the config file (see [Config File > Interpolation](../config/config-file.html#interpolation)).

Environment variable
: `${FOO}`

Environment variable with default value
: `${FOO:-default}`

Monitor variable
: `%{monitor.VAR}`

{: .note }
**NOTE**: `${}` and `%{}` are always required.

## Syntax
