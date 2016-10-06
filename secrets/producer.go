// Package secrets handles the production of and access to collections of
// secrets. i.e. files containing sensitive data such as passwords.
package secrets

import "github.com/negz/secret-volume/api"

// A Producer produces secrets files for the supplied api.Volume.
type Producer interface {
	// For returns the appropriate secrets files for the supplied api.Volume.
	For(*api.Volume) (api.Secrets, error)
}

// Producers maps api.SecretSources to the Producer that handles them.
type Producers map[api.SecretSource]Producer
