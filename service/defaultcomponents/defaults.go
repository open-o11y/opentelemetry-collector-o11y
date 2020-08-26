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

// Package defaultcomponents composes the default set of components used by the otel service
package defaultcomponents

import (
	"github.com/o11y/opentelemetry-collector-o11y/component"
	"github.com/o11y/opentelemetry-collector-o11y/component/componenterror"
	"github.com/o11y/opentelemetry-collector-o11y/exporter/loggingexporter"
	"github.com/o11y/opentelemetry-collector-o11y/exporter/prometheusremotewriteexporter"
	"github.com/o11y/opentelemetry-collector-o11y/extension/fluentbitextension"
	"github.com/o11y/opentelemetry-collector-o11y/extension/healthcheckextension"
	"github.com/o11y/opentelemetry-collector-o11y/extension/pprofextension"
	"github.com/o11y/opentelemetry-collector-o11y/extension/zpagesextension"
	"github.com/o11y/opentelemetry-collector-o11y/processor/batchprocessor"
	"github.com/o11y/opentelemetry-collector-o11y/receiver/otlpreceiver"
)

// Components returns the default set of components used by the
// OpenTelemetry collector.
func Components() (
	component.Factories,
	error,
) {
	var errs []error

	extensions, err := component.MakeExtensionFactoryMap(
		healthcheckextension.NewFactory(),
		pprofextension.NewFactory(),
		zpagesextension.NewFactory(),
		fluentbitextension.NewFactory(),
	)
	if err != nil {
		errs = append(errs, err)
	}

	receivers, err := component.MakeReceiverFactoryMap(
		otlpreceiver.NewFactory(),
	)
	if err != nil {
		errs = append(errs, err)
	}

	exporters, err := component.MakeExporterFactoryMap(
		prometheusremotewriteexporter.NewFactory(),
		loggingexporter.NewFactory(),
	)
	if err != nil {
		errs = append(errs, err)
	}

	processors, err := component.MakeProcessorFactoryMap(
		batchprocessor.NewFactory(),
	)
	if err != nil {
		errs = append(errs, err)
	}

	factories := component.Factories{
		Extensions: extensions,
		Receivers:  receivers,
		Processors: processors,
		Exporters:  exporters,
	}

	return factories, componenterror.CombineErrors(errs)
}
