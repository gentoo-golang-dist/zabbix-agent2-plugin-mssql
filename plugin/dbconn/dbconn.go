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
	"database/sql"
	"net/url"
	"strconv"
	"sync"
	"time"

	"git.zabbix.com/ap/mssql/plugin/handlers"
	"git.zabbix.com/ap/mssql/plugin/params"
	"git.zabbix.com/ap/plugin-support/log"
	"git.zabbix.com/ap/plugin-support/zbxerr"
)

// ConnConfig is a configuration for a connection to the database.
type ConnConfig struct {
	User     string
	Password string
	URI      string
}

// ConnCollection is a collection of connections to the database.
// Allows managing multiple connections.
type ConnCollection struct {
	mu         sync.Mutex
	conns      map[ConnConfig]*sql.DB
	keepAlive  int
	logr       log.Logger
	driverName string // always sqlserver, allow to change for unit tests.
}

// Init initializes a pre-allocated connection collection.
func (c *ConnCollection) Init(keepAlive int, logr log.Logger) {
	c.conns = make(map[ConnConfig]*sql.DB)
	c.keepAlive = keepAlive
	c.logr = logr
	c.driverName = "sqlserver"
}

func (c *ConnCollection) WithConnHandlerFunc(
	handler handlers.ConnHandlerFunc,
) handlers.HandlerFunc {
	return func(
		metricParams map[string]string, extraParams ...string,
	) (any, error) {
		conn, err := c.get(
			ConnConfig{
				URI:      metricParams[params.URI.Name()],
				User:     metricParams[params.User.Name()],
				Password: metricParams[params.Password.Name()],
			},
		)
		if err != nil {
			return nil, zbxerr.Wrap(err, "failed to get conn")
		}

		return handler(conn, metricParams, extraParams...)
	}
}

// PingHandler tries to ping the database, returning 1 on success 0 on failure.
func (c *ConnCollection) PingHandler(
	metricParams map[string]string, _ ...string,
) (any, error) {
	conn, err := c.get(
		ConnConfig{
			URI:      metricParams[params.URI.Name()],
			User:     metricParams[params.User.Name()],
			Password: metricParams[params.Password.Name()],
		},
	)
	if err != nil {
		c.logr.Infof("Failed go get connection for ping: %s", err.Error())

		return 0, nil
	}

	err = conn.Ping()
	if err != nil {
		c.logr.Infof("Failed to ping: %s", err.Error())

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

func (c *ConnCollection) get(conf ConnConfig) (*sql.DB, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	conn, ok := c.conns[conf]
	if ok {
		return conn, nil
	}

	conn, err := c.newConn(&conf)
	if err != nil {
		return nil, zbxerr.Wrap(err, "failed to create conn")
	}

	c.conns[conf] = conn

	return conn, nil
}

func (c *ConnCollection) newConn(conf *ConnConfig) (*sql.DB, error) {
	u, err := url.Parse(conf.URI)
	if err != nil {
		return nil, zbxerr.Wrap(err, "failed to parse URI")
	}

	u.User = url.UserPassword(conf.User, conf.Password)

	queryParams := u.Query()
	queryParams.Add("app name", "Zabbix agent 2 MSSQL plugin")
	queryParams.Add("keepAlive", strconv.Itoa(c.keepAlive))

	u.RawQuery = queryParams.Encode()

	db, err := sql.Open(c.driverName, u.String())
	if err != nil {
		return nil, zbxerr.Wrap(err, "failed to open DB connection")
	}

	db.SetConnMaxIdleTime(time.Duration(c.keepAlive) * time.Second)

	err = db.Ping()
	if err != nil {
		return nil, zbxerr.Wrap(err, "failed to ping")
	}

	return db, nil
}
