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

package filterprocessor

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/o11y/opentelemetry-collector-o11y/component/componenttest"
	"github.com/o11y/opentelemetry-collector-o11y/config/configmodels"
	"github.com/o11y/opentelemetry-collector-o11y/config/configtest"
	"github.com/o11y/opentelemetry-collector-o11y/internal/processor/filtermetric"
	"github.com/o11y/opentelemetry-collector-o11y/internal/processor/filterset"
	fsregexp "github.com/o11y/opentelemetry-collector-o11y/internal/processor/filterset/regexp"
)

// TestLoadingConfigRegexp tests loading testdata/config_strict.yaml
func TestLoadingConfigStrict(t *testing.T) {
	// list of filters used repeatedly on testdata/config_strict.yaml
	testDataFilters := []string{
		"hello_world",
		"hello/world",
	}

	testDataMetricProperties := &filtermetric.MatchProperties{
		Config: filterset.Config{
			MatchType: filterset.Strict,
		},
		MetricNames: testDataFilters,
	}

	factories, err := componenttest.ExampleComponents()
	assert.Nil(t, err)

	factory := NewFactory()
	factories.Processors[configmodels.Type(typeStr)] = factory
	config, err := configtest.LoadConfigFile(t, path.Join(".", "testdata", "config_strict.yaml"), factories)

	assert.Nil(t, err)
	require.NotNil(t, config)

	tests := []struct {
		filterName string
		expCfg     *Config
	}{
		{
			filterName: "filter/empty",
			expCfg: &Config{
				ProcessorSettings: configmodels.ProcessorSettings{
					NameVal: "filter/empty",
					TypeVal: typeStr,
				},
				Metrics: MetricFilters{
					Include: &filtermetric.MatchProperties{
						Config: filterset.Config{
							MatchType: filterset.Strict,
						},
					},
				},
			},
		}, {
			filterName: "filter/include",
			expCfg: &Config{
				ProcessorSettings: configmodels.ProcessorSettings{
					NameVal: "filter/include",
					TypeVal: typeStr,
				},
				Metrics: MetricFilters{
					Include: testDataMetricProperties,
				},
			},
		}, {
			filterName: "filter/exclude",
			expCfg: &Config{
				ProcessorSettings: configmodels.ProcessorSettings{
					NameVal: "filter/exclude",
					TypeVal: typeStr,
				},
				Metrics: MetricFilters{
					Exclude: testDataMetricProperties,
				},
			},
		}, {
			filterName: "filter/includeexclude",
			expCfg: &Config{
				ProcessorSettings: configmodels.ProcessorSettings{
					NameVal: "filter/includeexclude",
					TypeVal: typeStr,
				},
				Metrics: MetricFilters{
					Include: testDataMetricProperties,
					Exclude: &filtermetric.MatchProperties{
						Config: filterset.Config{
							MatchType: filterset.Strict,
						},
						MetricNames: []string{"hello_world"},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.filterName, func(t *testing.T) {
			cfg := config.Processors[test.filterName]
			assert.Equal(t, test.expCfg, cfg)
		})
	}
}

// TestLoadingConfigRegexp tests loading testdata/config_regexp.yaml
func TestLoadingConfigRegexp(t *testing.T) {
	// list of filters used repeatedly on testdata/config.yaml
	testDataFilters := []string{
		"prefix/.*",
		"prefix_.*",
		".*/suffix",
		".*_suffix",
		".*/contains/.*",
		".*_contains_.*",
		"full/name/match",
		"full_name_match",
	}

	testDataMetricProperties := &filtermetric.MatchProperties{
		Config: filterset.Config{
			MatchType: filterset.Regexp,
		},
		MetricNames: testDataFilters,
	}

	factories, err := componenttest.ExampleComponents()
	assert.Nil(t, err)

	factory := NewFactory()
	factories.Processors[typeStr] = factory
	config, err := configtest.LoadConfigFile(t, path.Join(".", "testdata", "config_regexp.yaml"), factories)

	assert.Nil(t, err)
	require.NotNil(t, config)

	tests := []struct {
		filterName string
		expCfg     *Config
	}{
		{
			filterName: "filter/include",
			expCfg: &Config{
				ProcessorSettings: configmodels.ProcessorSettings{
					NameVal: "filter/include",
					TypeVal: typeStr,
				},
				Metrics: MetricFilters{
					Include: testDataMetricProperties,
				},
			},
		}, {
			filterName: "filter/exclude",
			expCfg: &Config{
				ProcessorSettings: configmodels.ProcessorSettings{
					NameVal: "filter/exclude",
					TypeVal: typeStr,
				},
				Metrics: MetricFilters{
					Exclude: testDataMetricProperties,
				},
			},
		}, {
			filterName: "filter/unlimitedcache",
			expCfg: &Config{
				ProcessorSettings: configmodels.ProcessorSettings{
					NameVal: "filter/unlimitedcache",
					TypeVal: typeStr,
				},
				Metrics: MetricFilters{
					Include: &filtermetric.MatchProperties{
						Config: filterset.Config{
							MatchType: filterset.Regexp,
							RegexpConfig: &fsregexp.Config{
								CacheEnabled: true,
							},
						},
						MetricNames: testDataFilters,
					},
				},
			},
		}, {
			filterName: "filter/limitedcache",
			expCfg: &Config{
				ProcessorSettings: configmodels.ProcessorSettings{
					NameVal: "filter/limitedcache",
					TypeVal: typeStr,
				},
				Metrics: MetricFilters{
					Exclude: &filtermetric.MatchProperties{
						Config: filterset.Config{
							MatchType: filterset.Regexp,
							RegexpConfig: &fsregexp.Config{
								CacheEnabled:       true,
								CacheMaxNumEntries: 10,
							},
						},
						MetricNames: testDataFilters,
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.filterName, func(t *testing.T) {
			cfg := config.Processors[test.filterName]
			assert.Equal(t, test.expCfg, cfg)
		})
	}
}
