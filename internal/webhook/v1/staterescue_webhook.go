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

package v1

import (
	"context"
	"fmt"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	validationutils "k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	terraformv1 "github.com/hammadzf/tf-state-rescuer/api/v1"
)

// nolint:unused
// log is for logging in this package.
var staterescuelog = logf.Log.WithName("staterescue-resource")

// SetupStateRescueWebhookWithManager registers the webhook for StateRescue in the manager.
func SetupStateRescueWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&terraformv1.StateRescue{}).
		WithValidator(&StateRescueCustomValidator{}).
		Complete()
}

// +kubebuilder:webhook:path=/validate-terraform-hammadzf-github-io-v1-staterescue,mutating=false,failurePolicy=fail,sideEffects=None,groups=terraform.hammadzf.github.io,resources=staterescues,verbs=create;update,versions=v1,name=vstaterescue-v1.kb.io,admissionReviewVersions=v1

// StateRescueCustomValidator struct is responsible for validating the StateRescue resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type StateRescueCustomValidator struct {
}

var _ webhook.CustomValidator = &StateRescueCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type StateRescue.
func (v *StateRescueCustomValidator) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	staterescue, ok := obj.(*terraformv1.StateRescue)
	if !ok {
		return nil, fmt.Errorf("expected a StateRescue object but got %T", obj)
	}
	staterescuelog.Info("Validation for StateRescue upon creation", "name", staterescue.GetName())

	return nil, validateStateRescue(staterescue)
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type StateRescue.
func (v *StateRescueCustomValidator) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	staterescue, ok := newObj.(*terraformv1.StateRescue)
	if !ok {
		return nil, fmt.Errorf("expected a StateRescue object for the newObj but got %T", newObj)
	}
	staterescuelog.Info("Validation for StateRescue upon update", "name", staterescue.GetName())

	return nil, validateStateRescue(staterescue)
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type StateRescue.
func (v *StateRescueCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	staterescue, ok := obj.(*terraformv1.StateRescue)
	if !ok {
		return nil, fmt.Errorf("expected a StateRescue object but got %T", obj)
	}
	staterescuelog.Info("Validation for StateRescue upon deletion", "name", staterescue.GetName())

	// nothing to validate at delete

	return nil, nil
}

func validateStateRescue(sr *terraformv1.StateRescue) error {
	var allErrors field.ErrorList
	if err := validateStateRescueName(sr); err != nil {
		allErrors = append(allErrors, err)
	}
	if err := validateStateRescueSpec(sr); err != nil {
		allErrors = append(allErrors, err)
	}
	if len(allErrors) == 0 {
		return nil
	}
	return apierrors.NewInvalid(
		schema.GroupKind{Group: "terraform.hammadzf.github.io", Kind: "StateRescue"},
		sr.Name, allErrors,
	)
}

func validateStateRescueName(sr *terraformv1.StateRescue) *field.Error {
	if len(sr.Name) > validationutils.DNS1035LabelMaxLength {
		return field.Invalid(field.NewPath("metadata").Child("name"), sr.Name, "must not be longer than 63 characters")
	}
	return nil

}

func validateStateRescueSpec(sr *terraformv1.StateRescue) *field.Error {
	// The secret name in the StateRescue spec must follow the format `tfstate-{workspace}-{secret_suffix}`
	// to conform with the nomenclature that Terraform uses for naming secrets containing state file data
	// (https://developer.hashicorp.com/terraform/language/backend/kubernetes#configuration-variables)
	sp := strings.Split(sr.Spec.StateSecretName, "-")
	if len(sp) < 3 {
		return field.Invalid(field.NewPath("spec").Child("stateSecretName"), sr.Spec.StateSecretName, "does not match the format 'tfstate-{workspace}-{secret_suffix}'")
	}
	if sp[0] != "tfstate" {
		return field.Invalid(field.NewPath("spec").Child("stateSecretName"), sr.Spec.StateSecretName, "does not match the format 'tfstate-{workspace}-{secret_suffix}'")
	}
	return nil
}
