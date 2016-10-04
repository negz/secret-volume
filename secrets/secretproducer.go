package secrets

import (
	"errors"

	"github.com/negz/secret-volume/api"
)

var UnhandledSecretSourceError = errors.New("unhandled secret source")

type SecretProducer interface {
	For(*api.Volume) (api.Secrets, error)
}

type SecretProducers map[api.SecretSource]SecretProducer
