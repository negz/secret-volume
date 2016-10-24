package secrets

import (
	"io"
	"testing"

	"github.com/spf13/afero"

	"github.com/negz/secret-volume/api"
	"github.com/negz/secret-volume/fixtures"
)

var tarGzTests = []struct {
	f     string
	v     *api.Volume
	files int
	t     api.SecretType
}{
	{"../fixtures/yaml.tar.gz", fixtures.TestVolume, 3, api.YAMLSecretType},
}

func TestTarGz(t *testing.T) {
	for _, tt := range tarGzTests {
		fs := afero.NewOsFs()
		sd, _ := OpenTarGz(tt.v, fs, tt.f, TarGzSecretType(tt.t))
		t.Run("Extract", func(t *testing.T) {
			got := 0
			for {
				h, err := sd.Next()
				if err == io.EOF {
					if got != tt.files {
						t.Errorf("want %v, got %v", tt.files, got)
					}
					return
				}
				if err != nil {
					t.Errorf("sd.Next(): %v", err)
					return
				}

				if h.Type != tt.t {
					t.Errorf("h.Type: want %v, got %v", tt.t, h.Type)
				}

				got++
			}
		})
		t.Run("Close", func(t *testing.T) {
			if err := sd.Close(); err != nil {
				t.Errorf("sd.Close(): %v", err)
			}
		})
	}
}
