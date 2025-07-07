package hive

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/abdelaziz-ouhammou/go-impala/v3/services/cli_service"
	"github.com/apache/thrift/lib/go/thrift"
)

// Client represents Hive Client
type Client struct {
	client *cli_service.TCLIServiceClient
	opts   *Options
	log    *log.Logger
}

// Options for Hive Client
type Options struct {
	MaxRows                int64
	MemLimit               string
	QueryTimeout           int
	ParquetArrayResolution string
}

// NewClient creates Hive Client
func NewClient(client thrift.TClient, log *log.Logger, opts *Options) *Client {
	return &Client{
		client: cli_service.NewTCLIServiceClient(client),
		log:    log,
		opts:   opts,
	}
}

// OpenSession creates new hive session
func (c *Client) OpenSession(ctx context.Context) (*Session, error) {
	cfg := map[string]string{
		"MEM_LIMIT":                c.opts.MemLimit,
		"QUERY_TIMEOUT_S":          strconv.Itoa(c.opts.QueryTimeout),
		"PARQUET_ARRAY_RESOLUTION": c.opts.ParquetArrayResolution,
	}
	fmt.Println(cfg)

	req := cli_service.TOpenSessionReq{
		ClientProtocol: cli_service.TProtocolVersion_HIVE_CLI_SERVICE_PROTOCOL_V7,
		Configuration:  cfg,
	}

	resp, err := c.client.OpenSession(ctx, &req)
	if err != nil {
		return nil, err
	}
	if err := checkStatus(resp); err != nil {
		return nil, err
	}

	c.log.Printf("open session: %s", guid(resp.SessionHandle.GetSessionId().GUID))
	c.log.Printf("session config: %v", resp.Configuration)
	return &Session{h: resp.SessionHandle, hive: c}, nil
}
