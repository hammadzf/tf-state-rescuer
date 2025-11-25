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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	terraformv1 "github.com/hammadzf/tf-state-rescuer/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("StateRescue Webhook", func() {
	var (
		validObj       *terraformv1.StateRescue
		newValidObj    *terraformv1.StateRescue
		invalidNameObj *terraformv1.StateRescue
		invalidSpecObj *terraformv1.StateRescue
		validator      StateRescueCustomValidator
	)

	Context("When creating or updating StateRescue under Validating Webhook", func() {
		It("Should deny creation of StateRescue object if its name is not a valid DNS subdomain name", func() {
			By("simulating creation of StateRescue object with invalid name")
			invalidNameObj = &terraformv1.StateRescue{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "terraform.hammadzf.github.io/v1",
					Kind:       "StateRescue",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "!invalid-name",
				},
				Spec: terraformv1.StateRescueSpec{
					StateSecretName: "tf-default-state",
				},
			}
			Expect(validator.ValidateCreate(ctx, invalidNameObj)).Error().To(HaveOccurred())
		})
		It("Should deny creation of StateRescue object if the Terraform state secret name in its spec is not in a valid format", func() {
			By("simulating creation of StateRescue object with invalid spec")
			invalidSpecObj = &terraformv1.StateRescue{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "terraform.hammadzf.github.io/v1",
					Kind:       "StateRescue",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "valid-name",
				},
				Spec: terraformv1.StateRescueSpec{
					StateSecretName: "terraform-default-state",
				},
			}
			Expect(validator.ValidateCreate(ctx, invalidSpecObj)).Error().To(HaveOccurred())
		})
		It("Should admit creation of StateRescue object if the name and spec are valid", func() {
			By("simulating a valid creation scenario")
			validObj = &terraformv1.StateRescue{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "terraform.hammadzf.github.io/v1",
					Kind:       "StateRescue",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "valid-name",
				},
				Spec: terraformv1.StateRescueSpec{
					StateSecretName: "tfstate-default-state",
				},
			}
			Expect(validator.ValidateCreate(ctx, validObj)).To(BeNil())
		})
		It("Should validate updates correctly", func() {
			By("simulating a valid update scenario")
			newValidObj = &terraformv1.StateRescue{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "terraform.hammadzf.github.io/v1",
					Kind:       "StateRescue",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "valid-name",
				},
				Spec: terraformv1.StateRescueSpec{
					StateSecretName: "tfstate-default-state-0",
				},
			}
			Expect(validator.ValidateUpdate(ctx, validObj, newValidObj)).To(BeNil())
		})
	})

})
