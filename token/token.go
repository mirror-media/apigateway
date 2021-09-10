// Package token define the domain of token
package token

const (
	OK = "OK"
)

const TypeJWT = "JWT"

type Token interface {
	ExecuteTokenStateUpdate() error
	GetTokenString() (string, error)
	GetTokenState() string
	GetSubject() string
	GetEmail() (string, bool)
}
