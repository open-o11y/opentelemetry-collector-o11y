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

package logstest

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/o11y/opentelemetry-collector-o11y/consumer/pdata"
)

func TestLogs(t *testing.T) {
	logs := Logs(Log{
		Timestamp: 1,
		Body:      pdata.NewAttributeValueString("asdf"),
		Attributes: map[string]pdata.AttributeValue{
			"a": pdata.NewAttributeValueString("b"),
		},
	})

	require.Equal(t, 1, logs.LogRecordCount())
}
