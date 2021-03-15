package utils

import (
	cephv1 "github.com/LERSONG/cephrbdstorage/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func OwnerReference(obj metav1.Object, kindName string) []metav1.OwnerReference {
	return []metav1.OwnerReference{
		*metav1.NewControllerRef(obj,
			schema.GroupVersionKind{
				Group:   cephv1.GroupVersion.Group,
				Version: cephv1.GroupVersion.Version,
				Kind:    kindName,
			}),
	}
}
