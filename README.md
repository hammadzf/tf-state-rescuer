# tf-state-rescuer
TF State Rescuer is a Kubernetes operator that backs up Kubernetes Secrets containing Terraform state files in your Kubernetes cluster, and rescues the state in case original Secrets are deleted for some reason.

> Disclaimer: This is a pet project that I created while playing around with kubebuilder to build custom operators. The controller logic can be seen as a minimal working solution that has not been rigorously tested, especially its behavior against Terraform's state locking feature in case of dynamic updates to state Secrets. If you find any bugs or security issues, please raise an issue or preferebly a merge request.

## Description
The TF State Rescuer operator is built using [kubebuilder](https://book.kubebuilder.io/). A custom resource called StateRescue is used to specify the name of Terraform state Secret(s) that the corresponding controller should monitor for backup and rescue. This name should match the naming convention that Terraform uses for creating Kubernetes Secrets when the Kubernetes backend is used to store state remotely, which is `tfstate-{workspace}-{secret_suffix}` and the `secret-suffix` is the one which is specified by the Terraform user when configuring the Kubernetes backend. Refer to [Terraform documentation](https://developer.hashicorp.com/terraform/language/backend/kubernetes) for configuring a Kubernetes backend.

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

Once this StateRescue resource is created, the controller will monitor the corresponding Kubernetes Secret(s) containing state files for the Terraform project that is using the Kubernetes backend. In the above example, the controller looks for Secrets in the 'terraform' namespace as this is the namespace where the StateRescue resource is created. These Secrets are backed up by creating copies in the same namespace, and the LastBackupTime field in StateRescue resource's Status is updated accordingly. The controller looks out for any changes made in the Secret(s) containing Terraform state and updates backup Secrets accordingly in order to keep the latest state. 

In case the Secrets being read/updated by Terraform for keeping state are deleted for some reason, the controller rescues Terraform state from the the backup Secrets, and the LastRescueTime field in StateRescue's Status is updated accordingly.


## Getting Started

### Prerequisites
- go version v1.24.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

### To Deploy on the cluster
**Build and push your image to the location specified by `IMG`:**

```sh
make docker-build docker-push IMG=<some-registry>/tf-state-rescuer:tag
```

**NOTE:** This image ought to be published in the personal registry you specified.
And it is required to have access to pull the image from the working environment.
Make sure you have the proper permission to the registry if the above commands don’t work.

**Install the CRDs into the cluster:**

```sh
make install
```

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy IMG=<some-registry>/tf-state-rescuer:tag
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin
privileges or be logged in as admin.

**Create instances of your solution**
You can apply the samples (examples) from the config/sample:

```sh
kubectl apply -k config/samples/
```

>**NOTE**: Ensure that the samples has default values to test it out.

### To Uninstall
**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k config/samples/
```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```

## Project Distribution

Following the options to release and provide this solution to the users.

### By providing a bundle with all YAML files

1. Build the installer for the image built and published in the registry:

```sh
make build-installer IMG=<some-registry>/tf-state-rescuer:tag
```

**NOTE:** The makefile target mentioned above generates an 'install.yaml'
file in the dist directory. This file contains all the resources built
with Kustomize, which are necessary to install this project without its
dependencies.

2. Using the installer

Users can just run 'kubectl apply -f <URL for YAML BUNDLE>' to install
the project, i.e.:

```sh
kubectl apply -f https://raw.githubusercontent.com/<org>/tf-state-rescuer/<tag or branch>/dist/install.yaml
```

### By providing a Helm Chart

1. Build the chart using the optional helm plugin

```sh
kubebuilder edit --plugins=helm/v1-alpha
```

2. See that a chart was generated under 'dist/chart', and users
can obtain this solution from there.

**NOTE:** If you change the project, you need to update the Helm Chart
using the same command above to sync the latest changes. Furthermore,
if you create webhooks, you need to use the above command with
the '--force' flag and manually ensure that any custom configuration
previously added to 'dist/chart/values.yaml' or 'dist/chart/manager/manager.yaml'
is manually re-applied afterwards.

**NOTE:** Run `make help` for more information on all potential `make` targets

## Contributing
Contributions are welcome. Feel free to open a pull request.

## License

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

