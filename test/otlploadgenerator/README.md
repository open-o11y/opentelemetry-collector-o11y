# OTLP Load Generator
`data.go`: randomly generates and writes metrics in the following format to a text file:
```  		 
    name, type, label1 labelvalue1 , value1 value2 value3 value4 value5
```
gauge and counter has only one value. Output file path, metric type, number of metrics, labels, and value bounds of the 
generated metrics are all defined in `otlp.go`.

`otlp.go`: builds and send metrics to the collector. After the metric text is generated, it
waits for 10 seconds, then starts building OTLP metrics and sending.

`util.go`: contains utilities for building OTLP metrics.
