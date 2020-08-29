# Testing Pipeline for Prometheus Remote Write Exporter
The this package contains utilities for testing the Prometheus remote write exporter. 

- `otlploadgenerator` generates and
sends metric to OTLP receiver of the Collector. 
- `querier` validates the correctness of the metric by querying a backend.
- `otel-collector-config.yaml` specifies the configuration of the OpenTelemetry Collector.

To start a Collector instance and send to it using `otlploadgenerator`, run the following command:

```
make testaps
```
This command builds the Collector binary, runs the Collector instance based on `otel-collector-config.yaml`, then starts
`otlploadgenerator` to start sending metrics to the Collector. 

Note: With this command, the collector process has to be terminated manually after each run
## `otlploadgenerator`
The load generator first creates a `data.txt` file. This file is need so that the `querier` knows what the input data is.
Each line in the file represents and OTLP metric. Then, it parses each line from the file and build OTLP metric. 
It then creates a gRPC connection to the Collector, and sends the metric it builds. 

See more detail [here](./otlploadgenerator/README.md)

## `querier`
To be added.
