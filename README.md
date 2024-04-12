![knuu-logo](./docs/knuu-logo.png)

[![CodeQL](https://github.com/celestiaorg/knuu/workflows/CodeQL/badge.svg)](https://github.com/celestiaorg/knuu/actions) [![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0) [![OpenSSF Best Practices](https://bestpractices.coreinfrastructure.org/projects/7475/badge)](https://bestpractices.coreinfrastructure.org/projects/7475)

## Description

The goal of knuu is to provide a framework for writing integration tests.
The framework is written in Go and is designed to be used in Go projects.
The idea is to provide a framework that uses the power of containers and Kubernetes without the test writer having to know the details of how to use them.

We invite you to explore our codebase, contribute, and join us in developing a framework to help projects write integration tests.

## Features

Knuu is designed around `Instances`, which you can create, start, control, communicate with other Instances, stop, and destroy.

Some of the features of knuu are:

- Initialize an Instance from a Container/Docker image
- Configure startup commands
- Configure Networking
  - What ports to expose
  - Disable networking to simulate network outages
- Configure Storage
- Execute Commands
- Configure HW resources
- Create a pool of Instances and control them as a group
- Allow AddFile after Commit via ConfigMap
- Implement a TTL value for Pod cleanup
- Add a timeout variable
- See this issue for more upcoming features: [#91](https://github.com/celestiaorg/knuu/issues/91)

> If you have feedback on the framework, want to report a bug, or suggest an improvement, please create an issue [here](https://github.com/celestiaorg/knuu/issues/new/choose).

## Getting Started

This section will guide you on how to set up and run **knuu**.

### Prerequisites

1. **Docker**: Knuu requires Docker to run
   > You can install Docker by following the instructions [here](https://docs.docker.com/get-docker/).

2. **Kubernetes cluster**: Set up access to a Kubernetes cluster using a kubeconfig.
   > In case you have no Kubernets cluster running yet, you can get more information [here](https://kubernetes.io/docs/setup/).

3. **'test' Namespace**: Create a namespace called 'test' in your Kubernetes cluster.
   > **Note:** The used namespace can be changed by setting the `KNUU_NAMESPACE` environment variable.

### Writing Tests

The documentation you can find  [here](https://pkg.go.dev/github.com/celestiaorg/knuu).

Simple example:

1. Add the following to your `go.mod` file:

    ```go
    require (
        github.com/celestiaorg/knuu v0.8.2
        github.com/stretchr/testify v1.8.4
    )
    ```

2. Run `go mod tidy` to download the dependencies.

3. Create a file called `main_test.go` with the following content to initialize knuu:

    ```go
    package main

    import (
        "fmt"
        "os"
        "testing"
        "time"

        "github.com/celestiaorg/knuu/pkg/knuu"
    )

    func TestMain(m *testing.M) {
        err := knuu.Initialize()
        if err != nil {
           log.Fatalf("Error initializing knuu: %v:", err)
        }
        exitVal := m.Run()
        os.Exit(exitVal)
    }
    ```

4. Create a file called `example_test.go` with the following content:

    ```go
   package main

    import (
        "os"
        "testing"

        "github.com/celestiaorg/knuu/pkg/knuu"
        "github.com/stretchr/testify/assert"
    )

    func TestBasic(t *testing.T) {
        t.Parallel()
        // Setup

        instance, err := knuu.NewInstance("alpine")
        if err != nil {
            t.Fatalf("Error creating instance '%v':", err)
        }
        err = instance.SetImage("docker.io/alpine:latest")
        if err != nil {
            t.Fatalf("Error setting image: %v", err)
        }
        err = instance.SetCommand("sleep", "infinity")
        if err != nil {
            t.Fatalf("Error setting command: %v", err)
        }
        err = instance.Commit()
        if err != nil {
            t.Fatalf("Error committing instance: %v", err)
        }

        t.Cleanup(func() {
            // Cleanup
            if os.Getenv("KNUU_SKIP_CLEANUP") == "true" {
                t.Log("Skipping cleanup")
                return
            }

            err = instance.Destroy()
            if err != nil {
                t.Fatalf("Error destroying instance: %v", err)
            }
        })

        // Test logic

        err = instance.Start()
        if err != nil {
            t.Fatalf("Error starting instance: %v", err)
        }
        err = instance.WaitInstanceIsRunning()
        if err != nil {
            t.Fatalf("Error waiting for instance to be running: %v", err)
        }
        wget, err := instance.ExecuteCommand("echo", "Hello World!")
        if err != nil {
            t.Fatalf("Error executing command '%v':", err)
        }

        assert.Equal(t, wget, "Hello World!\n")
    }
    ```

You can find more examples in the following repositories:

- [celestiaorg/knuu-example](https://github.com/celestiaorg/knuu-example)
- [celestiaorg/celestia-app](https://github.com/celestiaorg/celestia-app/tree/main/test/e2e)

### Running Tests

You can use the built-in `go test` command to run the tests.

To run all tests in the current directory, you can run:

```shell
go test -v ./...
```

#### Environment Variables

You can set the following environment variables to change the behavior of knuu:

| Environment Variable        | Description                                                                                                     | Possible Values                  | Default  |
|-----------------------------|-----------------------------------------------------------------------------------------------------------------|----------------------------------|----------|
| `KNUU_TIMEOUT`              | The timeout for the tests.                                                                                      | Any valid duration               | `60m`    |
| `KNUU_NAMESPACE`            | The namespace where the instances will be created.                                                              | Any valid namespace name         | `test`   |
| `KNUU_BUILDER`              | The builder to use for building images.                                                                         | `docker`, `kubernetes`           | `docker` |
| `KNUU_DEDICATED_NAMESPACE`  | Creates and deletes a dedicated namespace for the test after `KNUU_TIMEOUT`. See note below for more details.*	 | `true`, `false`                  | `false`  |º
| `DEBUG_LEVEL`               | The debug level.                                                                                                | `debug`, `info`, `warn`, `error` | `info`   |

* Note on KNUU_DEDICATED_NAMESPACE: 

When set to true, this environment variable tells knuu to create a unique namespace for running tests, which will be automatically deleted after the specified timeout period (`KNUU_TIMEOUT`).
The created namespace will have a prefix of `knuu-` followed by a unique identifier (unique id of the test), which ensures that each test run has its isolated environment within the Kubernetes cluster.
The unique identifier is an autogenerated timestamp that uniquely identifies each test run, however, developers have the option to specify this identifier.

## Contributing

We warmly welcome and appreciate contributions.

By participating in this project, you agree to abide by the [CNCF Code of Conduct](https://github.com/cncf/foundation/blob/main/code-of-conduct.md).

See the [Contributing Guide](./CONTRIBUTING.md) for more information.

To ensure that your contribution is working as expected, please run [knuu-example](https://github.com/celestiaorg/knuu-example) with your fork and branch.

<!---
## Governance

[Describe the governance model for your project. Reference the GOVERNANCE.md file.]

## Adopters

[Provide information about the public adopters of your project. Reference the ADOPTERS.md file.]

## Security and Disclosure Information

See [SECURITY.md](SECURITY.md) for security and disclosure information.
--->

## Licensing

Knuu is licensed under the [Apache License 2.0](LICENSE).

<!---
## Contact

[Provide contact information for the project maintainers.]
--->
