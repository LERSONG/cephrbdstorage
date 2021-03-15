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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
	"time"
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

func (r *CephRBDStorageReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	logf := r.Log.WithValues("cephrbdstorage", req.NamespacedName)

	// your logic here
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

	if err := r.reconcileSecret(instance); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.reconcileStorageClass(instance); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *CephRBDStorageReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&yamecloudv1.CephRBDStorage{}).
		Owns(&storagev1.StorageClass{}).
		Complete(r)
}

func (r *CephRBDStorageReconciler) reconcileStorageClass(instance *cephv1.CephRBDStorage) error {
	storageClassName := instance.Spec.StorageClassName
	if storageClassName == "" {
		return nil
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
	}

	return nil
}

func (r *CephRBDStorageReconciler) reconcileSecret(instance *yamecloudv1.CephRBDStorage) error {
	storageClassName := instance.Spec.StorageClassName
	if storageClassName == "" {
		return nil
	}

	parameters := instance.Spec.Parameters
	if len(parameters) == 0 {
		return nil
	}
	adminSecretName := parameters["adminSecretName"]
	if adminSecretName == "" {
		adminSecretName = generateAdminSecretName(storageClassName)
		secret := &corev1.Secret{}
		err := r.Client.Get(context.Background(), types.NamespacedName{Namespace: "kube-system", Name: adminSecretName}, secret)
		if err != nil {
			if errors.IsNotFound(err) {
				secret.Name = adminSecretName
				secret.Namespace = "kube-system"
				secret.Type = "kubernetes.io/rbd"
				secretData := make(map[string]string, 0)
				adminSecretData := parameters["adminSecretData"]
				if adminSecretData == "" {
					return fmt.Errorf("adminSecretData must have value")
				}
				secretData["key"] = adminSecretData
				secret.StringData = secretData
				if err := r.Client.Create(context.Background(), secret); err != nil {
					return err
				}
			} else {
				return err
			}
		}
		parameters["adminSecretName"] = adminSecretName
	}
	userSecretName := parameters["userSecretName"]
	if userSecretName == "" {
		userSecretData := parameters["userSecretData"]
		if userSecretData == "" {
			return fmt.Errorf("userSecretData must have value")
		}
		userSecretName = generateUserSecretName(storageClassName)
		parameters["userSecretName"] = userSecretName
	}

	err := r.Client.Update(context.Background(), instance)
	if err != nil {
		return err
	}

	return nil
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
	//todo: generate adminSecretName/userSecretName
	storageClass.Parameters = instance.Spec.Parameters
	return storageClass
}
