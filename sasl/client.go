package sasl

// NewClient created new sasl client
func NewClient(opts *Options) Client {
	c := &client{opts: opts, m: newPlain(opts)}
	if opts.UseKerberos {
		c.m = newGSSAPI(opts)
	}
	return c
}

func (c *client) Start(mechlist []string) (string, []byte, bool, error) {
	return c.m.Start()
}

func (c *client) Step(challenge []byte) ([]byte, bool, error) {
	return c.m.Step(challenge)
}

func (c *client) Free() {}

type mech interface {
	Start() (mech string, initial []byte, done bool, err error)
	Step(challenge []byte) (response []byte, done bool, err error)
}

type client struct {
	m    mech
	opts *Options
}
