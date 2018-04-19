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

package k8sutil

import (
	"fmt"
	"strings"

	api "github.com/coreos/vault-operator/pkg/apis/vault/v1alpha1"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CascadeDeleteBackground returns a background delete policy option which causes the garbage collector to delete the dependents in the background.
func CascadeDeleteBackground() *metav1.DeleteOptions {
	return &metav1.DeleteOptions{
		PropagationPolicy: func() *metav1.DeletionPropagation {
			background := metav1.DeletePropagationBackground
			return &background
		}(),
	}
}

// PodDNSName constructs the dns name on which a pod can be addressed
func PodDNSName(p v1.Pod) string {
	podIP := strings.Replace(p.Status.PodIP, ".", "-", -1)
	return fmt.Sprintf("%s.%s.pod", podIP, p.Namespace)
}

// AddOwnerRefToObject appends the desired OwnerReference to the object
func AddOwnerRefToObject(o metav1.Object, r metav1.OwnerReference) {
	o.SetOwnerReferences(append(o.GetOwnerReferences(), r))
}

// AsOwner returns an owner reference set as the vault cluster CR
func AsOwner(v *api.VaultService) metav1.OwnerReference {
	trueVar := true
	return metav1.OwnerReference{
		APIVersion: api.SchemeGroupVersion.String(),
		Kind:       api.VaultServiceKind,
		Name:       v.Name,
		UID:        v.UID,
		Controller: &trueVar,
	}
}
