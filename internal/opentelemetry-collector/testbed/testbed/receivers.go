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

package testbed

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/config"
	sdConfig "github.com/prometheus/prometheus/discovery/config"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver/jaegerreceiver"
	"go.opentelemetry.io/collector/receiver/opencensusreceiver"
	"go.opentelemetry.io/collector/receiver/otlpreceiver"
	"go.opentelemetry.io/collector/receiver/prometheusreceiver"
	"go.opentelemetry.io/collector/receiver/zipkinreceiver"
)

// DataReceiver allows to receive traces or metrics. This is an interface that must
// be implemented by all protocols that want to be used in MockBackend.
// Note the terminology: testbed.DataReceiver is something that can listen and receive data
// from Collector and the corresponding entity in the Collector that sends this data is
// an exporter.
type DataReceiver interface {
	Start(tc consumer.TraceConsumer, mc consumer.MetricsConsumer, lc consumer.LogsConsumer) error
	Stop() error

	// Generate a config string to place in exporter part of collector config
	// so that it can send data to this receiver.
	GenConfigYAMLStr() string

	// Return protocol name to use in collector config pipeline.
	ProtocolName() string
}

// DataReceiverBase implement basic functions needed by all receivers.
type DataReceiverBase struct {
	// Port on which to listen.
	Port int
}

const DefaultHost = "localhost"

func (mb *DataReceiverBase) ReportFatalError(err error) {
	log.Printf("Fatal error reported: %v", err)
}

// GetFactory of the specified kind. Returns the factory for a component type.
func (mb *DataReceiverBase) GetFactory(_ component.Kind, _ configmodels.Type) component.Factory {
	return nil
}

// Return map of extensions. Only enabled and created extensions will be returned.
func (mb *DataReceiverBase) GetExtensions() map[configmodels.Extension]component.ServiceExtension {
	return nil
}

func (mb *DataReceiverBase) GetExporters() map[configmodels.DataType]map[configmodels.Exporter]component.Exporter {
	return nil
}

// OCDataReceiver implements OpenCensus format receiver.
type OCDataReceiver struct {
	DataReceiverBase
	traceReceiver   component.TraceReceiver
	metricsReceiver component.MetricsReceiver
}

// Ensure OCDataReceiver implements DataReceiver.
var _ DataReceiver = (*OCDataReceiver)(nil)

const DefaultOCPort = 56565

// NewOCDataReceiver creates a new OCDataReceiver that will listen on the specified port after Start
// is called.
func NewOCDataReceiver(port int) *OCDataReceiver {
	return &OCDataReceiver{DataReceiverBase: DataReceiverBase{Port: port}}
}

func (or *OCDataReceiver) Start(tc consumer.TraceConsumer, mc consumer.MetricsConsumer, _ consumer.LogsConsumer) error {
	factory := opencensusreceiver.NewFactory()
	cfg := factory.CreateDefaultConfig().(*opencensusreceiver.Config)
	cfg.SetName(or.ProtocolName())
	cfg.NetAddr = confignet.NetAddr{Endpoint: fmt.Sprintf("localhost:%d", or.Port), Transport: "tcp"}
	var err error
	params := component.ReceiverCreateParams{Logger: zap.NewNop()}
	if or.traceReceiver, err = factory.CreateTraceReceiver(context.Background(), params, cfg, tc); err != nil {
		return err
	}
	if or.metricsReceiver, err = factory.CreateMetricsReceiver(context.Background(), params, cfg, mc); err != nil {
		return err
	}
	if err = or.traceReceiver.Start(context.Background(), or); err != nil {
		return err
	}
	return or.metricsReceiver.Start(context.Background(), or)
}

func (or *OCDataReceiver) Stop() error {
	if err := or.traceReceiver.Shutdown(context.Background()); err != nil {
		return err
	}
	if err := or.metricsReceiver.Shutdown(context.Background()); err != nil {
		return err
	}
	return nil
}

func (or *OCDataReceiver) GenConfigYAMLStr() string {
	// Note that this generates an exporter config for agent.
	return fmt.Sprintf(`
  opencensus:
    endpoint: "localhost:%d"
    insecure: true`, or.Port)
}

func (or *OCDataReceiver) ProtocolName() string {
	return "opencensus"
}

// JaegerDataReceiver implements Jaeger format receiver.
type JaegerDataReceiver struct {
	DataReceiverBase
	receiver component.TraceReceiver
}

var _ DataReceiver = (*JaegerDataReceiver)(nil)

const DefaultJaegerPort = 14250

func NewJaegerDataReceiver(port int) *JaegerDataReceiver {
	return &JaegerDataReceiver{DataReceiverBase: DataReceiverBase{Port: port}}
}

func (jr *JaegerDataReceiver) Start(tc consumer.TraceConsumer, _ consumer.MetricsConsumer, _ consumer.LogsConsumer) error {
	factory := jaegerreceiver.NewFactory()
	cfg := factory.CreateDefaultConfig().(*jaegerreceiver.Config)
	cfg.SetName(jr.ProtocolName())
	cfg.Protocols.GRPC = &configgrpc.GRPCServerSettings{
		NetAddr: confignet.NetAddr{Endpoint: fmt.Sprintf("localhost:%d", jr.Port), Transport: "tcp"},
	}
	var err error
	params := component.ReceiverCreateParams{Logger: zap.NewNop()}
	jr.receiver, err = factory.CreateTraceReceiver(context.Background(), params, cfg, tc)
	if err != nil {
		return err
	}

	return jr.receiver.Start(context.Background(), jr)
}

func (jr *JaegerDataReceiver) Stop() error {
	return jr.receiver.Shutdown(context.Background())
}

func (jr *JaegerDataReceiver) GenConfigYAMLStr() string {
	// Note that this generates an exporter config for agent.
	return fmt.Sprintf(`
  jaeger:
    endpoint: "localhost:%d"
    insecure: true`, jr.Port)
}

func (jr *JaegerDataReceiver) ProtocolName() string {
	return "jaeger"
}

// OTLPDataReceiver implements OTLP format receiver.
type OTLPDataReceiver struct {
	DataReceiverBase
	traceReceiver   component.TraceReceiver
	metricsReceiver component.MetricsReceiver
	logReceiver     component.LogsReceiver
}

// Ensure OTLPDataReceiver implements DataReceiver.
var _ DataReceiver = (*OTLPDataReceiver)(nil)

const DefaultOTLPPort = 55680

// NewOTLPDataReceiver creates a new OTLPDataReceiver that will listen on the specified port after Start
// is called.
func NewOTLPDataReceiver(port int) *OTLPDataReceiver {
	return &OTLPDataReceiver{DataReceiverBase: DataReceiverBase{Port: port}}
}

func (or *OTLPDataReceiver) Start(tc consumer.TraceConsumer, mc consumer.MetricsConsumer, lc consumer.LogsConsumer) error {
	factory := otlpreceiver.NewFactory()
	cfg := factory.CreateDefaultConfig().(*otlpreceiver.Config)
	cfg.SetName(or.ProtocolName())
	cfg.GRPC.NetAddr = confignet.NetAddr{Endpoint: fmt.Sprintf("localhost:%d", or.Port), Transport: "tcp"}
	cfg.HTTP = nil
	var err error
	params := component.ReceiverCreateParams{Logger: zap.NewNop()}
	if or.traceReceiver, err = factory.CreateTraceReceiver(context.Background(), params, cfg, tc); err != nil {
		return err
	}
	if or.metricsReceiver, err = factory.CreateMetricsReceiver(context.Background(), params, cfg, mc); err != nil {
		return err
	}
	if or.logReceiver, err = factory.CreateLogsReceiver(context.Background(), params, cfg, lc); err != nil {
		return err
	}

	if err = or.traceReceiver.Start(context.Background(), or); err != nil {
		return err
	}
	if err = or.metricsReceiver.Start(context.Background(), or); err != nil {
		return err
	}
	return or.logReceiver.Start(context.Background(), or)
}

func (or *OTLPDataReceiver) Stop() error {
	if err := or.traceReceiver.Shutdown(context.Background()); err != nil {
		return err
	}
	if err := or.metricsReceiver.Shutdown(context.Background()); err != nil {
		return err
	}
	return nil
}

func (or *OTLPDataReceiver) GenConfigYAMLStr() string {
	// Note that this generates an exporter config for agent.
	return fmt.Sprintf(`
  otlp:
    endpoint: "localhost:%d"
    insecure: true`, or.Port)
}

func (or *OTLPDataReceiver) ProtocolName() string {
	return "otlp"
}

// ZipkinDataReceiver implements Zipkin format receiver.
type ZipkinDataReceiver struct {
	DataReceiverBase
	receiver component.TraceReceiver
}

var _ DataReceiver = (*ZipkinDataReceiver)(nil)

const DefaultZipkinAddressPort = 9411

func NewZipkinDataReceiver(port int) *ZipkinDataReceiver {
	return &ZipkinDataReceiver{DataReceiverBase: DataReceiverBase{Port: port}}
}

func (zr *ZipkinDataReceiver) Start(tc consumer.TraceConsumer, _ consumer.MetricsConsumer, _ consumer.LogsConsumer) error {
	factory := zipkinreceiver.NewFactory()
	cfg := factory.CreateDefaultConfig().(*zipkinreceiver.Config)
	cfg.SetName(zr.ProtocolName())
	cfg.Endpoint = fmt.Sprintf("localhost:%d", zr.Port)

	params := component.ReceiverCreateParams{Logger: zap.NewNop()}
	var err error
	zr.receiver, err = factory.CreateTraceReceiver(context.Background(), params, cfg, tc)

	if err != nil {
		return err
	}

	return zr.receiver.Start(context.Background(), zr)
}

func (zr *ZipkinDataReceiver) Stop() error {
	return zr.receiver.Shutdown(context.Background())
}

func (zr *ZipkinDataReceiver) GenConfigYAMLStr() string {
	// Note that this generates an exporter config for agent.
	return fmt.Sprintf(`
  zipkin:
    endpoint: http://localhost:%d/api/v2/spans
    format: json`, zr.Port)
}

func (zr *ZipkinDataReceiver) ProtocolName() string {
	return "zipkin"
}

// prometheus

type PrometheusDataReceiver struct {
	DataReceiverBase
	receiver component.MetricsReceiver
}

var _ DataReceiver = (*PrometheusDataReceiver)(nil)

func NewPrometheusDataReceiver(port int) *PrometheusDataReceiver {
	return &PrometheusDataReceiver{DataReceiverBase: DataReceiverBase{Port: port}}
}

func (dr *PrometheusDataReceiver) Start(_ consumer.TraceConsumer, mc consumer.MetricsConsumer, _ consumer.LogsConsumer) error {
	factory := prometheusreceiver.NewFactory()
	cfg := factory.CreateDefaultConfig().(*prometheusreceiver.Config)
	addr := fmt.Sprintf("0.0.0.0:%d", dr.Port)
	cfg.PrometheusConfig = &config.Config{
		ScrapeConfigs: []*config.ScrapeConfig{{
			JobName:        "testbed-job",
			ScrapeInterval: model.Duration(100 * time.Millisecond),
			ScrapeTimeout:  model.Duration(time.Second),
			ServiceDiscoveryConfig: sdConfig.ServiceDiscoveryConfig{
				StaticConfigs: []*targetgroup.Group{{
					Targets: []model.LabelSet{{
						"__address__":      model.LabelValue(addr),
						"__scheme__":       "http",
						"__metrics_path__": "/metrics",
					}},
				}},
			},
		}},
	}
	var err error
	dr.receiver, err = factory.CreateMetricsReceiver(context.Background(), component.ReceiverCreateParams{}, cfg, mc)
	if err != nil {
		return err
	}
	return dr.receiver.Start(context.Background(), dr)
}

func (dr *PrometheusDataReceiver) Stop() error {
	return dr.receiver.Shutdown(context.Background())
}

// Generate exporter yaml
func (dr *PrometheusDataReceiver) GenConfigYAMLStr() string {
	format := `
  prometheus:
    endpoint: "localhost:%d"
`
	return fmt.Sprintf(format, dr.Port)
}

func (dr *PrometheusDataReceiver) ProtocolName() string {
	return "prometheus"
}