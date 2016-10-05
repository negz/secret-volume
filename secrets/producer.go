package secrets

import (
	"errors"

	"github.com/negz/secret-volume/api"
)

var UnhandledSecretSourceError = errors.New("unhandled secret source")

type Producer interface {
	For(*api.Volume) (api.Secrets, error)
}

type Producers map[api.SecretSource]Producer
