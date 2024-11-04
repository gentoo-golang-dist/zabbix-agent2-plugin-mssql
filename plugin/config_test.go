/*
** Zabbix
** Copyright (C) 2001-2024 Zabbix SIA
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
				KeepAlive: 300,
				Timeout:   3,
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
				KeepAlive: 300,
				Timeout:   2,
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
				KeepAlive: 300,
				Timeout:   3,
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
		tt := tt
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
