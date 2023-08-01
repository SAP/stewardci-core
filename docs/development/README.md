# Develop Steward

## Folder structure

The project "Steward" sources are structured in the following folders:

- cmd

  The go coding for the command line / image for the pipeline executor

- docs/example

  This folder contains examples how to interact with the service

- pkg/apis

  The go api for the pipelineRun object

- pkg/client (generated)

  This folder contains the generated clientset, informers and listers for project "Steward". The generation is done via `hack/update-codegen.sh`.

  The controllers use client-go library extensively. The details of interaction points of the controllers with various mechanisms from this library are explained [here][sample-controller].

- pkg/runctl

  The go implementation of the controllers

## Versioning

Although the controller images and the Helm chart are independent they are always released together. Our **release pipeline** performs the following steps:

- Identify the new semver2 release version to release. This is done by taking the `version` from the Chart.yaml and removing the `-dev` suffix. This means, if we need to increase the major or minor version we can do so in the Chart.yaml before, while keeping the `-dev` suffix.
- Update the Helm chart with the new release version. This includes `version` and `appVersion` in the `Chart.yaml` and `runController.image.tag` in the `values.yaml`. The image tags are prepared with the new version, although the images do not exist yet.
- Push commit to GitHub into a `prepare-<version>` branch.
- Push the controller images built and validated earlier in the pipeline with the new version tag.
- Create a GitHub release tag based on the pushed commit with the chart version changes.
- Prepare the next dev version in `prepare-<version>` by updating `version` and `appVersion` in Chart.yaml with an incremented patch version and `-dev` suffix.
- Merge the `prepare-<version>` branch into the main branch and delete the `prepare-<version>` branch.

## Hotfix Releases

To release a hotfix based on a released version (assuming `v1.2.3` below) do the following.

- Prepare a new commit, based on the "next dev" commit which follows the release tag `v1.2.3`
- Adjust the `version` and `appVersion` in [/charts/steward/Chart.yaml](https://github.com/SAP/stewardci-core/blob/master/charts/steward/Chart.yaml) to `v1.2.3-hotfix1`
- Add a hotfix changelog entry to the changelog.yamls version "NEXT"
- Commit the changes via `git commit --amend` and rename the commit message (to get rid the "next dev" commit)
- Push the changes to a **branch** `v1.2.3-hotfix1` (or work with a pull request targeting branch `v1.2.3-hotfix1`)
- Trigger the release pipeline for branch `v1.2.3-hotfix1`
- Approve the release stage

The pipeline will now create a hotfix release.

:warning: Manual post-release steps!

- Merge the `v1.2.3-hotfix1` **tag** into the master branch. You will have to resolve conflicts - take the version from the master branch, make sure to keep all changelog entries.
- Delete the `v1.2.3-hotfix1` **branch**.

## Contribution

You are welcome to contribute to this project via Pull Requests.

## Development

### Prerequisites

```sh
# Prepare Code Generator
git clone https://github.com/kubernetes/code-generator.git
cd code-generator/

# We need a specific version matching to our K8s client-go version (currently kubernetes-1.14.9)
#
# Unfortunately old versions are not yet modularized.
#     We take the module info from a newer release.
git checkout kubernetes-1.14.9
git checkout kubernetes-1.16.1 -- go.mod go.sum

# CODEGEN_PKG is used by the script to find the code-generator
export CODEGEN_PKG=${PWD}
```

```sh
# Prepare mockgen tool
go get github.com/golang/mock/mockgen
```

### Build

To run build and test simply execute `./build.sh` from the project root folder.

To build only the controllers run:

```sh
# Build the run controller executable
go build -o runController ./cmd/run_controller/
```

### Code Generation

The generated clients and mocks have been committed into the project sources. Generation is not necessary in every build, but for some changes (e.g. API changes) the clients and mocks need to be generate again (and committed). This can be done using:

```sh
# Client generation
hack/update-codegen.sh
```

### Logging

Refer to [our message logging conventions](./logging.md) used in the project.

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
