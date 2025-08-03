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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StateRescueSpec defines the desired state of StateRescue
type StateRescueSpec struct {
	// Important: Run "make" to regenerate code after modifying this file
	// The following markers will use OpenAPI v3 schema to validate the value
	// More info: https://book.kubebuilder.io/reference/markers/crd-validation.html

	// specifies the name of the secret object containing terraform state file
	// is determined from terraform Kubernetes backend configurations (secret_suffix)
	// +required
	StateSecretName string `json:"stateSecretName,omitempty"`
}

// StateRescueStatus defines the observed state of StateRescue.
type StateRescueStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// time when the state file secrets were last backed up
	// +optional
	LastBackupTime metav1.Time `json:"lastBackupTime,omitempty"`
	// time when the state files were last rescued from backup
	// +optional
	LastRescueTime metav1.Time `json:"lastRescueTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// StateRescue is the Schema for the staterescues API
type StateRescue struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// spec defines the desired state of StateRescue
	// +required
	Spec StateRescueSpec `json:"spec"`

	// status defines the observed state of StateRescue
	// +optional
	Status StateRescueStatus `json:"status,omitempty,omitzero"`
}

// +kubebuilder:object:root=true

// StateRescueList contains a list of StateRescue
type StateRescueList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []StateRescue `json:"items"`
}

func init() {
	SchemeBuilder.Register(&StateRescue{}, &StateRescueList{})
}
