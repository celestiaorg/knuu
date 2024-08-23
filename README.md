![knuu-logo](./docs/knuu-logo.png)

![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/celestiaorg/knuu)
![GitHub Release](https://img.shields.io/github/v/release/celestiaorg/knuu)
[![CodeQL](https://github.com/celestiaorg/knuu/workflows/CodeQL/badge.svg)](https://github.com/celestiaorg/knuu/actions) [![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0) [![OpenSSF Best Practices](https://bestpractices.coreinfrastructure.org/projects/7475/badge)](https://bestpractices.coreinfrastructure.org/projects/7475)

---

## Description

The goal of Knuu is to provide a framework for writing integration tests.
The framework is written in Go and is designed to be used in Go projects.
The idea is to provide a framework that uses the power of containers and Kubernetes without the test writer having to know the details of how to use them.

We invite you to explore our codebase, contribute, and join us in developing a framework to help projects write integration tests.

---

## Features

Knuu is designed around `Instances`, which you can create, start, control, communicate with other Instances, stop, and destroy.

Some of the features of Knuu are:

- Initialize an Instance from a Container/Docker image or a git repository
- Configure startup commands
- Configure Networking
  - What ports to expose
  - Shape the network traffic
  - Disable networking to simulate network outages
- Configure Storage
- Execute Commands
- Configure HW resources
- Allow AddFile after Commit via ConfigMap
- Implement a TTL value for Pod cleanup
- Add a timeout variable

> If you have feedback on the framework, want to report a bug, or suggest an improvement, please create an issue [here](https://github.com/celestiaorg/knuu/issues/new/choose).

---

## Getting Started

This section will guide you on how to set up and run **Knuu**.

---

### Prerequisites

- **Kubernetes cluster**: Set up access to a Kubernetes cluster using a `kubeconfig`.
   > In case you have no Kubernetes cluster running yet, you can get more information [here](https://kubernetes.io/docs/setup/).


### Writing Tests

More details and examples on the new knuu can be found [here](./docs/knuu-new.md)

And some more real-world examples can be found in the following repositories:

- [celestiaorg/knuu e2e tests](https://github.com/celestiaorg/knuu/tree/main/e2e)
- [celestiaorg/celestia-app e2e tests](https://github.com/celestiaorg/celestia-app/tree/main/test/e2e)

---

### Running Tests

Depending on how you write your tests, you can use the built-in `go test` command to run the tests.

To run all tests in the current directory, you can run:

```shell
go test -v ./... -timeout=<timeout>m
```

**Note 1:** The timeout is set to 10 minutes by default. Make sure to set a timeout that is long enough to complete the test.
**Note 2:** Please note that the timeout flag in the example is the go test timeout and not related to the knuu timeout.

#### Environment Variables

You can set the following environment variable to change the behavior of knuu:

| Environment Variable | Description | Possible Values | Default |
| --- | --- | --- | --- |
| `LOG_LEVEL` | The debug level. | `debug`, `info`, `warn`, `error` | `info` |

**Note:** `knuu` does not load `.env` file, if you want to set environment variables, you can set them directly in the code or use a `.env` file and load it using `go-dotenv`.

---


# E2E

In the folder `e2e`, you will find some examples of how to use the [knuu](https://github.com/celestiaorg/knuu) Integration Test Framework.

## Setup

Set up access to a Kubernetes cluster using your `kubeconfig` and create the `test` namespace.

## Write Tests

You can find the relevant documentation in the `pkg/knuu` package at: <https://pkg.go.dev/github.com/celestiaorg/knuu>

## Run

You can use the Makefile commands to easily target whatever test by setting the pkg, run, or count flags.

Targeting a directory

```shell
make test pkgs=./e2e/basic
```

Targeting a Test in a directory

```shell
make test pkgs=./e2e/basic run=TestJustThisTest
```

Run a test in a loop to debug

```shell
make test pkgs=./e2e/basic run=TestJustThisTest10Times count=10
```

---

## Contributing

We warmly welcome and appreciate contributions.

By participating in this project, you agree to abide by the [CNCF Code of Conduct](https://github.com/cncf/foundation/blob/main/code-of-conduct.md).

See the [Contributing Guide](./CONTRIBUTING.md) for more information.

To ensure that your contribution is working as expected, please run the tests in the `e2e` folder.

<!---
## Governance

[Describe the governance model for your project. Reference the GOVERNANCE.md file.]

## Adopters

[Provide information about the public adopters of your project. Reference the ADOPTERS.md file.]

## Security and Disclosure Information

See [SECURITY.md](SECURITY.md) for security and disclosure information.
--->

## Licensing

Knuu is licensed under the [Apache License 2.0](./LICENSE).

<!---
## Contact

[Provide contact information for the project maintainers.]
--->
