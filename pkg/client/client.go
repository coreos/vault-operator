package client

import (
	"github.com/coreos-inc/vault-operator/pkg/spec"
	"github.com/coreos-inc/vault-operator/pkg/util/k8sutil"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
)

type Vaults interface {
	RESTClient() *rest.RESTClient
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
