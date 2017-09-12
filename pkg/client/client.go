package client

import (
	"context"

	api "github.com/coreos-inc/vault-operator/pkg/apis/vault/v1alpha1"
	"github.com/coreos-inc/vault-operator/pkg/util/k8sutil"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
)

type Vaults interface {
	RESTClient() *rest.RESTClient

	Get(ctx context.Context, namespace, name string) (*api.VaultService, error)
	Create(ctx context.Context, vault *api.VaultService) (*api.VaultService, error)
	Delete(ctx context.Context, namespace, name string) error
	Update(ctx context.Context, vault *api.VaultService) (*api.VaultService, error)
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
	cli, err := NewCRClient(cfg)
	if err != nil {
		panic(err)
	}
	return cli
}

// NewCRClient create a new vault client based on the Kubernetes client configuration passed in
func NewCRClient(cfg *rest.Config) (Vaults, error) {
	cli, crScheme, err := newClient(cfg)
	if err != nil {
		return nil, err
	}
	return &vaults{
		restCli:    cli,
		crScheme:   crScheme,
		paramCodec: runtime.NewParameterCodec(crScheme),
	}, nil
}

func newClient(cfg *rest.Config) (*rest.RESTClient, *runtime.Scheme, error) {
	crScheme := runtime.NewScheme()
	if err := api.AddToScheme(crScheme); err != nil {
		return nil, nil, err
	}

	config := *cfg
	config.GroupVersion = &api.SchemeGroupVersion
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
func (vs *vaults) Get(ctx context.Context, namespace, name string) (*api.VaultService, error) {
	v := &api.VaultService{}
	err := vs.restCli.Get().Context(ctx).
		Namespace(namespace).
		Resource(api.VaultServicePlural).
		Name(name).
		Do().
		Into(v)
	return v, err
}

// Create creates a Vault resource in the given namespace.
func (vs *vaults) Create(ctx context.Context, vault *api.VaultService) (*api.VaultService, error) {
	nv := &api.VaultService{}
	err := vs.restCli.Post().Context(ctx).
		Namespace(vault.Namespace).
		Resource(api.VaultServicePlural).
		Body(vault).
		Do().
		Into(nv)
	return nv, err
}

// Delete deletes the Vault resource in the given namespace.
func (vs *vaults) Delete(ctx context.Context, namespace, name string) error {
	return vs.restCli.Delete().Context(ctx).
		Namespace(namespace).
		Resource(api.VaultServicePlural).
		Name(name).
		Do().
		Error()
}

// Update updates the Vault resource in the given namespace.
func (vs *vaults) Update(ctx context.Context, vault *api.VaultService) (*api.VaultService, error) {
	nv := &api.VaultService{}
	err := vs.restCli.Put().Context(ctx).
		Namespace(vault.Namespace).
		Resource(api.VaultServicePlural).
		Name(vault.Name).
		Body(vault).
		Do().
		Into(nv)
	return nv, err
}
