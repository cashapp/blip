---
layout: default
parent: Cloud
title: AWS
---

# AWS

Blip has built-in supprot for Amazon RDS for MySQL and Amazon Aurora.
It supports fetching the MySQL password from AWS Secrets Manager and using IAM authentication tokens.

### IAM Authentication

```sql
CREATE USER 'blip' IDENTIFIED WITH AWSAuthenticationPlugin as 'RDS';
GRANT ALL PRIVILEGES ON `blip`.* TO 'blip'@'%' REQUIRE SSL;
GRANT SELECT ON `performance_schema`.* TO 'blip'@'%' REQUIRE SSL;
```
