//go:build !windows
// +build !windows

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

import (
	stdlog "log"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"golang.zabbix.com/sdk/log"
	"golang.zabbix.com/sdk/plugin"
)

//nolint:paralleltest,tparallel
func Test_mssqlPlugin_Configure(t *testing.T) {
	log.DefaultLogger = stdlog.New(os.Stdout, "", stdlog.LstdFlags)

	type fields struct {
		config *pluginConfig
	}

	type args struct {
		global  *plugin.GlobalOptions
		options any
	}

	tests := []struct {
		name       string
		fields     fields
		args       args
		wantConfig *pluginConfig
	}{
		{
			"+valid",
			fields{},
			args{
				&plugin.GlobalOptions{Timeout: 3},
				[]byte(`KeepAlive=300`),
			},
			&pluginConfig{
				KeepAlive:        300,
				Timeout:          3,
				CustomQueriesDir: "/usr/local/share/zabbix/custom-queries/mssql",
			},
		},
		{
			"+withTimeout",
			fields{},
			args{
				&plugin.GlobalOptions{Timeout: 3},
				[]byte(
					strings.Join([]string{"KeepAlive=300", "Timeout=2"}, "\n"),
				),
			},
			&pluginConfig{
				KeepAlive:        300,
				Timeout:          2,
				CustomQueriesDir: "/usr/local/share/zabbix/custom-queries/mssql",
			},
		},
		{
			"+prevConfig",
			fields{
				&pluginConfig{
					Timeout:          44,
					KeepAlive:        22,
					CustomQueriesDir: "aaa",
				},
			},
			args{
				&plugin.GlobalOptions{Timeout: 3},
				[]byte(`KeepAlive=300`),
			},
			&pluginConfig{
				KeepAlive:        300,
				Timeout:          3,
				CustomQueriesDir: "/usr/local/share/zabbix/custom-queries/mssql",
			},
		},
		{
			"+withCustomQueriesDir",
			fields{},
			args{
				&plugin.GlobalOptions{Timeout: 3},
				[]byte(
					strings.Join([]string{"CustomQueriesDir=custom/path/to/sql/folder"}, "\n"),
				),
			},
			&pluginConfig{
				KeepAlive:        300,
				Timeout:          3,
				CustomQueriesDir: "custom/path/to/sql/folder",
			},
		},
		{
			"-marshalErr",
			fields{},
			args{
				&plugin.GlobalOptions{Timeout: 3},
				[]byte(
					strings.Join(
						[]string{"KeepAlive=300", "Timeout=2", "invalid"},
						"\n",
					),
				),
			},
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := &mssqlPlugin{
				config: tt.fields.config,
				Base:   plugin.Base{Logger: log.New("test")},
			}

			p.Configure(tt.args.global, tt.args.options)

			if diff := cmp.Diff(tt.wantConfig, p.config); diff != "" {
				t.Errorf(
					"mssqlPlugin.Configure() mismatch (-want +got):\n%s",
					diff,
				)
			}
		})
	}
}
