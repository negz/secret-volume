// +build debug

package cmd

import "github.com/uber-go/zap"

var log = zap.New(
	zap.NewTextEncoder(),
	zap.AddCaller(),
	zap.DebugLevel,
)
