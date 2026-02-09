# tf-state-rescuer
TF State Rescuer is a Kubernetes operator that backs up Kubernetes Secrets containing Terraform state files in your Kubernetes cluster, and rescues the state in case original Secrets are deleted for some reason.

> Disclaimer: This is a pet project that I created while playing around with kubebuilder to build custom operators. The controller logic can be seen as a minimal working solution that has not been rigorously tested, especially its behavior against Terraform's state locking feature in case of dynamic updates to state Secrets. If you find any bugs or security issues, please raise an issue or preferebly a merge request.

## Description
The TF State Rescuer operator is built using [kubebuilder](https://book.kubebuilder.io/). A custom resource called StateRescue is used to specify the name of Terraform state Secret(s) that the corresponding controller should monitor for backup and rescue. This name should match the naming convention that Terraform uses for creating Kubernetes Secrets when the Kubernetes backend is used to store state remotely, which is `tfstate-{workspace}-{secret_suffix}` and the `secret-suffix` is the one which is specified by the Terraform user when configuring the Kubernetes backend. Refer to [Terraform documentation](https://developer.hashicorp.com/terraform/language/backend/kubernetes) for configuring a Kubernetes backend.

### Example manifest for StateRescue resource
An example YAML manifest for a StateRescue resource is provided below:  
```yaml
apiVersion: terraform.hammadzf.github.io/v1
kind: StateRescue
metadata:
  labels:
    app.kubernetes.io/name: tf-state-rescuer
    app.kubernetes.io/managed-by: kustomize
  name: staterescue-example
  namespace: terraform
spec:
  stateSecretName: "tfstate-default-state"
```

### How does it work?
Once this StateRescue resource is created, the controller will monitor the corresponding Kubernetes Secret(s) containing state files for the Terraform project that is using the Kubernetes backend. In the above example, the controller looks for Secrets in the 'terraform' namespace as this is the namespace where the StateRescue resource is created. These Secrets are backed up by creating copies in the same namespace, and the `LastBackupTime` field in StateRescue resource's Status is updated accordingly. The controller looks out for any changes made in the Secret(s) containing Terraform state and updates backup Secrets accordingly in order to keep the latest state. 

In case the Secrets being read/updated by Terraform for keeping state are deleted for some reason, the controller rescues Terraform state from the the backup Secrets, and the `LastRescueTime` field in StateRescue's Status is updated accordingly.

### Admission Controller (ValidatingAdmissionWebhook)
The controller manager for this operator also implements a validation webhook for admission control. It validates incoming (Create and Update) requests to the API server for the StateRescue custom resource. Two kinds of validation are performed, one on the name of the object of StateRescue custom resource and the other regarding its specification. 
- Name: Name of an object whose kind/resource is defined by a CRD must also be a valid DNS subdomain name ([source](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/#customresourcedefinitions)).
- Spec: The secret name in the StateRescue spec must follow the format `tfstate-{workspace}-{secret_suffix}` to conform with the nomenclature that Terraform uses for naming secrets containing state file data ([source](https://developer.hashicorp.com/terraform/language/backend/kubernetes#configuration-variables)).

The working of the validation webhook can be verified by attempting to create StateRescue objects with invalid name and spec using manifests in [config/samples](./config/samples/).


## Getting Started

### Prerequisites
- go version v1.24.0+
- docker version 17.03+
- kubectl version v1.11.3+
- Access to a Kubernetes v1.11.3+ cluster
- cert-manager installed on the Kubernetes cluster (for webhooks)

### To Deploy on the cluster

**Install cert-manager**

You can follow the [cert-manager documentation](https://cert-manager.io/docs/installation/) to install it.

**Build and push your image to the location specified by `IMG`:**

*Option#1: Remote image registry*

```sh
make docker-build docker-push IMG=<some-registry>/tf-rescuer:stable-1.0
```

>**NOTE:** This image ought to be published in the personal registry you specified.
>And it is required to have access to pull the image from the working environment.
>Make sure you have the proper permission to the registry if the above commands don’t work.

> **NOTE:** Update the `values.yaml` file with the appropriate value for image registry in the [helm/chart](./helm/chart/) directory.   

*Option#2: Building and loading the image to minikube (for dev and testing)*

Alternatively you can use minikube (or kind) for local dev and testing purposes.
In that case, you can first create the local image 

```sh
make docker-build IMG=tf-rescuer:stable-1.0
minikube image load tf-rescuer:stable-1.0
```

**Install the Helm Chart**

```sh
helm install <release-name> ./helm/chart/
```
**Create StateRescue resources**

To create a sample StateRescue custom resource, you can apply the samples (examples) from the config/sample:

```sh
kubectl apply -k config/samples/terraform_v1_staterescue.yaml
```

### To Uninstall

**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k config/samples/terraform_v1_staterescue.yaml
```

**Delete the APIs(CRDs) and controller (plus other K8s resources) from the cluster:**

```sh
helm uninstall <release-name>
```

**Delete the CRDs from the cluster:**

```sh
make uninstall
```

**Uninstall the cert-manager**

You can follow the [cert-manager documentation](https://cert-manager.io/docs/installation/kubectl/#uninstalling) to uninstall it.

## Contributing
Contributions are welcome. Feel free to open a pull request.

## License

Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

