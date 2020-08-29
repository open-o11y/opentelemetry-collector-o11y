// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package prometheusremotewriteexporter

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/prometheus/prometheus/prompb"

	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/consumer/pdatautil"
	otlp "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/metrics/v1old"
	"go.opentelemetry.io/collector/internal/dataold"
)

// PrwExporter converts OTLP metrics to Prometheus remote write TimeSeries and sends them to a remote endpoint
type PrwExporter struct {
	namespace   string
	endpointURL *url.URL
	client      *http.Client
	wg          *sync.WaitGroup
	closeChan   chan struct{}
}

// NewPrwExporter initializes a new PrwExporter instance and sets fields accordingly.
// client parameter cannot be nil.
func NewPrwExporter(namespace string, endpoint string, client *http.Client) (*PrwExporter, error) {

	if client == nil {
		return nil, fmt.Errorf("http client cannot be nil")
	}

	endpointURL, err := url.ParseRequestURI(endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint")
	}

	return &PrwExporter{
		namespace:   namespace,
		endpointURL: endpointURL,
		client:      client,
		wg:          new(sync.WaitGroup),
		closeChan:   make(chan struct{}),
	}, nil
}

// Shutdown stops the exporter from accepting incoming calls(and return error), and wait for current export operations
// to finish before returning
func (prwe *PrwExporter) Shutdown(context.Context) error {
	close(prwe.closeChan)
	prwe.wg.Wait()
	return nil
}

// PushMetrics converts metrics to Prometheus remote write TimeSeries and send to remote endpoint. It maintain a map of
// TimeSeries, validates and handles each individual metric, adding the converted TimeSeries to the map, and finally
// exports the map.
func (prwe *PrwExporter) PushMetrics(ctx context.Context, md pdata.Metrics) (int, error) {
	prwe.wg.Add(1)
	defer prwe.wg.Done()
	select {
	case <-prwe.closeChan:
		return pdatautil.MetricCount(md), fmt.Errorf("shutdown has been called")
	default:
		tsMap := map[string]*prompb.TimeSeries{}
		dropped := 0
		errs := []error{}

		resourceMetrics := dataold.MetricDataToOtlp(pdatautil.MetricsToOldInternalMetrics(md))
		for _, resourceMetric := range resourceMetrics {
			if resourceMetric == nil {
				continue
			}
			// TODO: add resource attributes as labels, probably in next PR
			for _, instrumentationMetrics := range resourceMetric.InstrumentationLibraryMetrics {
				if instrumentationMetrics == nil {
					continue
				}
				// TODO: decide if instrumentation library information should be exported as labels
				for _, metric := range instrumentationMetrics.Metrics {
					if metric == nil {
						continue
					}
					// check for valid type and temporality combination
					if ok := validateMetrics(metric.MetricDescriptor); !ok {
						dropped++
						errs = append(errs, fmt.Errorf("invalid temporality and type combination"))
						continue
					}
					// handle individual metric based on type
					switch metric.GetMetricDescriptor().GetType() {
					case otlp.MetricDescriptor_MONOTONIC_INT64, otlp.MetricDescriptor_INT64,
						otlp.MetricDescriptor_MONOTONIC_DOUBLE, otlp.MetricDescriptor_DOUBLE:
						if err := prwe.handleScalarMetric(tsMap, metric); err != nil {
							dropped++
							errs = append(errs, err)
						}
					case otlp.MetricDescriptor_HISTOGRAM:
						if err := prwe.handleHistogramMetric(tsMap, metric); err != nil {
							dropped++
							errs = append(errs, err)
						}
					case otlp.MetricDescriptor_SUMMARY:
						if err := prwe.handleSummaryMetric(tsMap, metric); err != nil {
							dropped++
							errs = append(errs, err)
						}
					default:
						dropped++
						errs = append(errs, fmt.Errorf("unsupported metric type"))
					}
				}
			}
		}

		if err := prwe.export(ctx, tsMap); err != nil {
			dropped = pdatautil.MetricCount(md)
			errs = append(errs, err)
		}

		if dropped != 0 {
			return dropped, componenterror.CombineErrors(errs)
		}

		return 0, nil
	}
}

// handleScalarMetric processes data points in a single OTLP scalar metric by adding the each point as a Sample into
// its corresponding TimeSeries in tsMap.
// tsMap and metric cannot be nil, and metric must have a non-nil descriptor
func (prwe *PrwExporter) handleScalarMetric(tsMap map[string]*prompb.TimeSeries, metric *otlp.Metric) error {
	mType := metric.MetricDescriptor.Type

	switch mType {
	// int points
	case otlp.MetricDescriptor_MONOTONIC_INT64, otlp.MetricDescriptor_INT64:
		if metric.Int64DataPoints == nil {
			return fmt.Errorf("nil data point field in metric %v", metric.GetMetricDescriptor().Name)
		}

		for _, pt := range metric.Int64DataPoints {
			if pt == nil {
				continue
			}
			// create parameters for addSample
			name := getPromMetricName(metric.GetMetricDescriptor(), prwe.namespace)
			labels := createLabelSet(pt.GetLabels(), nameStr, name)
			sample := &prompb.Sample{
				Value: float64(pt.Value),
				// convert ns to ms
				Timestamp: convertTimeStamp(pt.TimeUnixNano),
			}

			addSample(tsMap, sample, labels, metric.GetMetricDescriptor().GetType())
		}
		return nil

	// double points
	case otlp.MetricDescriptor_MONOTONIC_DOUBLE, otlp.MetricDescriptor_DOUBLE:
		if metric.DoubleDataPoints == nil {
			return fmt.Errorf("nil data point field in metric %v", metric.GetMetricDescriptor().Name)
		}
		for _, pt := range metric.DoubleDataPoints {
			if pt == nil {
				continue
			}
			// create parameters for addSample
			name := getPromMetricName(metric.GetMetricDescriptor(), prwe.namespace)
			labels := createLabelSet(pt.GetLabels(), nameStr, name)
			sample := &prompb.Sample{
				Value:     pt.Value,
				Timestamp: convertTimeStamp(pt.TimeUnixNano),
			}

			addSample(tsMap, sample, labels, metric.GetMetricDescriptor().GetType())
		}
		return nil
	}
	return fmt.Errorf("invalid metric type: wants int or double data points")
}

// handleHistogramMetric processes data points in a single OTLP histogram metric by mapping the sum, count and each
// bucket of every data point as a Sample, and adding each Sample to its corresponding TimeSeries.
// tsMap and metric cannot be nil.
func (prwe *PrwExporter) handleHistogramMetric(tsMap map[string]*prompb.TimeSeries, metric *otlp.Metric) error {

	if metric.HistogramDataPoints == nil {
		return fmt.Errorf("invalid metric type: wants histogram points")
	}

	for _, pt := range metric.HistogramDataPoints {
		if pt == nil {
			continue
		}
		time := convertTimeStamp(pt.TimeUnixNano)
		mType := metric.GetMetricDescriptor().GetType()

		// sum, count, and buckets of the histogram should append suffix to baseName
		baseName := getPromMetricName(metric.GetMetricDescriptor(), prwe.namespace)

		// treat sum as a sample in an individual TimeSeries
		sum := &prompb.Sample{
			Value:     pt.GetSum(),
			Timestamp: time,
		}
		sumlabels := createLabelSet(pt.GetLabels(), nameStr, baseName+sumStr)
		addSample(tsMap, sum, sumlabels, mType)

		// treat count as a sample in an individual TimeSeries
		count := &prompb.Sample{
			Value:     float64(pt.GetCount()),
			Timestamp: time,
		}
		countlabels := createLabelSet(pt.GetLabels(), nameStr, baseName+countStr)
		addSample(tsMap, count, countlabels, mType)

		// count for +Inf bound
		var totalCount uint64

		// process each bucket
		for le, bk := range pt.GetBuckets() {
			bucket := &prompb.Sample{
				Value:     float64(bk.Count),
				Timestamp: time,
			}
			boundStr := strconv.FormatFloat(pt.GetExplicitBounds()[le], 'f', -1, 64)
			labels := createLabelSet(pt.GetLabels(), nameStr, baseName+bucketStr, leStr, boundStr)
			addSample(tsMap, bucket, labels, mType)

			totalCount += bk.GetCount()
		}
		// add le=+Inf bucket
		infBucket := &prompb.Sample{
			Value:     float64(totalCount),
			Timestamp: time,
		}
		infLabels := createLabelSet(pt.GetLabels(), nameStr, baseName+bucketStr, leStr, pInfStr)
		addSample(tsMap, infBucket, infLabels, mType)
	}
	return nil
}

// handleSummaryMetric processes data points in a single OTLP summary metric by mapping the sum, count and each
// quantile of every data point as a Sample, and adding each Sample to its corresponding TimeSeries.
// tsMap and metric cannot be nil.
func (prwe *PrwExporter) handleSummaryMetric(tsMap map[string]*prompb.TimeSeries, metric *otlp.Metric) error {

	if metric.SummaryDataPoints == nil {
		return fmt.Errorf("invalid metric type: wants summary points")
	}

	for _, pt := range metric.SummaryDataPoints {

		time := convertTimeStamp(pt.TimeUnixNano)
		mType := metric.GetMetricDescriptor().GetType()

		// sum and count of the Summary should append suffix to baseName
		baseName := getPromMetricName(metric.GetMetricDescriptor(), prwe.namespace)

		// treat sum as sample in an individual TimeSeries
		sum := &prompb.Sample{
			Value:     pt.GetSum(),
			Timestamp: time,
		}
		sumlabels := createLabelSet(pt.GetLabels(), nameStr, baseName+sumStr)
		addSample(tsMap, sum, sumlabels, mType)

		// treat count as a sample in an individual TimeSeries
		count := &prompb.Sample{
			Value:     float64(pt.GetCount()),
			Timestamp: time,
		}
		countlabels := createLabelSet(pt.GetLabels(), nameStr, baseName+countStr)
		addSample(tsMap, count, countlabels, mType)

		// process each percentile/quantile
		for _, qt := range pt.GetPercentileValues() {
			quantile := &prompb.Sample{
				Value:     qt.Value,
				Timestamp: time,
			}
			percentileStr := strconv.FormatFloat(qt.Percentile, 'f', -1, 64)
			qtlabels := createLabelSet(pt.GetLabels(), nameStr, baseName, quantileStr, percentileStr)
			addSample(tsMap, quantile, qtlabels, mType)
		}
	}
	return nil
}

// export sends a Snappy-compressed WriteRequest containing TimeSeries to a remote write endpoint in order
func (prwe *PrwExporter) export(ctx context.Context, tsMap map[string]*prompb.TimeSeries) error {
	//Calls the helper function to convert the TsMap to the desired format
	req, err := wrapTimeSeries(tsMap)
	if err != nil {
		return err
	}
	//Uses proto.Marshal to convert the WriteRequest into bytes array
	data, err := proto.Marshal(req)
	if err != nil {
		return err
	}
	buf := make([]byte, len(data), cap(data))
	compressedData := snappy.Encode(buf, data)

	//Create the HTTP POST request to send to the endpoint
	httpReq, err := http.NewRequest("POST", prwe.endpointURL.String(), bytes.NewReader(compressedData))
	if err != nil {
		return err
	}

	// Add necessary headers specified by:
	// https://cortexmetrics.io/docs/apis/#remote-api
	httpReq.Header.Add("Content-Encoding", "snappy")
	httpReq.Header.Set("Content-Type", "application/x-protobuf")
	httpReq.Header.Set("X-Prometheus-Remote-Write-Version", "0.1.0")

	httpReq = httpReq.WithContext(ctx)

	_, cancel := context.WithTimeout(context.Background(), prwe.client.Timeout)
	defer cancel()

	httpResp, err := prwe.client.Do(httpReq)
	if err != nil {
		return err
	}

	if httpResp.StatusCode/100 != 2 {
		scanner := bufio.NewScanner(io.LimitReader(httpResp.Body, 256))
		line := ""
		if scanner.Scan() {
			line = scanner.Text()
		}
		errMsg := "server returned HTTP status " + httpResp.Status + ": " + line
		return fmt.Errorf(errMsg)
	}
	return nil
}
