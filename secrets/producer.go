package secrets

import (
	"errors"

	"github.com/negz/secret-volume/api"
)

var UnhandledSecretSourceError = errors.New("unhandled secret source")

// A Producer produces secrets files for the supplied api.Volume.
type Producer interface {
	// For returns the appropriate secrets files for the supplied api.Volume.
	For(*api.Volume) (api.Secrets, error)
}

// Producers maps api.SecretSources to the Producer that handles them.
type Producers map[api.SecretSource]Producer
