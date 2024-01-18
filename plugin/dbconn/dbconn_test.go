/*
** Zabbix
** Copyright 2001-2023 Zabbix SIA
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

package dbconn

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	stdlog "log"
	"os"
	"sync"
	"testing"

	"git.zabbix.com/ap/mssql/plugin/params"
	"git.zabbix.com/ap/plugin-support/log"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

var (
	_ driver.Driver        = (*driverMock)(nil)
	_ driver.DriverContext = (*driverMock)(nil)
	_ driver.Connector     = (*connectorMock)(nil)

	//nolint:gochecknoglobals // global driver instance.
	mockDriver = &driverMock{}
)

type connectorMock struct {
	driver driver.Driver
	name   string
}

type driverMock struct {
	openErr error
	driver  driver.Driver
}

//nolint:gochecknoinits
func init() {
	sql.Register("testdriver", mockDriver)
}

func (d *driverMock) Open(name string) (driver.Conn, error) {
	if d.openErr != nil {
		return nil, d.openErr
	}

	return d.driver.Open(name)
}

func (d *driverMock) OpenConnector(name string) (driver.Connector, error) {
	if d.openErr != nil {
		return nil, d.openErr
	}

	return &connectorMock{d.driver, name}, nil
}

func (d *driverMock) reset() {
	d.openErr = nil
	d.driver = nil
}

func (c *connectorMock) Connect(context.Context) (driver.Conn, error) {
	return c.driver.Open(c.name)
}

func (c *connectorMock) Driver() driver.Driver {
	return c.driver
}

func TestConnCollection_Init(t *testing.T) {
	t.Parallel()

	sampleLogr := &struct{ log.Logger }{}

	type fields struct {
		conns      map[ConnConfig]*sql.DB
		keepAlive  int
		logr       log.Logger
		driverName string
	}

	type args struct {
		keepAlive int
		logr      log.Logger
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		want   *ConnCollection
	}{
		{
			"+valid",
			fields{},
			args{10, sampleLogr},
			&ConnCollection{
				conns:      make(map[ConnConfig]*sql.DB),
				keepAlive:  10,
				logr:       sampleLogr,
				driverName: "sqlserver",
			},
		},
		{
			"-overwrite",
			fields{
				conns:      map[ConnConfig]*sql.DB{{}: nil},
				keepAlive:  3,
				logr:       log.New("aaa"),
				driverName: "lol",
			},
			args{10, sampleLogr},
			&ConnCollection{
				conns:      make(map[ConnConfig]*sql.DB),
				keepAlive:  10,
				logr:       sampleLogr,
				driverName: "sqlserver",
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := &ConnCollection{
				conns:      tt.fields.conns,
				keepAlive:  tt.fields.keepAlive,
				logr:       tt.fields.logr,
				driverName: tt.fields.driverName,
			}
			c.Init(tt.args.keepAlive, tt.args.logr)

			if diff := cmp.Diff(
				tt.want, c, cmp.AllowUnexported(ConnCollection{}, sync.Mutex{}),
			); diff != "" {
				t.Fatalf("ConnCollection.Init() = %s", diff)
			}
		})
	}
}

//nolint:paralleltest
func TestConnCollection_WithConnHandlerFunc(t *testing.T) {
	log.DefaultLogger = stdlog.New(os.Stdout, "", stdlog.LstdFlags)

	type fields struct {
		getErr error
		dsn    string
	}

	type args struct {
		metricParams map[string]string
		extraParams  []string
	}

	tests := []struct {
		name             string
		fields           fields
		args             args
		wantMetricParams map[string]string
		wantExtraParams  []string
		want             any
		wantErr          bool
	}{
		{
			"+valid",
			fields{
				dsn: "pigeon://8888:dddd@uri:1433?app+name=Zabbix+agent+2+MSSQL+plugin&keepAlive=0", //nolint:lll
			},
			args{
				metricParams: map[string]string{
					params.URI.Name():      "pigeon://uri",
					params.User.Name():     "8888",
					params.Password.Name(): "dddd",
				},
			},
			map[string]string{
				params.URI.Name():      "pigeon://uri",
				params.User.Name():     "8888",
				params.Password.Name(): "dddd",
			},
			nil,
			"handler called",
			false,
		},
		{
			"+extraParams",
			fields{
				dsn: "pigeon://7777:dddd@uri:1433?app+name=Zabbix+agent+2+MSSQL+plugin&keepAlive=0", //nolint:lll
			},
			args{
				metricParams: map[string]string{
					params.URI.Name():      "pigeon://uri",
					params.User.Name():     "7777",
					params.Password.Name(): "dddd",
					"extra":                "param",
				},
				extraParams: []string{"extra", "spicey"},
			},
			map[string]string{
				params.URI.Name():      "pigeon://uri",
				params.User.Name():     "7777",
				params.Password.Name(): "dddd",
				"extra":                "param",
			},
			[]string{"extra", "spicey"},
			"handler called",
			false,
		},
		{
			"-getErr",
			fields{
				dsn:    "pigeon://6666:dddd@uri:1433?app+name=Zabbix+agent+2+MSSQL+plugin&keepAlive=0", //nolint:lll
				getErr: errors.New("fail"),
			},
			args{
				metricParams: map[string]string{
					params.URI.Name():      "pigeon://uri",
					params.User.Name():     "6666",
					params.Password.Name(): "dddd",
				},
			},
			nil,
			nil,
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest
			c := &ConnCollection{
				conns:      map[ConnConfig]*sql.DB{},
				driverName: "testdriver",
				logr:       log.New("aaa"),
			}

			db, m, err := sqlmock.NewWithDSN(
				tt.fields.dsn,
				sqlmock.MonitorPingsOption(true),
			)
			if err != nil {
				t.Fatalf("failed to open sqlmock: %s", err.Error())
			}

			mockDriver.driver = db.Driver()

			defer mockDriver.reset()

			m.ExpectPing().WillReturnError(tt.fields.getErr)

			got, err := c.WithConnHandlerFunc(
				func(
					db *sql.DB,
					metricParams map[string]string,
					extraParams ...string,
				) (any, error) {
					if db == nil {
						t.Fatalf(
							"ConnCollection.WithConnHandlerFunc() db is nil",
						)
					}

					if diff := cmp.Diff(
						tt.wantMetricParams, metricParams,
					); diff != "" {
						t.Fatalf(
							"ConnCollection.WithConnHandlerFunc() = %s", diff,
						)
					}

					if diff := cmp.Diff(
						tt.wantExtraParams, extraParams,
					); diff != "" {
						t.Fatalf(
							"ConnCollection.WithConnHandlerFunc() = %s", diff,
						)
					}

					return "handler called", nil
				},
			)(tt.args.metricParams, tt.args.extraParams...)
			if (err != nil) != tt.wantErr {
				t.Fatalf(
					"ConnCollection.WithConnHandlerFunc() "+
						"error = %v, wantErr %v",
					err, tt.wantErr,
				)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatalf("ConnCollection.WithConnHandlerFunc() = %s", diff)
			}
		})
	}
}

//nolint:paralleltest
func TestConnCollection_PingHandler(t *testing.T) {
	log.DefaultLogger = stdlog.New(os.Stdout, "", stdlog.LstdFlags)

	type expect struct {
		ping bool
	}

	type fields struct {
		getErr  error
		pingErr error
		dsn     string
	}

	type args struct {
		metricParams map[string]string
	}

	tests := []struct {
		name    string
		expect  expect
		fields  fields
		args    args
		want    any
		wantErr bool
	}{
		{
			"+valid",
			expect{true},
			fields{
				dsn: "pigeon://aaaa:dddd@uri:1433?app+name=Zabbix+agent+2+MSSQL+plugin&keepAlive=0", //nolint:lll
			},
			args{
				metricParams: map[string]string{
					params.URI.Name():      "pigeon://uri",
					params.User.Name():     "aaaa",
					params.Password.Name(): "dddd",
				},
			},
			1,
			false,
		},
		{
			"-getErr",
			expect{false},
			fields{
				getErr: errors.New("fail"),
				dsn:    "pigeon://aaaa:bbbb@uri:1433?app+name=Zabbix+agent+2+MSSQL+plugin&keepAlive=0", //nolint:lll
			},
			args{
				metricParams: map[string]string{
					params.URI.Name():      "pigeon://uri",
					params.User.Name():     "aaaa",
					params.Password.Name(): "bbbb",
				},
			},
			0,
			false,
		},
		{
			"-pingErr",
			expect{true},
			fields{
				pingErr: errors.New("fail"),
				dsn:     "pigeon://aaaa:cccc@uri:1433?app+name=Zabbix+agent+2+MSSQL+plugin&keepAlive=0", //nolint:lll
			},
			args{
				metricParams: map[string]string{
					params.URI.Name():      "pigeon://uri",
					params.User.Name():     "aaaa",
					params.Password.Name(): "cccc",
				},
			},
			0,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest
			c := &ConnCollection{
				conns:      map[ConnConfig]*sql.DB{},
				logr:       log.New("test"),
				driverName: "testdriver",
			}

			db, m, err := sqlmock.NewWithDSN(
				tt.fields.dsn,
				sqlmock.MonitorPingsOption(true),
			)
			if err != nil {
				t.Fatalf("failed to open sqlmock: %s", err.Error())
			}

			mockDriver.driver = db.Driver()

			defer mockDriver.reset()

			m.ExpectPing().WillReturnError(tt.fields.getErr)

			if tt.expect.ping {
				m.ExpectPing().WillReturnError(tt.fields.pingErr)
			}

			got, err := c.PingHandler(tt.args.metricParams)
			if (err != nil) != tt.wantErr {
				t.Fatalf(
					"ConnCollection.PingHandler() error = %v, wantErr %v",
					err, tt.wantErr,
				)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatalf("ConnCollection.PingHandler() = %s", diff)
			}

			if err := m.ExpectationsWereMet(); err != nil {
				t.Fatalf(
					"ConnCollection.PingHandler() "+
						"expectations where not met: %s",
					err.Error(),
				)
			}
		})
	}
}

//nolint:paralleltest,tparallel
func TestConnCollection_Close(t *testing.T) {
	log.DefaultLogger = stdlog.New(os.Stdout, "", stdlog.LstdFlags)

	type fields struct {
		closeErr error
	}

	tests := []struct {
		name   string
		fields fields
	}{
		{"+valid", fields{}},
		{"-closeErr", fields{errors.New("fail")}},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest
			t.Parallel()

			db, m, err := sqlmock.New()
			if err != nil {
				t.Fatalf("failed to open sqlmock: %s", err.Error())
			}

			m.ExpectClose().WillReturnError(tt.fields.closeErr)

			c := &ConnCollection{
				conns: map[ConnConfig]*sql.DB{{}: db},
				logr:  log.New("test"),
			}
			c.Close()

			if err := m.ExpectationsWereMet(); err != nil {
				t.Fatalf("ConnCollection.Close() = %s", err.Error())
			}
		})
	}
}

//nolint:paralleltest
func TestConnCollection_get(t *testing.T) {
	log.DefaultLogger = stdlog.New(os.Stdout, "", stdlog.LstdFlags)

	type expect struct {
		newConn bool
	}

	type fields struct {
		conns      map[ConnConfig]*sql.DB
		dsn        string
		newConnErr error
		driverName string
	}

	type args struct {
		conf ConnConfig
	}

	tests := []struct {
		name         string
		expect       expect
		fields       fields
		args         args
		wantReceiver *ConnCollection
		wantNil      bool
		wantErr      bool
	}{
		{
			"+validExisting",
			expect{false},
			fields{
				conns: map[ConnConfig]*sql.DB{
					{URI: "pigeon://uri", User: "aaaa", Password: "bbbb"}: {},
				},
				driverName: "testdriver",
			},
			args{
				conf: ConnConfig{
					URI:      "pigeon://uri",
					User:     "aaaa",
					Password: "bbbb",
				},
			},
			&ConnCollection{
				conns: map[ConnConfig]*sql.DB{
					{URI: "pigeon://uri", User: "aaaa", Password: "bbbb"}: {},
				},
				driverName: "testdriver",
			},
			false,
			false,
		},
		{
			"+validNew",
			expect{true},
			fields{
				conns:      map[ConnConfig]*sql.DB{},
				dsn:        "pigeon://rrrr:tttt@uri:1433?app+name=Zabbix+agent+2+MSSQL+plugin&keepAlive=0", //nolint:lll
				driverName: "testdriver",
			},
			args{
				conf: ConnConfig{
					User:     "rrrr",
					Password: "tttt",
					URI:      "pigeon://uri",
				},
			},
			&ConnCollection{
				conns: map[ConnConfig]*sql.DB{
					{User: "rrrr", Password: "tttt", URI: "pigeon://uri"}: {},
				},
				driverName: "testdriver",
			},
			false,
			false,
		},
		{
			"+prevCons",
			expect{true},
			fields{
				conns: map[ConnConfig]*sql.DB{
					{}: {},
				},
				dsn:        "pigeon://jjjj:tttt@uri:1433?app+name=Zabbix+agent+2+MSSQL+plugin&keepAlive=0", //nolint:lll
				driverName: "testdriver",
			},
			args{
				conf: ConnConfig{
					User:     "jjjj",
					Password: "tttt",
					URI:      "pigeon://uri",
				},
			},
			&ConnCollection{
				conns: map[ConnConfig]*sql.DB{
					{}: {},
					{User: "jjjj", Password: "tttt", URI: "pigeon://uri"}: {},
				},
				driverName: "testdriver",
			},
			false,
			false,
		},
		{
			"-newConnErr",
			expect{true},
			fields{
				conns:      map[ConnConfig]*sql.DB{},
				dsn:        "pigeon://kkkk:tttt@uri:1433?app+name=Zabbix+agent+2+MSSQL+plugin&keepAlive=0", //nolint:lll
				newConnErr: errors.New("fail"),
				driverName: "testdriver",
			},
			args{
				conf: ConnConfig{
					User:     "kkkk",
					Password: "tttt",
					URI:      "pigeon://uri",
				},
			},
			&ConnCollection{
				conns:      map[ConnConfig]*sql.DB{},
				driverName: "testdriver",
			},
			true,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest
			var (
				db  *sql.DB
				m   sqlmock.Sqlmock
				err error
			)

			if tt.expect.newConn {
				db, m, err = sqlmock.NewWithDSN(
					tt.fields.dsn,
					sqlmock.MonitorPingsOption(true),
				)
				if err != nil {
					t.Fatalf("failed to open sqlmock: %s", err.Error())
				}

				mockDriver.driver = db.Driver()

				defer mockDriver.reset()

				m.ExpectPing().WillReturnError(tt.fields.newConnErr)
			}

			c := &ConnCollection{
				conns:      tt.fields.conns,
				driverName: tt.fields.driverName,
				logr:       log.New("test"),
			}

			got, err := c.get(tt.args.conf)
			if (err != nil) != tt.wantErr {
				t.Fatalf(
					"ConnCollection.get() error = %v, wantErr %v",
					err, tt.wantErr,
				)
			}

			if (got == nil) != tt.wantNil {
				t.Fatalf(
					"ConnCollection.get() got = %v, wantNil %v",
					got, tt.wantNil,
				)
			}

			if diff := cmp.Diff(
				tt.wantReceiver, c,
				cmp.AllowUnexported(ConnCollection{}, sync.Mutex{}),
				cmp.Comparer(
					func(x, y *sql.DB) bool {
						return (x == nil) == (y == nil)
					},
				),
				cmpopts.SortMaps(
					func(x, y ConnConfig) bool {
						return x.User < y.User
					},
				),
				cmpopts.IgnoreFields(ConnCollection{}, "logr"),
			); diff != "" {
				t.Fatalf("ConnCollection.get() = %s", diff)
			}
			if m != nil {
				if err := m.ExpectationsWereMet(); err != nil {
					t.Fatalf("ConnCollection.get() = %s", err.Error())
				}
			}
		},
		)
	}
}

//nolint:paralleltest
func TestConnCollection_newConn(t *testing.T) {
	log.DefaultLogger = stdlog.New(os.Stdout, "", stdlog.LstdFlags)

	type expect struct {
		open bool
		ping bool
	}

	type fields struct {
		keepAlive     int
		openErr       error
		pingErr       error
		dsn           string
		driverName    string
		defaultScheme string
	}

	type args struct {
		conf *ConnConfig
	}

	tests := []struct {
		name    string
		expect  expect
		fields  fields
		args    args
		wantNil bool
		wantErr bool
	}{
		{
			"+valid",
			expect{true, true},
			fields{
				keepAlive:  4,
				dsn:        "pigeon://aaaa:bbbb@uri:1433?app+name=Zabbix+agent+2+MSSQL+plugin&keepAlive=4", //nolint:lll
				driverName: "testdriver",
			},
			args{&ConnConfig{
				User:     "aaaa",
				Password: "bbbb",
				URI:      "pigeon://uri",
			}},
			false,
			false,
		},
		{
			"+validWithTLS",
			expect{true, true},
			fields{
				keepAlive: 4,
				dsn: "pigeon://aaaa:bbbb@uri:1433?" +
					"TrustServerCertificate=false&" +
					"app+name=Zabbix+agent+2+MSSQL+plugin&" +
					"certificate=%2Fa%2Fb%2Fc&" +
					"encrypt=true&" +
					"hostNameInCertificate=server&" +
					"keepAlive=4&" +
					"tlsMinVersion=1.3",
				driverName: "testdriver",
			},
			args{&ConnConfig{
				User:                   "aaaa",
				Password:               "bbbb",
				URI:                    "pigeon://uri",
				CACertPath:             "/a/b/c",
				TrustServerCertificate: "false",
				HostNameInCertificate:  "server",
				Encrypt:                "true",
				TLSMinVersion:          "1.3",
			}},
			false,
			false,
		},
		{
			"-newConnURIErr",
			expect{false, false},
			fields{
				keepAlive:  4,
				openErr:    nil,
				driverName: "testdriver",
			},
			args{&ConnConfig{
				User:     "aaaa",
				Password: "bbbb",
				URI:      "://",
			}},
			true,
			true,
		},
		{
			"-openErr",
			expect{true, false},
			fields{
				keepAlive:  4,
				openErr:    errors.New("fail"),
				dsn:        "pigeon://cccc:bbbb@uri:1433?app+name=Zabbix+agent+2+MSSQL+plugin&keepAlive=4", //nolint:lll
				driverName: "testdriver",
			},
			args{&ConnConfig{
				User:     "cccc",
				Password: "bbbb",
				URI:      "pigeon://uri",
			}},
			true,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) { //nolint:paralleltest
				if tt.fields.defaultScheme != "" {
					prevScheme := params.URIDefaults.Scheme

					defer func() {
						params.URIDefaults.Scheme = prevScheme
					}()

					params.URIDefaults.Scheme = tt.fields.defaultScheme
				}

				var (
					db  *sql.DB
					m   sqlmock.Sqlmock
					err error
				)

				if tt.expect.open {
					db, m, err = sqlmock.NewWithDSN(
						tt.fields.dsn,
						sqlmock.MonitorPingsOption(true),
					)
					if err != nil {
						t.Fatalf("failed to open sqlmock: %s", err.Error())
					}

					mockDriver.openErr = tt.fields.openErr
					mockDriver.driver = db.Driver()

					defer mockDriver.reset()

					if tt.expect.ping {
						m.ExpectPing().WillReturnError(tt.fields.pingErr)
					}
				}

				c := &ConnCollection{
					keepAlive:  tt.fields.keepAlive,
					driverName: tt.fields.driverName,
					logr:       log.New("test"),
				}

				got, err := c.newConn(tt.args.conf)
				if (err != nil) != tt.wantErr {
					t.Fatalf(
						"ConnCollection.newConn() error = %v, wantErr %v",
						err, tt.wantErr,
					)
				}
				if (got == nil) != tt.wantNil {
					t.Fatalf(
						"ConnCollection.newConn() got = %v, wantNil %v",
						got, tt.wantNil,
					)
				}
				if m != nil {
					if err := m.ExpectationsWereMet(); err != nil {
						t.Fatalf(
							"ConnCollection.newConn() "+
								"expectations where not met: %s",
							err.Error(),
						)
					}
				}
			},
		)
	}
}
