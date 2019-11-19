#!/bin/bash

function die() {
    if [[ -n "$*" ]]; then
        echo "$@" >&2
    fi
    exit 1
}

PROJECT_ROOT=$(cd "$(dirname "$BASH_SOURCE")/.." && pwd) || die
export CODEGEN_PKG=${PROJECT_ROOT}/temp/code-generator

# Prepare Code Generator
rm -rf ${CODEGEN_PKG}
git clone https://github.com/kubernetes/code-generator.git ${CODEGEN_PKG} || die
cd  ${CODEGEN_PKG} || die

# We need a specific version matching to our Tekton version (currently kubernetes-1.13.11)
#
# Unfortunately old versions are not yet modularized.
#     We take the module info from a newer release.
git checkout kubernetes-1.13.11 || die
git checkout kubernetes-1.16.1 -- go.mod go.sum || die

cd -