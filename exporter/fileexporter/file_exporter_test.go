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
package fileexporter

import (
	"context"
	"testing"
	"time"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/o11y/opentelemetry-collector-o11y/consumer/pdata"
	"github.com/o11y/opentelemetry-collector-o11y/consumer/pdatautil"
	"github.com/o11y/opentelemetry-collector-o11y/internal/data"
	collectorlogs "github.com/o11y/opentelemetry-collector-o11y/internal/data/opentelemetry-proto-gen/collector/logs/v1"
	collectormetrics "github.com/o11y/opentelemetry-collector-o11y/internal/data/opentelemetry-proto-gen/collector/metrics/v1"
	collectortrace "github.com/o11y/opentelemetry-collector-o11y/internal/data/opentelemetry-proto-gen/collector/trace/v1"
	otlpcommon "github.com/o11y/opentelemetry-collector-o11y/internal/data/opentelemetry-proto-gen/common/v1"
	logspb "github.com/o11y/opentelemetry-collector-o11y/internal/data/opentelemetry-proto-gen/logs/v1"
	otresourcepb "github.com/o11y/opentelemetry-collector-o11y/internal/data/opentelemetry-proto-gen/resource/v1"
	"github.com/o11y/opentelemetry-collector-o11y/internal/data/testdata"
	"github.com/o11y/opentelemetry-collector-o11y/testutil"
)

func TestFileTraceExporterNoErrors(t *testing.T) {
	mf := &testutil.LimitedWriter{}
	lte := &fileExporter{file: mf}
	require.NotNil(t, lte)

	td := testdata.GenerateTraceDataTwoSpansSameResource()

	assert.NoError(t, lte.ConsumeTraces(context.Background(), td))
	assert.NoError(t, lte.Shutdown(context.Background()))

	var unmarshaler = &jsonpb.Unmarshaler{}
	var j collectortrace.ExportTraceServiceRequest
	assert.NoError(t, unmarshaler.Unmarshal(mf, &j))

	assert.EqualValues(t, pdata.TracesToOtlp(td), j.ResourceSpans)
}

func TestFileMetricsExporterNoErrors(t *testing.T) {
	mf := &testutil.LimitedWriter{}
	lme := &fileExporter{file: mf}
	require.NotNil(t, lme)

	md := pdatautil.MetricsFromInternalMetrics(testdata.GenerateMetricDataTwoMetrics())
	assert.NoError(t, lme.ConsumeMetrics(context.Background(), md))
	assert.NoError(t, lme.Shutdown(context.Background()))

	var unmarshaler = &jsonpb.Unmarshaler{}
	var j collectormetrics.ExportMetricsServiceRequest
	assert.NoError(t, unmarshaler.Unmarshal(mf, &j))

	assert.EqualValues(t, data.MetricDataToOtlp(pdatautil.MetricsToInternalMetrics(md)), j.ResourceMetrics)
}

func TestFileLogsExporterNoErrors(t *testing.T) {
	mf := &testutil.LimitedWriter{}
	exporter := &fileExporter{file: mf}
	require.NotNil(t, exporter)

	now := time.Now()
	ld := []*logspb.ResourceLogs{
		{
			Resource: &otresourcepb.Resource{
				Attributes: []*otlpcommon.KeyValue{
					{
						Key:   "attr1",
						Value: &otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "value1"}},
					},
				},
			},
			InstrumentationLibraryLogs: []*logspb.InstrumentationLibraryLogs{
				{
					Logs: []*logspb.LogRecord{
						{
							TimeUnixNano: uint64(now.UnixNano()),
							Name:         "logA",
						},
						{
							TimeUnixNano: uint64(now.UnixNano()),
							Name:         "logB",
						},
					},
				},
			},
		},
		{
			Resource: &otresourcepb.Resource{
				Attributes: []*otlpcommon.KeyValue{
					{
						Key:   "attr2",
						Value: &otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "value2"}},
					},
				},
			},
			InstrumentationLibraryLogs: []*logspb.InstrumentationLibraryLogs{
				{
					Logs: []*logspb.LogRecord{
						{
							TimeUnixNano: uint64(now.UnixNano()),
							Name:         "logC",
						},
					},
				},
			},
		},
	}
	assert.NoError(t, exporter.ConsumeLogs(context.Background(), pdata.LogsFromOtlp(ld)))
	assert.NoError(t, exporter.Shutdown(context.Background()))

	var unmarshaler = &jsonpb.Unmarshaler{}
	var j collectorlogs.ExportLogsServiceRequest

	assert.NoError(t, unmarshaler.Unmarshal(mf, &j))
	assert.EqualValues(t, ld, j.ResourceLogs)
}

func TestFileLogsExporterErrors(t *testing.T) {

	now := time.Now()
	ld := []*logspb.ResourceLogs{
		{
			Resource: &otresourcepb.Resource{
				Attributes: []*otlpcommon.KeyValue{
					{
						Key:   "attr1",
						Value: &otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "value1"}},
					},
				},
			},
			InstrumentationLibraryLogs: []*logspb.InstrumentationLibraryLogs{
				{
					Logs: []*logspb.LogRecord{
						{
							TimeUnixNano: uint64(now.UnixNano()),
							Name:         "logA",
						},
						{
							TimeUnixNano: uint64(now.UnixNano()),
							Name:         "logB",
						},
					},
				},
			},
		},
		{
			Resource: &otresourcepb.Resource{
				Attributes: []*otlpcommon.KeyValue{
					{
						Key:   "attr2",
						Value: &otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "value2"}},
					},
				},
			},
			InstrumentationLibraryLogs: []*logspb.InstrumentationLibraryLogs{
				{
					Logs: []*logspb.LogRecord{
						{
							TimeUnixNano: uint64(now.UnixNano()),
							Name:         "logC",
						},
					},
				},
			},
		},
	}

	cases := []struct {
		Name   string
		MaxLen int
	}{
		{
			Name:   "opening",
			MaxLen: 1,
		},
		{
			Name:   "resource",
			MaxLen: 16,
		},
		{
			Name:   "log_start",
			MaxLen: 78,
		},
		{
			Name:   "logs",
			MaxLen: 128,
		},
	}

	for i := range cases {
		maxLen := cases[i].MaxLen
		t.Run(cases[i].Name, func(t *testing.T) {
			mf := &testutil.LimitedWriter{
				MaxLen: maxLen,
			}
			exporter := &fileExporter{file: mf}
			require.NotNil(t, exporter)

			assert.Error(t, exporter.ConsumeLogs(context.Background(), pdata.LogsFromOtlp(ld)))
			assert.NoError(t, exporter.Shutdown(context.Background()))
		})
	}
}
