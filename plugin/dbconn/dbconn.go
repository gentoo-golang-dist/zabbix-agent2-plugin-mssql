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
	"fmt"
	"net/url"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"golang.zabbix.com/plugin/mssql/plugin/handlers"
	"golang.zabbix.com/plugin/mssql/plugin/params"
	"golang.zabbix.com/sdk/errs"
	"golang.zabbix.com/sdk/log"
	"golang.zabbix.com/sdk/uri"
)

var (
	_ handlers.HandlerFunc = (*ConnCollection)(nil).WithConnHandlerFunc(nil)
	_ handlers.HandlerFunc = (*ConnCollection)(nil).PingHandler
)

// connConfig is a configuration for a connection to the database.
type connConfig struct {
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

// ConnCollection is a collection of connections to the database.
// Allows managing multiple connections.
type ConnCollection struct {
	mu         sync.Mutex
	conns      map[connConfig]*sql.DB
	keepAlive  int
	logr       log.Logger
	driverName string // always sqlserver, allow to change for unit tests.
}

// Init initializes a pre-allocated connection collection.
func (c *ConnCollection) Init(keepAlive int, logr log.Logger) {
	c.conns = make(map[connConfig]*sql.DB)
	c.keepAlive = keepAlive
	c.logr = logr
	c.driverName = "sqlserver"
}

// WithConnHandlerFunc creates a new function that creates or gets cached DB
// connection for the given metric parameters and calls the given handler
// function with the connection.
func (c *ConnCollection) WithConnHandlerFunc(
	handler handlers.ConnHandlerFunc,
) handlers.HandlerFunc {
	return func(
		timeout time.Duration, metricParams map[string]string, extraParams ...string,
	) (any, error) {
		ctx, cancel := context.WithTimeout(
			context.Background(),
			timeout,
		)
		defer cancel()

		conn, err := c.get(ctx, newConnConfig(metricParams))
		if err != nil {
			c.logr.Errf("Failed to get connection: %s", err.Error())

			return nil, errs.Wrap(err, "failed to get conn")
		}

		return handler(ctx, conn, metricParams, extraParams...)
	}
}

// PingHandler tries to ping the database, returning 1 on success 0 on failure.
func (c *ConnCollection) PingHandler(
	timeout time.Duration, metricParams map[string]string, _ ...string,
) (any, error) {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		timeout,
	)
	defer cancel()

	conn, err := c.get(ctx, newConnConfig(metricParams))
	if err != nil {
		c.logr.Errf("Failed to get connection for ping: %s", err.Error())

		return 0, nil
	}

	err = conn.PingContext(ctx)
	if err != nil {
		c.logr.Debugf("Failed to ping: %s", err.Error())

		return 0, nil
	}

	return 1, nil
}

// Close closes all connections in the collection.
func (c *ConnCollection) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for conf, conn := range c.conns {
		err := conn.Close()
		if err != nil {
			c.logr.Errf("Failed to close connection: %s", err.Error())
		}

		delete(c.conns, conf)
	}
}

//nolint:gocritic // need conf by value.
func (c *ConnCollection) get(
	ctx context.Context,
	conf connConfig,
) (*sql.DB, error) {
	conn := c.getConn(conf)
	if conn != nil {
		return conn, nil
	}

	conn, err := c.newConn(ctx, &conf)
	if err != nil {
		return nil, errs.Wrap(err, "failed to create conn")
	}

	return c.setConn(conf, conn), nil
}

// getConn concurrent connections cache getter.
func (c *ConnCollection) getConn(conf connConfig) *sql.DB { //nolint:gocritic
	c.mu.Lock()
	defer c.mu.Unlock()

	conn, ok := c.conns[conf]
	if !ok {
		return nil
	}

	return conn
}

// setConn concurrent connections cache setter.
//
// Returns the cached connection. If the provider connection is already present
// in cache, it is closed.
//
//nolint:gocritic
func (c *ConnCollection) setConn(
	conf connConfig,
	conn *sql.DB,
) *sql.DB {
	c.mu.Lock()
	defer c.mu.Unlock()

	existingConn, ok := c.conns[conf]
	if ok {
		defer conn.Close() //nolint:errcheck

		c.logr.Debugf("Closed redundant connection: %s", conf.URI)

		return existingConn
	}

	c.conns[conf] = conn

	return conn
}

//nolint:gocyclo,cyclop
//revive:disable:cognitive-complexity
func (c *ConnCollection) newConn(
	ctx context.Context,
	conf *connConfig,
) (*sql.DB, error) {
	c.logr.Debugf(
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
	queryParams.Add("app name", "Zabbix agent 2 MSSQL plugin")
	queryParams.Add("keepAlive", strconv.Itoa(c.keepAlive))

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

	u.RawQuery = queryParams.Encode()

	db, err := sql.Open(c.driverName, u.String())
	if err != nil {
		return nil, errs.Wrap(err, "failed to open DB connection")
	}

	db.SetConnMaxIdleTime(time.Duration(c.keepAlive) * time.Second)

	err = db.PingContext(ctx)
	if err != nil {
		return nil, errs.Wrap(err, "failed to ping")
	}

	return db, nil
}

func newConnConfig(metricParams map[string]string) connConfig {
	return connConfig{
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
