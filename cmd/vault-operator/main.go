package main

import (
	"context"
	"runtime"

	"github.com/coreos-inc/vault-operator/pkg/operator"

	"github.com/Sirupsen/logrus"
)

func main() {
	// nothing interesting
	logrus.Infof("Go Version: %s", runtime.Version())
	logrus.Infof("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)

	v := operator.New()
	err := v.Start(context.TODO())
	if err != nil {
		logrus.Infof("operator stopped with %v", err)
	}
}
