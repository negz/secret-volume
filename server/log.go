// +build !debug

package server

import "github.com/uber-go/zap"

var log = zap.New(zap.NewJSONEncoder())
