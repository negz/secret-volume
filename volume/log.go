// +build !debug

package volume

import "github.com/uber-go/zap"

var log = zap.New(zap.NewJSONEncoder())
