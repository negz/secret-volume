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
}{
	{"../fixtures/yaml.tar.gz", fixtures.TestVolume, 3},
}

func TestTarGzSecrets(t *testing.T) {
	for _, tt := range tarGzTests {
		fs := afero.NewOsFs()
		sd, err := OpenTarGzSecrets(tt.v, fs, tt.f)
		if err != nil {
			t.Errorf("OpenTarGzSecrets(%v, %v, %v): %v", tt.v, fs, tt.f, err)
			continue
		}
		t.Run("Extract", func(t *testing.T) {
			found := 0
			for {
				h, err := sd.Next()
				if err == io.EOF {
					if found != tt.files {
						t.Errorf("found == %v, wanted %v", found, tt.files)
					}
					return
				}
				if err != nil {
					t.Errorf("sd.Next(): %v", err)
					return
				}
				t.Logf("Found: %v", h.Path)
				found += 1
			}
		})
		t.Run("Close", func(t *testing.T) {
			if err = sd.Close(); err != nil {
				t.Errorf("sd.Close(): %v", err)
			}
		})
	}
}
