/*
** Zabbix
** Copyright (C) 2001-2026 Zabbix SIA
**
** Licensed under the Apache License, Version 2.0 (the "License");
** you may not use this file except in compliance with the License.
** You may obtain a copy of the License at
**
**     http://www.apache.org/licenses/LICENSE-2.0
**
** Unless required by applicable law or agreed to in writing, software
** distributed under the License is distributed on an "AS IS" BASIS,
** WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
** See the License for the specific language governing permissions and
** limitations under the License.
**/

package params

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"golang.zabbix.com/sdk/metric"
)

func TestJoin(t *testing.T) {
	t.Parallel()

	type args struct {
		params [][]*metric.Param
	}

	tests := []struct {
		name string
		args args
		want []*metric.Param
	}{
		{
			"+valid",
			args{[][]*metric.Param{BaseParams, TLSParams}},
			append(BaseParams, TLSParams...),
		},
		{"+single", args{[][]*metric.Param{BaseParams}}, BaseParams},
		{"-empty", args{[][]*metric.Param{}}, nil},
		{"-nil", args{}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := Join(tt.args.params...)
			if diff := cmp.Diff(
				tt.want, got,
				cmp.AllowUnexported(metric.Param{}),
			); diff != "" {
				t.Fatalf("Join() = %s", diff)
			}
		})
	}
}
