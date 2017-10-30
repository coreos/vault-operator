/*
Copyright 2017 The vault-operator Authors

Commercial software license.
*/
package fake

import (
	v1alpha1 "github.com/coreos-inc/vault-operator/pkg/apis/vault/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeVaultServices implements VaultServiceInterface
type FakeVaultServices struct {
	Fake *FakeVaultV1alpha1
	ns   string
}

var vaultservicesResource = schema.GroupVersionResource{Group: "vault.security.coreos.com", Version: "v1alpha1", Resource: "vaultservices"}

var vaultservicesKind = schema.GroupVersionKind{Group: "vault.security.coreos.com", Version: "v1alpha1", Kind: "VaultService"}

// Get takes name of the vaultService, and returns the corresponding vaultService object, and an error if there is any.
func (c *FakeVaultServices) Get(name string, options v1.GetOptions) (result *v1alpha1.VaultService, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(vaultservicesResource, c.ns, name), &v1alpha1.VaultService{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.VaultService), err
}

// List takes label and field selectors, and returns the list of VaultServices that match those selectors.
func (c *FakeVaultServices) List(opts v1.ListOptions) (result *v1alpha1.VaultServiceList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(vaultservicesResource, vaultservicesKind, c.ns, opts), &v1alpha1.VaultServiceList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.VaultServiceList{}
	for _, item := range obj.(*v1alpha1.VaultServiceList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested vaultServices.
func (c *FakeVaultServices) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(vaultservicesResource, c.ns, opts))

}

// Create takes the representation of a vaultService and creates it.  Returns the server's representation of the vaultService, and an error, if there is any.
func (c *FakeVaultServices) Create(vaultService *v1alpha1.VaultService) (result *v1alpha1.VaultService, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(vaultservicesResource, c.ns, vaultService), &v1alpha1.VaultService{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.VaultService), err
}

// Update takes the representation of a vaultService and updates it. Returns the server's representation of the vaultService, and an error, if there is any.
func (c *FakeVaultServices) Update(vaultService *v1alpha1.VaultService) (result *v1alpha1.VaultService, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(vaultservicesResource, c.ns, vaultService), &v1alpha1.VaultService{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.VaultService), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeVaultServices) UpdateStatus(vaultService *v1alpha1.VaultService) (*v1alpha1.VaultService, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(vaultservicesResource, "status", c.ns, vaultService), &v1alpha1.VaultService{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.VaultService), err
}

// Delete takes name of the vaultService and deletes it. Returns an error if one occurs.
func (c *FakeVaultServices) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(vaultservicesResource, c.ns, name), &v1alpha1.VaultService{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeVaultServices) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(vaultservicesResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.VaultServiceList{})
	return err
}

// Patch applies the patch and returns the patched vaultService.
func (c *FakeVaultServices) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.VaultService, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(vaultservicesResource, c.ns, name, data, subresources...), &v1alpha1.VaultService{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.VaultService), err
}
