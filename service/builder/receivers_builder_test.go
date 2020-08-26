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

package builder

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/o11y/opentelemetry-collector-o11y/component"
	"github.com/o11y/opentelemetry-collector-o11y/component/componenttest"
	"github.com/o11y/opentelemetry-collector-o11y/config/configmodels"
	"github.com/o11y/opentelemetry-collector-o11y/config/configtest"
	"github.com/o11y/opentelemetry-collector-o11y/consumer"
	"github.com/o11y/opentelemetry-collector-o11y/consumer/pdata"
	"github.com/o11y/opentelemetry-collector-o11y/consumer/pdatautil"
	"github.com/o11y/opentelemetry-collector-o11y/internal/data/testdata"
	"github.com/o11y/opentelemetry-collector-o11y/processor/attributesprocessor"
	"github.com/o11y/opentelemetry-collector-o11y/receiver/zipkinreceiver"
)

type testCase struct {
	name                      string
	receiverName              string
	exporterNames             []string
	spanDuplicationByExporter map[string]int
	hasTraces                 bool
	hasMetrics                bool
}

func TestReceiversBuilder_Build(t *testing.T) {
	tests := []testCase{
		{
			name:          "one-exporter",
			receiverName:  "examplereceiver",
			exporterNames: []string{"exampleexporter"},
			hasTraces:     true,
			hasMetrics:    true,
		},
		{
			name:          "multi-exporter",
			receiverName:  "examplereceiver/2",
			exporterNames: []string{"exampleexporter", "exampleexporter/2"},
			hasTraces:     true,
		},
		{
			name:          "multi-metrics-receiver",
			receiverName:  "examplereceiver/3",
			exporterNames: []string{"exampleexporter", "exampleexporter/2"},
			hasTraces:     false,
			hasMetrics:    true,
		},
		{
			name:          "multi-receiver-multi-exporter",
			receiverName:  "examplereceiver/multi",
			exporterNames: []string{"exampleexporter", "exampleexporter/2"},

			// Check pipelines_builder.yaml to understand this case.
			// We have 2 pipelines, one exporting to one exporter, the other
			// exporting to both exporters, so we expect a duplication on
			// one of the exporters, but not on the other.
			spanDuplicationByExporter: map[string]int{
				"exampleexporter": 2, "exampleexporter/2": 1,
			},
			hasTraces: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testReceivers(t, test)
		})
	}
}

func testReceivers(
	t *testing.T,
	test testCase,
) {
	factories, err := componenttest.ExampleComponents()
	assert.NoError(t, err)

	attrFactory := attributesprocessor.NewFactory()
	factories.Processors[attrFactory.Type()] = attrFactory
	cfg, err := configtest.LoadConfigFile(t, "testdata/pipelines_builder.yaml", factories)
	require.Nil(t, err)

	// Build the pipeline
	allExporters, err := NewExportersBuilder(zap.NewNop(), cfg, factories.Exporters).Build()
	assert.NoError(t, err)
	pipelineProcessors, err := NewPipelinesBuilder(zap.NewNop(), cfg, allExporters, factories.Processors).Build()
	assert.NoError(t, err)
	receivers, err := NewReceiversBuilder(zap.NewNop(), cfg, pipelineProcessors, factories.Receivers).Build()

	assert.NoError(t, err)
	require.NotNil(t, receivers)

	receiver := receivers[cfg.Receivers[test.receiverName]]

	// Ensure receiver has its fields correctly populated.
	require.NotNil(t, receiver)

	assert.NotNil(t, receiver.receiver)

	// Compose the list of created exporters.
	var exporters []*builtExporter
	for _, name := range test.exporterNames {
		// Ensure exporter is created.
		exp := allExporters[cfg.Exporters[name]]
		require.NotNil(t, exp)
		exporters = append(exporters, exp)
	}

	// Send TraceData via receiver and verify that all exporters of the pipeline receive it.

	// First check that there are no traces in the exporters yet.
	for _, exporter := range exporters {
		consumer := exporter.te.(*componenttest.ExampleExporterConsumer)
		require.Equal(t, len(consumer.Traces), 0)
		require.Equal(t, len(consumer.Metrics), 0)
	}

	if test.hasTraces {
		traceProducer := receiver.receiver.(*componenttest.ExampleReceiverProducer)
		traceProducer.TraceConsumer.ConsumeTraces(context.Background(), testdata.GenerateTraceDataOneSpan())
	}

	metrics := pdatautil.MetricsFromInternalMetrics(testdata.GenerateMetricDataOneMetric())
	if test.hasMetrics {
		metricsProducer := receiver.receiver.(*componenttest.ExampleReceiverProducer)
		metricsProducer.MetricsConsumer.ConsumeMetrics(context.Background(), metrics)
	}

	// Now verify received data.
	for _, name := range test.exporterNames {
		// Check that the data is received by exporter.
		exporter := allExporters[cfg.Exporters[name]]

		// Validate traces.
		if test.hasTraces {
			var spanDuplicationCount int
			if test.spanDuplicationByExporter != nil {
				spanDuplicationCount = test.spanDuplicationByExporter[name]
			} else {
				spanDuplicationCount = 1
			}

			traceConsumer := exporter.te.(*componenttest.ExampleExporterConsumer)
			require.Equal(t, spanDuplicationCount, len(traceConsumer.Traces))

			for i := 0; i < spanDuplicationCount; i++ {
				assert.EqualValues(t, generateTestTracesWithAttributes(), traceConsumer.Traces[i])
			}
		}

		// Validate metrics.
		if test.hasMetrics {
			metricsConsumer := exporter.me.(*componenttest.ExampleExporterConsumer)
			require.Equal(t, 1, len(metricsConsumer.Metrics))
			assert.EqualValues(t, metrics, metricsConsumer.Metrics[0])
		}
	}
}

func TestReceiversBuilder_BuildCustom(t *testing.T) {
	factories := createExampleFactories()

	tests := []struct {
		dataType   string
		shouldFail bool
	}{
		{
			dataType:   "logs",
			shouldFail: false,
		},
		{
			dataType:   "nosuchdatatype",
			shouldFail: true,
		},
	}

	for _, test := range tests {
		t.Run(test.dataType, func(t *testing.T) {
			dataType := test.dataType

			cfg := createExampleConfig(dataType)

			// Build the pipeline
			allExporters, err := NewExportersBuilder(zap.NewNop(), cfg, factories.Exporters).Build()
			if test.shouldFail {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			pipelineProcessors, err := NewPipelinesBuilder(zap.NewNop(), cfg, allExporters, factories.Processors).Build()
			assert.NoError(t, err)
			receivers, err := NewReceiversBuilder(zap.NewNop(), cfg, pipelineProcessors, factories.Receivers).Build()

			assert.NoError(t, err)
			require.NotNil(t, receivers)

			receiver := receivers[cfg.Receivers["examplereceiver"]]

			// Ensure receiver has its fields correctly populated.
			require.NotNil(t, receiver)

			assert.NotNil(t, receiver.receiver)

			// Compose the list of created exporters.
			exporterNames := []string{"exampleexporter"}
			var exporters []*builtExporter
			for _, name := range exporterNames {
				// Ensure exporter is created.
				exp := allExporters[cfg.Exporters[name]]
				require.NotNil(t, exp)
				exporters = append(exporters, exp)
			}

			// Send Data via receiver and verify that all exporters of the pipeline receive it.

			// First check that there are no traces in the exporters yet.
			for _, exporter := range exporters {
				consumer := exporter.le.(*componenttest.ExampleExporterConsumer)
				require.Equal(t, len(consumer.Logs), 0)
			}

			// Send one data.
			log := pdata.Logs{}
			producer := receiver.receiver.(*componenttest.ExampleReceiverProducer)
			producer.LogConsumer.ConsumeLogs(context.Background(), log)

			// Now verify received data.
			for _, name := range exporterNames {
				// Check that the data is received by exporter.
				exporter := allExporters[cfg.Exporters[name]]

				// Validate exported data.
				consumer := exporter.le.(*componenttest.ExampleExporterConsumer)
				require.Equal(t, 1, len(consumer.Logs))
				assert.EqualValues(t, log, consumer.Logs[0])
			}
		})
	}
}

func TestReceiversBuilder_DataTypeError(t *testing.T) {
	factories, err := componenttest.ExampleComponents()
	assert.NoError(t, err)

	attrFactory := attributesprocessor.NewFactory()
	factories.Processors[attrFactory.Type()] = attrFactory
	cfg, err := configtest.LoadConfigFile(t, "testdata/pipelines_builder.yaml", factories)
	assert.NoError(t, err)

	// Make examplereceiver to "unsupport" trace data type.
	receiver := cfg.Receivers["examplereceiver"]
	receiver.(*componenttest.ExampleReceiver).FailTraceCreation = true

	// Build the pipeline
	allExporters, err := NewExportersBuilder(zap.NewNop(), cfg, factories.Exporters).Build()
	assert.NoError(t, err)
	pipelineProcessors, err := NewPipelinesBuilder(zap.NewNop(), cfg, allExporters, factories.Processors).Build()
	assert.NoError(t, err)
	receivers, err := NewReceiversBuilder(zap.NewNop(), cfg, pipelineProcessors, factories.Receivers).Build()

	// This should fail because "examplereceiver" is attached to "traces" pipeline
	// which is a configuration error.
	assert.NotNil(t, err)
	assert.Nil(t, receivers)
}

func TestReceiversBuilder_StartAll(t *testing.T) {
	receivers := make(Receivers)
	rcvCfg := &configmodels.ReceiverSettings{}

	receiver := &componenttest.ExampleReceiverProducer{}

	receivers[rcvCfg] = &builtReceiver{
		logger:   zap.NewNop(),
		receiver: receiver,
	}

	assert.Equal(t, false, receiver.Started)

	err := receivers.StartAll(context.Background(), componenttest.NewNopHost())
	assert.NoError(t, err)

	assert.Equal(t, true, receiver.Started)
}

func TestReceiversBuilder_StopAll(t *testing.T) {
	receivers := make(Receivers)
	rcvCfg := &configmodels.ReceiverSettings{}

	receiver := &componenttest.ExampleReceiverProducer{}

	receivers[rcvCfg] = &builtReceiver{
		logger:   zap.NewNop(),
		receiver: receiver,
	}

	assert.Equal(t, false, receiver.Stopped)

	assert.NoError(t, receivers.ShutdownAll(context.Background()))

	assert.Equal(t, true, receiver.Stopped)
}

func TestReceiversBuilder_ErrorOnNilReceiver(t *testing.T) {
	factories, err := componenttest.ExampleComponents()
	assert.NoError(t, err)

	bf := &badReceiverFactory{}
	factories.Receivers[bf.Type()] = bf

	cfg, err := configtest.LoadConfigFile(t, "testdata/bad_receiver_factory.yaml", factories)
	require.Nil(t, err)

	// Build the pipeline
	allExporters, err := NewExportersBuilder(zap.NewNop(), cfg, factories.Exporters).Build()
	assert.NoError(t, err)
	pipelineProcessors, err := NewPipelinesBuilder(zap.NewNop(), cfg, allExporters, factories.Processors).Build()
	assert.NoError(t, err)

	// First test only trace receivers by removing the metrics pipeline.
	metricsPipeline := cfg.Service.Pipelines["metrics"]
	delete(cfg.Service.Pipelines, "metrics")
	require.Equal(t, 1, len(cfg.Service.Pipelines))

	receivers, err := NewReceiversBuilder(zap.NewNop(), cfg, pipelineProcessors, factories.Receivers).Build()
	assert.Error(t, err)
	assert.Zero(t, len(receivers))

	// Now test the metric pipeline.
	delete(cfg.Service.Pipelines, "traces")
	cfg.Service.Pipelines["metrics"] = metricsPipeline
	require.Equal(t, 1, len(cfg.Service.Pipelines))

	receivers, err = NewReceiversBuilder(zap.NewNop(), cfg, pipelineProcessors, factories.Receivers).Build()
	assert.Error(t, err)
	assert.Zero(t, len(receivers))
}

func TestReceiversBuilder_Unused(t *testing.T) {
	factories, err := componenttest.ExampleComponents()
	assert.NoError(t, err)

	attrFactory := attributesprocessor.NewFactory()
	factories.Processors[attrFactory.Type()] = attrFactory

	zpkFactory := zipkinreceiver.NewFactory()
	factories.Receivers[zpkFactory.Type()] = zpkFactory
	cfg, err := configtest.LoadConfigFile(t, "testdata/unused_receiver.yaml", factories)
	assert.NoError(t, err)

	// Build the pipeline
	allExporters, err := NewExportersBuilder(zap.NewNop(), cfg, factories.Exporters).Build()
	assert.NoError(t, err)
	pipelineProcessors, err := NewPipelinesBuilder(zap.NewNop(), cfg, allExporters, factories.Processors).Build()
	assert.NoError(t, err)
	receivers, err := NewReceiversBuilder(zap.NewNop(), cfg, pipelineProcessors, factories.Receivers).Build()
	assert.NoError(t, err)
	assert.NotNil(t, receivers)

	assert.NoError(t, receivers.StartAll(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, receivers.ShutdownAll(context.Background()))
}

func TestReceiversBuilder_InternalToOcTraceConverter(t *testing.T) {
	factories, err := componenttest.ExampleComponents()
	assert.NoError(t, err)

	newStyleReceiver := &newStyleReceiverFactory{}
	factories.Receivers[newStyleReceiver.Type()] = newStyleReceiver

	cfg, err := configtest.LoadConfigFile(t, "testdata/new_style_receiver_factory.yaml", factories)
	require.Nil(t, err)

	// Build the pipeline
	allExporters, err := NewExportersBuilder(zap.NewNop(), cfg, factories.Exporters).Build()
	assert.NoError(t, err)
	pipelineProcessors, err := NewPipelinesBuilder(zap.NewNop(), cfg, allExporters, factories.Processors).Build()
	assert.NoError(t, err)
	receivers, err := NewReceiversBuilder(zap.NewNop(), cfg, pipelineProcessors, factories.Receivers).Build()
	assert.NoError(t, err)
	assert.NotNil(t, receivers)

	receiver := receivers[cfg.Receivers["newstylereceiver"]].receiver
	assert.NotNil(t, receiver)
}

// badReceiverFactory is a factory that returns no error but returns a nil object.
type badReceiverFactory struct{}

func (b *badReceiverFactory) Type() configmodels.Type {
	return "bf"
}

func (b *badReceiverFactory) CreateDefaultConfig() configmodels.Receiver {
	return &configmodels.ReceiverSettings{}
}

func (b *badReceiverFactory) CreateTraceReceiver(
	context.Context,
	component.ReceiverCreateParams,
	configmodels.Receiver,
	consumer.TraceConsumer,
) (component.TraceReceiver, error) {
	return nil, nil
}

func (b *badReceiverFactory) CreateMetricsReceiver(
	context.Context,
	component.ReceiverCreateParams,
	configmodels.Receiver,
	consumer.MetricsConsumer,
) (component.MetricsReceiver, error) {
	return nil, nil
}

// newStyleReceiverFactory defines FactoryV2 interface
type newStyleReceiverFactory struct{}

func (b *newStyleReceiverFactory) Type() configmodels.Type {
	return "newstylereceiver"
}

func (b *newStyleReceiverFactory) CreateDefaultConfig() configmodels.Receiver {
	return &configmodels.ReceiverSettings{}
}

func (b *newStyleReceiverFactory) CreateTraceReceiver(
	_ context.Context,
	_ component.ReceiverCreateParams,
	_ configmodels.Receiver,
	_ consumer.TraceConsumer,
) (component.TraceReceiver, error) {
	return &componenttest.ExampleReceiverProducer{}, nil
}

func (b *newStyleReceiverFactory) CreateMetricsReceiver(
	_ context.Context,
	_ component.ReceiverCreateParams,
	_ configmodels.Receiver,
	_ consumer.MetricsConsumer,
) (component.MetricsReceiver, error) {
	return &componenttest.ExampleReceiverProducer{}, nil
}
