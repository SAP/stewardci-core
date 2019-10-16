# Develop Steward

## Folder structure

The project "Steward" sources are structured in the following folders:

- backend-k8s

  This folder contains the components which need to be applied to a k8s cluster to provide the backend.

- cmd

  The go coding for the command line / image for the pipeline executor

- docs/example

  This folder contains examples how to interact with the service

- pkg/apis

  The go api for the pipelineRun object

- pkg/client (generated)

  This folder contains the generated clientset, informers and listers for project "Steward". The generation is done via `hack/update-codegen.sh`.

  The controllers use client-go library extensively. The details of interaction points of the controllers with various mechanisms from this library are explained [here][sample-controller].

- pkg/runctl & pkg/tenantctl

  The go implementation of the controllers



## Contribution

You are welcome to contribute to this project via Pull Requests.


## Development

### Prerequisites

```sh
# Prepare Code Generator
git clone https://github.com/kubernetes/code-generator.git
cd code-generator/

# We need a specific version matching to our Tekton version (currently kubernetes-1.13.11)
#
# Unfortunately old versions are not yet modularized.
#     We take the module info from a newer release.
git checkout kubernetes-1.13.11
git checkout kubernetes-1.16.1 -- go.mod go.sum

# CODEGEN_PKG is used by the script to find the code-generator
export CODEGEN_PKG=${PWD}
```

```sh
# Prepare gomock tool
go get github.com/golang/mock/gomock
```

### Build

To run build and test simply execute `./build.sh` from the project root folder.

To build only the controllers run:

```sh
# Build the run controller executable
go build -o runController ./cmd/run_controller/

# Build the tenant controller executable
go build -o tenantController ./cmd/tenant_controller/
```

### Code Generation

The generated clients and mocks have been committed into the project sources. Generation is not necessary in every build, but for some changes (e.g. API changes) the clients and mocks need to be generate again (and committed). This can be done using:

```sh
# Client generation
hack/update-codegen.sh
```


### Test

```sh
go test -coverprofile coverage.txt ./...
go tool cover -html=coverage.txt -o coverage.html
```


## Known Issues

### 'unknown escape sequence' during generation

For some reason `\` characters are generated into imports on Windows.
Those are interpreted as (wrong) escape chars which fails the generation.
```go
import (
	stewardv1alpha1 "github.com/SAP/stewardci-core\..."
)
```

Solution: Linux or Ubuntu sub system on Windows. Cygwin does not help.
See also [issue #68](https://github.com/kubernetes/code-generator/issues/68)




[sample-controller]: https://github.com/kubernetes/sample-controller/blob/master/docs/controller-client-go.md
