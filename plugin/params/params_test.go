/*
** Copyright (C) 2001-2025 Zabbix SIA
**
** This program is free software: you can redistribute it and/or modify it under the terms of
** the GNU Affero General Public License as published by the Free Software Foundation, version 3.
**
** This program is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY;
** without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.
** See the GNU Affero General Public License for more details.
**
** You should have received a copy of the GNU Affero General Public License along with this program.
** If not, see <https://www.gnu.org/licenses/>.
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
		tt := tt
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
