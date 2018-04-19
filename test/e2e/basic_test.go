// Copyright 2018 The vault-operator Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package e2e

import (
	"testing"

	api "github.com/coreos/vault-operator/pkg/apis/vault/v1alpha1"
	"github.com/coreos/vault-operator/test/e2e/e2eutil"
	"github.com/coreos/vault-operator/test/e2e/framework"
)

func TestCreateHAVault(t *testing.T) {
	f := framework.Global
	vaultCR, tlsConfig, rootToken := e2eutil.SetupUnsealedVaultCluster(t, f.KubeClient, f.VaultsCRClient, f.Namespace)
	defer func(vaultCR *api.VaultService) {
		if err := e2eutil.DeleteCluster(t, f.VaultsCRClient, vaultCR); err != nil {
			t.Fatalf("failed to delete vault cluster: %v", err)
		}
	}(vaultCR)
	vClient, keyPath, secretData, podName := e2eutil.WriteSecretData(t, vaultCR, f.KubeClient, tlsConfig, rootToken, f.Namespace)
	e2eutil.VerifySecretData(t, vClient, secretData, keyPath, podName)
}
