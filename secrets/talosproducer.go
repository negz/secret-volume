package secrets

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/net/context/ctxhttp"

	"github.com/negz/secret-volume/api"
	"github.com/pkg/errors"

	"github.com/benschw/srv-lb/lb"
	"github.com/uber-go/zap"
)

type talosProducer struct {
	lb  lb.LoadBalancer
	ctx context.Context
}

// A TalosProducerOption represents an argument to NewTalosProducer.
type TalosProducerOption func(sp *talosProducer) error

// WithContext provides an alternative parent context.Context() for HTTP
// requests to Talos. context.Background() is used by default.
func WithContext(ctx context.Context) TalosProducerOption {
	return func(sp *talosProducer) error {
		sp.ctx = ctx
		return nil
	}
}

// NewTalosProducer builds a Producer backed by https://github.com/spotify/talos
// The supplied lb.LoadBalancer should return the address of a Talos HTTP
// backend.
func NewTalosProducer(lb lb.LoadBalancer, spo ...TalosProducerOption) (Producer, error) {
	sp := &talosProducer{lb, context.Background()}
	for _, o := range spo {
		if err := o(sp); err != nil {
			return nil, errors.Wrapf(err, "cannot apply Talos producer option")
		}
	}
	return sp, nil
}

func httpClientFor(v *api.Volume) (*http.Client, error) {
	crt, err := v.KeyPair.ToCertificate()
	if err != nil {
		return nil, errors.Wrapf(err, "cannot parse keypair for %v", v)
	}

	cfg := &tls.Config{Certificates: []tls.Certificate{crt}, InsecureSkipVerify: true}
	cfg.BuildNameToCertificate()

	return &http.Client{Transport: &http.Transport{TLSClientConfig: cfg}}, nil
}

func (sp *talosProducer) url(tags url.Values) (string, error) {
	h, err := sp.lb.Next()
	if err != nil {
		return "", errors.Wrap(err, "cannot determine next talos endpoint")
	}
	return fmt.Sprintf("https://%v?%v", h, tags.Encode()), nil
}

func (sp *talosProducer) For(v *api.Volume) (api.Secrets, error) {
	url, err := sp.url(v.Tags)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot build URL for %v", v.Tags)
	}
	log.Debug("fetching secrets", zap.String("url", url))
	ctx, cancel := context.WithTimeout(sp.ctx, 15*time.Second)
	defer cancel()
	c, err := httpClientFor(v)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot build HTTP client for %v", v)
	}
	r, err := ctxhttp.Get(ctx, c, url)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot fetch secrets from %v", url)
	}
	s, err := NewTarGz(v, r.Body)
	return s, errors.Wrap(err, "cannot build tar.gz secrets")
}
