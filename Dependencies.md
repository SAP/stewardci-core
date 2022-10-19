# Managing Go Dependencies

This page describes Go dependency management for Steward CI.

## Version Restrictions

### Kubernetes libraries

Steward depends on multiple libraries belonging to the Kubernetes main release.
For all those libraries the version of one single Kubernetes release should be
used.

`replace` directives for all Kubernetes libraries from the main release are
maintained in `go.mod`, because other dependencies may depend on a higher
version of one or more of those libraries which would then be selected by Go's
minimum version selection instead of the desired ones.

The targeted Kubernetes version must also be stored in file `K8S_VERSION`.

### `knative.dev/pkg`

This library builds against Kubernetes libraries and provides interaction with
Kubernetes at runtime. Therefore a version should be used that fits to the
Kubernetes release version targeted by Steward. The versions are not required to
be identical, but the difference should not be larger than one minor version.

A `replace` directive for `knative.dev/pkg` is maintained in `go.mod`, because
other dependencies may depend on a higher version which would then be selected
by Go's minimum version selection instead of the desired one.

## Upgrading Dependencies

Steward binaries should be built with the most up-to-date dependencies to
receive functional corrections and security fixes. Therefore, upgrades to the
_highest compatible_ version are intended. In particular it is _not_ sufficient
to only update patch versions, because functional corrections and security fixes
are often applied to the latest minor version only.

In general, finding the highest compatible version of a dependency is not
trivial:

-   Many dependencies have `v0` versions where minor version increments can
    contain incompatible changes.

-   Some projects break the SemVer rules.

-   A newer version of a dependency requires a higher version of another
    dependency which contradicts [Stewards version restrictions](#version-restrictions).

-   Other non-code changes can break the dependency upgrade or build.

    Example:

    -   The Git repository and the module path of a dependency have been
        renamed. Upgrading to a higher version does not work because the module
        path changed.

Upgrading _all_ dependencies with the following single command typically does
_NOT_ work, because of the aforementioned reasons:

    go get -t -u all

Therefore, dependencies must be updated one by one with build verification in
between.

### Prerequisites

Use the Go SDK version in file `GOLANG_VERSION` to run any of the commands.
Using another version may lead to different and possibly wrong results.

### Process

1.  Update the _direct_ dependencies.

    Direct dependencies are updated first, because this may change indirect
    dependencies.

    Find direct dependencies where updates are available:

        go list -m -u -f '{{if and .Update (or .Main .Indirect | not)}}{{.String}}{{end}}' all

    Decide which dependencies should be upgraded. Take the [version
    restrictions](#version-restrictions) described above into account.

    For each dependency (or group of related dependencies) to be updated:

    1.  Update one (or multiple) dependencies:

            go get MODULES && go mod tidy -v

        `MODULES` is one or more module names optionally with a version query.
        See [`go get`][go_get_mod] in the "Go Modules Reference" for details.

        For [Kubernetes libraries](#kubernetes-libraries) make sure to update them
        consistently. Also adapt the replace directives.

    2.  View the change:

            git diff go.mod

    3.  Run the build:

            ./build.sh

    4.  If successfull, add the changes to the Git index and proceed with the next
        dependency.

2.  Update the _indirect_ dependencies:

    Find indirect dependencies where updates are available:

        go list -m -u -f '{{if and .Update .Indirect}}{{.String}}{{end}}' all

    Note that not all dependencies in the returned list are actually used (see
    [_Module Graph Pruning_][mod_graph_pruning] in the "Go Modules Reference").
    Pick those where entries in `go.mod` exist.

    Perform the same steps as for direct dependencies to update dependencies one
    by one or in chunks.

3.  Check if further [Kubernetes libraries](#kubernetes-libraries) are contained
    in the module graph:

        go list -m -f '{{if and (eq (slice .Path 0 7) "k8s.io/") (.Replace | not)}}{{.String}}{{end}}' all | less

    Note that _not all_ returned modules belong to the Kubernetes main release.

    For all additional Kubernetes libraries belonging to the Kubernetes main
    release, __add corresponding `replace` directives__ to enforce the right
    version.

### Further Helpful Commands

Show the list of modules in the module graph:

    go list -m all

Find which modules require a given module:

    go mod graph | grep -F ' <MODULE>@'

## Further Reading

- Go documentation:
    - [_Managing dependencies_][managing_deps]
    - [_Go Modules Reference_][go_modules_ref]


[managing_deps]: https://go.dev/doc/modules/managing-dependencies
[go_modules_ref]: https://go.dev/ref/mod
[go_get_mod]: https://go.dev/ref/mod#go-get
[mod_graph_pruning]: https://go.dev/ref/mod#graph-pruning
