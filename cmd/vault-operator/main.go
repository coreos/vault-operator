package main

import (
	"context"
	"runtime"

	"github.com/coreos-inc/vault-operator/pkg/operator"
	"github.com/coreos-inc/vault-operator/version"

	"github.com/Sirupsen/logrus"
)

func main() {
	logrus.Infof("Go Version: %s", runtime.Version())
	logrus.Infof("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)
	logrus.Infof("vault-operator Version: %v", version.Version)
	logrus.Infof("Git SHA: %s", version.GitSHA)

	v := operator.New()
	err := v.Start(context.TODO())
	if err != nil {
		logrus.Infof("operator stopped with %v", err)
	}
}
