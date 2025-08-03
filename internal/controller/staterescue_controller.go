/*
Copyright 2025.

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

package controller

import (
	"context"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	terraformv1 "github.com/hammadzf/tf-state-rescuer/api/v1"
)

const (
	TfStateLabelKey   = "app.kubernetes.io/managed-by"
	TfStateLabelValue = "terraform"
)

// StateRescueReconciler reconciles a StateRescue object
type StateRescueReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=terraform.hammadzf.github.io,resources=staterescues,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=terraform.hammadzf.github.io,resources=staterescues/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=terraform.hammadzf.github.io,resources=staterescues/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets/data,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *StateRescueReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	stateSecrets := &corev1.SecretList{}
	originalSecrets := &corev1.SecretList{}
	backupSecrets := &corev1.SecretList{}

	// load the state rescuer object
	var stateRescue terraformv1.StateRescue
	if err := r.Get(ctx, req.NamespacedName, &stateRescue); err != nil {
		if errors.IsNotFound(err) {
			log.Info("State rescue resource not found in the requested namespace")
			return ctrl.Result{}, nil
		} else {
			log.Error(err, "unable to fetch StateRescue resource from the requested namespace")
			return ctrl.Result{}, err
		}
	}

	// Load Kubernetes secrets that contains terraform state files in the state rescue namespace
	if err := r.List(ctx, stateSecrets, client.InNamespace(stateRescue.Namespace), client.MatchingLabels{TfStateLabelKey: TfStateLabelValue}); err != nil {
		if errors.IsNotFound(err) {
			log.Error(err, "no secrets containing tf state found in the namespace of state rescue resource")
			return ctrl.Result{}, client.IgnoreNotFound(err)
		} else {
			log.Error(err, "unable to fetch secrets in the state rescue resource namespace")
			return ctrl.Result{}, err
		}
	}
	// Sort them in original and backup secrets
	for _, item := range stateSecrets.Items {
		if _, found := strings.CutPrefix(item.Name, stateRescue.Spec.StateSecretName); found {
			originalSecrets.Items = append(originalSecrets.Items, item)
		} else if _, found := strings.CutPrefix(item.Name, "backup-"+stateRescue.Spec.StateSecretName); found {
			backupSecrets.Items = append(backupSecrets.Items, item)
		}
	}
	// call backup and rescue logic to complete reconcilliation process
	return r.backupAndRescue(ctx, stateRescue, originalSecrets, backupSecrets)
}

// SetupWithManager sets up the controller with the Manager.
func (r *StateRescueReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&terraformv1.StateRescue{}).
		Owns(&corev1.Secret{}).
		Watches(
			&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
				log := logf.FromContext(ctx)
				// check if the secret is associated with a terraform state contains TF state label
				if val, ok := obj.GetLabels()[TfStateLabelKey]; ok && val == TfStateLabelValue {
					var stateRescueList terraformv1.StateRescueList
					if err := r.List(ctx, &stateRescueList); err != nil {
						if errors.IsNotFound(err) {
							log.Info("no state rescue resources found")
							return []reconcile.Request{}
						} else {
							log.Error(err, "unable to list stateRescue resources")
						}
					}
					for _, item := range stateRescueList.Items {
						if _, found := strings.CutPrefix(obj.GetName(), item.Spec.StateSecretName); found {
							log.Info("TF state secret triggered a reconciliation event",
								"Namespace", obj.GetNamespace(), "Secret", obj.GetName(),
							)
							return []reconcile.Request{
								{
									NamespacedName: types.NamespacedName{
										Name:      item.Name,
										Namespace: item.Namespace,
									},
								},
							}
						}
					}
				}
				return []reconcile.Request{}
			}),
		).
		Named("staterescue").
		Complete(r)
}

// backupSecretForStaterescue returns a secret object for creating backup of an original secret
// handled as a separate function to create a binding between backup objects and the CR
// once the StateResuce CR is deleted, the controller will automatically delete backup objects
// that were created during the lifecycle of the StateRescue resource
func (r *StateRescueReconciler) backupsecretForStaterescue(ctx context.Context, staterescue *terraformv1.StateRescue, secret *corev1.Secret) (*corev1.Secret, error) {
	log := logf.FromContext(ctx)
	backupString := "backup-"
	// create backup Secret object
	backupSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        backupString + secret.Name,
			Namespace:   secret.Namespace,
			Labels:      secret.Labels,
			Annotations: secret.Annotations,
		},
		Data: secret.Data,
	}
	// Set the ownerRef for the backup Secret, ensuring that the
	// Secret will be deleted when the StateRescue CR is deleted.
	if err := controllerutil.SetControllerReference(staterescue, backupSecret, r.Scheme); err != nil {
		log.Error(err, "could not set controller reference for the backup secret")
		return nil, err
	}
	return backupSecret, nil
}

// logic for creating backup secrets and rescuing originals if they are deleted
// original secrets have the tfstate label set to true while it is false for backup secrets
// to avoid issues when reading/updating state in the original secret(s) by the terraform client
func (r *StateRescueReconciler) backupAndRescue(ctx context.Context, stateRescue terraformv1.StateRescue, original *corev1.SecretList, backup *corev1.SecretList) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// check if original secret is missing against a backup one
	// and rescue the original from back up if needed
	for _, item := range backup.Items {
		origSecretNameStr := strings.TrimPrefix(item.Name, "backup-")
		originalSecret := &corev1.Secret{}

		if err := r.Get(ctx, types.NamespacedName{Name: origSecretNameStr, Namespace: item.Namespace}, originalSecret); err != nil {
			if errors.IsNotFound(err) {
				log.Info("original secret with terraform state not found in the state rescue namespace")
				// create state secret object from backup data
				originalSecret = &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:        origSecretNameStr,
						Namespace:   item.Namespace,
						Labels:      item.Labels,
						Annotations: item.Annotations,
					},
					Data: item.Data,
				}
				// update tfstate label to true for the original secret
				originalSecret.Labels["tfstate"] = "true"
				// create secret
				log.Info("creating an original secret from backup secret", "Secret", item.Name)
				if err := r.Create(ctx, originalSecret); err != nil {
					log.Error(err, "unable to create the original secret")
					return ctrl.Result{}, err
				}
				// update rescue time
				stateRescue.Status.LastRescueTime = metav1.Now()
				if err := r.Status().Update(ctx, &stateRescue); err != nil {
					log.Error(err, "unable to update state rescue resource")
					return ctrl.Result{}, err
				}
			} else {
				log.Error(err, "unable to fetch the original secret")
				return ctrl.Result{}, err
			}
		}
	}

	// check if backup secrets exist against the original ones
	// create or update backup secrets if not found
	for _, item := range original.Items {
		backupSecret := &corev1.Secret{}
		if err := r.Get(ctx, types.NamespacedName{Name: "backup-" + item.Name, Namespace: item.Namespace}, backupSecret); err != nil {
			if errors.IsNotFound(err) {
				// create backup secret for the original one
				if backupSecret, err = r.backupsecretForStaterescue(ctx, &stateRescue, &item); err != nil {
					log.Error(err, "unable to fetch backup secret object")
					return ctrl.Result{}, err
				}
				// update tfstate label to false for the backup secret
				backupSecret.Labels["tfstate"] = "false"
				log.Info("Creating the backup state secret for the original secret", "Secret", item.Name)
				if err := r.Create(ctx, backupSecret); err != nil {
					log.Error(err, "unable to create the backup secret")
					return ctrl.Result{}, err
				}
				// update backup time
				stateRescue.Status.LastBackupTime = metav1.Now()
				if err := r.Status().Update(ctx, &stateRescue); err != nil {
					log.Error(err, "unable to update state rescue resource")
					return ctrl.Result{}, err
				}
				// continue to the next iteration
				continue
			} else {
				log.Error(err, "unable to fetch backup secret")
				return ctrl.Result{}, err
			}
		}
		// if backup secret already exists, then only update its data
		log.Info("Updating the backup secret of the original secret", "Secret", item.Name)
		// copy data of original state file secret to backup secret
		backupSecret.Data = item.Data
		if err := r.Update(ctx, backupSecret); err != nil {
			log.Error(err, "unable to update backup secret")
			return ctrl.Result{}, err
		}
		// update backup time
		stateRescue.Status.LastBackupTime = metav1.Now()
		if err := r.Update(ctx, &stateRescue); err != nil {
			log.Error(err, "unable to update state rescue resource")
			return ctrl.Result{}, err
		}
	}

	// successfully return after updating backup and rescuing
	return ctrl.Result{}, nil

}
