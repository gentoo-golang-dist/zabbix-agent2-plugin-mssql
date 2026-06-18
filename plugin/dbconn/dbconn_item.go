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
	"fmt"
	"net/url"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"golang.zabbix.com/plugin/mssql/plugin/params"
	"golang.zabbix.com/sdk/errs"
	"golang.zabbix.com/sdk/log"
	"golang.zabbix.com/sdk/uri"
)

// ConnConfig is a configuration for a connection to the database.
type ConnConfig struct {
	URI                    string
	User                   string
	Password               string
	CACertPath             string
	TrustServerCertificate string
	HostNameInCertificate  string
	Encrypt                string
	TLSMinVersion          string
	Database               string
}

// ConnItem type is a connection item, and it keeps a database handle with its service and configuration fields.
type ConnItem struct {
	db *sql.DB
	mu sync.Mutex

	keepAlive  int
	logr       log.Logger
	driverName string
}

// newConnConfig function creates connection config instance.
func newConnConfig(metricParams map[string]string) *ConnConfig {
	return &ConnConfig{
		URI:                    metricParams[params.URI.Name()],
		User:                   metricParams[params.User.Name()],
		Password:               metricParams[params.Password.Name()],
		CACertPath:             metricParams[params.CACertPath.Name()],
		TrustServerCertificate: metricParams[params.TrustServerCertificate.Name()],
		HostNameInCertificate:  metricParams[params.HostNameInCertificate.Name()],
		Encrypt:                metricParams[params.Encrypt.Name()],
		TLSMinVersion:          metricParams[params.TLSMinVersion.Name()],
		Database:               metricParams[params.Database.Name()],
	}
}

// newConnItem function creates a new connection item instance by its configuration but without opening.
func newConnItem(keepAlive int, logr log.Logger, driverName string) *ConnItem {
	s := ConnItem{
		keepAlive:  keepAlive,
		logr:       logr,
		driverName: driverName,
	}

	return &s
}

// getDbConn function initializes a pre-allocated database handle. First, it must be created with newConnItem function.
func (s *ConnItem) getDbConn(connectionTimeout int, conf *ConnConfig) (*sql.DB, error) {
	s.logr.Debugf(
		"Creating new connection to %q, with user %q to database %q, "+
			"with CA certificate %q, "+
			"trust server certificate %q, host name in certificate %q "+
			"encrypt %q, TLS min version %q",
		conf.URI,
		conf.User,
		conf.Database,
		conf.CACertPath,
		conf.TrustServerCertificate,
		conf.HostNameInCertificate,
		conf.Encrypt,
		conf.TLSMinVersion,
	)

	connURI, err := uri.NewWithCreds(
		conf.URI, conf.User, conf.Password, &uri.Defaults{Scheme: params.URIDefaults.Scheme},
	)
	if err != nil {
		return nil, errs.Wrap(err, "failed to set URI defaults")
	}

	u, err := url.Parse(connURI.String())
	if err != nil {
		return nil, errs.Wrap(err, "failed to parse URI")
	}

	if connURI.Port() == "" && connURI.Path() == "" {
		u.Host = fmt.Sprintf("%s:%s", u.Hostname(), params.URIDefaults.Port)
	}

	u.Path = connURI.Path()

	queryParams := u.Query()
	s.composeQueryParams(queryParams, conf)

	// other database libraries accept the connection timeout as a separate value,
	// but MSSQL wants it to be a part of the connection string.
	// to maintain consistency with other integrations, we attach it here.
	if connectionTimeout != 0 {
		queryParams.Add("connection timeout", strconv.Itoa(connectionTimeout))
	}

	u.RawQuery = queryParams.Encode()

	db, err := sql.Open(s.driverName, u.String())
	if err != nil {
		return nil, errs.Wrap(err, "failed to open DB connection")
	}

	db.SetConnMaxIdleTime(time.Duration(s.keepAlive) * time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(connectionTimeout)*time.Second)

	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		return nil, errs.Wrap(err, "failed to ping")
	}

	return db, nil
}

func (s *ConnItem) composeQueryParams(queryParams url.Values, conf *ConnConfig) {
	queryParams.Add("app name", "Zabbix agent 2 MSSQL plugin")
	queryParams.Add("keepAlive", strconv.Itoa(s.keepAlive))

	if conf.Database != "" {
		queryParams.Add("database", conf.Database)
	}

	if conf.CACertPath != "" {
		queryParams.Add("certificate", filepath.Clean(conf.CACertPath))
	}

	if conf.TrustServerCertificate != "" {
		queryParams.Add("TrustServerCertificate", conf.TrustServerCertificate)
	}

	if conf.HostNameInCertificate != "" {
		queryParams.Add("hostNameInCertificate", conf.HostNameInCertificate)
	}

	if conf.Encrypt != "" {
		queryParams.Add("encrypt", conf.Encrypt)
	}

	if conf.TLSMinVersion != "" {
		queryParams.Add("tlsMinVersion", conf.TLSMinVersion)
	}
}

// closeDb function closes a connection item's database handle.
func (s *ConnItem) closeDb() {
	if s.db != nil { // item can be created but yet uninitialised
		err := s.db.Close()
		if err != nil {
			s.logr.Errf("Failed to close connection: %s", err.Error())
		}
	}
}
