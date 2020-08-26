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

package tailsamplingprocessor

import (
	"context"
	"time"

	"github.com/o11y/opentelemetry-collector-o11y/component"
	"github.com/o11y/opentelemetry-collector-o11y/config/configmodels"
	"github.com/o11y/opentelemetry-collector-o11y/consumer"
	"github.com/o11y/opentelemetry-collector-o11y/processor/processorhelper"
)

const (
	// The value of "type" Tail Sampling in configuration.
	typeStr = "tail_sampling"
)

// NewFactory returns a new factory for the Tail Sampling processor.
func NewFactory() component.ProcessorFactory {
	return processorhelper.NewFactory(
		typeStr,
		createDefaultConfig,
		processorhelper.WithTraces(createTraceProcessor))
}

func createDefaultConfig() configmodels.Processor {
	return &Config{
		DecisionWait: 30 * time.Second,
		NumTraces:    50000,
	}
}

func createTraceProcessor(
	_ context.Context,
	params component.ProcessorCreateParams,
	cfg configmodels.Processor,
	nextConsumer consumer.TraceConsumer,
) (component.TraceProcessor, error) {
	tCfg := cfg.(*Config)
	return newTraceProcessor(params.Logger, nextConsumer, *tCfg)
}
