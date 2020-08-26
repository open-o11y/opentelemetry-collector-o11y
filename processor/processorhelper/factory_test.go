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

package processorhelper

import (
	"context"
	"errors"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"

	"github.com/o11y/opentelemetry-collector-o11y/component"
	"github.com/o11y/opentelemetry-collector-o11y/config/configmodels"
	"github.com/o11y/opentelemetry-collector-o11y/consumer"
)

const typeStr = "test"

var defaultCfg = &configmodels.ProcessorSettings{
	TypeVal: typeStr,
	NameVal: typeStr,
}

func TestNewTrace(t *testing.T) {
	factory := NewFactory(
		typeStr,
		defaultConfig)
	assert.EqualValues(t, typeStr, factory.Type())
	assert.EqualValues(t, defaultCfg, factory.CreateDefaultConfig())
	_, ok := factory.(component.ConfigUnmarshaler)
	assert.False(t, ok)
	_, err := factory.CreateTraceProcessor(context.Background(), component.ProcessorCreateParams{}, nil, defaultCfg)
	assert.Error(t, err)
	_, err = factory.CreateMetricsProcessor(context.Background(), component.ProcessorCreateParams{}, nil, defaultCfg)
	assert.Error(t, err)

	lfactory := factory.(component.LogsProcessorFactory)
	_, err = lfactory.CreateLogsProcessor(context.Background(), component.ProcessorCreateParams{}, defaultCfg, nil)
	assert.Error(t, err)
}

func TestNewMetrics_WithConstructors(t *testing.T) {
	factory := NewFactory(
		typeStr,
		defaultConfig,
		WithTraces(createTraceProcessor),
		WithMetrics(createMetricsProcessor),
		WithLogs(createLogsProcessor),
		WithCustomUnmarshaler(customUnmarshaler))
	assert.EqualValues(t, typeStr, factory.Type())
	assert.EqualValues(t, defaultCfg, factory.CreateDefaultConfig())

	fu, ok := factory.(component.ConfigUnmarshaler)
	assert.True(t, ok)
	assert.Equal(t, errors.New("my error"), fu.Unmarshal(nil, nil))

	_, err := factory.CreateTraceProcessor(context.Background(), component.ProcessorCreateParams{}, nil, defaultCfg)
	assert.NoError(t, err)

	_, err = factory.CreateMetricsProcessor(context.Background(), component.ProcessorCreateParams{}, nil, defaultCfg)
	assert.NoError(t, err)

	lfactory := factory.(component.LogsProcessorFactory)
	_, err = lfactory.CreateLogsProcessor(context.Background(), component.ProcessorCreateParams{}, defaultCfg, nil)
	assert.NoError(t, err)
}

func defaultConfig() configmodels.Processor {
	return defaultCfg
}

func createTraceProcessor(context.Context, component.ProcessorCreateParams, configmodels.Processor, consumer.TraceConsumer) (component.TraceProcessor, error) {
	return nil, nil
}

func createMetricsProcessor(context.Context, component.ProcessorCreateParams, configmodels.Processor, consumer.MetricsConsumer) (component.MetricsProcessor, error) {
	return nil, nil
}

func createLogsProcessor(context.Context, component.ProcessorCreateParams, configmodels.Processor, consumer.LogsConsumer) (component.LogsProcessor, error) {
	return nil, nil
}

func customUnmarshaler(*viper.Viper, interface{}) error {
	return errors.New("my error")
}
