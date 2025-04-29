# opsmate-operator

> **⚠️ This project is under active development. Features are subject to change. Use at your own risk.**

`opsmate-operator` is a Kubernetes operator for managing and orchestrating [opsmate](https://github.com/opsmate-ai/opsmate) environments and tasks. The operator simplifies the deployment, management, and lifecycle of ephemeral environments for operations tasks.

## Description

The opsmate-operator extends the Kubernetes API with custom resources for managing execution environments (EnvironmentBuild) and tasks (Task). It automates the provisioning of complete environments with required infrastructure (pods, services, ingresses) for running operational tasks within Kubernetes.

The operator maintains the full lifecycle of these environments - from creation, scheduling, running, to termination - and handles service exposure, and task cleanup. It's especially useful for organizations that need consistent, ephemeral Opsmate environments for LLM-powered operational tasks with isolated resources.

## Install using Helm

The project is distributed as a Helm chart, which is automatically published to GitHub Container Registry on each push to main branch.

To install using Helm:

```sh
helm install opsmate-operator oci://ghcr.io/opsmate-ai/opsmate-operator/opsmate-operator --version 0.2.0
```

## Up and Running

Checkout the [Opsmate Production Guide](https://docs.tryopsmate.ai/production/#environment-build) to find out to have on-demand Opsmate environments running in your Kubernetes cluster.

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
