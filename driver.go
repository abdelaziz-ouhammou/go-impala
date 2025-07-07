package impala

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql/driver"
	"errors"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/abdelaziz-ouhammou/go-impala/v3/hive"
	"github.com/abdelaziz-ouhammou/go-impala/v3/sasl"
	"github.com/apache/thrift/lib/go/thrift"
)

var (
	// ErrNotSupported means this operation is not supported by impala driver
	ErrNotSupported = errors.New("impala: not supported")
)

// Driver to impala
type Driver struct{}

// Open creates new connection to impala
func (d *Driver) Open(uri string) (driver.Conn, error) {
	opts, err := parseURI(uri)
	if err != nil {
		return nil, err
	}

	log.Printf("opts: %v", opts)

	conn, err := connect(opts)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func parseURI(uri string) (*Options, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	if u.Scheme != "impala" {
		return nil, fmt.Errorf("scheme %s not recognized", u.Scheme)
	}

	opts := DefaultOptions

	if u.User != nil {
		opts.Username = u.User.Username()
		password, ok := u.User.Password()
		if ok {
			opts.Password = password
		}
	}

	if !strings.Contains(u.Host, ":") {
		u.Host = fmt.Sprintf("%s:%s", u.Host, DefaultOptions.Port)
	}

	host, port, err := net.SplitHostPort(u.Host)
	if err != nil {
		return nil, err
	}

	opts.Host = host
	opts.Port = port

	query := u.Query()
	auth := query.Get("auth")
	if auth == "ldap" {
		opts.UseLDAP = true
	}
	if auth == "kerberos" {
		opts.UseKerberos = true
	}
	service := query.Get("service")
	if service != "" {
		opts.Service = service
	}
	krbhost := query.Get("krb_host")
	if krbhost != "" {
		opts.KrbHost = krbhost
	}
	tls, ok := query["tls"]
	if ok {
		v, err := strconv.ParseBool(tls[0])
		if err != nil {
			return nil, err
		}
		opts.UseTLS = v
	}
	insecure := query.Get("secure")
	if insecure == "false" {
		opts.SkipServerCertCheck = true
	}
	caCert, ok := query["ca-cert"]
	if ok {
		opts.CACertPath = caCert[0]
	}

	batchSize, ok := query["batch-size"]
	if ok {
		size, err := strconv.Atoi(batchSize[0])
		if err != nil {
			return nil, err
		}
		opts.BatchSize = size
	}

	bufferSize, ok := query["buffer-size"]
	if ok {
		size, err := strconv.Atoi(bufferSize[0])
		if err != nil {
			return nil, err
		}
		opts.BufferSize = size
	}

	memLimit, ok := query["mem-limit"]
	if ok {
		opts.MemoryLimit = memLimit[0]
	}

	queryTimeout, ok := query["query-timeout"]
	if ok {
		qTimeout, err := strconv.Atoi(queryTimeout[0])
		if err != nil {
			return nil, err
		}
		opts.QueryTimeout = qTimeout
	}
	opts.ParquetArrayResolution = "THREE_LEVEL"
	parquetArrayResolution, ok := query["parquet-array-resolution"]
	if ok {
		opts.ParquetArrayResolution = parquetArrayResolution[0]
	}
	return &opts, nil
}

// OpenConnector parses name and return connector with fixed options
func (d *Driver) OpenConnector(name string) (driver.Connector, error) {

	opts, err := parseURI(name)
	if err != nil {
		return nil, err
	}

	return &connector{opts: opts}, nil
}

type connector struct {
	d    *Driver
	opts *Options
}

// NewConnector creates connector with specified options
func NewConnector(opts *Options) driver.Connector {
	return &connector{opts: opts}
}

func (c *connector) Connect(ctx context.Context) (driver.Conn, error) {
	return connect(c.opts)
}

func (c *connector) Driver() driver.Driver {
	return c.d
}

func connect(opts *Options) (*Conn, error) {
	addr := net.JoinHostPort(opts.Host, opts.Port)
	var socket thrift.TTransport
	var err error
	if opts.UseTLS || opts.UseKerberos {
		var caCertPool *x509.CertPool
		if !opts.SkipServerCertCheck {
			if opts.CACertPath == "" {
				return nil, errors.New("Please provide CA certificate path")
			}
			caCert, err := os.ReadFile(opts.CACertPath)
			if err != nil {
				return nil, err
			}
			caCertPool = x509.NewCertPool()
			caCertPool.AppendCertsFromPEM(caCert)
		}
		tlsConfig := tls.Config{
			// RootCAs:            caCertPool,
			CipherSuites:       []uint16{tls.TLS_RSA_WITH_AES_256_GCM_SHA384},
			InsecureSkipVerify: opts.SkipServerCertCheck,
		}
		socket = thrift.NewTSSLSocketConf(addr, &thrift.TConfiguration{
			TLSConfig: &tlsConfig,
		})
	} else {
		socket = thrift.NewTSocketConf(addr, &thrift.TConfiguration{})
	}
	var transport thrift.TTransport
	if opts.UseLDAP {

		if opts.Username == "" {
			return nil, errors.New("Please provide username for LDAP auth")
		}

		if opts.Password == "" {
			return nil, errors.New("Please provide password for LDAP auth")
		}

		transport, err = sasl.NewTSaslTransport(socket, &sasl.Options{
			Host:     opts.Host,
			Username: opts.Username,
			Password: opts.Password,
		})
		if err != nil {
			return nil, err
		}
	} else if opts.UseKerberos {
		transport, err = sasl.NewTSaslTransport(socket, &sasl.Options{
			Service:     opts.Service,
			Host:        opts.KrbHost,
			UseKerberos: true,
		})
		if err != nil {
			return nil, err
		}
	} else {
		transport = thrift.NewTBufferedTransport(socket, opts.BufferSize)
	}
	strictRead := false
	strictWrite := true
	protocol := thrift.NewTBinaryProtocolConf(transport, &thrift.TConfiguration{
		TBinaryStrictRead:  &strictRead,
		TBinaryStrictWrite: &strictWrite,
	})
	if err := transport.Open(); err != nil {
		return nil, err
	}

	logger := log.New(opts.LogOut, "impala: ", log.LstdFlags)

	tclient := thrift.NewTStandardClient(protocol, protocol)
	client := hive.NewClient(tclient, logger, &hive.Options{
		MaxRows:                int64(opts.BatchSize),
		MemLimit:               opts.MemoryLimit,
		QueryTimeout:           opts.QueryTimeout,
		ParquetArrayResolution: opts.ParquetArrayResolution,
	})

	return &Conn{client: client, t: transport, log: logger}, nil
}
