package dbconn

import (
	"database/sql"
	"net/url"
	"sync"

	"git.zabbix.com/ap/plugin-support/zbxerr"
)

var (
	_ Queryer = (*Conn)(nil)
	_ Rows    = (*sql.Rows)(nil)
)

// Queryer describes the interface for querying the database.
type Queryer interface {
	Query(query string, args ...any) (Rows, error)
}

// Rows describes the interface for rows returned by a query.
type Rows interface {
	Columns() ([]string, error)
	Close() error
	Next() bool
	Scan(dest ...any) error
}

// Conn is a wrapper for sql.DB. Implements Queryer.
type Conn struct {
	db *sql.DB
}

// ConnConfig is a configuration for a connection to the database.
type ConnConfig struct {
	User     string
	Password string
	URI      string
}

// ConnCollection is a collection of connections to the database.
// Allows managing multiple connections.
type ConnCollection struct {
	mu    sync.Mutex
	conns map[ConnConfig]*Conn
}

// NewConnCollection creates a new ConnCollection.
func NewConnCollection() *ConnCollection {
	return &ConnCollection{conns: make(map[ConnConfig]*Conn)}
}

// Get returns a connection from the collection. If the connection with the
// provided configuration does not exist a new connection is going to be
// created and stored for subsequent call to Get.
func (c *ConnCollection) Get(conf ConnConfig) (*Conn, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	conn, ok := c.conns[conf]
	if ok {
		return conn, nil
	}

	conn, err := newConn(&conf)
	if err != nil {
		return nil, zbxerr.Wrap(err, "failed to create conn")
	}

	err = conn.db.Ping()
	if err != nil {
		return nil, zbxerr.Wrap(err, "failed to ping DB")
	}

	c.conns[conf] = conn

	return conn, nil
}

// Query wrapper for go-mssqldb Query.
func (c *Conn) Query(query string, args ...any) (Rows, error) {
	rows, err := c.db.Query(query, args...) //nolint:rowserrcheck
	if err != nil {
		return nil, zbxerr.Wrap(err, "failed to query MSSQL DB")
	}

	return rows, nil
}

func newConn(conf *ConnConfig) (*Conn, error) {
	u, err := url.Parse(conf.URI)
	if err != nil {
		return nil, zbxerr.Wrap(err, "failed to parse URI")
	}

	u.User = url.UserPassword(conf.User, conf.Password)

	queryParams := u.Query()
	queryParams.Add("app name", "Zabbix agent 2 MSSQL plugin")

	u.RawQuery = queryParams.Encode()

	dsn := u.String()

	dbConn, err := sql.Open("sqlserver", dsn)
	if err != nil {
		return nil, zbxerr.Wrap(err, "failed to open DB connection")
	}

	return &Conn{db: dbConn}, nil
}
