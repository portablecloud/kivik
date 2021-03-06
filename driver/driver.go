package driver

import (
	"context"
	"encoding/json"
	"io"
	"time"
)

// Driver is the interface that must be implemented by a database driver.
type Driver interface {
	// NewClient returns a connection handle to the database. The name is in a
	// driver-specific format.
	NewClientContext(ctx context.Context, name string) (Client, error)
}

// ServerInfo represents the response a server gives witha GET request to '/'.
type ServerInfo interface {
	// Response is the full response, unparsed.
	Response() json.RawMessage
	// Version should return the version string from the top level of the response.
	Version() string
	// Vendor should return the name of the vendor.
	Vendor() string
	// VendorVersion should return the vendor version number.
	VendorVersion() string
}

// Client is a connection to a database server.
type Client interface {
	// VersionInfo returns the server implementation's details.
	ServerInfoContext(ctx context.Context) (ServerInfo, error)
	AllDBsContext(ctx context.Context) ([]string, error)
	// DBExists returns true if the database exists.
	DBExistsContext(ctx context.Context, dbName string) (bool, error)
	// CreateDB creates the requested DB. The dbName is validated as a valid
	// CouchDB database name prior to calling this function, so the driver can
	// assume a valid name.
	CreateDBContext(ctx context.Context, dbName string) error
	// DestroyDB deletes the requested DB.
	DestroyDBContext(ctx context.Context, dbName string) error
	// DB returns a handleto the requested database
	DBContext(ctx context.Context, dbName string) (DB, error)
	// SetDefault allows the user to set a default option to be propogated to
	// future DB instances.
	SetDefault(key string, value interface{}) error
}

// Authenticator is an optional interface that may be implemented by a Client
// that supports authenitcated connections.
type Authenticator interface {
	// Authenticate attempts to authenticate the client using an authenticator.
	// If the authenticator is not known to the client, an error should be
	// returned.
	AuthenticateContext(ctx context.Context, authenticator interface{}) error
}

// UUIDer is an optional interface that may be implemented by a Client. Generally,
// this should not be used, but it is part of the CouchDB spec, so it is included
// for completeness.
type UUIDer interface {
	UUIDsContext(ctx context.Context, count int) ([]string, error)
}

// LogReader is an optional interface that may be implemented by a Client.
type LogReader interface {
	// Log reads the server log, up to length bytes, beginning offset bytes from
	// the end.
	LogContext(ctx context.Context, length, offset int64) (io.ReadCloser, error)
}

// Cluster is an optional interface that may be implemented by a Client for
// servers that support clustering operations (specifically CouchDB 2.0)
type Cluster interface {
	MembershipContext(ctx context.Context) (allNodes []string, clusterNodes []string, err error)
}

// DBInfo provides statistics about a database.
type DBInfo struct {
	Name           string `json:"db_name"`
	CompactRunning bool   `json:"compact_running"`
	DocCount       int64  `json:"doc_count"`
	DeletedCount   int64  `json:"doc_del_count"`
	UpdateSeq      string `json:"update_seq"`
	DiskSize       int64  `json:"disk_size"`
	ActiveSize     int64  `json:"data_size"`
	ExternalSize   int64  `json:"-"`
}

// Members represents the members of a database security document.
type Members struct {
	Names []string `json:"names,omitempty"`
	Roles []string `json:"roles,omitempty"`
}

// Security represents a database security document.
type Security struct {
	Admins  Members `json:"admins"`
	Members Members `json:"members"`
}

// DB is a database handle.
type DB interface {
	// SetOption allows setting a database-specific option.
	SetOption(key string, value interface{}) error
	// AllDocsContext returns all of the documents in the database, subject
	// to the options provided.
	AllDocsContext(ctx context.Context, options map[string]interface{}) (Rows, error)
	// GetContext fetches the requested document from the database, and unmarshals it
	// into doc.
	GetContext(ctx context.Context, docID string, doc interface{}, options map[string]interface{}) error
	// CreateDocContext creates a new doc, with a server-generated ID.
	CreateDocContext(ctx context.Context, doc interface{}) (docID, rev string, err error)
	// PutContext writes the document in the database.
	PutContext(ctx context.Context, docID string, doc interface{}) (rev string, err error)
	// DeleteContext marks the specified document as deleted.
	DeleteContext(ctx context.Context, docID, rev string) (newRev string, err error)
	// InfoContext returns information about the database
	InfoContext(ctx context.Context) (*DBInfo, error)
	// CompactContext initiates compaction of the database.
	CompactContext(ctx context.Context) error
	// CompactViewContext initiates compaction of the view.
	CompactViewContext(ctx context.Context, ddocID string) error
	// ViewCleanupContext cleans up stale view files.
	ViewCleanupContext(ctx context.Context) error
	// SecurityContext returns the database's security document.
	SecurityContext(ctx context.Context) (*Security, error)
	// SetSecurityContext sets the database's security document.
	SetSecurityContext(ctx context.Context, security *Security) error
	// RevsLimitContext returns the the maximum number of document revisions
	// that will be tracked.
	RevsLimitContext(ctx context.Context) (limit int, err error)
	// SetRevsLimitContext sets the maximum number of document revisions that
	// will be tracked.
	SetRevsLimitContext(ctx context.Context, limit int) error
	// ChangesContext returns a Rows iterator for the changes feed. In
	// continuous mode, the iterator will continue indefinately, until Close is
	// called.
	ChangesContext(ctx context.Context, options map[string]interface{}) (Rows, error)
	// BulkDocsContext alls bulk create, update and/or delete operations. It
	// returns an iterator over the results.
	BulkDocsContext(ctx context.Context, docs ...interface{}) (BulkResults, error)
	// PutAttachmentContext uploads an attachment to the specified document,
	// returning the new revision.
	PutAttachmentContext(ctx context.Context, docID, rev, filename, contentType string, body io.Reader) (newRev string, err error)
	// GetAttachmentContext fetches an attachment for the associated document ID.
	// rev may be empty, to fetch the most recent document version.
	GetAttachmentContext(ctx context.Context, docID, rev, filename string) (contentType string, md5sum Checksum, body io.ReadCloser, err error)
	// DeleteAttachmentContext deletes an attachment from a document, returning
	// the document's new revision.
	DeleteAttachmentContext(ctx context.Context, docID, rev, filename string) (newRev string, err error)
	// QueryContext performs a query against a view, subject to the options
	// provided.
	// ddoc will be the design doc name without the '_design/' previx.
	// view will be the view name without the '_view/' prefix.
	QueryContext(ctx context.Context, ddoc, view string, options map[string]interface{}) (Rows, error)
}

// Finder is an optional interface which may be implemented by a database. The
// Finder interface provides access to the new (in CouchDB 2.0) MongoDB-style
// query interface.
type Finder interface {
	// FindContext executes a query using the new /_find interface. If query is
	// a string, []byte, or json.RawMessage, it should be treated as a raw JSON
	// payload. Any other type should be marshaled to JSON.
	FindContext(ctx context.Context, query interface{}) (Rows, error)
	// CreateIndexContext creates an index if it doesn't already exist. If the
	// index already exists, it should do nothing. ddoc and name may be empty,
	// in which case they should be provided by the backend. If index is a
	// string, []byte, or json.RawMessage, it should be treated as a raw JSON
	// payload. Any other type should be marshaled to JSON.
	CreateIndexContext(ctx context.Context, ddoc, name string, index interface{}) error
	// GetIndexesContext returns a list of all indexes in the database.
	GetIndexesContext(ctx context.Context) ([]Index, error)
	// DeleteIndex deletes the requested index.
	DeleteIndexContext(ctx context.Context, ddoc, name string) error
}

// Index is a MonboDB-style index definition.
type Index struct {
	DesignDoc  string      `json:"ddoc,omitempty"`
	Name       string      `json:"name"`
	Type       string      `json:"type"`
	Definition interface{} `json:"def"`
}

// Checksum is a 128-bit MD5 checksum of a file's content.
type Checksum [16]byte

// AttachmentMetaer is an optional interface which may be satisfied by a
// DB. If satisfied, it may be used to fetch meta data about an attachment. If
// not satisfied, GetAttachmentContext will be used instead.
type AttachmentMetaer interface {
	// GetAttachmentMetaContext returns meta information about an attachment.
	GetAttachmentMetaContext(ctx context.Context, docID, rev, filename string) (contentType string, md5sum Checksum, err error)
}

// BulkResult is the result of a single doc update in a BulkDocs request.
type BulkResult struct {
	ID    string `json:"id"`
	Rev   string `json:"rev"`
	Error error
}

// BulkResults is an iterator over the results for a BulkDocs call.
type BulkResults interface {
	// Next is called to populate *BulkResult with the values of the next bulk
	// result in the set.
	//
	// Next should return io.EOF when there are no more results.
	Next(*BulkResult) error
	// Close closes the bulk results iterator.
	Close() error
}

// Rever is an optional interface that may be implemented by a database. If not
// implemented by the driver, the GetContext method will be used to emulate
// the functionality.
type Rever interface {
	// RevContext returns the most current revision of the requested document.
	RevContext(ctx context.Context, docID string) (rev string, err error)
}

// DBFlusher is an optional interface that may be implemented by a database
// that can force a flush of the database backend file(s) to disk or other
// permanent storage.
type DBFlusher interface {
	// Flush requests a flush of disk cache to disk or other permanent storage.
	// The response a timestamp when the database backend opened the storage
	// backend.
	//
	// See http://docs.couchdb.org/en/2.0.0/api/database/compact.html#db-ensure-full-commit
	FlushContext(ctx context.Context) (time.Time, error)
}

// Copier is an optional interface that may be implemented by a DB.
//
// If a DB does implement Copier, Copy() functions will use it.  If a DB does
// not implement the Copier interface, or if a call to Copy() returns an
// http.StatusUnimplemented, the driver will emulate a copy by doing
// a GET followed by PUT.
type Copier interface {
	CopyContext(ctx context.Context, targetID, sourceID string, options map[string]interface{}) (targetRev string, err error)
}

// Configer is an optional interface that may be implemented by a Client.
//
// If a Client does implement Configer, it allows backend configuration
// to be queried and modified via the API.
type Configer interface {
	ConfigContext(ctx context.Context) (Config, error)
}

// Config is the minimal interface that a Config backend must implement.
type Config interface {
	GetAllContext(ctx context.Context) (config map[string]map[string]string, err error)
	SetContext(ctx context.Context, secName, key, value string) error
	DeleteContext(ctx context.Context, secName, key string) error
}

// ConfigSection is an optional interface that may be implemented by a Config
// backend. If not implemented, it will be emulated with GetAll() and SetAll().
// The only reason for a config backend to implement this interface is if
// reading a config section alone can be more efficient than reading the entire
// configuration for the specific storage backend.
type ConfigSection interface {
	GetSectionContext(ctx context.Context, secName string) (section map[string]string, err error)
}

// ConfigItem is an optional interface that may be implemented by a Config
// backend. If not implemented, it will be emulated with GetAll() and SetAll().
// The only reason for a config backend to implement this interface is if
// reading a single config value alone can be more efficient than reading the
// entire configuration for the specific storage backend.
type ConfigItem interface {
	GetContext(ctx context.Context, secName, key string) (value string, err error)
}
