package k8sutil

import (
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
