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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	terraformv1 "github.com/hammadzf/tf-state-rescuer/api/v1"
)

var _ = Describe("StateRescue Controller", func() {
	const (
		StateRescueName      = "test-staterescue"
		StateRescueNamespace = "default"
		SecretName           = "test-secret"
		timeout              = time.Second * 10
		duration             = time.Second * 10
		interval             = time.Millisecond * 250
	)
	Context("When updating StateResuce status", func() {
		It("Should update LastBackupTime and LastRescueTime when TF state secrets are backed up and rescued", func() {
			By("By creating a new StateRescue resource")
			ctx := context.Background()
			stateRescue := &terraformv1.StateRescue{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "terraform.hammadzf.github.io/v1",
					Kind:       "StateRescue",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      StateRescueName,
					Namespace: StateRescueNamespace,
				},
				Spec: terraformv1.StateRescueSpec{
					StateSecretName: SecretName,
				},
			}
			Expect(k8sClient.Create(ctx, stateRescue)).To(Succeed())

			// check if the StateRescue resource has been created
			stateRescueLookupKey := types.NamespacedName{Name: StateRescueName, Namespace: StateRescueNamespace}
			createdStateRescue := &terraformv1.StateRescue{}

			// retry getting this newly created StateRescue resource, given that creation may not immediately happen.
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stateRescueLookupKey, createdStateRescue)).To(Succeed())
			}, timeout, interval).Should(Succeed())
			// make sure our StateSecretName string value was properly converted/handled.
			Expect(createdStateRescue.Spec.StateSecretName).To(Equal(SecretName))

			By("Creating a test Secret containing TF state")
			// create a test secret emulating TF state secret to trigger controller logic
			label := make(map[string]string)
			label["tfstate"] = "true"
			label["app.kubernetes.io/managed-by"] = "terraform"
			testSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      SecretName,
					Namespace: StateRescueNamespace,
					Labels:    label,
				},
			}
			Expect(k8sClient.Create(ctx, testSecret)).To(Succeed())

			// controller should create a backup secret after finding the test secret in the cluster
			By("Controller creating a backup Secret")
			// check if the backup Secret has been created
			// backup secret contains the prefix "backup-"
			// and the lable "tfstate" = "false"
			backupSecretName := "backup-" + SecretName
			backupSecretLookupKey := types.NamespacedName{Name: backupSecretName, Namespace: StateRescueNamespace}
			createdBackupSecret := &corev1.Secret{}

			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, backupSecretLookupKey, createdBackupSecret)).To(Succeed())
			}, timeout, interval).Should(Succeed())
			// Make sure the backup secret has the proper tfstate label
			Expect(createdBackupSecret.Labels["tfstate"]).To(Equal("false"))

			// check that the backup time was updated in the StateRescue resource
			By("Updating the LastBackupTime in the StateRescue resource status")
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stateRescueLookupKey, createdStateRescue)).To(Succeed())
			}, timeout, interval).Should(Succeed())
			// make sure lastBackupTime is not empty
			Expect(createdStateRescue.Status.LastBackupTime.String()).ToNot(BeEmpty())

			By("Rescuing the TF state from last backup")
			// delete the original test secret containing TF state
			Expect(k8sClient.Delete(ctx, testSecret)).To(Succeed())

			// the controller should now rescue the TF state from backup secret
			// and eventually the original TF state secret should be available
			secretLookupKey := types.NamespacedName{Name: SecretName, Namespace: StateRescueNamespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, secretLookupKey, testSecret)).To(Succeed())
			}, timeout, interval).Should(Succeed())
			// Make sure the original secret has the proper tfstate label
			Expect(testSecret.Labels["tfstate"]).To(Equal("true"))

			// check that the rescue time was updated in the StateRescue resource
			By("Updating the LastRescueTime in the StateRescue resource status")
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stateRescueLookupKey, createdStateRescue)).To(Succeed())
			}, timeout, interval).Should(Succeed())
			// make sure lastRescueTime is not empty
			Expect(createdStateRescue.Status.LastRescueTime.String()).ShouldNot(BeEmpty())

			// cleanup (remove StateRescue resource and test Secret)
			// backup secret should automatically be deleted
			By("Cleanup the specific resource instance StateRescue")
			Expect(k8sClient.Delete(ctx, stateRescue)).To(Succeed())

			By("Cleanup the test secret")
			Expect(k8sClient.Delete(ctx, testSecret)).To(Succeed())

			// By("Checking if the backup secret was also deleted")
			// Expect(k8sClient.Get(ctx, backupSecretLookupKey, createdBackupSecret)).To(Succeed())
			// Expect(createdBackupSecret).To(BeNil())
		})
	})
})
