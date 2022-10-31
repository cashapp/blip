---
layout: default
parent: Plans
title: "Table"
---

# Plan Table


```sql
CREATE TABLE plans (
  name       varchar(100) not null,
  levels     blob not null
  monitor_id varchar(1000) null default null
  PRIMARY KEY (name),
  INDEX (monitorId)
)
```

```
-- Default single state plan (DSSP)
("default", "{...}", NULL)

-- Default multi-state plans
("readonly", "{...}", NULL)
("active", "{...}", NULL)

-- Per-monitor single state plans
("mon1", "{...}", "mon1")
("mon2", "{...}", "mon2")

-- Mixed plans
("default", "{...}", NULL)
("mon1-ro", "{...}", "mon1")
("mon1-rw", "{...}", "mon1")
("mon2", "{...}", "mon2")
-- mon3 uses default plan
```
