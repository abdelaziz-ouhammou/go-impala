package sasl

import (
	"bytes"
	"fmt"

	"github.com/golang-auth/go-gssapi/v2"
	_ "github.com/golang-auth/go-gssapi/v2/krb5"
)

type kerberos struct {
	opts *Options
	ctx  gssapi.Mech
}

func newGSSAPI(opts *Options) mech {
	return &kerberos{ctx: gssapi.NewMech("kerberos_v5"), opts: opts}
}

func (m *kerberos) Start() (string, []byte, bool, error) {
	err := m.ctx.Initiate(m.opts.Service+"/"+m.opts.Host,
		gssapi.ContextFlagSequence|gssapi.ContextFlagMutual|gssapi.ContextFlagConf, nil)
	if err != nil {
		return "", nil, false, fmt.Errorf("sasl: kerberos: %v", err)
	}
	token, err := m.ctx.Continue(nil)
	if err != nil {
		return "", nil, false, fmt.Errorf("sasl: kerberos: %v", err)
	}
	return "GSSAPI", token, false, nil
}

func (m *kerberos) Step(challenge []byte) ([]byte, bool, error) {
	token, err := m.ctx.Continue(challenge)
	if err != nil {
		return nil, false, fmt.Errorf("sasl: kerberos: %v", err)
	}
	//check if we need to unwrap the token and send it back
	if len(token) == 0 && bytes.Equal(challenge[0:2], []byte{0x05, 0x04}) {
		token, _, err = m.ctx.Unwrap(challenge)
		if err != nil {
			return nil, false, fmt.Errorf("sasl: kerberos: gssapi: unwrap: %v", err)
		}
		token, err = m.ctx.Wrap(token, false)
		if err != nil {
			return nil, false, fmt.Errorf("sasl: kerberos: gssapi: wrap: %v", err)
		}
		return token, true, nil
	}
	return token, false, nil
}
