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

package probabilisticsamplerprocessor

import (
	"context"

	"github.com/o11y/opentelemetry-collector-o11y/component"
	"github.com/o11y/opentelemetry-collector-o11y/config/configmodels"
	"github.com/o11y/opentelemetry-collector-o11y/consumer"
	"github.com/o11y/opentelemetry-collector-o11y/processor/processorhelper"
)

const (
	// The value of "type" trace-samplers in configuration.
	typeStr = "probabilistic_sampler"
)

// NewFactory returns a new factory for the Probabilistic sampler processor.
func NewFactory() component.ProcessorFactory {
	return processorhelper.NewFactory(
		typeStr,
		createDefaultConfig,
		processorhelper.WithTraces(createTraceProcessor))
}

func createDefaultConfig() configmodels.Processor {
	return &Config{
		ProcessorSettings: configmodels.ProcessorSettings{
			TypeVal: typeStr,
			NameVal: typeStr,
		},
	}
}

// CreateTraceProcessor creates a trace processor based on this config.
func createTraceProcessor(
	_ context.Context,
	_ component.ProcessorCreateParams,
	cfg configmodels.Processor,
	nextConsumer consumer.TraceConsumer,
) (component.TraceProcessor, error) {
	oCfg := cfg.(*Config)
	return newTraceProcessor(nextConsumer, *oCfg)
}
