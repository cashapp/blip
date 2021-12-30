# OpenMetrics

The `openmetrics` package contains only the protobuf build of [OpenMetrics](https://github.com/OpenObservability/OpenMetrics). The Blip [Chronosphere](https://chronosphere.io/) sink writes (pushes) metrics to a Chronosphere "collector" using OpenMetrics.

### Building

To build `openmetrics_data_model.pb.go` in this directory:

1. Check out [OpenMetrics](https://github.com/OpenObservability/OpenMetrics). The protobuf is defintion is `proto/openmetrics_data_model.proto`.

2. Change to the `proto/` directory in the OpenMetrics repo and run:
```bash
protoc \
  --go_out=$BLIP_REPO \
  --go_opt=Mopenmetrics_data_model.proto=./openmetrics \
  ./openmetrics_data_model.proto
```
Replace `$BLIP_REPO` with the full path to the Blip repo root directly.

3. In the Blip repo, change to directory `sink/` and `go build` to ensure that `chronosphere.go` compiles with the new OpenMetrics protobuf definition.

### Data Structures

Following is a high-level view of the main OpenMetric data structs. This is not a comprehensive guide; it's only a quick visual guide for Blip developers.

```yaml
MetricFamily: -------------- One metric with potentially several data points (values)
  Name: -------------------- Name of metric
  Type: -------------------- Type of metric
  Help: -------------------- Description of metric

  Metrics: ----------------- Data points (values)

    - Labels: -------------- Unique identity of metric  | First instances of metric,
        - Name:  method ----                            | identified by its unique label set,
          Value: GET    ---- metric{method=GET}         | reporting one data point (value)
      MetricPoints: -------- Data point                 |
        - Value: ----------- 123 or 3.14 (e.g.)         |
          Timestamp: ------- Timestamp                  |
                                                        +-------------------------------------
    - Labels: -------------- Unique identity of metric  | Second instance of (same) metric,
        - Name:  method ----                            | made unique by different label set,
          Value: POST   ---- metric{method=POST}        | also one data point (value)
      MetricPoints: -------- Data point                 |
        - Value: ----------- 5 or 5.5 (e.g.)            |
          Timestamp: ------- Timestamp                  |
```

```yaml
MetricFamily:
  Name: "Queries"
  Type: COUNTER
  Help: "Number of queries executed"
  Metrics:
    - Labels:
        - Name:  env
          Value: staging
       	- Name:  hostname
          Value: db.local
      MetricPoints:
        - Value: 43789
          Timestamp: 1637165074

  Name: "Threads_running"
  Type: GAUGE
  Help: "Number of threads running"
  Metrics:
    - Labels:
        - Name:  env
          Value: staging
       	- Name:  hostname
          Value: db.local
      MetricPoints:
        - Value: 9
          Timestamp: 1637165074
```

```yaml
MetricFamily:
  Name: "users"
  Type: COUNTER
  Help: "/users endpoint access rates"
  Metrics:
    - Labels:
        - Name:  method
          Value: GET         # users{method=GET}
        - Name:  env
          Value: staging
       	- Name:  hostname
          Value: db.local
      MetricPoints:
        - Value: 555
          Timestamp: 1637165074
    - Labels:
        - Name:  method
          Value: POST        # users{method=POST}
        - Name:  env
          Value: staging
       	- Name:  hostname
          Value: db.local
      MetricPoints:
        - Value: 20
          Timestamp: 1637165074
```
