// +build debug

package secrets

import "github.com/uber-go/zap"

var log = zap.New(
	zap.NewTextEncoder(),
	zap.AddCaller(),
	zap.DebugLevel,
)
