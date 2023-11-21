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

package handlers

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"io"
	"io/fs"
	"path/filepath"
	"strings"

	"git.zabbix.com/ap/mssql/plugin/params"
	"git.zabbix.com/ap/plugin-support/log"
	"git.zabbix.com/ap/plugin-support/zbxerr"
	mssql "github.com/microsoft/go-mssqldb"
)

var (
	_ sql.Scanner    = (*nullUniqueIdentifier)(nil)
	_ driver.Valuer  = (*nullUniqueIdentifier)(nil)
	_ json.Marshaler = (*nullUniqueIdentifier)(nil)
)

// HandlerFunc describes the signature all metric handler functions must have.
type HandlerFunc func(
	metricParams map[string]string, extraParams ...string,
) (any, error)

type ConnHandlerFunc func(
	conn *sql.DB, metricParams map[string]string, extraParams ...string,
) (any, error)

// CustomQueries stores user defined custom queries.
type CustomQueries map[string]string

type nullUniqueIdentifier struct {
	UUID  *mssql.UniqueIdentifier
	Valid bool
}

// Scan implements the Scanner interface.
func (nuid *nullUniqueIdentifier) Scan(value any) error {
	if value == nil {
		nuid.UUID = nil
		nuid.Valid = false

		return nil
	}

	nuid.UUID = &mssql.UniqueIdentifier{}

	err := nuid.UUID.Scan(value)
	if err != nil {
		return zbxerr.Wrap(err, "failed to scan UniqueIdentifier")
	}

	nuid.Valid = true

	return nil
}

// Value implements the driver Valuer interface.
func (nuid nullUniqueIdentifier) Value() (driver.Value, error) {
	if !nuid.Valid {
		return nil, nil
	}

	v, err := nuid.UUID.Value()
	if err != nil {
		return nil, zbxerr.Wrap(err, "failed to get value of UniqueIdentifier")
	}

	return v, nil
}

// MarshalJSON implements the json.Marshaler interface for nullable
// UniqueIdentifier.
func (nuid nullUniqueIdentifier) MarshalJSON() ([]byte, error) {
	var val any

	if nuid.Valid {
		val = nuid.UUID.String()
	}

	b, err := json.Marshal(val)
	if err != nil {
		return nil, zbxerr.Wrap(err, "failed to marshal UniqueIdentifier")
	}

	return b, nil
}

// Loads user defined custom queries form a config specified directory.
func (cq CustomQueries) Load(customQueriesDirFS fs.FS, logr log.Logger) error {
	queryFilePaths, err := fs.Glob(customQueriesDirFS, "*.sql")
	if err != nil {
		return zbxerr.Wrap(err, "failed to match glob pattern")
	}

	queries := make(map[string]string)

	for _, qfp := range queryFilePaths {
		f, err := customQueriesDirFS.Open(qfp)
		if err != nil {
			return zbxerr.Wrap(err, "failed to open custom query file")
		}

		defer f.Close() //nolint:gocritic,revive // closure over scoped var.

		data, err := io.ReadAll(f)
		if err != nil {
			return zbxerr.Wrapf(
				err,
				"failed to read contents of custom query file %s",
				qfp,
			)
		}

		qName := strings.TrimSuffix(filepath.Base(qfp), filepath.Ext(qfp))
		queries[qName] = string(data)

		logr.Infof("Loaded custom query from file %q with name %q", qfp, qName)
	}

	for k, v := range queries {
		cq[k] = v
	}

	return nil
}

// HandlerFunc handles a single metric request to execute a custom query.
func (cq CustomQueries) HandlerFunc(
	conn *sql.DB, metricParams map[string]string, extraParams ...string,
) (any, error) {
	name := metricParams[params.QueryName.Name()]

	query, ok := cq[name]
	if !ok {
		return nil, zbxerr.Errorf("custom query %q not found", name)
	}

	return QueryHandlerFunc(query)(conn, metricParams, extraParams...)
}

// QueryHandlerFunc returns a handler function that will execute the specified
// query with arguments, formatting result rows as JSON.
func QueryHandlerFunc(query string) ConnHandlerFunc {
	return func(
		conn *sql.DB, _ map[string]string, extraParams ...string,
	) (any, error) {
		args := make([]any, 0, len(extraParams))

		for _, p := range extraParams {
			args = append(args, p)
		}

		rows, err := conn.Query(query, args...)
		if err != nil {
			return nil, zbxerr.Wrap(err, "failed to query")
		}

		defer func() { rows.Close() }()

		res, err := rowsToJSON(rows)
		if err != nil {
			return nil, zbxerr.Wrap(err, "failed to convert rows to json")
		}

		return res, nil
	}
}

// VersionHandler handler func that returns the version of the database server.
func VersionHandler(
	conn *sql.DB, _ map[string]string, _ ...string,
) (any, error) {
	const query = "SELECT SERVERPROPERTY('productversion')"

	row := conn.QueryRow(query)

	var version string

	err := row.Scan(&version)
	if err != nil {
		return nil, zbxerr.Wrap(err, "failed to scan version")
	}

	err = row.Err()
	if err != nil {
		return nil, zbxerr.Wrap(err, "failed to iterate over rows")
	}

	return version, nil
}

func rowsToJSON(rows *sql.Rows) ([]map[string]any, error) {
	cols, err := rows.ColumnTypes()
	if err != nil {
		return nil, zbxerr.Wrap(err, "failed to get column types")
	}

	results := []map[string]any{}

	for rows.Next() {
		// make new dest for each row, cause it's all pointer.
		dest := make([]any, 0, len(cols))

		for _, col := range cols {
			var val any

			switch col.DatabaseTypeName() {
			case "UNIQUEIDENTIFIER":
				val = &nullUniqueIdentifier{}
			default:
				var v any
				val = &v
			}

			dest = append(dest, val)
		}

		err = rows.Scan(dest...)
		if err != nil {
			return nil, zbxerr.Wrap(err, "failed to scan row")
		}

		res := make(map[string]any)

		for idx := range cols {
			res[cols[idx].Name()] = dest[idx]
		}

		results = append(results, res)
	}

	err = rows.Err()
	if err != nil {
		return nil, zbxerr.Wrap(err, "failed to iterate over rows")
	}

	return results, nil
}
