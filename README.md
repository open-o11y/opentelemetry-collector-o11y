# OpenTelemetry Collector O11y

This repository is focused on building and packaging the OpenTelemetry Collector with a Cortex exporter
 supporting Sig V4 to export to AWS services. The exporter is built on top of the Prometheus remote write exporter from
 upstream repository. See [this package](./exporter/cortexexporter) for implementation detail of the Cortex exporter. 

## Components

Most upstream components are removed and not included in the build. Available components are:

* Receiver: OpenTelemetry Collector default receivers 
* Processor: OpenTelemetry Collector default processors 
* Exporter: OpenTelemetry Collector default processors and a Cortex Exporter supporting AWS Sig V4 signing. 

An example Collector pipeline is illustrated below:

![Image: Repo README.png](./img/Pipeline.png)

## Building

To build the Collector, run the following command: 

```
make otelcol
```
The resultant binary is under `/bin`.

## Sample Configuration

The following is a configuration for a Collector instance that receives gRPC OTLP metrics on `localhost:55680`, a 
logging exporter that logs metric to `stdout`, and a Prometheus remote write exporter sending to an endpoint with AWS 
Sig V4 support enabled. 

```
receivers:
     otlp:
      protocols:
         grpc:
exporters:
  prometheusremotewrite:
    endpoint: "http://localhost:9009"
    namespace: otel-collector
    auth:
      region: "us-west-2"
      service: "aps"
    timeout: 10s
  logging:
    loglevel: debug

service:
  extensions:
  pipelines:
    metrics:
      receivers: [otlp]
      exporters: [logging,prometheusremotewrite]
```

see a complete list of configuration options and explanation of the prometheus remote write exporter [here](./exporter/prometheusremotewriteexporter/README.md)
