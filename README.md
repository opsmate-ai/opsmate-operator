# opsmate-operator

`opsmate-operator` is a Kubernetes operator for managing and orchestrating [opsmate](https://github.com/opsmate-ai/opsmate) environments and tasks. The operator simplifies the deployment, management, and lifecycle of ephemeral environments for operations tasks.

## Description

The opsmate-operator extends the Kubernetes API with custom resources for managing execution environments (EnvironmentBuild) and tasks (Task). It automates the provisioning of complete environments with required infrastructure (pods, services, ingresses) for running operational tasks within Kubernetes.

The operator maintains the full lifecycle of these environments - from creation, scheduling, running, to termination - and handles service exposure, and task cleanup. It's especially useful for organizations that need consistent, ephemeral Opsmate environments for LLM-powered operational tasks with isolated resources.

## Getting Started

### Prerequisites

- go version v1.22.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

### To Deploy on the cluster

**Build and push your image to GitHub Container Registry:**

```sh
make docker-build docker-push
```

**NOTE:** The image is published to GitHub Container Registry (ghcr.io) by default.
The current image is available at `ghcr.io/jingkaihe/opsmate-controller-manager:0.1.4.alpha2`.
If you're pushing to your own registry, set the `IMG` variable to your desired location.
Make sure you have the proper permissions to the registry if the above commands don't work.

**Install the CRDs into the cluster:**

```sh
make install
```

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy
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

Following are the steps to build the installer and distribute this project to users.

1. Build the installer for the image built and published in the registry:

```sh
make build-installer IMG=ghcr.io/jingkaihe/opsmate-controller-manager:0.1.4.alpha2
```

NOTE: The makefile target mentioned above generates an 'install.yaml'
file in the dist directory. This file contains all the resources built
with Kustomize, which are necessary to install this project without
its dependencies.

2. Using the installer

Users can just run kubectl apply -f <URL for YAML BUNDLE> to install the project, i.e.:

```sh
kubectl apply -f https://raw.githubusercontent.com/jingkaihe/opsmate-operator/main/dist/install.yaml
```

## Contributing
Contributions to opsmate-operator are welcome! Here's how you can contribute:

1. **Fork the Repository**: Create your own fork of the project.
2. **Create a Feature Branch**: Make your changes in a new branch.
3. **Follow the Coding Standards**: Maintain the existing code style and patterns.
4. **Add Tests**: Include tests for new features or bug fixes.
5. **Submit a Pull Request**: Open a PR with a clear description of your changes.

All contributions must pass the existing test suite and linting checks before they can be merged.

**NOTE:** Run `make help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

See [LICENSE](./LICENSE) for details.