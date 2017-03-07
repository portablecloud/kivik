package kivik

import (
	"fmt"
	"io"
	"net/http"
	"regexp"

	"github.com/flimzy/kivik/driver"
	"github.com/flimzy/kivik/errors"
)

// Client is a client connection handle to a CouchDB-like server.
type Client struct {
	dsn          string
	driverClient driver.Client
}

// Options is a collection of options. The keys and values are backend specific.
type Options map[string]interface{}

// New creates a new client object specified by its database driver name
// and a driver-specific data source name.
func New(driverName, dataSourceName string) (*Client, error) {
	driversMu.RLock()
	driveri, ok := drivers[driverName]
	driversMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("kivik: unknown driver %q (forgotten import?)", driverName)
	}
	client, err := driveri.NewClient(dataSourceName)
	if err != nil {
		return nil, err
	}
	return &Client{
		dsn:          dataSourceName,
		driverClient: client,
	}, nil
}

// ServerInfo returns version and vendor info about the backend.
func (c *Client) ServerInfo() (driver.ServerInfo, error) {
	return c.driverClient.ServerInfo()
}

// DB returns a handle to the requested database.
func (c *Client) DB(dbName string) (*DB, error) {
	db, err := c.driverClient.DB(dbName)
	return &DB{
		driverDB: db,
	}, err
}

// AllDBs returns a list of all databases.
func (c *Client) AllDBs() ([]string, error) {
	return c.driverClient.AllDBs()
}

// UUIDs returns one or more UUIDs as generated by the CouchDB server. This
// method may not be implemented by all backends, in which case an error will
// be returned. Generally, there are better ways to generate UUIDs.
func (c *Client) UUIDs(count int) ([]string, error) {
	if count < 0 {
		return nil, errors.Status(http.StatusBadRequest, "count must be a positive integer")
	}
	if uuider, ok := c.driverClient.(driver.UUIDer); ok {
		return uuider.UUIDs(count)
	}
	return nil, ErrNotImplemented
}

// Membership returns the list of nodes that are part of the cluster as
// clusterNodes, and all known nodes, including cluster nodes, as allNodes.
// Not all servers or clients will support this method.
func (c *Client) Membership() (allNodes []string, clusterNodes []string, err error) {
	if cluster, ok := c.driverClient.(driver.Cluster); ok {
		return cluster.Membership()
	}
	return nil, nil, ErrNotImplemented
}

// Log reads the server log, if supported by the client driver. This method will
// read up to len(buf) bytes of logs from the server, ending at offset bytes from
// the end, placing the logs in buf. The number of read bytes will be returned.
func (c *Client) Log(buf []byte, offset int) (int, error) {
	if logger, ok := c.driverClient.(driver.Logger); ok {
		return logger.Log(buf, offset)
	}
	return 0, ErrNotImplemented
}

// DBExists returns true if the specified database exists.
func (c *Client) DBExists(dbName string) (bool, error) {
	return c.driverClient.DBExists(dbName)
}

// Copied verbatim from http://docs.couchdb.org/en/2.0.0/api/database/common.html#head--db
var validDBName = regexp.MustCompile("^[a-z][a-z0-9_$()+/-]*$")

// CreateDB creates a DB of the requested name.
func (c *Client) CreateDB(dbName string) error {
	if !validDBName.MatchString(dbName) {
		return errors.Status(errors.StatusBadRequest, "invalid database name")
	}
	return c.driverClient.CreateDB(dbName)
}

// DestroyDB deletes the requested DB.
func (c *Client) DestroyDB(dbName string) error {
	return c.driverClient.DestroyDB(dbName)
}

// Authenticate authenticates the client with the passed authenticator, which
// is driver-specific. If the driver does not understand the authenticator, an
// error will be returned.
func (c *Client) Authenticate(a interface{}) error {
	if auth, ok := c.driverClient.(driver.Authenticator); ok {
		return auth.Authenticate(a)
	}
	return ErrNotImplemented
}

// HTTPRequest returns an HTTP request to the CouchDB server. The path is
// expected to be a path relative to the CouchDB root. Any authentication
// headers will be set in the returned request.
func (c *Client) HTTPRequest(method, path string, body io.Reader) (*http.Request, *http.Client, error) {
	if reqer, ok := c.driverClient.(driver.HTTPRequester); ok {
		return reqer.HTTPRequest(method, path, body)
	}
	return nil, nil, ErrNotImplemented
}
