# Cortex Exporter

This Exporter sends metrics data in Prometheus TimeSeries format to Cortex and signs each outgoing HTTP request following
the AWS Signature Version 4 signing process. AWS region and service must be provided in the configuration file, and AWS
credentials are retrieved from the [default credential chain](https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html#specifying-credentials)
of the AWS SDK for Go.

Note: this exporter intends to import and use the [Prometheus remote write exporter](https://github.com/open-telemetry/opentelemetry-collector/tree/master/exporter/prometheusremotewriteexporter)
from upstream; However, since the Prometheus remote write exporter has not been fully merged upstream, the OpenTelemtry
Collector code has been copied to `/internal` folder of this project, and a replace directive in `go.mod` redirects 
imports of `go.opentelemtry.io/collector` to `/internal`. Once upstream code is fully merged and stable, the Collector
copy and the replace directive in this project can be removed. 

Same as the Prometheus remote write exporter, this exporter checks the temporality and the type of each incoming metric 
and only exports the following combination:

- Int64 or Double type with any temporality
- MonotonicInt64, MonotonicDouble, Histogram, or Summary with only Cumulative temporality.

## Configuration
The following settings are required:
- `endpoint`: protocol:host:port to which the exporter is going to send traces or metrics, using the HTTP/HTTPS protocol. 

The following settings can be optionally configured:
- `namespace`: prefix attached to each exported metric name.
- `headers`: additional headers attached to each HTTP request. If `X-Prometheus-Remote-Write-Version` is set by user, its value must be `0.1.0`
- `insecure` (default = false): whether to enable client transport security for the exporter's connection.
- `ca_file`: path to the CA cert. For a client this verifies the server certificate. Should only be used if `insecure` is set to true.
- `cert_file`: path to the TLS cert to use for TLS required connections. Should only be used if `insecure` is set to true.
- `key_file`: path to the TLS key to use for TLS required connections. Should only be used if `insecure` is set to true.
- `timeout` (default = 5s): How long to wait until the connection is close.
- `read_buffer_size` (default = 0): ReadBufferSize for HTTP client.
- `write_buffer_size` (default = 512 * 1024): WriteBufferSize for HTTP client.
- `aws_auth`: whether each request should be singed with AWS Sig v4
            `region`: region string used for AWS Sig V4 signing
            `service`: service string used for AWS Sig V4 signing
            `debug`: whether the Sig V4 signature as well as each of the HTTP request and response should be printed. 
Example:

```yaml
exporters:
  prometheusremotewrite:
    endpoint: "http://some.url:9411/api/prom/push"
```
The full list of settings exposed for this exporter are documented [here](./config.go)
with detailed sample configurations [here](./testdata/config.yaml).

_Here is a link to the overall project [design](./DESIGN.md)_