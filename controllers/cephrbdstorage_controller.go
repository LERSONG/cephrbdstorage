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

package controllers

import (
	"context"
	"fmt"
	cephv1 "github.com/LERSONG/cephrbdstorage/api/v1"
	yamecloudv1 "github.com/LERSONG/cephrbdstorage/api/v1"
	"github.com/LERSONG/cephrbdstorage/utils"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
	"time"
)

const (
	AdminSecretName  = "adminSecretName"
	AdminSecretValue = "adminSecretValue"
	UserSecretName   = "userSecretName"
	UserSecretValue  = "userSecretValue"
	SecretType       = "kubernetes.io/rbd"
	SecretDataKey    = "key"
	LabelHideKey     = "hide"
	LabelHideValue   = "1"
)

// CephRBDStorageReconciler reconciles a CephRBDStorage object
type CephRBDStorageReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=yamecloud.io,resources=cephrbdstorages,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=yamecloud.io,resources=cephrbdstorages/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=storage.k8s.io,resources=storages,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=storage.k8s.io,resources=storages/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=,resources=secrets/status,verbs=get;update;patch

func (r *CephRBDStorageReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	logf := r.Log.WithValues("cephrbdstorage", req.NamespacedName)

	if req.Namespace == "" {
		req.Namespace = "kube-system"
	}
	instance := &cephv1.CephRBDStorage{}
	if err := r.Client.Get(ctx, req.NamespacedName, instance); err != nil {
		if errors.IsNotFound(err) {
			logf.Info("receive request could not be found cephrbdstorage resource or non specified resources",
				"namespace",
				req.Namespace, "name", req.Name,
			)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if err := r.syncSecret(instance); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.syncStorageClass(instance); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *CephRBDStorageReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&yamecloudv1.CephRBDStorage{}).
		Owns(&storagev1.StorageClass{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}

func (r *CephRBDStorageReconciler) syncStorageClass(instance *cephv1.CephRBDStorage) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		instanceChanged := false
		storageClassName := instance.Spec.StorageClassName
		if storageClassName == "" {
			storageClassName = generateStorageClassName(instance.Name)
			instanceChanged = true
			instance.Spec.StorageClassName = storageClassName
		}
		storageClass := &storagev1.StorageClass{}
		namespacedName := types.NamespacedName{
			Namespace: "",
			Name:      storageClassName,
		}
		if err := r.Client.Get(context.Background(), namespacedName, storageClass); err != nil {
			if errors.IsNotFound(err) {
				storageClass = makeStorageClass(instance)
				if err := r.Client.Create(context.Background(), storageClass); err != nil {
					return err
				}
			} else {
				return err
			}
		} else {
			changed, err := r.checkChangeForStorageClass(instance, storageClass)
			if err != nil {
				return err
			}
			if changed {
				if err := r.Client.Update(context.Background(), storageClass); err != nil {
					return err
				}
			}
		}
		if instanceChanged {
			if err := r.Client.Update(context.Background(), instance); err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *CephRBDStorageReconciler) syncSecret(instance *yamecloudv1.CephRBDStorage) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		instanceChanged := false
		parameters := instance.Spec.Parameters
		if len(parameters) == 0 {
			return nil
		}
		adminSecretName := parameters[AdminSecretName]
		if adminSecretName == "" {
			adminSecretName = generateAdminSecretName(instance.Name)
			parameters[AdminSecretName] = adminSecretName
			instanceChanged = true
		}
		secret := &corev1.Secret{}
		err := r.Client.Get(context.Background(), types.NamespacedName{Namespace: "kube-system", Name: adminSecretName}, secret)
		if err != nil {
			if errors.IsNotFound(err) {
				secret.Name = adminSecretName
				secret.Namespace = "kube-system"
				labels := make(map[string]string, 0)
				labels[LabelHideKey] = LabelHideValue
				secret.SetLabels(labels)
				secret.SetOwnerReferences(utils.OwnerReference(instance, "CephRBDStorage"))
				secret.Type = SecretType

				if parameters[AdminSecretValue] != "" {
					secretData := make(map[string]string, 0)
					secretData[SecretDataKey] = parameters[AdminSecretValue]
					secret.StringData = secretData
				}
				if err := r.Client.Create(context.Background(), secret); err != nil {
					return err
				}
			} else {
				return err
			}
		} else {
			changed, err := r.checkChangeForSecret(instance, secret)
			if err != nil {
				return err
			}
			if changed {
				if err := r.Client.Update(context.Background(), secret); err != nil {
					return err
				}
			}
		}

		userSecretName := parameters[UserSecretName]
		if userSecretName == "" {
			userSecretData := parameters[UserSecretValue]
			if userSecretData == "" {
				return fmt.Errorf("userSecretValue must have value")
			}
			userSecretName = generateUserSecretName(instance.Name)
			parameters[UserSecretName] = userSecretName
			instanceChanged = true
		}
		if instanceChanged {
			instance.Spec.Parameters = parameters
			if err := r.Client.Update(context.Background(), instance); err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *CephRBDStorageReconciler) checkChangeForStorageClass(instance *yamecloudv1.CephRBDStorage, storageClass *storagev1.StorageClass) (bool, error) {
	isChange := false
	if storageClass.Provisioner != instance.Spec.Provisioner {
		isChange = true
		storageClass.Provisioner = instance.Spec.Provisioner
	}
	if storageClass.ReclaimPolicy != instance.Spec.ReclaimPolicy {
		isChange = true
		storageClass.ReclaimPolicy = instance.Spec.ReclaimPolicy
	}
	if storageClass.VolumeBindingMode != instance.Spec.VolumeBindingMode {
		isChange = true
		storageClass.VolumeBindingMode = instance.Spec.VolumeBindingMode
	}

	if storageClass.Parameters == nil && instance.Spec.Parameters == nil {
		return isChange, nil
	}
	if instance.Spec.Parameters == nil {
		isChange = true
		storageClass.Parameters = nil
		return isChange, nil
	}
	if storageClass.Parameters == nil {
		storageClass.Parameters = makeStorageClassParameters(instance.Spec.Parameters)
		isChange = true
		return isChange, nil
	}

	for key, value := range instance.Spec.Parameters {
		if key == AdminSecretValue || key == UserSecretValue {
			continue
		}
		if storageClass.Parameters[key] != value {
			isChange = true
			storageClass.Parameters[key] = value
		}
	}

	return isChange, nil
}

func (r *CephRBDStorageReconciler) checkChangeForSecret(instance *yamecloudv1.CephRBDStorage, secret *corev1.Secret) (bool, error) {
	changed := false
	adminSecretValue := instance.Spec.Parameters[AdminSecretValue]
	data := secret.Data
	if data == nil {
		data = make(map[string][]byte)
	}
	if adminSecretValue != string(data[SecretDataKey]) {
		changed = true
		data[SecretDataKey] = []byte(adminSecretValue)
	}

	return changed, nil
}

func generateStorageClassName(name string) string {
	return name + "-storage-class-" + strconv.FormatInt(time.Now().Unix(), 10)
}

func generateAdminSecretName(storageClassName string) string {
	return storageClassName + "-admin-secret-" + strconv.FormatInt(time.Now().Unix(), 10)
}

func generateUserSecretName(storageClassName string) string {
	return storageClassName + "-user-secret-" + strconv.FormatInt(time.Now().Unix(), 10)
}

func makeStorageClass(instance *yamecloudv1.CephRBDStorage) *storagev1.StorageClass {
	storageClass := &storagev1.StorageClass{
		TypeMeta: metav1.TypeMeta{APIVersion: "storage.k8s.io/v1", Kind: "StorageClass"},
		ObjectMeta: metav1.ObjectMeta{
			Name:            instance.Spec.StorageClassName,
			OwnerReferences: utils.OwnerReference(instance, "CephRBDStorage"),
		},
	}
	storageClass.Provisioner = instance.Spec.Provisioner
	storageClass.ReclaimPolicy = instance.Spec.ReclaimPolicy
	storageClass.VolumeBindingMode = instance.Spec.VolumeBindingMode
	storageClass.Parameters = makeStorageClassParameters(instance.Spec.Parameters)
	return storageClass
}

func makeStorageClassParameters(crsParameters map[string]string) map[string]string {
	if crsParameters == nil {
		return nil
	}
	scParameters := make(map[string]string, 0)
	for key, value := range crsParameters {
		if key == AdminSecretValue || key == UserSecretValue {
			continue
		}
		scParameters[key] = value
	}
	return scParameters
}
