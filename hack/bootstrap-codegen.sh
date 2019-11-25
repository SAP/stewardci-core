#!/bin/bash
set -u -o pipefail

CODEGEN_GIT_URL='https://github.com/kubernetes/code-generator.git'


function die() {
    if [[ -n "$*" ]]; then
        echo "$@" >&2
    fi
    exit 1
}

function is_git_dir() {
    local result
    result=$(git rev-parse --git-dir 2>&1)
    case $? in
        0)
            return 0;;
        128)
            return 1;;
        *)
            echo "$result" >&2
            die "failed to execute `git rev-parse --git-dir`"
    esac
}

function is_inside_git_worktree() {
    is_git_dir || return 1
    local result
    result=$(git rev-parse --is-inside-work-tree) || die "failed to run `git rev-parse`"
    [[ $result == "true" ]]
}

function is_git_origin_url() {
    local url="$1"
    is_git_dir || return 1
    local result
    result=$(git remote get-url origin) || die "failed to run `git remote get-url origin`"
    [[ $result == "$url" ]]
}

HERE=$(cd "$(dirname "$BASH_SOURCE")" && pwd) || die
source "$HERE/.setpaths"
K8S_VERSION=$(cat "${PROJECT_ROOT}/K8S_VERSION") || die


### main ###

if [[ ! -d $CODEGEN_PKG ]]; then
    git clone "$CODEGEN_GIT_URL" "$CODEGEN_PKG" || die "could not clone code generator Git repository"
fi

cd "$CODEGEN_PKG" || die
is_inside_git_worktree || die "\$CODEGEN_PKG ('${CODEGEN_PKG}'): not in a Git work tree"
is_git_origin_url "$CODEGEN_GIT_URL" || die "\$CODEGEN_PKG ('${CODEGEN_PKG}'): unexpected origin URL"

# We need a specific version matching to our Kubernetes and Tekton dependencies
# (see https://github.com/kubernetes/code-generator#compatibility)
git clean -dxf || die "failed to execute `git clean -dxf`"
git reset --hard || die "failed to execute `git reset --hard`"
git checkout "$K8S_VERSION" || die "\$CODEGEN_PKG ('${CODEGEN_PKG}'): could not checkout revision $K8S_VERSION"
if [[ ! -f "go.mod" ]]; then
    # this revision is not a Go module
    # take the module descriptor from another revision hoping that it fits
    git checkout kubernetes-1.16.1 -- go.mod go.sum || die "failed to checkout go.mod from other branch"
fi
