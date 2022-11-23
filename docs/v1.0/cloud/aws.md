---
layout: default
parent: Cloud
title: AWS
---


# AWS
{: .no_toc }

{: .warn }
Blip works with AWS, but AWS does not support or contribute to Blip.

Blip has built-in support for Amazon RDS for MySQL and Amazon Aurora.
The following documentation presumes proficiency with AWS.
It does not explain, for example, how to set up IAM roles or configure RDS.

* TOC
{:toc}


## Region

By default, Blip tries to auto-detect the region by querying [IMDB](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-instance-metadata.html).
This works if running on an EC2 instance.

If auto-detection fails or [`config.aws.region`](../config/config-file#region) is set, the value of this variable is used.

If that variable is not set, Blip tries to auto-detect the region from the MySQL endpoint by splitting the fields of the DNS address.
This should work because Amazon RDS endpoints have the format `<db-id>.<cluster>.<region>.rds.amazonaws.com`.

If that fails, no region is used.
Not setting the region in Blip might work if [AWS SDK for Go v2](https://pkg.go.dev/github.com/aws/aws-sdk-go-v2) can auto-detect the region.

## Credentials

Currently, Blip relies on the [AWS SDK for Go v2](https://pkg.go.dev/github.com/aws/aws-sdk-go-v2) to detect and set the credentials through its various conventions.
For example, Blip does not currently have a config variable to specify an AWS profile.
The only information Blip passes to the AWS SDK is the region:

```go
config.LoadDefaultConfig(ctx, config.WithRegion("..."))
```

Section [Region](#region) above explains how the region is auto-detected or configured.

Use the [`blip.AWSConfigFactory`](https://pkg.go.dev/github.com/cashapp/blip#AWSConfigFactory) to load a custome AWS configuration (which includes the credentials).
See [Develop / Intergration API](../develop/integration-api) for more information.

## Authentication

Blip supports AWS IAM auth and fetching the password from Secrets Manager.
Like all MySQL credentials, **Blip automatically [reloads the password](../monitors/mysql-connection#password-reloading) on "access denied" from MySQL.**
This is especially important for IAM auth since the token is valid only 15 minutes: Blip automatically regenerates a new token.

### IAM Authentication

Blip can use an IAM authentication token as the password:

* [IAM database authentication for MariaDB, MySQL, and PostgreSQL](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/UsingWithRDS.IAMDBAuth.html)
* [How do I allow users to authenticate to an Amazon RDS MySQL DB instance using their IAM credentials?](https://aws.amazon.com/premiumsupport/knowledge-center/users-connect-rds-iam/)

IAM auth must be enabled on the RDS instance first.
To check, execute:

```sql
SELECT * FROM INFORMATION_SCHEMA.PLUGINS WHERE plugin_name='AWSAuthenticationPlugin'\G

*************************** 1. row ***************************
           PLUGIN_NAME: AWSAuthenticationPlugin
        PLUGIN_VERSION: 1.0
         PLUGIN_STATUS: ACTIVE
           PLUGIN_TYPE: AUTHENTICATION
   PLUGIN_TYPE_VERSION: 2.0
        PLUGIN_LIBRARY: aws_auth.so
PLUGIN_LIBRARY_VERSION: 1.10
         PLUGIN_AUTHOR: AWSRDS
    PLUGIN_DESCRIPTION: Aurora AWS Authentication Plugin
        PLUGIN_LICENSE: PROPRIETARY
           LOAD_OPTION: ON
```

The output above means IAM is enabled on the instance.

Once IAM auth is enable, first create the Blip user as documented by AWS:

```sql
CREATE USER 'blip' IDENTIFIED WITH AWSAuthenticationPlugin as 'RDS' REQUIRE SSL;
GRANT SELECT, PROCESS, REPLICATION CLIENT ON *.* TO 'blip'@'%';
```

Then you need an IAM policy that allows the `blip` MySQL user to connect using IAM auth:

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "AllowBlipToConnect",
            "Effect": "Allow",
            "Action": "rds-db:connect",
            "Resource": "arn:aws:rds-db:*:999999999999:dbuser:*/blip"
        }
    ]
}
```

Replace `999999999999` with your AWS account ID.

Then set [`config.aws.iam-auth`](../config/config-file#iam-auth) to true and start Blip.
If everything is configured correctly in both Blip and AWS, Blip should work as usual.

### Secrets Manager

Blip can fetch its MySQL password from a secret in [AWS Secrets Manager](https://aws.amazon.com/secrets-manager/).
Set [`config.aws.password-secret`](../config/config-file#password-secret) to the ARN of the secret.
The secret value must be a key-value map with a `password` key, like:

```json
{
  "user": "blip",    // IGNORED
  "password": "..."  // Blip uses only this value
}
```

Blip ignores other keys in the secret value; it only reads the `password` key-value.

## TLS

When connecting an Amazon RDS instances (RDS or Aurora), Blip automatically detects and enables TLS using the Amazon RDS CA-2019 certificate, which is hard-coded into Blip at the end of [`blip/aws/rds.go`](https://github.com/cashapp/blip/blob/main/aws/rds.go).

You can disable this by explicitly setting any [`config.tls`](../config/config-file#tls) variable, or by setting [`config.aws.disable-auto-tls`](../config/config-file#disable-auto-tls) to true.
