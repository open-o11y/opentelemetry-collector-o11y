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

package kafkareceiver

import (
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/o11y/opentelemetry-collector-o11y/component/componenttest"
	"github.com/o11y/opentelemetry-collector-o11y/config/configmodels"
	"github.com/o11y/opentelemetry-collector-o11y/config/configtest"
	"github.com/o11y/opentelemetry-collector-o11y/exporter/kafkaexporter"
)

func TestLoadConfig(t *testing.T) {
	factories, err := componenttest.ExampleComponents()
	assert.NoError(t, err)

	factory := NewFactory()
	factories.Receivers[typeStr] = factory
	cfg, err := configtest.LoadConfigFile(t, path.Join(".", "testdata", "config.yaml"), factories)
	require.NoError(t, err)
	require.Equal(t, 1, len(cfg.Receivers))

	r := cfg.Receivers[typeStr].(*Config)
	assert.Equal(t, &Config{
		ReceiverSettings: configmodels.ReceiverSettings{
			NameVal: typeStr,
			TypeVal: typeStr,
		},
		Topic:    "spans",
		Brokers:  []string{"foo:123", "bar:456"},
		ClientID: "otel-collector",
		GroupID:  "otel-collector",
		Metadata: kafkaexporter.Metadata{
			Full: true,
			Retry: kafkaexporter.MetadataRetry{
				Max:     10,
				Backoff: time.Second * 5,
			},
		},
	}, r)
}
