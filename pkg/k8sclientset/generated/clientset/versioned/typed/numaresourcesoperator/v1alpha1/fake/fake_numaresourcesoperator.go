/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v1alpha1 "github.com/openshift-kni/numaresources-operator/api/numaresourcesoperator/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeNUMAResourcesOperators implements NUMAResourcesOperatorInterface
type FakeNUMAResourcesOperators struct {
	Fake *FakeNumaresourcesoperatorV1alpha1
	ns   string
}

var numaresourcesoperatorsResource = schema.GroupVersionResource{Group: "numaresourcesoperator", Version: "v1alpha1", Resource: "numaresourcesoperators"}

var numaresourcesoperatorsKind = schema.GroupVersionKind{Group: "numaresourcesoperator", Version: "v1alpha1", Kind: "NUMAResourcesOperator"}

// Get takes name of the nUMAResourcesOperator, and returns the corresponding nUMAResourcesOperator object, and an error if there is any.
func (c *FakeNUMAResourcesOperators) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.NUMAResourcesOperator, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(numaresourcesoperatorsResource, c.ns, name), &v1alpha1.NUMAResourcesOperator{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.NUMAResourcesOperator), err
}

// List takes label and field selectors, and returns the list of NUMAResourcesOperators that match those selectors.
func (c *FakeNUMAResourcesOperators) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.NUMAResourcesOperatorList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(numaresourcesoperatorsResource, numaresourcesoperatorsKind, c.ns, opts), &v1alpha1.NUMAResourcesOperatorList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.NUMAResourcesOperatorList{ListMeta: obj.(*v1alpha1.NUMAResourcesOperatorList).ListMeta}
	for _, item := range obj.(*v1alpha1.NUMAResourcesOperatorList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested nUMAResourcesOperators.
func (c *FakeNUMAResourcesOperators) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(numaresourcesoperatorsResource, c.ns, opts))

}

// Create takes the representation of a nUMAResourcesOperator and creates it.  Returns the server's representation of the nUMAResourcesOperator, and an error, if there is any.
func (c *FakeNUMAResourcesOperators) Create(ctx context.Context, nUMAResourcesOperator *v1alpha1.NUMAResourcesOperator, opts v1.CreateOptions) (result *v1alpha1.NUMAResourcesOperator, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(numaresourcesoperatorsResource, c.ns, nUMAResourcesOperator), &v1alpha1.NUMAResourcesOperator{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.NUMAResourcesOperator), err
}

// Update takes the representation of a nUMAResourcesOperator and updates it. Returns the server's representation of the nUMAResourcesOperator, and an error, if there is any.
func (c *FakeNUMAResourcesOperators) Update(ctx context.Context, nUMAResourcesOperator *v1alpha1.NUMAResourcesOperator, opts v1.UpdateOptions) (result *v1alpha1.NUMAResourcesOperator, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(numaresourcesoperatorsResource, c.ns, nUMAResourcesOperator), &v1alpha1.NUMAResourcesOperator{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.NUMAResourcesOperator), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeNUMAResourcesOperators) UpdateStatus(ctx context.Context, nUMAResourcesOperator *v1alpha1.NUMAResourcesOperator, opts v1.UpdateOptions) (*v1alpha1.NUMAResourcesOperator, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(numaresourcesoperatorsResource, "status", c.ns, nUMAResourcesOperator), &v1alpha1.NUMAResourcesOperator{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.NUMAResourcesOperator), err
}

// Delete takes name of the nUMAResourcesOperator and deletes it. Returns an error if one occurs.
func (c *FakeNUMAResourcesOperators) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(numaresourcesoperatorsResource, c.ns, name), &v1alpha1.NUMAResourcesOperator{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeNUMAResourcesOperators) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(numaresourcesoperatorsResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.NUMAResourcesOperatorList{})
	return err
}

// Patch applies the patch and returns the patched nUMAResourcesOperator.
func (c *FakeNUMAResourcesOperators) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.NUMAResourcesOperator, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(numaresourcesoperatorsResource, c.ns, name, pt, data, subresources...), &v1alpha1.NUMAResourcesOperator{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.NUMAResourcesOperator), err
}
