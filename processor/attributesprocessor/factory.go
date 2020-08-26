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

package attributesprocessor

import (
	"context"
	"fmt"

	"github.com/o11y/opentelemetry-collector-o11y/component"
	"github.com/o11y/opentelemetry-collector-o11y/config/configmodels"
	"github.com/o11y/opentelemetry-collector-o11y/consumer"
	"github.com/o11y/opentelemetry-collector-o11y/internal/processor/filterspan"
	"github.com/o11y/opentelemetry-collector-o11y/processor/processorhelper"
)

const (
	// typeStr is the value of "type" key in configuration.
	typeStr = "attributes"
)

var processorCapabilities = component.ProcessorCapabilities{MutatesConsumedData: true}

// NewFactory returns a new factory for the Attributes processor.
func NewFactory() component.ProcessorFactory {
	return processorhelper.NewFactory(
		typeStr,
		createDefaultConfig,
		processorhelper.WithTraces(createTraceProcessor))
}

// Note: This isn't a valid configuration because the processor would do no work.
func createDefaultConfig() configmodels.Processor {
	return &Config{
		ProcessorSettings: configmodels.ProcessorSettings{
			TypeVal: typeStr,
			NameVal: typeStr,
		},
	}
}

func createTraceProcessor(
	_ context.Context,
	_ component.ProcessorCreateParams,
	cfg configmodels.Processor,
	nextConsumer consumer.TraceConsumer,
) (component.TraceProcessor, error) {
	oCfg := cfg.(*Config)
	if len(oCfg.Actions) == 0 {
		return nil, fmt.Errorf("error creating \"attributes\" processor due to missing required field \"actions\" of processor %q", cfg.Name())
	}
	attrProc, err := processorhelper.NewAttrProc(&oCfg.Settings)
	if err != nil {
		return nil, fmt.Errorf("error creating \"attributes\" processor: %w of processor %q", err, cfg.Name())
	}
	include, err := filterspan.NewMatcher(oCfg.Include)
	if err != nil {
		return nil, err
	}
	exclude, err := filterspan.NewMatcher(oCfg.Exclude)
	if err != nil {
		return nil, err
	}

	return processorhelper.NewTraceProcessor(
		cfg,
		nextConsumer,
		newAttributesProcessor(attrProc, include, exclude),
		processorhelper.WithCapabilities(processorCapabilities))
}
