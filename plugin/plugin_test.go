/*
** Copyright (C) 2001-2026 Zabbix SIA
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
	_ "embed"
	"errors"
	"fmt"
	stdlog "log"
	"os"
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
	"golang.zabbix.com/sdk/zbxsync"
)

type mockCtx struct {
	plugin.ContextProvider
	timeout int
}

func (m *mockCtx) Timeout() int {
	return m.timeout
}

//nolint:paralleltest,tparallel
func Test_mssqlPlugin_Start(t *testing.T) {
	log.DefaultLogger = stdlog.New(os.Stdout, "", stdlog.LstdFlags)

	sampleConnCollection := &dbconn.ConnManager{}
	sampleConnCollection.Init(30, &mssqlPlugin{})

	type fields struct {
		Base          plugin.Base
		mgr           *dbconn.ConnManager
		config        *pluginConfig
		customQueries handlers.CustomQueries
	}

	tests := []struct {
		name              string
		fields            fields
		wantCons          *dbconn.ConnManager
		wantCustomQueries handlers.CustomQueries
	}{
		{
			"+valid",
			fields{
				Base: plugin.Base{
					Logger: log.New("test"),
				},
				mgr: &dbconn.ConnManager{},
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
				conns:         tt.fields.mgr,
				config:        tt.fields.config,
				customQueries: tt.fields.customQueries,
			}

			p.Start()

			if diff := cmp.Diff(
				tt.wantCons, p.conns,
				cmp.AllowUnexported(dbconn.ConnManager{}, mssqlPlugin{}),
				cmpopts.IgnoreUnexported(zbxsync.SyncMap[dbconn.ConnConfig, *dbconn.ConnItem]{}),
				cmpopts.IgnoreFields(dbconn.ConnManager{}, "logr"),
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
		mgr *dbconn.ConnManager
	}

	tests := []struct {
		name   string
		fields fields
	}{
		{"+valid", fields{mgr: &dbconn.ConnManager{}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tt.fields.mgr.Init(0, nil)
			p := &mssqlPlugin{conns: tt.fields.mgr}

			p.Stop()
		})
	}
}

func Test_mssqlPlugin_Export(t *testing.T) {
	t.Parallel()

	newHandler := func(err error, failMarshal bool) handlers.HandlerFunc {
		return func(
			timeout time.Duration, metricParams map[string]string, extraParams ...string,
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

	newTimeoutHandler := func() handlers.HandlerFunc {
		return func(
			timeout time.Duration, _ map[string]string, _ ...string,
		) (any, error) {
			return timeout, nil
		}
	}

	type fields struct {
		Base          plugin.Base
		conns         *dbconn.ConnManager
		config        *pluginConfig
		metrics       map[mssqlMetricKey]*mssqlMetric
		customQueries handlers.CustomQueries
	}

	type args struct {
		key       string
		rawParams []string
		pluginCtx plugin.ContextProvider
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
				conns:  &dbconn.ConnManager{},
				config: &pluginConfig{},
			},
			args{
				key: string(dbGet),
				rawParams: []string{
					"sqlserver://uri", "dddd", "8888", "extra", "param",
				},
				pluginCtx: &mockCtx{},
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
				conns:  &dbconn.ConnManager{},
				config: &pluginConfig{},
			},
			args{
				key: "unknown",
				rawParams: []string{
					"sqlserver://uri", "dddd", "8888", "extra", "param",
				},
				pluginCtx: &mockCtx{},
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
				conns:  &dbconn.ConnManager{},
				config: &pluginConfig{},
			},
			args{
				key: string(dbGet),
				rawParams: []string{
					"sqlserver://uri", "dddd", "8888", "extra", "param",
				},
				pluginCtx: &mockCtx{},
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
				conns:  &dbconn.ConnManager{},
				config: &pluginConfig{},
			},
			args{
				key: string(dbGet),
				rawParams: []string{
					"sqlserver://uri", "dddd", "8888", "extra", "param",
				},
				pluginCtx: &mockCtx{},
			},
			nil,
			true,
		},
		{
			"+itemTimeoutLargerThanConfigTimeout",
			fields{
				metrics: map[mssqlMetricKey]*mssqlMetric{
					dbGet: {
						metric: metric.New(
							"Test.",
							nil,
							false,
						),
						handler: newTimeoutHandler(),
					},
				},
				conns: &dbconn.ConnManager{},
				config: &pluginConfig{
					Timeout: 3,
				},
			},
			args{
				key:       string(dbGet),
				rawParams: []string{},
				pluginCtx: &mockCtx{timeout: 10},
			},
			time.Second * 10,
			false,
		},
		{
			"+itemTimeoutSmallerThanConfigTimeout",
			fields{
				metrics: map[mssqlMetricKey]*mssqlMetric{
					dbGet: {
						metric: metric.New(
							"Test.",
							nil,
							false,
						),
						handler: newTimeoutHandler(),
					},
				},
				conns: &dbconn.ConnManager{},
				config: &pluginConfig{
					Timeout: 8,
				},
			},
			args{
				key:       string(dbGet),
				rawParams: []string{},
				pluginCtx: &mockCtx{timeout: 3},
			},
			time.Second * 8,
			false,
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

			got, err := p.Export(
				tt.args.key,
				tt.args.rawParams,
				tt.args.pluginCtx,
			)
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
