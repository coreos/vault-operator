package upgradetest

import (
	"os"
	"testing"

	"github.com/coreos-inc/vault-operator/test/e2e/upgradetest/framework"

	"github.com/sirupsen/logrus"
)

func TestMain(m *testing.M) {
	if err := framework.Setup(); err != nil {
		logrus.Errorf("fail to setup framework: %v", err)
		os.Exit(1)
	}

	code := m.Run()

	if err := framework.TearDown(); err != nil {
		logrus.Errorf("fail to teardown framework: %v", err)
		os.Exit(1)
	}
	os.Exit(code)
}
