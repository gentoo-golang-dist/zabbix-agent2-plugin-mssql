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
	"io"
	"io/fs"
	"path/filepath"
	"strings"

	"git.zabbix.com/ap/mssql/plugin/params"
	"git.zabbix.com/ap/plugin-support/log"
	"git.zabbix.com/ap/plugin-support/zbxerr"
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
	cols, err := rows.Columns()
	if err != nil {
		return nil, zbxerr.Wrap(err, "failed to get columns")
	}

	var (
		values = make([]sql.RawBytes, len(cols))
		dest   = make([]any, len(cols))
	)

	for idx := range dest {
		dest[idx] = &values[idx]
	}

	results := []map[string]any{}

	for rows.Next() {
		err := rows.Scan(dest...)
		if err != nil {
			return nil, zbxerr.Wrap(err, "failed to scan row")
		}

		res := make(map[string]any)

		for idx := range cols {
			var val any

			if values[idx] != nil {
				val = string(values[idx])
			}

			res[cols[idx]] = val
		}

		results = append(results, res)
	}

	err = rows.Err()
	if err != nil {
		return nil, zbxerr.Wrap(err, "failed to iterate over rows")
	}

	return results, nil
}
