/*


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

package v1

import (
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CephRBDStorageSpec defines the desired state of CephRBDStorage
type CephRBDStorageSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// StorageClassName
	// +optional
	StorageClassName string `json:"storageClassName,omitempty"`
	// Provisioner indicates the type of the provisioner.
	Provisioner string `json:"provisioner" protobuf:"bytes,2,opt,name=provisioner"`

	// Parameters holds the parameters for the provisioner that should
	// create volumes of this storage class.
	// +optional
	Parameters map[string]string `json:"parameters,omitempty" protobuf:"bytes,3,rep,name=parameters"`

	// Dynamically provisioned PersistentVolumes of this storage class are
	// created with this reclaimPolicy. Defaults to Delete.
	// +optional
	ReclaimPolicy *v1.PersistentVolumeReclaimPolicy `json:"reclaimPolicy,omitempty" protobuf:"bytes,4,opt,name=reclaimPolicy,casttype=k8s.io/api/core/v1.PersistentVolumeReclaimPolicy"`

	// VolumeBindingMode indicates how PersistentVolumeClaims should be
	// provisioned and bound.  When unset, VolumeBindingImmediate is used.
	// This field is only honored by servers that enable the VolumeScheduling feature.
	// +optional
	VolumeBindingMode *storagev1.VolumeBindingMode `json:"volumeBindingMode,omitempty" protobuf:"bytes,7,opt,name=volumeBindingMode"`

	//provisioner: ceph.com/rbd
	//parameters:
	//adminId: admin
	//adminSecretName: ceph-admin-secret
	//adminSecretNamespace: kube-system
	//imageFeatures: layering
	//imageFormat: '2'
	//monitors: '10.200.45.30:6789,10.200.45.31:6789,10.200.45.32:6789'
	//pool: kube
	//userId: kube
	//userSecretName: ceph-kube-secret
	//reclaimPolicy: Retain
	//volumeBindingMode: Immediate
}

// CephRBDStorageStatus defines the observed state of CephRBDStorage
type CephRBDStorageStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true

// CephRBDStorage is the Schema for the cephrbdstorages API
type CephRBDStorage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CephRBDStorageSpec   `json:"spec,omitempty"`
	Status CephRBDStorageStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CephRBDStorageList contains a list of CephRBDStorage
type CephRBDStorageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CephRBDStorage `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CephRBDStorage{}, &CephRBDStorageList{})
}
