package impala

import (
	"database/sql"
	"io"
	"io/ioutil"
)

func init() {
	sql.Register("impala", &Driver{})
}

// Options for impala driver connection
type Options struct {
	Host     string
	Port     string
	Username string
	Password string

	UseLDAP                bool
	UseKerberos            bool
	Service                string
	KrbHost                string
	UseTLS                 bool
	SkipServerCertCheck    bool
	CACertPath             string
	BufferSize             int
	BatchSize              int
	MemoryLimit            string
	QueryTimeout           int
	ParquetArrayResolution string
	LogOut                 io.Writer
}

var (
	// DefaultOptions for impala driver
	DefaultOptions = Options{BatchSize: 1024, BufferSize: 4096, Port: "21050", LogOut: ioutil.Discard}
)
