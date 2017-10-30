/*
Copyright 2017 The vault-operator Authors

Commercial software license.
*/
package v1alpha1

import (
	v1alpha1 "github.com/coreos-inc/vault-operator/pkg/apis/vault/v1alpha1"
	scheme "github.com/coreos-inc/vault-operator/pkg/generated/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// VaultServicesGetter has a method to return a VaultServiceInterface.
// A group's client should implement this interface.
type VaultServicesGetter interface {
	VaultServices(namespace string) VaultServiceInterface
}

// VaultServiceInterface has methods to work with VaultService resources.
type VaultServiceInterface interface {
	Create(*v1alpha1.VaultService) (*v1alpha1.VaultService, error)
	Update(*v1alpha1.VaultService) (*v1alpha1.VaultService, error)
	UpdateStatus(*v1alpha1.VaultService) (*v1alpha1.VaultService, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.VaultService, error)
	List(opts v1.ListOptions) (*v1alpha1.VaultServiceList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.VaultService, err error)
	VaultServiceExpansion
}

// vaultServices implements VaultServiceInterface
type vaultServices struct {
	client rest.Interface
	ns     string
}

// newVaultServices returns a VaultServices
func newVaultServices(c *VaultV1alpha1Client, namespace string) *vaultServices {
	return &vaultServices{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the vaultService, and returns the corresponding vaultService object, and an error if there is any.
func (c *vaultServices) Get(name string, options v1.GetOptions) (result *v1alpha1.VaultService, err error) {
	result = &v1alpha1.VaultService{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("vaultservices").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of VaultServices that match those selectors.
func (c *vaultServices) List(opts v1.ListOptions) (result *v1alpha1.VaultServiceList, err error) {
	result = &v1alpha1.VaultServiceList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("vaultservices").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested vaultServices.
func (c *vaultServices) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("vaultservices").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a vaultService and creates it.  Returns the server's representation of the vaultService, and an error, if there is any.
func (c *vaultServices) Create(vaultService *v1alpha1.VaultService) (result *v1alpha1.VaultService, err error) {
	result = &v1alpha1.VaultService{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("vaultservices").
		Body(vaultService).
		Do().
		Into(result)
	return
}

// Update takes the representation of a vaultService and updates it. Returns the server's representation of the vaultService, and an error, if there is any.
func (c *vaultServices) Update(vaultService *v1alpha1.VaultService) (result *v1alpha1.VaultService, err error) {
	result = &v1alpha1.VaultService{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("vaultservices").
		Name(vaultService.Name).
		Body(vaultService).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *vaultServices) UpdateStatus(vaultService *v1alpha1.VaultService) (result *v1alpha1.VaultService, err error) {
	result = &v1alpha1.VaultService{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("vaultservices").
		Name(vaultService.Name).
		SubResource("status").
		Body(vaultService).
		Do().
		Into(result)
	return
}

// Delete takes name of the vaultService and deletes it. Returns an error if one occurs.
func (c *vaultServices) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("vaultservices").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *vaultServices) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("vaultservices").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched vaultService.
func (c *vaultServices) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.VaultService, err error) {
	result = &v1alpha1.VaultService{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("vaultservices").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
