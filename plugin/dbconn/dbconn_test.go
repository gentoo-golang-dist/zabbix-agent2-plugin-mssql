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
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"golang.zabbix.com/plugin/mssql/plugin/params"
	"golang.zabbix.com/sdk/log"
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
		conns      map[connConfig]*sql.DB
		keepAlive  int
		logr       log.Logger
		driverName string
	}

	type args struct {
		keepAlive     int
		queryTimetout int
		logr          log.Logger
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
			args{10, 11, sampleLogr},
			&ConnCollection{
				conns:      make(map[connConfig]*sql.DB),
				keepAlive:  10,
				logr:       sampleLogr,
				driverName: "sqlserver",
			},
		},
		{
			"-overwrite",
			fields{
				conns:      map[connConfig]*sql.DB{{}: nil},
				keepAlive:  3,
				logr:       log.New("aaa"),
				driverName: "lol",
			},
			args{10, 11, sampleLogr},
			&ConnCollection{
				conns:      make(map[connConfig]*sql.DB),
				keepAlive:  10,
				logr:       sampleLogr,
				driverName: "sqlserver",
			},
		},
	}
	for _, tt := range tests {
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
		timeout      time.Duration
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
				dsn: "pigeon://8888:dddd@uri:1433?app+name=Zabbix+agent+2+MSSQL+plugin&keepAlive=0",
			},
			args{
				timeout: time.Second * 10,
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
				dsn: "pigeon://7777:dddd@uri:1433?app+name=Zabbix+agent+2+MSSQL+plugin&keepAlive=0",
			},
			args{
				timeout: time.Second * 10,
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
				dsn:    "pigeon://6666:dddd@uri:1433?app+name=Zabbix+agent+2+MSSQL+plugin&keepAlive=0",
				getErr: errors.New("fail"),
			},
			args{
				timeout: time.Second * 10,
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
				conns:      map[connConfig]*sql.DB{},
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
					ctx context.Context,
					db *sql.DB,
					metricParams map[string]string,
					extraParams ...string,
				) (any, error) {
					deadline, deadlineSet := ctx.Deadline()
					if !deadlineSet {
						t.Fatal(
							"ConnCollection.WithConnHandlerFunc() context deadline is not set",
						)
					}

					ctxTimeout := time.Until(deadline).Round(time.Second)
					if ctxTimeout != tt.args.timeout {
						t.Fatalf(
							"ConnCollection.WithConnHandlerFunc() "+
								"context timeout is not correctly set: "+
								"got %v, expected %v", ctxTimeout, tt.args.timeout,
						)
					}

					if db == nil {
						t.Fatal(
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
			)(tt.args.timeout, tt.args.metricParams, tt.args.extraParams...)
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
		timeout      time.Duration
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
				dsn: "pigeon://aaaa:dddd@uri:1433?app+name=Zabbix+agent+2+MSSQL+plugin&keepAlive=0",
			},
			args{
				metricParams: map[string]string{
					params.URI.Name():      "pigeon://uri",
					params.User.Name():     "aaaa",
					params.Password.Name(): "dddd",
				},
				timeout: time.Second * 10,
			},
			1,
			false,
		},
		{
			"-getErr",
			expect{false},
			fields{
				getErr: errors.New("fail"),
				dsn:    "pigeon://aaaa:bbbb@uri:1433?app+name=Zabbix+agent+2+MSSQL+plugin&keepAlive=0",
			},
			args{
				metricParams: map[string]string{
					params.URI.Name():      "pigeon://uri",
					params.User.Name():     "aaaa",
					params.Password.Name(): "bbbb",
				},
				timeout: time.Second * 10,
			},
			0,
			false,
		},
		{
			"-pingErr",
			expect{true},
			fields{
				pingErr: errors.New("fail"),
				dsn:     "pigeon://aaaa:cccc@uri:1433?app+name=Zabbix+agent+2+MSSQL+plugin&keepAlive=0",
			},
			args{
				metricParams: map[string]string{
					params.URI.Name():      "pigeon://uri",
					params.User.Name():     "aaaa",
					params.Password.Name(): "cccc",
				},
				timeout: time.Second * 10,
			},
			0,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest
			c := &ConnCollection{
				conns:      map[connConfig]*sql.DB{},
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

			got, err := c.PingHandler(tt.args.timeout, tt.args.metricParams)
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
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest
			t.Parallel()

			db, m, err := sqlmock.New()
			if err != nil {
				t.Fatalf("failed to open sqlmock: %s", err.Error())
			}

			m.ExpectClose().WillReturnError(tt.fields.closeErr)

			c := &ConnCollection{
				conns: map[connConfig]*sql.DB{{}: db},
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
		conns      map[connConfig]*sql.DB
		dsn        string
		newConnErr error
		driverName string
	}

	type args struct {
		conf connConfig
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
				conns: map[connConfig]*sql.DB{
					{URI: "pigeon://uri", User: "aaaa", Password: "bbbb"}: {},
				},
				driverName: "testdriver",
			},
			args{
				conf: connConfig{
					URI:      "pigeon://uri",
					User:     "aaaa",
					Password: "bbbb",
				},
			},
			&ConnCollection{
				conns: map[connConfig]*sql.DB{
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
				conns:      map[connConfig]*sql.DB{},
				dsn:        "pigeon://rrrr:tttt@uri:1433?app+name=Zabbix+agent+2+MSSQL+plugin&keepAlive=0",
				driverName: "testdriver",
			},
			args{
				conf: connConfig{
					User:     "rrrr",
					Password: "tttt",
					URI:      "pigeon://uri",
				},
			},
			&ConnCollection{
				conns: map[connConfig]*sql.DB{
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
				conns: map[connConfig]*sql.DB{
					{}: {},
				},
				dsn:        "pigeon://jjjj:tttt@uri:1433?app+name=Zabbix+agent+2+MSSQL+plugin&keepAlive=0",
				driverName: "testdriver",
			},
			args{
				conf: connConfig{
					User:     "jjjj",
					Password: "tttt",
					URI:      "pigeon://uri",
				},
			},
			&ConnCollection{
				conns: map[connConfig]*sql.DB{
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
				conns:      map[connConfig]*sql.DB{},
				dsn:        "pigeon://kkkk:tttt@uri:1433?app+name=Zabbix+agent+2+MSSQL+plugin&keepAlive=0",
				newConnErr: errors.New("fail"),
				driverName: "testdriver",
			},
			args{
				conf: connConfig{
					User:     "kkkk",
					Password: "tttt",
					URI:      "pigeon://uri",
				},
			},
			&ConnCollection{
				conns:      map[connConfig]*sql.DB{},
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

			got, err := c.get(context.Background(), tt.args.conf)
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
					func(x, y connConfig) bool {
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
		conf *connConfig
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
				dsn:        "pigeon://aaaa:bbbb@uri:1433?app+name=Zabbix+agent+2+MSSQL+plugin&keepAlive=4",
				driverName: "testdriver",
			},
			args{&connConfig{
				User:     "aaaa",
				Password: "bbbb",
				URI:      "pigeon://uri",
			}},
			false,
			false,
		},
		{
			"+named",
			expect{true, true},
			fields{
				keepAlive:  4,
				dsn:        "pigeon://aaaa:bbbb@uri/InstanceName?app+name=Zabbix+agent+2+MSSQL+plugin&keepAlive=4",
				driverName: "testdriver",
			},
			args{&connConfig{
				User:     "aaaa",
				Password: "bbbb",
				URI:      "pigeon://uri/InstanceName",
			}},
			false,
			false,
		},
		{
			"+namedWithPort",
			expect{true, true},
			fields{
				keepAlive:  4,
				dsn:        "pigeon://aaaa:bbbb@uri:1435/InstanceName?app+name=Zabbix+agent+2+MSSQL+plugin&keepAlive=4",
				driverName: "testdriver",
			},
			args{&connConfig{
				User:     "aaaa",
				Password: "bbbb",
				URI:      "pigeon://uri:1435/InstanceName",
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
					"certificate=abc&" +
					"encrypt=true&" +
					"hostNameInCertificate=server&" +
					"keepAlive=4&" +
					"tlsMinVersion=1.3",
				driverName: "testdriver",
			},
			args{&connConfig{
				User:                   "aaaa",
				Password:               "bbbb",
				URI:                    "pigeon://uri",
				CACertPath:             "abc",
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
			args{&connConfig{
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
				dsn:        "pigeon://cccc:bbbb@uri:1433?app+name=Zabbix+agent+2+MSSQL+plugin&keepAlive=4",
				driverName: "testdriver",
			},
			args{&connConfig{
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

				got, err := c.newConn(context.Background(), tt.args.conf)
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

func Test_newConnConfig(t *testing.T) {
	t.Parallel()

	type args struct {
		metricParams map[string]string
	}

	tests := []struct {
		name string
		args args
		want connConfig
	}{
		{
			"+valid",
			args{
				map[string]string{
					"URI":                    "pigeon://uri",
					"User":                   "aaaa",
					"Password":               "bbbb",
					"CACertPath":             "/a/b/c",
					"TrustServerCertificate": "false",
					"HostNameInCertificate":  "true",
					"Encrypt":                "false",
					"TLSMinVersion":          "1.2",
					"Database":               "testdb",
				},
			},
			connConfig{
				URI:                    "pigeon://uri",
				User:                   "aaaa",
				Password:               "bbbb",
				CACertPath:             "/a/b/c",
				TrustServerCertificate: "false",
				HostNameInCertificate:  "true",
				Encrypt:                "false",
				TLSMinVersion:          "1.2",
				Database:               "testdb",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := newConnConfig(tt.args.metricParams)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatalf("newConnConfig() = %s", diff)
			}
		})
	}
}
