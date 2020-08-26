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

package logs

import (
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/o11y/opentelemetry-collector-o11y/consumer"
	"github.com/o11y/opentelemetry-collector-o11y/consumer/pdata"
	"github.com/o11y/opentelemetry-collector-o11y/exporter/exportertest"
	collectorlog "github.com/o11y/opentelemetry-collector-o11y/internal/data/opentelemetry-proto-gen/collector/logs/v1"
	otlplog "github.com/o11y/opentelemetry-collector-o11y/internal/data/opentelemetry-proto-gen/logs/v1"
	"github.com/o11y/opentelemetry-collector-o11y/obsreport"
	"github.com/o11y/opentelemetry-collector-o11y/testutil"
)

var _ collectorlog.LogsServiceServer = (*Receiver)(nil)

func TestExport(t *testing.T) {
	// given

	logSink := new(exportertest.SinkLogsExporter)

	port, doneFn := otlpReceiverOnGRPCServer(t, logSink)
	defer doneFn()

	traceClient, traceClientDoneFn, err := makeLogsServiceClient(port)
	require.NoError(t, err, "Failed to create the TraceServiceClient: %v", err)
	defer traceClientDoneFn()

	// when

	unixnanos := uint64(12578940000000012345)

	traceID, err := base64.StdEncoding.DecodeString("SEhaOVO7YSQ=")
	assert.NoError(t, err)

	spanID, err := base64.StdEncoding.DecodeString("QuHicGYRg4U=")
	assert.NoError(t, err)

	resourceLogs := []*otlplog.ResourceLogs{
		{
			InstrumentationLibraryLogs: []*otlplog.InstrumentationLibraryLogs{
				{
					Logs: []*otlplog.LogRecord{
						{
							TraceId:      traceID,
							SpanId:       spanID,
							Name:         "operationB",
							TimeUnixNano: unixnanos,
						},
					},
				},
			},
		},
	}

	// Keep log data to compare the test result against it
	// Clone needed because OTLP proto XXX_ fields are altered in the GRPC downstream
	traceData := pdata.LogsFromOtlp(resourceLogs).Clone()

	req := &collectorlog.ExportLogsServiceRequest{
		ResourceLogs: resourceLogs,
	}

	resp, err := traceClient.Export(context.Background(), req)
	require.NoError(t, err, "Failed to export trace: %v", err)
	require.NotNil(t, resp, "The response is missing")

	// assert

	require.Equal(t, 1, len(logSink.AllLogs()), "unexpected length: %v", len(logSink.AllLogs()))

	assert.EqualValues(t, traceData, logSink.AllLogs()[0])
}

func TestExport_EmptyRequest(t *testing.T) {
	logSink := new(exportertest.SinkLogsExporter)

	port, doneFn := otlpReceiverOnGRPCServer(t, logSink)
	defer doneFn()

	logClient, logClientDoneFn, err := makeLogsServiceClient(port)
	require.NoError(t, err, "Failed to create the TraceServiceClient: %v", err)
	defer logClientDoneFn()

	resp, err := logClient.Export(context.Background(), &collectorlog.ExportLogsServiceRequest{})
	assert.NoError(t, err, "Failed to export trace: %v", err)
	assert.NotNil(t, resp, "The response is missing")
}

func TestExport_ErrorConsumer(t *testing.T) {
	logSink := new(exportertest.SinkLogsExporter)
	logSink.SetConsumeLogError(fmt.Errorf("error"))

	port, doneFn := otlpReceiverOnGRPCServer(t, logSink)
	defer doneFn()

	logClient, logClientDoneFn, err := makeLogsServiceClient(port)
	require.NoError(t, err, "Failed to create the TraceServiceClient: %v", err)
	defer logClientDoneFn()

	req := &collectorlog.ExportLogsServiceRequest{
		ResourceLogs: []*otlplog.ResourceLogs{
			{
				InstrumentationLibraryLogs: []*otlplog.InstrumentationLibraryLogs{
					{
						Logs: []*otlplog.LogRecord{
							{
								Name: "operationB",
							},
						},
					},
				},
			},
		},
	}

	resp, err := logClient.Export(context.Background(), req)
	assert.EqualError(t, err, "rpc error: code = Unknown desc = error")
	assert.Nil(t, resp)
}

func makeLogsServiceClient(port int) (collectorlog.LogsServiceClient, func(), error) {
	addr := fmt.Sprintf(":%d", port)
	cc, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return nil, nil, err
	}

	logClient := collectorlog.NewLogsServiceClient(cc)

	doneFn := func() { _ = cc.Close() }
	return logClient, doneFn, nil
}

func otlpReceiverOnGRPCServer(t *testing.T, tc consumer.LogsConsumer) (int, func()) {
	ln, err := net.Listen("tcp", "localhost:")
	require.NoError(t, err, "Failed to find an available address to run the gRPC server: %v", err)

	doneFnList := []func(){func() { ln.Close() }}
	done := func() {
		for _, doneFn := range doneFnList {
			doneFn()
		}
	}

	_, port, err := testutil.HostPortFromAddr(ln.Addr())
	if err != nil {
		done()
		t.Fatalf("Failed to parse host:port from listener address: %s error: %v", ln.Addr(), err)
	}

	r := New(receiverTagValue, tc)
	require.NoError(t, err)

	// Now run it as a gRPC server
	srv := obsreport.GRPCServerWithObservabilityEnabled()
	collectorlog.RegisterLogsServiceServer(srv, r)
	go func() {
		_ = srv.Serve(ln)
	}()

	return port, done
}
