---
title: "aws.rds"
---

The `aws.rds` domain includes [Amazon RDS metrics](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/monitoring-cloudwatch.html#rds-metrics).

{{< toc >}}

## Usage

This domain queries the AWS API, so the Blip compute instance needs AWS credentials that, at minimum, allow `cloudwatch:GetMetricData`.
This is an example of full permissions policy:

```json
{
    "Statement": [
        {
            "Action": [
                "rds:ListTagsForResource",
                "rds:DescribeDBInstances"
            ],
            "Effect": "Allow",
            "Resource": "arn:aws:rds:*:AWS_ACCOUNT_NUMBER:db:*",
            "Sid": ""
        },
        {
            "Action": "secretsmanager:GetSecretValue",
            "Effect": "Allow",
            "Resource": "arn:aws:secretsmanager:*:AWS_ACCOUNT_NUMBER:secret:*metrics_password-*",
            "Sid": ""
        },
        {
            "Action": "cloudwatch:GetMetricData",
            "Effect": "Allow",
            "Resource": "*",
            "Sid": ""
        }
    ],
    "Version": "2012-10-17"
}
```

Metrics are named exactly as they are in the AWS API.
For example:

```yaml
level:
  freq: 60s
  collect:
    aws.rds:
      options:
        db-id: db-west-1
      metrics:
        - BinLogDiskUsage  # <-- HERE
        - CPUUtilization   # <-- AND HERE
``` 

Since AWS CloudWatch metrics have _1 minute_ resolution by default and can be delayed by up to 15 minutes, the level frequency should be 1 minute or longer.

Blip handles delayed AWS metrics by using the AWS timestamp for the metrics.
For example, at 13:00:00 Blip might collect AWS metrics from 2 minutes ago, so the metrics will have AWS timestamp 12:58:00.

See [Cloud / AWS]({{< ref "cloud/aws/" >}}) for more details.

## Derived Metrics

None.

## Options

### `db-id`

| | |
|---|---|
|Value|AWS database instance ID|
|Default|Blip monitor ID|

The `db-id` value is used to filter metrics from AWS CloudWatch.
If not value is provided in the Blip config, the monitor ID is used.

## Group Keys

None.

## Meta

None.

## Error Policies

None.

## MySQL Config

None.

## Changelog

|Blip Version|Change|
|------------|------|
|v1.0.0      |Domain added| 
