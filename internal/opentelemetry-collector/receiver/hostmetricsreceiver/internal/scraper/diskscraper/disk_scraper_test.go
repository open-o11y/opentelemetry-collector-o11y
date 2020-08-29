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

package diskscraper

import (
	"context"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/dataold"
	"go.opentelemetry.io/collector/internal/processor/filterset"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal"
)

func TestScrapeMetrics(t *testing.T) {
	type testCase struct {
		name          string
		config        Config
		expectMetrics bool
		newErrRegex   string
	}

	testCases := []testCase{
		{
			name:          "Standard",
			expectMetrics: true,
		},
		{
			name:          "Include Filter that matches nothing",
			config:        Config{Include: MatchConfig{filterset.Config{MatchType: "strict"}, []string{"@*^#&*$^#)"}}},
			expectMetrics: false,
		},
		{
			name:        "Invalid Include Filter",
			config:      Config{Include: MatchConfig{Devices: []string{"test"}}},
			newErrRegex: "^error creating device include filters:",
		},
		{
			name:        "Invalid Exclude Filter",
			config:      Config{Exclude: MatchConfig{Devices: []string{"test"}}},
			newErrRegex: "^error creating device exclude filters:",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			scraper, err := newDiskScraper(context.Background(), &test.config)
			if test.newErrRegex != "" {
				require.Error(t, err)
				require.Regexp(t, test.newErrRegex, err)
				return
			}
			require.NoError(t, err, "Failed to create disk scraper: %v", err)

			err = scraper.Initialize(context.Background())
			require.NoError(t, err, "Failed to initialize disk scraper: %v", err)
			defer func() { assert.NoError(t, scraper.Close(context.Background())) }()

			metrics, err := scraper.ScrapeMetrics(context.Background())
			require.NoError(t, err, "Failed to scrape metrics: %v", err)

			if !test.expectMetrics {
				assert.Equal(t, 0, metrics.Len())
				return
			}

			assert.GreaterOrEqual(t, metrics.Len(), 4)

			assertInt64DiskMetricValid(t, metrics.At(0), diskIODescriptor, 0)
			assertInt64DiskMetricValid(t, metrics.At(1), diskOpsDescriptor, 0)
			assertDoubleDiskMetricValid(t, metrics.At(2), diskTimeDescriptor, 0)
			assertDiskPendingOperationsMetricValid(t, metrics.At(3))

			if runtime.GOOS == "linux" {
				assertInt64DiskMetricValid(t, metrics.At(4), diskMergedDescriptor, 0)
			}

			internal.AssertSameTimeStampForAllMetrics(t, metrics)
		})
	}
}

func assertInt64DiskMetricValid(t *testing.T, metric dataold.Metric, expectedDescriptor dataold.MetricDescriptor, startTime pdata.TimestampUnixNano) {
	internal.AssertDescriptorEqual(t, expectedDescriptor, metric.MetricDescriptor())
	if startTime != 0 {
		internal.AssertInt64MetricStartTimeEquals(t, metric, startTime)
	}
	assert.GreaterOrEqual(t, metric.Int64DataPoints().Len(), 2)
	internal.AssertInt64MetricLabelExists(t, metric, 0, deviceLabelName)
	internal.AssertInt64MetricLabelHasValue(t, metric, 0, directionLabelName, readDirectionLabelValue)
	internal.AssertInt64MetricLabelHasValue(t, metric, 1, directionLabelName, writeDirectionLabelValue)
}

func assertDoubleDiskMetricValid(t *testing.T, metric dataold.Metric, expectedDescriptor dataold.MetricDescriptor, startTime pdata.TimestampUnixNano) {
	internal.AssertDescriptorEqual(t, expectedDescriptor, metric.MetricDescriptor())
	if startTime != 0 {
		internal.AssertInt64MetricStartTimeEquals(t, metric, startTime)
	}
	assert.GreaterOrEqual(t, metric.DoubleDataPoints().Len(), 2)
	internal.AssertDoubleMetricLabelExists(t, metric, 0, deviceLabelName)
	internal.AssertDoubleMetricLabelHasValue(t, metric, 0, directionLabelName, readDirectionLabelValue)
	internal.AssertDoubleMetricLabelHasValue(t, metric, metric.DoubleDataPoints().Len()-1, directionLabelName, writeDirectionLabelValue)
}

func assertDiskPendingOperationsMetricValid(t *testing.T, metric dataold.Metric) {
	internal.AssertDescriptorEqual(t, diskPendingOperationsDescriptor, metric.MetricDescriptor())
	assert.GreaterOrEqual(t, metric.Int64DataPoints().Len(), 1)
	internal.AssertInt64MetricLabelExists(t, metric, 0, deviceLabelName)
}
