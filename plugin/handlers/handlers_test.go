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

package handlers

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	stdlog "log"
	"os"
	"testing"
	"testing/fstest"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/go-cmp/cmp"
	mssql "github.com/microsoft/go-mssqldb"
	"golang.zabbix.com/sdk/log"
)

var (
	_ fs.FS   = (*mockFS)(nil)
	_ fs.File = (*mockFile)(nil)
)

type mockFS struct {
	fileMaps fstest.MapFS
	globErr  error
	openErr  error
	readErr  error
}

type mockFile struct {
	fs.File
	err error
}

func (m *mockFile) Read(p []byte) (int, error) {
	if m.err != nil {
		return 0, m.err
	}

	return m.File.Read(p)
}

func (m *mockFS) Open(name string) (fs.File, error) {
	if m.openErr != nil {
		return nil, m.openErr
	}

	f, err := m.fileMaps.Open(name)
	if err != nil {
		return nil, err
	}

	if m.readErr != nil {
		return &mockFile{f, m.readErr}, nil //nolint:nilerr
	}

	return f, nil
}

func (m *mockFS) Glob(pattern string) ([]string, error) {
	if m.globErr != nil {
		return nil, m.globErr
	}

	return m.fileMaps.Glob(pattern)
}

func Test_nullUniqueIdentifier_Scan(t *testing.T) {
	t.Parallel()

	type fields struct {
		UUID  *mssql.UniqueIdentifier
		Valid bool
	}

	type args struct {
		value any
	}

	tests := []struct {
		name     string
		fields   fields
		args     args
		wantNUID *nullUniqueIdentifier
		wantErr  bool
	}{
		{
			"+valid",
			fields{},
			args{make([]byte, 16)},
			&nullUniqueIdentifier{
				uuid:  &mssql.UniqueIdentifier{},
				valid: true,
			},
			false,
		},
		{
			"-prevValues",
			fields{
				UUID: &mssql.UniqueIdentifier{
					1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16,
				},
			},
			args{make([]byte, 16)},
			&nullUniqueIdentifier{
				uuid:  &mssql.UniqueIdentifier{},
				valid: true,
			},
			false,
		},
		{
			"-nilValue",
			fields{},
			args{},
			&nullUniqueIdentifier{},
			false,
		},
		{
			"-scanErr",
			fields{},
			args{make([]byte, 1)},
			&nullUniqueIdentifier{
				uuid: &mssql.UniqueIdentifier{},
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			nuid := &nullUniqueIdentifier{
				uuid:  tt.fields.UUID,
				valid: tt.fields.Valid,
			}

			if err := nuid.Scan(tt.args.value); (err != nil) != tt.wantErr {
				t.Fatalf(
					"nullUniqueIdentifier.Scan() error = %v, wantErr %v",
					err, tt.wantErr,
				)
			}

			if diff := cmp.Diff(
				tt.wantNUID, nuid,
				cmp.AllowUnexported(nullUniqueIdentifier{}),
			); diff != "" {
				t.Fatalf(
					"nullUniqueIdentifier.Scan() mismatch (+want -got):\n%s",
					diff,
				)
			}
		})
	}
}

func Test_nullUniqueIdentifier_Value(t *testing.T) {
	t.Parallel()

	type fields struct {
		UUID  *mssql.UniqueIdentifier
		Valid bool
	}

	tests := []struct {
		name    string
		fields  fields
		want    driver.Value
		wantErr bool
	}{
		{
			"+valid",
			fields{
				UUID: &mssql.UniqueIdentifier{
					1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 12, 13, 14, 15, 16,
				},
				Valid: true,
			},
			"01020304-0506-0708-090A-0C0D0E0F1000",
			false,
		},
		{
			"-invalid",
			fields{
				UUID: &mssql.UniqueIdentifier{
					1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 12, 13, 14, 15, 16,
				},
				Valid: false,
			},
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			nuid := nullUniqueIdentifier{
				uuid:  tt.fields.UUID,
				valid: tt.fields.Valid,
			}

			got, err := nuid.Value()
			if (err != nil) != tt.wantErr {
				t.Fatalf(
					"nullUniqueIdentifier.Value() error = %v, wantErr %v",
					err,
					tt.wantErr,
				)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatalf("nullUniqueIdentifier.Value() = %s", diff)
			}
		})
	}
}

func Test_nullBool_Value(t *testing.T) {
	t.Parallel()

	type fields struct {
		NullBool sql.NullBool
	}

	tests := []struct {
		name    string
		fields  fields
		want    driver.Value
		wantErr bool
	}{
		{
			"+validTrue",
			fields{NullBool: sql.NullBool{Bool: true, Valid: true}},
			int64(1),
			false,
		},
		{
			"+validFalse",
			fields{NullBool: sql.NullBool{Bool: false, Valid: true}},
			int64(0),
			false,
		},
		{
			"-invalid",
			fields{NullBool: sql.NullBool{}},
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			b := nullBool{
				NullBool: tt.fields.NullBool,
			}

			got, err := b.Value()
			if (err != nil) != tt.wantErr {
				t.Fatalf(
					"nullBool.Value() error = %v, wantErr %v",
					err,
					tt.wantErr,
				)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatalf("nullBool.Value() = %s", diff)
			}
		})
	}
}

func TestWithJSONResponse(t *testing.T) {
	newHandlerFunc := func(handlerErr error, invalidJSON bool) HandlerFunc {
		return func(
			metricParams map[string]string, extraParams ...string,
		) (any, error) {
			if handlerErr != nil {
				return nil, handlerErr
			}

			if diff := cmp.Diff(
				map[string]string{
					"param1": "value1",
					"param2": "value2",
				},
				metricParams,
			); diff != "" {
				t.Fatalf("metricParams mismatch (+want -got):\n%s", diff)
			}

			if diff := cmp.Diff(
				[]string{"extra1", "extra2", "extra3"},
				extraParams,
			); diff != "" {
				t.Fatalf("extraParams mismatch (+want -got):\n%s", diff)
			}

			if invalidJSON {
				return time.Unix(100000000000000000, 0), nil
			}

			return map[string]any{"a": 1, "b": 2, "c": "3"}, nil
		}
	}

	t.Parallel()

	type args struct {
		handler     HandlerFunc
		params      map[string]string
		extraParams []string
	}

	tests := []struct {
		name    string
		args    args
		want    any
		wantErr bool
	}{
		{
			"+valid",
			args{
				handler: newHandlerFunc(nil, false),
				params: map[string]string{
					"param1": "value1",
					"param2": "value2",
				},
				extraParams: []string{"extra1", "extra2", "extra3"},
			},
			`{"a":1,"b":2,"c":"3"}`,
			false,
		},
		{
			"-handlerErr",
			args{
				handler: newHandlerFunc(errors.New("fail"), false),
				params: map[string]string{
					"param1": "value1",
					"param2": "value2",
				},
				extraParams: []string{"extra1", "extra2", "extra3"},
			},
			nil,
			true,
		},
		{
			"-marshalErr",
			args{
				handler: newHandlerFunc(nil, true),
				params: map[string]string{
					"param1": "value1",
					"param2": "value2",
				},
				extraParams: []string{"extra1", "extra2", "extra3"},
			},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := WithJSONResponse(
				tt.args.handler,
			)(
				tt.args.params,
				tt.args.extraParams...,
			)
			if (err != nil) != tt.wantErr {
				t.Fatalf(
					"WithJSONResponse() error = %v, wantErr %v",
					err, tt.wantErr,
				)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatalf("WithJSONResponse() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

//nolint:paralleltest,tparallel
func TestCustomQueries_Load(t *testing.T) {
	log.DefaultLogger = stdlog.New(os.Stdout, "", stdlog.LstdFlags)

	type args struct {
		customQueriesDirFS fs.FS
	}

	tests := []struct {
		name    string
		args    args
		cq      CustomQueries
		want    CustomQueries
		wantErr bool
	}{
		{
			"+valid",
			args{&mockFS{
				fileMaps: fstest.MapFS{
					"test.sql": &fstest.MapFile{
						Data: []byte("SELECT A, B FROM C"),
					},
					"test2.sql": &fstest.MapFile{
						Data: []byte("SELECT D, E FROM F"),
					},
				},
			}},
			CustomQueries{},
			CustomQueries{
				"test":  "SELECT A, B FROM C",
				"test2": "SELECT D, E FROM F",
			},
			false,
		},
		{
			"+prevQueries",
			args{&mockFS{
				fileMaps: fstest.MapFS{
					"test.sql": &fstest.MapFile{
						Data: []byte("SELECT A, B FROM C"),
					},
					"test2.sql": &fstest.MapFile{
						Data: []byte("SELECT D, E FROM F"),
					},
				},
			}},
			CustomQueries{
				"test3": "aaaa",
				"test4": "bbbbb",
			},
			CustomQueries{
				"test":  "SELECT A, B FROM C",
				"test2": "SELECT D, E FROM F",
				"test3": "aaaa",
				"test4": "bbbbb",
			},
			false,
		},
		{
			"-overwrite",
			args{&mockFS{
				fileMaps: fstest.MapFS{
					"test.sql": &fstest.MapFile{
						Data: []byte("SELECT A, B FROM C"),
					},
					"test2.sql": &fstest.MapFile{
						Data: []byte("SELECT D, E FROM F"),
					},
				},
			}},
			CustomQueries{
				"test":  "aaaa",
				"test2": "bbbb",
			},
			CustomQueries{
				"test":  "SELECT A, B FROM C",
				"test2": "SELECT D, E FROM F",
			},
			false,
		},

		{
			"-noCustomQueries",
			args{&mockFS{fileMaps: fstest.MapFS{}}},
			CustomQueries{},
			CustomQueries{},
			false,
		},
		{
			"-globErr",
			args{&mockFS{
				globErr: errors.New("fail"),
				fileMaps: fstest.MapFS{
					"test.sql": &fstest.MapFile{
						Data: []byte("SELECT A, B FROM C"),
					},
					"test2.sql": &fstest.MapFile{
						Data: []byte("SELECT D, E FROM F"),
					},
				},
			}},
			CustomQueries{},
			CustomQueries{},
			true,
		},
		{
			"-openErr",
			args{&mockFS{
				openErr: errors.New("fail"),
				fileMaps: fstest.MapFS{
					"test.sql": &fstest.MapFile{
						Data: []byte("SELECT A, B FROM C"),
					},
					"test2.sql": &fstest.MapFile{
						Data: []byte("SELECT D, E FROM F"),
					},
				},
			}},
			CustomQueries{},
			CustomQueries{},
			true,
		},
		{
			"-readAllErr",
			args{&mockFS{
				readErr: errors.New("fail"),
				fileMaps: fstest.MapFS{
					"test.sql": &fstest.MapFile{
						Data: []byte("SELECT A, B FROM C"),
					},
					"test2.sql": &fstest.MapFile{
						Data: []byte("SELECT D, E FROM F"),
					},
				},
			}},
			CustomQueries{},
			CustomQueries{},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.cq.Load(tt.args.customQueriesDirFS, log.New("test"))
			if (err != nil) != tt.wantErr {
				t.Fatalf(
					"CustomQueries.Load() error = %v, wantErr %v",
					err,
					tt.wantErr,
				)
			}

			if diff := cmp.Diff(tt.want, tt.cq); diff != "" {
				t.Fatalf("CustomQueries.Load() = %s", diff)
			}
		})
	}
}

func TestCustomQueries_HandlerFunc(t *testing.T) {
	t.Parallel()

	type expect struct {
		query bool
	}

	type args struct {
		metricParams map[string]string
		extraParams  []string
	}

	type fields struct {
		query    string
		args     []driver.Value
		queryErr error
	}

	tests := []struct {
		name    string
		expect  expect
		args    args
		fields  fields
		cq      CustomQueries
		want    string
		wantErr bool
	}{
		{
			"+valid",
			expect{true},
			args{metricParams: map[string]string{"QueryName": "test"}},
			fields{query: "SELECT A, B FROM C"},
			map[string]string{"test": "SELECT A, B FROM C"},
			`[{"a":1,"b":2},{"a":3,"b":4}]`,
			false,
		},
		{
			"+validWithArgs",
			expect{true},
			args{
				metricParams: map[string]string{"QueryName": "test"},
				extraParams:  []string{"10"},
			},
			fields{
				query: "SELECT A, B FROM C WHERE A = @p1",
				args:  []driver.Value{"10"},
			},
			map[string]string{"test": "SELECT A, B FROM C WHERE A = @p1"},
			`[{"a":1,"b":2},{"a":3,"b":4}]`,
			false,
		},
		{
			"-customQueryNotFound",
			expect{false},
			args{metricParams: map[string]string{"QueryName": "test"}},
			fields{query: "SELECT A, B FROM C"},
			map[string]string{},
			"",
			true,
		},
		{
			"-QueryHandlerFuncErr",
			expect{true},
			args{metricParams: map[string]string{"QueryName": "test"}},
			fields{query: "SELECT A, B FROM C", queryErr: errors.New("fail")},
			map[string]string{"test": "SELECT A, B FROM C"},
			"",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db, m, err := sqlmock.New()
			if err != nil {
				t.Fatalf("failed to create mock DB: %s", err)
			}

			if tt.expect.query {
				m.ExpectQuery(fmt.Sprintf("^%s$", tt.fields.query)).
					WithArgs(tt.fields.args...).
					WillReturnRows(
						sqlmock.NewRows([]string{"a", "b"}).
							AddRow(1, 2).
							AddRow(3, 4),
					).
					WillReturnError(tt.fields.queryErr)
			}

			got, err := tt.cq.HandlerFunc(
				context.Background(),
				db,
				tt.args.metricParams,
				tt.args.extraParams...)
			if (err != nil) != tt.wantErr {
				t.Fatalf(
					"CustomQueries.HandlerFunc() error = %v, wantErr %v",
					err, tt.wantErr,
				)
			}

			if err == nil {
				b, err := json.Marshal(got)
				if err != nil {
					t.Fatalf(
						"CustomQueries.HandlerFunc() "+
							"failed to marshal handler result: %v",
						err,
					)
				}

				if diff := cmp.Diff(tt.want, string(b)); diff != "" {
					t.Fatalf("CustomQueries.HandlerFunc() = %s", diff)
				}
			}

			if err := m.ExpectationsWereMet(); err != nil {
				t.Fatalf("DB expectations unmet: %s", err)
			}
		})
	}
}

func TestQueryHandlerFunc(t *testing.T) {
	t.Parallel()

	type args struct {
		query       string
		extraParams []string
	}

	type fields struct {
		args     []driver.Value
		queryErr error
		rowsErr  error
	}

	tests := []struct {
		name    string
		args    args
		fields  fields
		want    string
		wantErr bool
	}{
		{
			"+valid",
			args{"SELECT A, B FROM C", []string{}},
			fields{},
			`[{"a":1,"b":2},{"a":3,"b":4}]`,
			false,
		},
		{
			"+validWithArgs",
			args{"SELECT A, B FROM C WHERE A = @p1", []string{"10"}},
			fields{args: []driver.Value{"10"}},
			`[{"a":1,"b":2},{"a":3,"b":4}]`,
			false,
		},
		{
			"-queryErr",
			args{"SELECT A, B FROM C", []string{}},
			fields{queryErr: errors.New("fail")},
			"",
			true,
		},
		{
			"-rowsToJSONErr",
			args{"SELECT A, B FROM C", []string{}},
			fields{rowsErr: errors.New("fail")},
			"",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db, m, err := sqlmock.New()
			if err != nil {
				t.Fatalf("failed to create mock DB: %s", err)
			}

			m.ExpectQuery(fmt.Sprintf("^%s$", tt.args.query)).
				WithArgs(tt.fields.args...).
				WillReturnRows(
					sqlmock.NewRows([]string{"a", "b"}).
						AddRow(1, 2).
						AddRow(3, 4).
						RowError(1, tt.fields.rowsErr),
				).
				WillReturnError(tt.fields.queryErr)

			resp, err := QueryHandlerFunc(tt.args.query)(
				context.Background(),
				db,
				nil,
				tt.args.extraParams...,
			)
			if (err != nil) != tt.wantErr {
				t.Fatalf(
					"QueryHandlerFunc() error = %v, wantErr %v",
					err,
					tt.wantErr,
				)
			}

			if err == nil {
				b, err := json.Marshal(resp)
				if err != nil {
					t.Fatalf(
						"QueryHandlerFunc() "+
							"failed to marshal handler result: %v",
						err,
					)
				}

				if diff := cmp.Diff(tt.want, string(b)); diff != "" {
					t.Fatalf("QueryHandlerFunc() = %s", diff)
				}
			}

			if err := m.ExpectationsWereMet(); err != nil {
				t.Fatalf("DB expectations unmet: %s", err)
			}
		})
	}
}

func TestVersionHandler(t *testing.T) {
	t.Parallel()

	type fields struct {
		resp     any
		queryErr error
		rowsErr  error
	}

	tests := []struct {
		name    string
		fields  fields
		want    any
		wantErr bool
	}{
		{
			"+valid",
			fields{resp: "1.2.3"},
			"1.2.3 lvl 69 nice",
			false,
		},
		{
			"-queryErr",
			fields{resp: "1.2.3", queryErr: errors.New("fail")},
			nil,
			true,
		},
		{
			"-scanErr",
			fields{resp: "1.2.3", rowsErr: errors.New("fail")},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db, m, err := sqlmock.New()
			if err != nil {
				t.Fatalf("failed to create mock DB: %s", err)
			}

			m.ExpectQuery(
				`SELECT
                     SERVERPROPERTY\('productversion'\),
                     SERVERPROPERTY\('productlevel'\),
                     SERVERPROPERTY\('edition'\)`,
			).
				WillReturnRows(
					sqlmock.NewRows(
						[]string{"productversion", "productlevel", "edition"},
					).
						AddRow("1.2.3", "lvl 69", "nice").
						RowError(0, tt.fields.rowsErr),
				).
				WillReturnError(tt.fields.queryErr)

			got, err := VersionHandler(context.Background(), db, nil)
			if (err != nil) != tt.wantErr {
				t.Fatalf(
					"VersionHandler() error = %v, wantErr %v",
					err, tt.wantErr,
				)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatalf("VersionHandler() = %s", diff)
			}

			if err := m.ExpectationsWereMet(); err != nil {
				t.Fatalf("DB expectations unmet: %s", err)
			}
		})
	}
}

func Test_rowsToJSON(t *testing.T) {
	t.Parallel()

	now := time.Now()

	wrapAny := func(v any) any {
		return &v
	}

	type args struct {
		rows *sqlmock.Rows
	}

	tests := []struct {
		name    string
		args    args
		want    []map[string]any
		wantErr bool
	}{
		{
			"+valid",
			args{sqlmock.NewRows([]string{"a", "b"}).AddRow(1, 2).AddRow(3, 4)},
			[]map[string]any{
				{"a": wrapAny(int64(1)), "b": wrapAny(int64(2))},
				{"a": wrapAny(int64(3)), "b": wrapAny(int64(4))},
			},
			false,
		},
		{
			"+allTypes",
			args{
				sqlmock.NewRows([]string{"type", "val"}).
					AddRow("int", 1).
					AddRow("float", 1.1).
					AddRow("string", "abc").
					AddRow("bool", true).
					AddRow("nil", nil).
					AddRow("bytes", []byte("abc")).
					AddRow("time", now).
					AddRow("valid bool", &nullBool{
						NullBool: sql.NullBool{Bool: true, Valid: true},
					}).
					AddRow("null bool", &nullBool{}).
					AddRow("valid uuid", &nullUniqueIdentifier{
						uuid: &mssql.UniqueIdentifier{
							1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 12, 13, 14, 15, 16,
						},
						valid: true,
					}).
					AddRow("null uuid", &nullUniqueIdentifier{valid: false}),
			},
			[]map[string]any{
				{"type": wrapAny("int"), "val": wrapAny(int64(1))},
				{"type": wrapAny("float"), "val": wrapAny(1.1)},
				{"type": wrapAny("string"), "val": wrapAny(string("abc"))},
				{"type": wrapAny("bool"), "val": wrapAny(true)},
				{"type": wrapAny("nil"), "val": wrapAny(nil)},
				{"type": wrapAny("bytes"), "val": wrapAny([]byte("abc"))},
				{"type": wrapAny("time"), "val": wrapAny(now)},
				{"type": wrapAny("valid bool"), "val": wrapAny(int64(1))},
				{"type": wrapAny("null bool"), "val": wrapAny(nil)},
				{
					"type": wrapAny("valid uuid"),
					"val":  wrapAny("01020304-0506-0708-090A-0C0D0E0F1000"),
				},
				{"type": wrapAny("null uuid"), "val": wrapAny(nil)},
			},
			false,
		},
		{
			"-rowErr",
			args{
				sqlmock.NewRows([]string{"a", "b"}).
					AddRow(1, 2).
					AddRow(3, 4).
					RowError(1, errors.New("fail")),
			},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db, m, err := sqlmock.New()
			if err != nil {
				t.Fatalf("failed to create mock DB: %s", err)
			}

			m.ExpectQuery(".*").WillReturnRows(tt.args.rows)

			rows, err := db.Query("SELECT")
			if err != nil {
				t.Fatalf("failed to query mock DB: %s", err)
			}

			got, err := rowsToJSON(rows)
			if (err != nil) != tt.wantErr {
				t.Fatalf("rowsToJSON() error = %v, wantErr %v", err, tt.wantErr)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatalf("rowsToJSON() result diff\n%s", diff)
			}

			if err := m.ExpectationsWereMet(); err != nil {
				t.Fatalf("DB expectations unmet: %s", err)
			}
		})
	}
}
