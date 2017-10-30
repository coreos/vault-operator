/*
Copyright 2017 The vault-operator Authors

Commercial software license.
*/
package fake

import (
	v1alpha1 "github.com/coreos-inc/vault-operator/pkg/generated/clientset/versioned/typed/vault/v1alpha1"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeVaultV1alpha1 struct {
	*testing.Fake
}

func (c *FakeVaultV1alpha1) VaultServices(namespace string) v1alpha1.VaultServiceInterface {
	return &FakeVaultServices{c, namespace}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeVaultV1alpha1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
