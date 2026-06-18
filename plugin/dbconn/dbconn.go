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

	"golang.zabbix.com/plugin/mssql/plugin/handlers"
	"golang.zabbix.com/sdk/errs"
	"golang.zabbix.com/sdk/log"
	"golang.zabbix.com/sdk/zbxsync"
)

var (
	_ handlers.HandlerFunc = (*ConnManager)(nil).WithConnHandlerFunc(nil)
	_ handlers.HandlerFunc = (*ConnManager)(nil).PingHandler
)

// ConnManager is a collection of connections to the database.
// Allows managing multiple connections.
type ConnManager struct {
	conns *zbxsync.SyncMap[ConnConfig, *ConnItem]

	keepAlive  int
	logr       log.Logger
	driverName string // always sqlserver, allow changing for unit tests.
}

// Init initializes a pre-allocated connection collection.
func (c *ConnManager) Init(keepAlive int, logr log.Logger) {
	c.conns = &zbxsync.SyncMap[ConnConfig, *ConnItem]{}
	c.keepAlive = keepAlive
	c.logr = logr
	c.driverName = "sqlserver"
}

// WithConnHandlerFunc creates a new function that creates or gets cached DB
// connection for the given metric parameters and calls the given handler
// function with the connection.
func (c *ConnManager) WithConnHandlerFunc(
	handler handlers.ConnHandlerFunc,
) handlers.HandlerFunc {
	return func(
		ctx context.Context,
		connectionTimeout int,
		metricParams map[string]string,
		extraParams ...string,
	) (any, error) {
		//nolint:contextcheck // connection get does not need ctx.
		conn, err := c.get(connectionTimeout, newConnConfig(metricParams))
		if err != nil {
			c.logr.Errf("Failed to get connection: %s", err.Error())

			return nil, errs.Wrap(err, "failed to get conn")
		}

		return handler(ctx, conn.db, metricParams, extraParams...)
	}
}

// PingHandler tries to ping the database, returning 1 on success 0 on failure.
func (c *ConnManager) PingHandler(
	ctx context.Context,
	connectionTimeout int,
	metricParams map[string]string,
	_ ...string,
) (any, error) {
	//nolint:contextcheck // connection get does not need ctx.
	conn, err := c.get(connectionTimeout, newConnConfig(metricParams))
	if err != nil {
		c.logr.Errf("Failed to get connection for ping: %s", err.Error())

		return 0, nil
	}

	err = conn.db.PingContext(ctx)
	if err != nil {
		c.logr.Debugf("Failed to ping: %s", err.Error())

		return 0, nil
	}

	return 1, nil
}

// Close closes all connections in the collection.
func (c *ConnManager) Close() {
	c.conns.Range(func(conf ConnConfig, item *ConnItem) bool {
		item.closeDb()
		c.conns.Delete(conf)

		return true
	})
}

func (c *ConnManager) get(
	connectionTimeout int,
	conf *ConnConfig,
) (*ConnItem, error) {
	conn, _ := c.conns.LoadOrStore(*conf, newConnItem(c.keepAlive, c.logr, c.driverName))

	conn.mu.Lock() // to implement singleflight pattern on db connection creation.
	defer conn.mu.Unlock()

	if conn.db == nil {
		dbConn, err := conn.getDbConn(connectionTimeout, conf)
		if err != nil {
			return nil, errs.Wrap(err, "failed to create conn")
		}

		conn.db = dbConn
	}

	return conn, nil
}
