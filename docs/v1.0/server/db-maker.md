---
layout: default
parent: Server
title: "Db Maker"
nav_order: 4
---

# Db Maker

The database connection factory&mdash;_db maker_ for short&mdash;makes connections to MySQL.

## Authentication

1. Amazon RDS IAM authentication ([`config.aws-rds.iam-auth-token`](../config/config-file.html#iam-auth-token))
1. Amazon Secrets Manager ([`config.aws-rds.password-secret`](../config/config-file.html#password-secret))
1. TLS certificate ([`config.tls`](../config/config-file.html#tls))
1. Password file ([`config.mysql.password-file`](../config/config-file.html#password-file))
1. my.cnf ([`config.mysql.mycnf`](../config/config-file.html#mycnf))
1. Password ([`config.mysql.password`](../config/config-file.html#password))
1. No password

{: .src }
Source code: [dbconn/factory.go](https://github.com/cashapp/blip/blob/main/dbconn/factory.go)

## Password Reloading

Blip automatically reloads the password for all authentication methods.
It uses [go-mysql/hotswap-dsn-driver](https://github.com/go-mysql/hotswap-dsn-driver).
