# `knuu` Package User Guide

The `knuu` package is a powerful and flexible test framework designed to simplify the process of writing integration tests for Go projects. It leverages the capabilities of containers and Kubernetes to create a robust testing environment, abstracting away the complexities of container orchestration and resource management. With Knuu, developers can focus on writing comprehensive integration tests without needing in-depth knowledge of Kubernetes operations, making it an ideal solution for teams looking to enhance their testing practices in containerized environments.

## Table of Contents

- [Installation](#installation)
- [Creating a `Knuu` Object](#creating-a-knuu-object)
- [Handling Stop Signals](#handling-stop-signals)
- [Cleaning Up Resources](#cleaning-up-resources)

## Installation

To use the `knuu` package, ensure you have the necessary dependencies installed:

```bash
go get github.com/celestiaorg/knuu
```

## Creating a `Knuu` Object

To create a `Knuu` object, you need to provide several options encapsulated in the `Options` struct. The object is initialized with a Kubernetes client, MinIO client, image builder, and other configurations.
To use the default knuu object, an empty `Options` struct can be passed to the `New` function.

### Options

- `K8sClient`: A Kubernetes client used to interact with your Kubernetes cluster (_default: k8s config from the running environment i.e. kubeconfig file_).
- `MinioClient`: A custom MinIO client for managing object storage (_default: nil_).
- `ImageBuilder`: A custom builder for creating container images (_default: kaniko builder_).
- `Scope`: A unique identifier for the resources managed by this knuu object (_default: a pseudo random string_).
- `ProxyEnabled`: A boolean to enable or disable a reverse proxy (_default: false_).
- `Timeout`: Duration after which the resources will be automatically cleaned up (_default: 60 minutes_).
- `Logger`: A logger instance for logging (_default: logrus with info level_).

### Example

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/celestiaorg/knuu/pkg/k8s"
    "github.com/celestiaorg/knuu/pkg/minio"
    "github.com/celestiaorg/knuu/pkg/builder"
    "github.com/sirupsen/logrus"
    "github.com/celestiaorg/knuu"
)

func main() {
    ctx := context.Background()

    options := knuu.Options{
        Scope:        "example-scope",
        Timeout:      30 * time.Minute,   // This is an internal timeout for the knuu object to clean up the resources if the program exits unexpectedly
        Logger:       logrus.New().WithLevel(logrus.ErrorLevel),
    }

    kn, err := knuu.New(ctx, options)
    if err != nil {
        log.Fatalf("Failed to create Knuu object: %v", err)
    }
    kn.HandleStopSignal(ctx) // explained in the next section

    defer func() {
        err := kn.CleanUp(ctx)
        if err != nil {
            log.Fatalf("Failed to clean up resources: %v", err)
        }
    }()


    sampleInstance, err := kn.NewInstance("my-builder-instance")
    if err != nil {
        log.Fatalf("Failed to create instance: %v", err)
    }

    // When using a git repo as the builder, the repo is cloned to the builder container and the build is done inside the container
    // It is expected to have a Dockerfile in the root of the repo
    err = sampleInstance.Build().SetGitRepo(ctx, builder.GitContext{
		Repo:     "https://github.com/<sample-repo>.git",
		Branch:   "<desired-branch>",
		Username: "<git-username>", // Leave empty if repo is public
		Password: "<git-password>", // Leave empty if repo is public
	})
    if err != nil {
        log.Fatalf("Failed to build from a git repo: %v", err)
    }

    // optionally can set image directly instead of building from git repo
    // err = sampleInstance.Build().SetImage(ctx, "docker.io/<example-image>:<tag>")
    // if err != nil {
    //     log.Fatalf("Failed to set image: %v", err)
    // }

    err = sampleInstance.Build().SetStartCommand("<start-command>", "<arg1>", "<arg2>",...)
    if err != nil {
        log.Fatalf("Failed to set start command: %v", err)
    }

    err = sampleInstance.Build().SetEnvironmentVariable("<env-var-name>", "<env-var-value>")
    if err != nil {
        log.Fatalf("Failed to set env var: %v", err)
    }

    // Adding file before commit will add it to the builder
    // Therefore the image will be rebuilt with the new file
    err = sampleInstance.Storage().AddFile("<source-path>", "<destination-path>", "<permissions>")
    if err != nil {
        log.Fatalf("Failed to add file: %v", err)
    }

    err = sampleInstance.Build().Commit(ctx)
    if err != nil {
        log.Fatalf("Failed to commit: %v", err)
    }

    // Adding file after commit will add it to the deployment of the instance
    // Therefore the is no image rebuilt, however the file must be very small (config maps are used, so a few KBs are fine)
    // and should not be used to transport large files otherwise the deployment will fail
    err = sampleInstance.Storage().AddFile("<source-path>", "<destination-path>", "<permissions>")
    if err != nil {
        log.Fatalf("Failed to add file: %v", err)
    }

    err = sampleInstance.Execution().Start(ctx)
    if err != nil {
        log.Fatalf("Failed to start instance: %v", err)
    }


    // the rest of the test...
}
```

## Handling Stop Signals

The `knuu` package can handle system signals like `SIGINT` and `SIGTERM` (e.g. when user presses ctrl+c) to perform cleanup operations when the application is stopped. This is useful to ensure that all resources are properly deleted when the application exits.

### Example

```go
kn.HandleStopSignal(ctx)
```

Once this is called, the program will listen for interrupt signals and clean up resources if such a signal is received.

## Cleaning Up Resources

You can manually clean up resources managed by the `Knuu` instance by calling the `CleanUp` method.

### Example

```go
err := kn.CleanUp(ctx)
if err != nil {
    log.Fatalf("Failed to clean up resources: %v", err)
}
```

This method deletes the namespace associated with the `Knuu` object, ensuring that all resources are cleaned up.
