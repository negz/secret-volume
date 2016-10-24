package secrets

import (
	"encoding/json"
	"io"

	"github.com/cloudfoundry-incubator/candiedyaml"
	"github.com/negz/secret-volume/api"
	"github.com/pkg/errors"
	"github.com/uber-go/zap"
)

// WriteJSON merges a set of Secrets into a single map, then encodes that map as
// JSON to the supplied io.Writer. If duplicate secret keys are found across the
// files contained by Secrets the value of the first key will take precedence.
func WriteJSON(s api.Secrets, w io.Writer) error {
	m, err := secretsToMap(s)
	if err != nil {
		return errors.Wrap(err, "cannot parse secrets")
	}
	return errors.Wrap(json.NewEncoder(w).Encode(m), "cannot encode secrets to JSON")
}

func secretsToMap(s api.Secrets) (map[string]string, error) {
	m := make(map[string]string)
	for {
		h, err := s.Next()
		if err == io.EOF {
			return m, nil
		}
		if err != nil {
			return nil, errors.Wrap(err, "cannot produce merged secrets map")
		}
		if h.FileInfo.IsDir() {
			continue
		}
		chunk, err := fileToMap(s, h.Type)
		if err != nil {
			log.Debug("cannot parse secret file",
				zap.String("path", h.Path),
				zap.String("type", h.Type.String()),
				zap.Error(err))
			continue
		}
		for k, v := range chunk {
			if _, ok := m[k]; ok {
				// We saw this secret in an earlier file. Leave it intact.
				// TODO(negz): Make the priority configurable, i.e. earlier
				// secrets win or later secrets win.
				continue
			}
			m[k] = v
		}
	}
}

func fileToMap(r io.Reader, t api.SecretType) (map[string]string, error) {
	var d map[string]string
	var err error
	switch t {
	case api.JSONSecretType:
		err = json.NewDecoder(r).Decode(&d)
	case api.YAMLSecretType:
		err = candiedyaml.NewDecoder(r).Decode(&d)
	default:
		return nil, errors.New("unknown secret file type")
	}
	return d, errors.Wrap(err, "cannot decode secret file")
}
