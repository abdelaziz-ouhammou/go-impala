package sasl

import "errors"

// Common SASL errors.
var (
	ErrUnexpectedClientResponse  = errors.New("sasl: unexpected client response")
	ErrUnexpectedServerChallenge = errors.New("sasl: unexpected server challenge")
)

// Mech is SASL mechanism token
type Mech string

// SASL mechanism tokens
const (
	MechPlain    = "PLAIN"
	MechKerberos = "GSSAPI"
)

// Options contains data related to SASL negotiation
type Options struct {
	Service     string
	Host        string
	UseKerberos bool
	Username    string
	Password    string
}

// Client is SASL client
type Client interface {
	Start(mechlist []string) (mech string, initial []byte, done bool, err error)
	Step(challenge []byte) (response []byte, done bool, err error)
	Free()
}
