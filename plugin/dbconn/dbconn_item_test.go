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
	"errors"
	stdlog "log"
	"net/url"
	"os"
	"sync"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"golang.zabbix.com/plugin/mssql/plugin/params"
	"golang.zabbix.com/sdk/log"
)

func TestConnItem_newConnItem(t *testing.T) {
	t.Parallel()

	sampleLogr := &struct{ log.Logger }{}

	type fields struct {
		db *sql.DB

		keepAlive  int
		logr       log.Logger
		driverName string
	}

	type args struct {
		keepAlive  int
		logr       log.Logger
		driverName string
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			"+valid",
			fields{
				db:         nil,
				keepAlive:  10,
				logr:       sampleLogr,
				driverName: "test",
			},
			args{
				keepAlive:  10,
				logr:       sampleLogr,
				driverName: "test",
			},
			true,
		},
		{
			"+overwriteFields",
			fields{
				db:         &sql.DB{},
				keepAlive:  99,
				logr:       nil,
				driverName: "test3",
			},
			args{
				keepAlive:  20,
				logr:       sampleLogr,
				driverName: "test2",
			},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			want := &ConnItem{
				db:         tt.fields.db,
				keepAlive:  tt.fields.keepAlive,
				logr:       tt.fields.logr,
				driverName: tt.fields.driverName,
			}

			got := newConnItem(tt.args.keepAlive, tt.args.logr, tt.args.driverName)
			if got == nil {
				t.Fatal("ConnItem.newConnItem() returned nil")
			}

			diff := cmp.Diff(got, want,
				cmp.AllowUnexported(ConnItem{}),
				cmpopts.IgnoreUnexported(sync.Mutex{}),
				cmp.Comparer(func(x, y log.Logger) bool { return x == y }),
			)
			if diff != "" && tt.wantErr {
				t.Errorf("ConnItem.newConnItem() = %s", diff)
			}
		})
	}
}

func TestConnItem_newConnConfig(t *testing.T) {
	t.Parallel()

	type args struct {
		metricParams map[string]string
	}

	tests := []struct {
		name string
		args args
		want ConnConfig
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
			ConnConfig{
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
			if diff := cmp.Diff(tt.want, *got); diff != "" {
				t.Fatalf("newConnConfig() = %s", diff)
			}
		})
	}
}

//nolint:paralleltest
func TestConnItem_initDB(t *testing.T) {
	log.DefaultLogger = stdlog.New(os.Stdout, "", stdlog.LstdFlags)

	type fields struct {
		keepAlive     int
		openErr       error
		pingErr       error
		dsn           string
		driverName    string
		defaultScheme string
	}

	type args struct {
		ctx  context.Context //nolint:containedctx
		conf *ConnConfig
	}

	type expect struct {
		open bool
		ping bool
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		expect  expect
		wantErr bool
	}{
		{
			"+valid",
			fields{
				keepAlive:  4,
				dsn:        "pigeon://aaaa:bbbb@uri:1433?app+name=Zabbix+agent+2+MSSQL+plugin&keepAlive=4",
				driverName: "testdriver",
			},
			args{
				t.Context(),
				newConnConfig(map[string]string{
					"User":     "aaaa",
					"Password": "bbbb",
					"URI":      "pigeon://uri",
				}),
			},
			expect{true, true},
			false,
		},
		{
			"+named",
			fields{
				keepAlive:  4,
				dsn:        "pigeon://aaaa:bbbb@uri/InstanceName?app+name=Zabbix+agent+2+MSSQL+plugin&keepAlive=4",
				driverName: "testdriver",
			},
			args{
				t.Context(),
				newConnConfig(map[string]string{
					"User":     "aaaa",
					"Password": "bbbb",
					"URI":      "pigeon://uri/InstanceName",
				}),
			},
			expect{true, true},
			false,
		},
		{
			"+namedWithPort",
			fields{
				keepAlive:  4,
				dsn:        "pigeon://aaaa:bbbb@uri:1435/InstanceName?app+name=Zabbix+agent+2+MSSQL+plugin&keepAlive=4",
				driverName: "testdriver",
			},
			args{
				t.Context(),
				newConnConfig(map[string]string{
					"User":     "aaaa",
					"Password": "bbbb",
					"URI":      "pigeon://uri:1435/InstanceName",
				}),
			},
			expect{true, true},
			false,
		},
		{
			"-newConnURIErr",
			fields{
				keepAlive:  4,
				openErr:    nil,
				driverName: "testdriver",
			},
			args{
				t.Context(),
				newConnConfig(map[string]string{
					"User":     "aaaa",
					"Password": "bbbb",
					"URI":      "://",
				}),
			},
			expect{false, false},
			true,
		},
		{
			"-openErr",
			fields{
				keepAlive:  4,
				openErr:    errors.New("fail"),
				dsn:        "pigeon://cccc:bbbb@uri:1433?app+name=Zabbix+agent+2+MSSQL+plugin&keepAlive=4",
				driverName: "testdriver",
			},
			args{
				t.Context(),
				newConnConfig(map[string]string{
					"User":     "cccc",
					"Password": "bbbb",
					"URI":      "pigeon://uri",
				}),
			},
			expect{true, false},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			item := newConnItem(
				tt.fields.keepAlive,
				log.New("test"),
				tt.fields.driverName,
			)

			_, err = item.getDbConn(t.Context(), tt.args.conf)
			if (err != nil) != tt.wantErr {
				t.Fatalf(
					"ConnItem.getDbConn() error = %v, wantErr %v",
					err, tt.wantErr,
				)
			}

			if m != nil {
				if err := m.ExpectationsWereMet(); err != nil {
					t.Fatalf(
						"ConnItem.getDbConn() expectations where not met: %s",
						err.Error(),
					)
				}
			}
		},
		)
	}
}

func TestConnItem_composeQueryParams(t *testing.T) {
	t.Parallel()

	type fields struct {
		queryParams map[string]string
	}

	type args struct {
		conf *ConnConfig
	}

	type expect struct {
		queryParams map[string]string
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		expect  expect
		wantErr bool
	}{
		{
			"+emptyInitial",
			fields{},
			args{
				newConnConfig(map[string]string{
					"CACertPath":             "cacert",
					"TrustServerCertificate": "trcert",
					"HostNameInCertificate":  "hostcert",
					"Encrypt":                "encr",
					"TLSMinVersion":          "1.2",
					"Database":               "db",
				}),
			},
			expect{map[string]string{
				"app name":               "Zabbix agent 2 MSSQL plugin",
				"keepAlive":              "10",
				"certificate":            "cacert",
				"TrustServerCertificate": "trcert",
				"hostNameInCertificate":  "hostcert",
				"encrypt":                "encr",
				"tlsMinVersion":          "1.2",
				"database":               "db",
			}},
			false,
		},
		{
			"+preset",
			fields{map[string]string{
				"certificate":            "cacert",
				"TrustServerCertificate": "trcert",
				"hostNameInCertificate":  "hostcert",
				"encrypt":                "encr",
				"tlsMinVersion":          "1.2",
				"database":               "db",
			}},
			args{
				newConnConfig(map[string]string{
					"CACertPath":             "",
					"TrustServerCertificate": "",
					"HostNameInCertificate":  "",
					"Encrypt":                "",
					"TLSMinVersion":          "",
					"Database":               "",
				}),
			},
			expect{map[string]string{
				"app name":               "Zabbix agent 2 MSSQL plugin",
				"keepAlive":              "10",
				"certificate":            "cacert",
				"TrustServerCertificate": "trcert",
				"hostNameInCertificate":  "hostcert",
				"encrypt":                "encr",
				"tlsMinVersion":          "1.2",
				"database":               "db",
			}},
			false,
		},
		{
			"+differs",
			fields{map[string]string{
				"certificate":            "aaa",
				"TrustServerCertificate": "trcert",
				"hostNameInCertificate":  "hostcert",
				"encrypt":                "encr",
				"tlsMinVersion":          "1.2",
				"database":               "db",
			}},
			args{
				newConnConfig(map[string]string{
					"CACertPath":             "",
					"TrustServerCertificate": "",
					"HostNameInCertificate":  "",
					"Encrypt":                "",
					"TLSMinVersion":          "",
					"Database":               "",
				}),
			},
			expect{map[string]string{
				"app name":               "Zabbix agent 2 MSSQL plugin",
				"keepAlive":              "10",
				"certificate":            "cacert",
				"TrustServerCertificate": "trcert",
				"hostNameInCertificate":  "hostcert",
				"encrypt":                "encr",
				"tlsMinVersion":          "1.2",
				"database":               "db",
			}},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			item := newConnItem(10, nil, "")

			u, err := url.Parse("pigeon://uri")
			if err != nil {
				t.Fatalf("failed to parse URI: %v", err)
			}

			gotQueryParams := u.Query()
			for k, v := range tt.fields.queryParams {
				gotQueryParams.Add(k, v)
			}

			item.composeQueryParams(gotQueryParams, tt.args.conf)

			wantQueryParams := url.Values{}
			for k, v := range tt.expect.queryParams {
				wantQueryParams.Add(k, v)
			}

			diff := cmp.Diff(gotQueryParams, wantQueryParams)

			if diff != "" && !tt.wantErr {
				t.Errorf("TestConnItem_composeQueryParams() = %s", diff)
			}
		})
	}
}
