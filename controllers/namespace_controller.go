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
	"encoding/json"
	"fmt"
	cephv1 "github.com/LERSONG/cephrbdstorage/api/v1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	AllowStorageClassAnnotationName = "fuxi.kubernetes.io/default_storage_limit"
)

// CephRBDStorageReconciler reconciles a CephRBDStorage object
type NamespaceReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=,resources=namespaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=,resources=namespaces/status,verbs=get;update;patch

func (r *NamespaceReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	logf := r.Log.WithValues("namespace", req.NamespacedName)

	// your logic here
	instance := &corev1.Namespace{}
	if err := r.Client.Get(ctx, req.NamespacedName, instance); err != nil {
		if errors.IsNotFound(err) {
			logf.Info("receive request could not be found namespace resource or non specified resources",
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

	return ctrl.Result{}, nil
}

func (r *NamespaceReconciler) syncSecret(instance *corev1.Namespace) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		annotations := instance.GetAnnotations()
		allowStorageClass, ok := annotations[AllowStorageClassAnnotationName]
		if ok {
			var scArr []string
			if err := json.Unmarshal([]byte(allowStorageClass), &scArr); err != nil {
				return err
			}
			storageClass := &storagev1.StorageClass{}
			for _, sc := range scArr {
				if sc == "" {
					continue
				}
				if err := r.Client.Get(context.Background(), types.NamespacedName{Name: sc}, storageClass); err != nil {
					return err
				}
				references := storageClass.GetOwnerReferences()
				if references != nil && len(references) > 0 {
					for _, reference := range references {
						if reference.Kind == "CephRBDStorage" {
							cephRBDStorage := &cephv1.CephRBDStorage{}
							if err := r.Client.Get(context.Background(), types.NamespacedName{Name: reference.Name, Namespace: "kube-system"}, cephRBDStorage); err != nil {
								continue
							}
							parameters := cephRBDStorage.Spec.Parameters
							userSecretName := parameters[UserSecretName]
							if userSecretName == "" {
								return fmt.Errorf("userSecretName must have value")
							}
							secret := &corev1.Secret{}
							if err := r.Client.Get(context.Background(), types.NamespacedName{Name: userSecretName, Namespace: instance.Name}, secret); err != nil {
								if errors.IsNotFound(err) {
									secret.Name = userSecretName
									secret.Namespace = instance.Name
									secret.Type = SecretType
									labels := make(map[string]string, 0)
									labels[LabelHideKey] = LabelHideValue
									secret.SetLabels(labels)
									secretData := make(map[string]string, 0)
									userSecretValue := parameters[UserSecretValue]
									if userSecretValue == "" {
										return fmt.Errorf("userSecretData must have value")
									}
									secretData[SecretDataKey] = userSecretValue
									secret.StringData = secretData
									if err := r.Client.Create(context.Background(), secret); err != nil {
										return err
									}
								} else {
									return err
								}
							} else {
								userSecretValue := parameters[UserSecretValue]
								if userSecretValue != "" {
									data := secret.Data
									if data == nil {
										data = make(map[string][]byte, 0)
									}
									if userSecretValue == string(data[SecretDataKey]) {
										return nil
									}
									data[SecretDataKey] = []byte(userSecretValue)
									secret.Data = data
									if err := r.Client.Update(context.Background(), secret); err != nil {
										return err
									}
								}
							}

						}
					}
				}

			}
		}
		return nil
	})

}

func (r *NamespaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Namespace{}).
		Complete(r)
}
