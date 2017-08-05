package k8sutil

import (
	"fmt"
	"strings"

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
func PodDNSName(podIP, namespace string) string {
	podIP = strings.Replace(podIP, ".", "-", -1)
	return fmt.Sprintf("%s.%s.pod", podIP, namespace)
}
