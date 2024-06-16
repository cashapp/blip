---
title: "size.binlog"
---

The `size.binlog` domain includes metrics about the size of binary logs.

{{< toc >}}

## Usage

This domain reports one derived metric: `bytes`.
It's calculated by totaling the `File_size` column of `SHOW BINARY LOGS`.

## Derived Metrics

### `bytes`

| | |
|---|---|
|**Metric Type**|gauge|
|**Value Units**|bytes|

Total size of all binary logs in bytes.

## Options

None.

## Group Keys

None.

## Meta

None.

## Error Policies

|Name|MySQL Error|
|---|---|
|access-denied|1227: access denied on 'SHOW BINARY LOGS' (need REPLICATION CLIENT priv)|
|binlog-not-enabled|1381: binary logging not enabled|

## MySQL Config

None.

## Changelog

|Blip Version|Change|
|------------|------|
|v1.0.0      |Domain added|
