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
	"github.com/o11y/opentelemetry-collector-o11y/config/confighttp"
	"github.com/o11y/opentelemetry-collector-o11y/config/configmodels"
	"github.com/o11y/opentelemetry-collector-o11y/exporter/exporterhelper"
)

// Config defines configuration for Remote Write exporter.
type Config struct {
	// squash ensures fields are correctly decoded in embedded struct.
	configmodels.ExporterSettings  `mapstructure:",squash"`
	exporterhelper.TimeoutSettings `mapstructure:",squash"`
	exporterhelper.QueueSettings   `mapstructure:"sending_queue"`
	exporterhelper.RetrySettings   `mapstructure:"retry_on_failure"`

	// prefix attached to each exported metric name
	// See: https://prometheus.io/docs/practices/naming/#metric-names
	Namespace string `mapstructure:"namespace"`

	AuthCfg map[string]string `mapstructure:"auth"`

	HTTPClientSettings confighttp.HTTPClientSettings `mapstructure:",squash"`
}
