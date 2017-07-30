package client

import (
	"context"

	"github.com/coreos-inc/vault-operator/pkg/spec"
	"github.com/coreos-inc/vault-operator/pkg/util/k8sutil"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
)

type Vaults interface {
	RESTClient() *rest.RESTClient

	Get(ctx context.Context, namespace, name string) (*spec.Vault, error)
	Create(ctx context.Context, vault *spec.Vault) (*spec.Vault, error)
	Delete(ctx context.Context, namespace, name string) error
	Update(ctx context.Context, vault *spec.Vault) (*spec.Vault, error)
}

type vaults struct {
	restCli    *rest.RESTClient
	crScheme   *runtime.Scheme
	paramCodec runtime.ParameterCodec
}

func (v *vaults) RESTClient() *rest.RESTClient {
	return v.restCli
}

func MustNewInCluster() Vaults {
	cfg, err := k8sutil.InClusterConfig()
	if err != nil {
		panic(err)
	}
	cli, crScheme, err := newClient(cfg)
	if err != nil {
		panic(err)
	}
	return &vaults{
		restCli:    cli,
		crScheme:   crScheme,
		paramCodec: runtime.NewParameterCodec(crScheme),
	}
}

func newClient(cfg *rest.Config) (*rest.RESTClient, *runtime.Scheme, error) {
	crScheme := runtime.NewScheme()
	if err := spec.AddToScheme(crScheme); err != nil {
		return nil, nil, err
	}

	config := *cfg
	config.GroupVersion = &spec.SchemeGroupVersion
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: serializer.NewCodecFactory(crScheme)}

	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, nil, err
	}

	return client, crScheme, nil
}

// Get returns a Vault resource for the given name in the given namespace.
func (vs *vaults) Get(ctx context.Context, namespace, name string) (*spec.Vault, error) {
	v := &spec.Vault{}
	err := vs.restCli.Get().Context(ctx).
		Namespace(namespace).
		Resource(spec.VaultResourcePlural).
		Name(name).
		Do().
		Into(v)
	return v, err
}

// Create creates a Vault resource in the given namespace.
func (vs *vaults) Create(ctx context.Context, vault *spec.Vault) (*spec.Vault, error) {
	nv := &spec.Vault{}
	err := vs.restCli.Post().Context(ctx).
		Namespace(vault.Namespace).
		Resource(spec.VaultResourcePlural).
		Body(vault).
		Do().
		Into(nv)
	return nv, err
}

// Delete deletes the Vault resource in the given namespace.
func (vs *vaults) Delete(ctx context.Context, namespace, name string) error {
	return vs.restCli.Delete().Context(ctx).
		Namespace(namespace).
		Resource(spec.VaultResourcePlural).
		Name(name).
		Do().
		Error()
}

// Update updates the Vault resource in the given namespace.
func (vs *vaults) Update(ctx context.Context, vault *spec.Vault) (*spec.Vault, error) {
	nv := &spec.Vault{}
	err := vs.restCli.Put().Context(ctx).
		Namespace(vault.Namespace).
		Resource(spec.VaultResourcePlural).
		Name(vault.Name).
		Body(vault).
		Do().
		Into(nv)
	return nv, err
}
