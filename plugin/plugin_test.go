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

package plugin

import (
	_ "embed"
	"errors"
	"fmt"
	stdlog "log"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"golang.zabbix.com/plugin/mssql/plugin/dbconn"
	"golang.zabbix.com/plugin/mssql/plugin/handlers"
	"golang.zabbix.com/plugin/mssql/plugin/params"
	"golang.zabbix.com/sdk/log"
	"golang.zabbix.com/sdk/metric"
	"golang.zabbix.com/sdk/plugin"
)

//nolint:paralleltest,tparallel
func Test_mssqlPlugin_Start(t *testing.T) {
	log.DefaultLogger = stdlog.New(os.Stdout, "", stdlog.LstdFlags)

	sampleConnCollection := &dbconn.ConnCollection{}
	sampleConnCollection.Init(30, 29, &mssqlPlugin{})

	type fields struct {
		Base          plugin.Base
		conns         *dbconn.ConnCollection
		config        *pluginConfig
		customQueries handlers.CustomQueries
	}

	tests := []struct {
		name              string
		fields            fields
		wantCons          *dbconn.ConnCollection
		wantCustomQueries handlers.CustomQueries
	}{
		{
			"+valid",
			fields{
				Base: plugin.Base{
					Logger: log.New("test"),
				},
				conns: &dbconn.ConnCollection{},
				config: &pluginConfig{
					KeepAlive:        30,
					Timeout:          29,
					CustomQueriesDir: "",
				},
				customQueries: handlers.CustomQueries{},
			},
			sampleConnCollection,
			handlers.CustomQueries{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := &mssqlPlugin{
				Base:          tt.fields.Base,
				conns:         tt.fields.conns,
				config:        tt.fields.config,
				customQueries: tt.fields.customQueries,
			}

			p.Start()

			if diff := cmp.Diff(
				tt.wantCons, p.conns,
				cmp.AllowUnexported(dbconn.ConnCollection{}, sync.Mutex{}, mssqlPlugin{}),
				cmpopts.IgnoreFields(dbconn.ConnCollection{}, "mu"),
				cmpopts.IgnoreTypes(mssqlPlugin{}),
			); diff != "" {
				t.Fatalf("mssqlPlugin.Start() = %s", diff)
			}

			if diff := cmp.Diff(
				tt.wantCustomQueries, p.customQueries,
			); diff != "" {
				t.Fatalf("mssqlPlugin.Start() = %s", diff)
			}
		})
	}
}

func Test_mssqlPlugin_Stop(t *testing.T) {
	t.Parallel()

	type fields struct {
		conns *dbconn.ConnCollection
	}

	tests := []struct {
		name   string
		fields fields
	}{
		{"+valid", fields{conns: &dbconn.ConnCollection{}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := &mssqlPlugin{conns: tt.fields.conns}

			p.Stop()
		})
	}
}

func Test_mssqlPlugin_Export(t *testing.T) {
	t.Parallel()

	newHandler := func(err error, failMarshal bool) handlers.HandlerFunc {
		return func(
			metricParams map[string]string, extraParams ...string,
		) (any, error) {
			if err != nil {
				return nil, err
			}

			wantMetricParams := map[string]string{
				params.URI.Name():      "sqlserver://uri",
				params.User.Name():     "dddd",
				params.Password.Name(): "8888",
			}

			wantExtraParams := []string{"extra", "param"}

			if diff := cmp.Diff(wantMetricParams, metricParams); diff != "" {
				return nil, fmt.Errorf("metricParams = %s", diff)
			}

			if diff := cmp.Diff(wantExtraParams, extraParams); diff != "" {
				return nil, fmt.Errorf("extraParams = %s", diff)
			}

			if failMarshal {
				return time.Unix(1000000000000000, 0), nil
			}

			return "handler called", nil
		}
	}

	type fields struct {
		Base          plugin.Base
		conns         *dbconn.ConnCollection
		config        *pluginConfig
		metrics       map[mssqlMetricKey]*mssqlMetric
		customQueries handlers.CustomQueries
	}

	type args struct {
		key       string
		rawParams []string
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    any
		wantErr bool
	}{
		{
			"+valid",
			fields{
				metrics: map[mssqlMetricKey]*mssqlMetric{
					dbGet: {
						metric: metric.New(
							"Returns the availabile databases.",
							[]*metric.Param{
								params.URI, params.User, params.Password,
							},
							true,
						),
						handler: newHandler(nil, false),
					},
				},
				conns:  &dbconn.ConnCollection{},
				config: &pluginConfig{},
			},
			args{
				key: string(dbGet),
				rawParams: []string{
					"sqlserver://uri", "dddd", "8888", "extra", "param",
				},
			},
			"handler called",
			false,
		},
		{
			"-unknownMetric",
			fields{
				metrics: map[mssqlMetricKey]*mssqlMetric{
					dbGet: {
						metric: metric.New(
							"Returns the availabile databases.",
							[]*metric.Param{
								params.URI, params.User, params.Password,
							},
							true,
						),
						handler: newHandler(nil, false),
					},
				},
				conns:  &dbconn.ConnCollection{},
				config: &pluginConfig{},
			},
			args{
				key: "unknown",
				rawParams: []string{
					"sqlserver://uri", "dddd", "8888", "extra", "param",
				},
			},
			nil,
			true,
		},
		{
			"-evalParamsErrExtraParams",
			fields{
				metrics: map[mssqlMetricKey]*mssqlMetric{
					dbGet: {
						metric: metric.New(
							"Returns the availabile databases.",
							[]*metric.Param{
								params.URI, params.User, params.Password,
							},
							false,
						),
						handler: newHandler(nil, false),
					},
				},
				conns:  &dbconn.ConnCollection{},
				config: &pluginConfig{},
			},
			args{
				key: string(dbGet),
				rawParams: []string{
					"sqlserver://uri", "dddd", "8888", "extra", "param",
				},
			},
			nil,
			true,
		},
		{
			"-handlerErr",
			fields{
				metrics: map[mssqlMetricKey]*mssqlMetric{
					dbGet: {
						metric: metric.New(
							"Returns the availabile databases.",
							[]*metric.Param{
								params.URI, params.User, params.Password,
							},
							true,
						),
						handler: newHandler(errors.New("fail"), false),
					},
				},
				conns:  &dbconn.ConnCollection{},
				config: &pluginConfig{},
			},
			args{
				key: string(dbGet),
				rawParams: []string{
					"sqlserver://uri", "dddd", "8888", "extra", "param",
				},
			},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := &mssqlPlugin{
				Base:          tt.fields.Base,
				conns:         tt.fields.conns,
				config:        tt.fields.config,
				metrics:       tt.fields.metrics,
				customQueries: tt.fields.customQueries,
			}

			got, err := p.Export(tt.args.key, tt.args.rawParams, nil)
			if (err != nil) != tt.wantErr {
				t.Fatalf(
					"mssqlPlugin.Export() error = %v, wantErr %v",
					err,
					tt.wantErr,
				)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatalf("mssqlPlugin.Export() = %s", diff)
			}
		})
	}
}

func Test_mssqlPlugin_registerMetrics(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			"+valid",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := (&mssqlPlugin{}).registerMetrics()
			if (err != nil) != tt.wantErr {
				t.Fatalf(
					"mssqlPlugin.registerMetrics() error = %v, wantErr %v",
					err, tt.wantErr,
				)
			}
		})
	}
}
