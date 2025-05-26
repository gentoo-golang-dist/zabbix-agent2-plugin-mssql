/*
** Zabbix
** Copyright (C) 2001-2025 Zabbix SIA
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
	"io"
	"io/fs"
	"path/filepath"
	"strings"

	mssql "github.com/microsoft/go-mssqldb"
	"golang.zabbix.com/plugin/mssql/plugin/params"
	"golang.zabbix.com/sdk/errs"
	"golang.zabbix.com/sdk/log"
)

var (
	_ sql.Scanner   = (*nullUniqueIdentifier)(nil)
	_ driver.Valuer = (*nullUniqueIdentifier)(nil)

	_ sql.Scanner   = (*nullBool)(nil)
	_ driver.Valuer = (*nullBool)(nil)

	_ HandlerFunc     = WithJSONResponse(nil)
	_ ConnHandlerFunc = QueryHandlerFunc("")
	_ ConnHandlerFunc = VersionHandler
	_ ConnHandlerFunc = CustomQueries(nil).HandlerFunc
)

// HandlerFunc describes the signature all metric handler functions must have.
type HandlerFunc func(
	metricParams map[string]string, extraParams ...string,
) (any, error)

// ConnHandlerFunc describes the signature all connection handler functions
// must have.
type ConnHandlerFunc func(
	ctx context.Context,
	conn *sql.DB,
	metricParams map[string]string,
	extraParams ...string,
) (any, error)

// CustomQueries stores user defined custom queries.
type CustomQueries map[string]string

// nullUniqueIdentifier is a wrapper for mssql.UniqueIdentifier that allows for
// NULL values and parses valid values as string representations of the UUID.
type nullUniqueIdentifier struct {
	uuid  *mssql.UniqueIdentifier
	valid bool
}

// nullBool is a wrapper for sql.NullBool that parses to 0 or 1 instead of
// false or true.
type nullBool struct {
	sql.NullBool
}

// Scan implements the Scanner interface.
func (nuid *nullUniqueIdentifier) Scan(value any) error {
	if value == nil {
		nuid.uuid = nil
		nuid.valid = false

		return nil
	}

	nuid.uuid = &mssql.UniqueIdentifier{}

	err := nuid.uuid.Scan(value)
	if err != nil {
		return errs.Wrap(err, "failed to scan UniqueIdentifier")
	}

	nuid.valid = true

	return nil
}

// Value implements the driver Valuer interface.
func (nuid *nullUniqueIdentifier) Value() (driver.Value, error) {
	if !nuid.valid {
		// here nil is a valid return representing NULL value from DB
		return nil, nil //nolint:nilnil
	}

	// check that the underlying UUID is valid.
	_, err := nuid.uuid.Value()
	if err != nil {
		return nil, errs.Wrap(err, "failed to get UniqueIdentifier value")
	}

	return nuid.uuid.String(), nil
}

// Value implements the driver Valuer interface.
func (b nullBool) Value() (driver.Value, error) {
	valuer, err := b.NullBool.Value()
	if err != nil {
		return nil, errs.Wrap(err, "failed to get bool value")
	}

	if valuer == nil {
		// here nil is a valid return representing NULL value from DB
		return nil, nil //nolint:nilnil
	}

	v, ok := valuer.(bool)
	if !ok {
		return nil, errs.Errorf("failed cast NullBool.Value() as bool")
	}

	if v {
		return int64(1), nil
	}

	return int64(0), nil
}

// WithJSONResponse wraps a handler function, marshaling its response
// to a JSON object and returning it as string.
func WithJSONResponse(handler HandlerFunc) HandlerFunc {
	return func(
		metricParams map[string]string, extraParams ...string,
	) (any, error) {
		res, err := handler(metricParams, extraParams...)
		if err != nil {
			return nil, errs.Wrap(err, "failed to execute handler")
		}

		jsonRes, err := json.Marshal(res)
		if err != nil {
			return nil, errs.Wrap(err, "failed to marshal result to JSON")
		}

		return string(jsonRes), nil
	}
}

// Load loads user defined custom queries form a config specified directory.
func (cq CustomQueries) Load(customQueriesDirFS fs.FS, logr log.Logger) error {
	queryFilePaths, err := fs.Glob(customQueriesDirFS, "*.sql")
	if err != nil {
		return errs.Wrap(err, "failed to match glob pattern")
	}

	queries := make(map[string]string)

	for _, qfp := range queryFilePaths {
		// nameless clojure to trigger defers on end of each iteration.
		err := func() error {
			f, err := customQueriesDirFS.Open(qfp)
			if err != nil {
				return errs.Wrap(err, "failed to open custom query file")
			}

			defer f.Close() //nolint:errcheck // not checking err.

			data, err := io.ReadAll(f)
			if err != nil {
				return errs.Wrapf(
					err,
					"failed to read contents of custom query file %s",
					qfp,
				)
			}

			qName := strings.TrimSuffix(filepath.Base(qfp), filepath.Ext(qfp))
			queries[qName] = string(data)

			logr.Infof(
				"Loaded custom query from file %q with name %q",
				qfp,
				qName,
			)

			return nil
		}()
		if err != nil {
			return err
		}
	}

	for k, v := range queries {
		cq[k] = v
	}

	return nil
}

// HandlerFunc handles a single metric request to execute a custom query.
func (cq CustomQueries) HandlerFunc(
	ctx context.Context,
	conn *sql.DB,
	metricParams map[string]string,
	extraParams ...string,
) (any, error) {
	name := metricParams[params.QueryName.Name()]

	query, ok := cq[name]
	if !ok {
		return nil, errs.Errorf("custom query %q not found", name)
	}

	return QueryHandlerFunc(query)(ctx, conn, metricParams, extraParams...)
}

// QueryHandlerFunc returns a handler function that will execute the specified
// query with arguments, formatting result rows as JSON.
func QueryHandlerFunc(query string) ConnHandlerFunc {
	return func(
		ctx context.Context,
		conn *sql.DB,
		_ map[string]string,
		extraParams ...string,
	) (any, error) {
		args := make([]any, 0, len(extraParams))

		for _, p := range extraParams {
			args = append(args, p)
		}

		rows, err := conn.QueryContext(ctx, query, args...)
		if err != nil {
			return nil, errs.Wrap(err, "failed to query")
		}

		defer rows.Close() //nolint:errcheck // not checking err.

		res, err := rowsToJSON(rows)
		if err != nil {
			return nil, errs.Wrap(err, "failed to convert rows to json")
		}

		return res, nil
	}
}

// VersionHandler handler func that returns the version of the database server.
func VersionHandler(
	ctx context.Context, conn *sql.DB, _ map[string]string, _ ...string,
) (any, error) {
	const query = `SELECT
                     SERVERPROPERTY('productversion'),
                     SERVERPROPERTY('productlevel'),
                     SERVERPROPERTY('edition')`

	row := conn.QueryRowContext(ctx, query)

	var (
		productVersion string
		productLevel   string
		edition        string
	)

	err := row.Scan(&productVersion, &productLevel, &edition)
	if err != nil {
		return nil, errs.Wrap(err, "failed to scan version")
	}

	err = row.Err()
	if err != nil {
		return nil, errs.Wrap(err, "failed to iterate over rows")
	}

	return strings.Join(
		[]string{productVersion, productLevel, edition},
		" ",
	), nil
}

//nolint:gocyclo,cyclop // it's not that big (thats what she said).
func rowsToJSON(rows *sql.Rows) ([]map[string]any, error) {
	cols, err := rows.ColumnTypes()
	if err != nil {
		return nil, errs.Wrap(err, "failed to get column types")
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
			case "DECIMAL":
				val = &sql.NullFloat64{}
			case "BIT":
				val = &nullBool{}
			default:
				var v any
				val = &v
			}

			dest = append(dest, val)
		}

		err = rows.Scan(dest...)
		if err != nil {
			return nil, errs.Wrap(err, "failed to scan row")
		}

		res := make(map[string]any)

		for idx := range cols {
			val := dest[idx]

			valuer, ok := dest[idx].(driver.Valuer)
			if ok {
				val, err = valuer.Value()
				if err != nil {
					return nil, errs.Wrap(err, "failed to get value")
				}
			}

			res[cols[idx].Name()] = val
		}

		results = append(results, res)
	}

	err = rows.Err()
	if err != nil {
		return nil, errs.Wrap(err, "failed to iterate over rows")
	}

	return results, nil
}
