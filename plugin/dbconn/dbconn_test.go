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
	"golang.zabbix.com/sdk/zbxsync"
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
		keepAlive  int
		logr       log.Logger
		driverName string
	}

	type args struct {
		keepAlive    int
		queryTimeout int
		logr         log.Logger
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		want   *ConnManager
	}{
		{
			"+valid",
			fields{},
			args{10, 11, sampleLogr},
			&ConnManager{
				conns:      &zbxsync.SyncMap[ConnConfig, *ConnItem]{},
				keepAlive:  10,
				logr:       sampleLogr,
				driverName: "sqlserver",
			},
		},
		{
			"-overwrite",
			fields{
				keepAlive:  3,
				logr:       log.New("aaa"),
				driverName: "lol",
			},
			args{10, 11, sampleLogr},
			&ConnManager{
				conns:      &zbxsync.SyncMap[ConnConfig, *ConnItem]{},
				keepAlive:  10,
				logr:       sampleLogr,
				driverName: "sqlserver",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := &ConnManager{
				keepAlive:  tt.fields.keepAlive,
				logr:       tt.fields.logr,
				driverName: tt.fields.driverName,
			}
			c.Init(tt.args.keepAlive, tt.args.logr)

			if diff := cmp.Diff(
				tt.want, c,
				cmp.AllowUnexported(ConnManager{}, ConnConfig{}, ConnItem{}),
				cmpopts.IgnoreUnexported(zbxsync.SyncMap[ConnConfig, *ConnItem]{}),
			); diff != "" {
				t.Fatalf("ConnManager.Init() = %s", diff)
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
		timeout      int
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
				timeout: 10,
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
				timeout: 10,
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
				timeout: 10,
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
			c := &ConnManager{
				conns:      &zbxsync.SyncMap[ConnConfig, *ConnItem]{},
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

			ctx, cancel := context.WithTimeout(t.Context(), time.Duration(tt.args.timeout)*time.Second)
			defer cancel()

			got, err := c.WithConnHandlerFunc(
				func(
					ctx context.Context,
					db *sql.DB,
					metricParams map[string]string,
					extraParams ...string,
				) (any, error) {
					if db == nil {
						t.Fatal(
							"ConnManager.WithConnHandlerFunc() db is nil",
						)
					}

					if diff := cmp.Diff(
						tt.wantMetricParams, metricParams,
					); diff != "" {
						t.Fatalf(
							"ConnManager.WithConnHandlerFunc() = %s", diff,
						)
					}

					if diff := cmp.Diff(
						tt.wantExtraParams, extraParams,
					); diff != "" {
						t.Fatalf(
							"ConnManager.WithConnHandlerFunc() = %s", diff,
						)
					}

					return "handler called", nil
				},
			)(ctx, 0, tt.args.metricParams, tt.args.extraParams...)
			if (err != nil) != tt.wantErr {
				t.Fatalf(
					"ConnManager.WithConnHandlerFunc() "+
						"error = %v, wantErr %v",
					err, tt.wantErr,
				)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatalf("ConnManager.WithConnHandlerFunc() = %s", diff)
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
		timeout      int
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
				timeout: 10,
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
				timeout: 10,
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
				timeout: 10,
			},
			0,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest
			c := &ConnManager{
				conns:      &zbxsync.SyncMap[ConnConfig, *ConnItem]{},
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

			ctx, cancel := context.WithTimeout(t.Context(), time.Duration(tt.args.timeout)*time.Second)
			defer cancel()

			got, err := c.PingHandler(ctx, 0, tt.args.metricParams)
			if (err != nil) != tt.wantErr {
				t.Fatalf(
					"ConnManager.PingHandler() error = %v, wantErr %v",
					err, tt.wantErr,
				)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatalf("ConnManager.PingHandler() = %s", diff)
			}

			if err := m.ExpectationsWereMet(); err != nil {
				t.Fatalf(
					"ConnManager.PingHandler() "+
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

			conf := ConnConfig{URI: "uri"}
			logr := log.New("test")

			item := newConnItem(0, logr, "")
			item.db = db

			conns := &zbxsync.SyncMap[ConnConfig, *ConnItem]{}
			conns.Store(conf, item)

			c := &ConnManager{
				conns: conns,
				logr:  logr,
			}
			c.Close()

			if err := m.ExpectationsWereMet(); err != nil {
				t.Fatalf("ConnManager.Close() = %s", err.Error())
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
		conns      *zbxsync.SyncMap[ConnConfig, *ConnItem]
		dsn        string
		newConnErr error
		driverName string
	}

	type args struct {
		conf *ConnConfig
	}

	tests := []struct {
		name         string
		expect       expect
		fields       fields
		args         args
		wantReceiver *ConnManager
		wantNil      bool
		wantErr      bool
	}{
		{
			"+validExisting",
			expect{false},
			fields{
				conns: func() *zbxsync.SyncMap[ConnConfig, *ConnItem] {
					conf := ConnConfig{URI: "pigeon://uri", User: "aaaa", Password: "bbbb"}
					m := &zbxsync.SyncMap[ConnConfig, *ConnItem]{}
					m.Store(conf, &ConnItem{db: &sql.DB{}})

					return m
				}(),
				driverName: "testdriver",
			},
			args{
				conf: &ConnConfig{
					URI:      "pigeon://uri",
					User:     "aaaa",
					Password: "bbbb",
				},
			},
			&ConnManager{
				conns: func() *zbxsync.SyncMap[ConnConfig, *ConnItem] {
					conf := ConnConfig{}
					m := &zbxsync.SyncMap[ConnConfig, *ConnItem]{}
					m.Store(conf, &ConnItem{db: &sql.DB{}})

					return m
				}(),
				driverName: "testdriver",
			},
			false,
			false,
		},
		{
			"+validNew",
			expect{true},
			fields{
				conns:      &zbxsync.SyncMap[ConnConfig, *ConnItem]{},
				dsn:        "pigeon://rrrr:tttt@uri:1433?app+name=Zabbix+agent+2+MSSQL+plugin&keepAlive=0",
				driverName: "testdriver",
			},
			args{
				conf: &ConnConfig{
					User:     "rrrr",
					Password: "tttt",
					URI:      "pigeon://uri",
				},
			},
			&ConnManager{
				conns: func() *zbxsync.SyncMap[ConnConfig, *ConnItem] {
					conf := ConnConfig{
						User:     "rrrr",
						Password: "tttt",
						URI:      "pigeon://uri",
					}
					m := &zbxsync.SyncMap[ConnConfig, *ConnItem]{}
					m.Store(conf, &ConnItem{db: &sql.DB{}})

					return m
				}(),
				driverName: "testdriver",
			},
			false,
			false,
		},
		{
			"+prevCons",
			expect{true},
			fields{
				conns: func() *zbxsync.SyncMap[ConnConfig, *ConnItem] {
					m := &zbxsync.SyncMap[ConnConfig, *ConnItem]{}
					m.Store(ConnConfig{}, &ConnItem{})

					return m
				}(),
				dsn:        "pigeon://jjjj:tttt@uri:1433?app+name=Zabbix+agent+2+MSSQL+plugin&keepAlive=0",
				driverName: "testdriver",
			},
			args{
				conf: &ConnConfig{
					User:     "jjjj",
					Password: "tttt",
					URI:      "pigeon://uri",
				},
			},
			&ConnManager{
				conns: func() *zbxsync.SyncMap[ConnConfig, *ConnItem] {
					m := &zbxsync.SyncMap[ConnConfig, *ConnItem]{}
					m.Store(ConnConfig{}, &ConnItem{})
					m.Store(
						ConnConfig{User: "rrrr", Password: "tttt", URI: "pigeon://uri"},
						&ConnItem{db: &sql.DB{}})

					return m
				}(),
				driverName: "testdriver",
			},
			false,
			false,
		},
		{
			"-newConnErr",
			expect{true},
			fields{
				conns:      &zbxsync.SyncMap[ConnConfig, *ConnItem]{},
				dsn:        "pigeon://kkkk:tttt@uri:1433?app+name=Zabbix+agent+2+MSSQL+plugin&keepAlive=0",
				newConnErr: errors.New("fail"),
				driverName: "testdriver",
			},
			args{
				conf: &ConnConfig{
					User:     "kkkk",
					Password: "tttt",
					URI:      "pigeon://uri",
				},
			},
			&ConnManager{
				conns:      &zbxsync.SyncMap[ConnConfig, *ConnItem]{},
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

			c := &ConnManager{
				conns:      tt.fields.conns,
				driverName: tt.fields.driverName,
				logr:       log.New("test"),
			}

			got, err := c.get(t.Context(), 0, tt.args.conf)
			if (err != nil) != tt.wantErr {
				t.Fatalf(
					"ConnManager.get() error = %v, wantErr %v",
					err, tt.wantErr,
				)
			}

			if (got == nil) != tt.wantNil {
				t.Fatalf(
					"ConnManager.get() got = %v, wantNil %v",
					got, tt.wantNil,
				)
			}

			if diff := cmp.Diff(
				tt.wantReceiver, c,
				cmp.AllowUnexported(ConnManager{}, ConnConfig{}, ConnItem{}),
				cmpopts.IgnoreUnexported(zbxsync.SyncMap[ConnConfig, *ConnItem]{}),
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
				cmpopts.IgnoreFields(ConnManager{}, "logr"),
			); diff != "" {
				t.Fatalf("ConnManager.get() = %s", diff)
			}

			if m != nil {
				if err := m.ExpectationsWereMet(); err != nil {
					t.Fatalf("ConnManager.get() = %s", err.Error())
				}
			}
		},
		)
	}
}

//nolint:paralleltest
func TestConnCollection_get_ConcurrentAccess(t *testing.T) {
	const goroutineCount = 10

	log.DefaultLogger = stdlog.New(os.Stdout, "", stdlog.LstdFlags)

	log.IncreaseLogLevel()
	log.IncreaseLogLevel()
	log.IncreaseLogLevel()
	log.IncreaseLogLevel()

	db, m, err := sqlmock.NewWithDSN(
		"pigeon://rrrr:tttt@concurrent:1433?app+name=Zabbix+agent+2+MSSQL+plugin&keepAlive=0",
		sqlmock.MonitorPingsOption(true),
	)
	if err != nil {
		t.Fatalf("failed to open sqlmock: %s", err.Error())
	}

	mockDriver.driver = db.Driver()
	defer mockDriver.reset()

	// Ping is done only after a connection is created.
	// As only a single connection should be opened, ping should be done only once.
	m.ExpectPing()

	ccol := &ConnManager{}
	ccol.Init(0, log.New("test"))
	ccol.driverName = "testdriver"

	var (
		wg sync.WaitGroup
	)

	//results := make([]*sql.DB, 0, goroutineCount)
	conf := &ConnConfig{
		URI:      "pigeon://concurrent",
		User:     "rrrr",
		Password: "tttt",
	}

	for range goroutineCount {
		wg.Go(func() {
			conn, err := ccol.get(t.Context(), 0, conf)
			if conn == nil {
				t.Errorf("unexpected error from get(): %v", err)

				return
			}
		})
	}

	wg.Wait()

	// Check that only one connection is stored
	connCnt := ccol.conns.Len()
	if connCnt != 1 {
		t.Errorf("expected 1 connection in map, got %d", connCnt)
	}
}
