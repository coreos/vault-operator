package k8sutil

import (
	"fmt"
	"strings"

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
