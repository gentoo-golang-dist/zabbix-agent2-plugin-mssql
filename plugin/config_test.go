/*
** Copyright (C) 2001-2024 Zabbix SIA
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

package plugin

import "testing"

func Test_mssqlPlugin_Validate(t *testing.T) {
	t.Parallel()

	type args struct {
		options any
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"+valid",
			args{[]byte(`KeepAlive=300`)},
			false,
		},
		{
			"-marshalErr",
			args{[]byte(`KeepDead=300`)},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := (&mssqlPlugin{}).Validate(tt.args.options)
			if (err != nil) != tt.wantErr {
				t.Fatalf(
					"mssqlPlugin.Validate() error = %v, wantErr %v",
					err, tt.wantErr,
				)
			}
		})
	}
}
