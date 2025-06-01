//go:build !windows

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

package plugin

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

const validTestPath = "/valid/abs/path"

func Test_pluginConfig_setCustomQueriesDirDefault(t *testing.T) {
	t.Parallel()

	type fields struct {
		CustomQueriesDir     string
		CustomQueriesEnabled bool
	}

	tests := []struct {
		name   string
		fields fields
		want   *pluginConfig
	}{
		{
			"+valid",
			fields{
				CustomQueriesDir:     "path/to/dir",
				CustomQueriesEnabled: true,
			},
			&pluginConfig{
				CustomQueriesDir:     "path/to/dir",
				CustomQueriesEnabled: true,
			},
		},
		{
			"+default",
			fields{
				CustomQueriesEnabled: true,
			},
			&pluginConfig{
				CustomQueriesDir:     "/usr/local/share/zabbix/custom-queries/mssql",
				CustomQueriesEnabled: true,
			},
		},
		{
			"-empty",
			fields{},
			&pluginConfig{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pc := pluginConfig{
				CustomQueriesDir:     tt.fields.CustomQueriesDir,
				CustomQueriesEnabled: tt.fields.CustomQueriesEnabled,
			}

			pc.setCustomQueriesDirDefault()

			if diff := cmp.Diff(tt.want, &pc); diff != "" {
				t.Fatalf("pluginConfig.setCustomQueriesDirDefault() = %s", diff)
			}
		})
	}
}
