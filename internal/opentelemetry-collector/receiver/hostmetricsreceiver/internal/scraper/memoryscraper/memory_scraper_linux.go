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

// +build linux

package memoryscraper

import (
	"github.com/shirou/gopsutil/mem"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/dataold"
)

const memStatesLen = 6

func appendMemoryUsageStateDataPoints(idps dataold.Int64DataPointSlice, now pdata.TimestampUnixNano, memInfo *mem.VirtualMemoryStat) {
	initializeMemoryUsageDataPoint(idps.At(0), now, usedStateLabelValue, int64(memInfo.Used))
	initializeMemoryUsageDataPoint(idps.At(1), now, freeStateLabelValue, int64(memInfo.Free))
	initializeMemoryUsageDataPoint(idps.At(2), now, bufferedStateLabelValue, int64(memInfo.Buffers))
	initializeMemoryUsageDataPoint(idps.At(3), now, cachedStateLabelValue, int64(memInfo.Cached))
	initializeMemoryUsageDataPoint(idps.At(4), now, slabReclaimableStateLabelValue, int64(memInfo.SReclaimable))
	initializeMemoryUsageDataPoint(idps.At(5), now, slabUnreclaimableStateLabelValue, int64(memInfo.SUnreclaim))
}
