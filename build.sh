#!/bin/bash
set -eu -o pipefail
exec 1>&2 <&-

declare -r -a GO_PACKAGES_TEST=(
    "./test/..."
)

declare -r -a GO_PACKAGES_ALL=(
    "./cmd/..."
    "./pkg/..."
    "${GO_PACKAGES_TEST[@]}"
)

declare -r HELM_TEST_DIR="charts/steward/test"

HERE=$(cd "$(dirname "$BASH_SOURCE")" && pwd) || {
    echo "error: could not determine script location" >&2
    exit 1
}

# set from options/arguments
P_FULL=
unset GOLANG_VERSION

function main() {
    parse_args "$@"

    if ! which go &>/dev/null; then
        die "error: go not found"$'\n\n'"Install Go and add its bin directory to your PATH!"
    fi

    GOPATH=${GOPATH:-$(go env GOPATH)} || die "error: could not determine GOPATH"
    export GOPATH
    GOPATH_1=${GOPATH%%:*}  # the first entry of the GOPATH

    if [[ -z $GOPATH_1 ]]; then
        die "error: GOPATH not set"
    fi

    check_dependencies

    banner1 "Settings"
    info \
        "GOLANG_VERSION=$GOLANG_VERSION" \
        "GOPATH=$GOPATH" \
        "GOLINT=$GOLINT_EXE" \
        "LOGCHECK=$LOGCHECK_EXE"

    banner1 "go build"
    go build "${GO_PACKAGES_ALL[@]}" || die "" "FAILED"

    banner1 "go vet"
    (
        local rc=0
        go vet "${GO_PACKAGES_ALL[@]}" || rc=1
        ( cd "$HELM_TEST_DIR" && go vet "./..." ) || rc=1
        exit $rc
    ) || {
        echo $'\n'"Check whether the issues above are real issues that must be fixed!"
    }

    banner1 "go test"
    go test -coverprofile coverage.txt "${GO_PACKAGES_ALL[@]}" || die "" "FAILED"
    go tool cover -html=coverage.txt -o coverage.html || die "" "FAILED"

    ( cd "$HELM_TEST_DIR" && go test "./..." ) || die "" "FAILED"

    if [[ $P_FULL ]]; then
        # compile tests in ./test/.. without running them
        local err=
        for tags in "frameworktest" "loadtest" "opennet" "closednet" "e2e"; do
            info "" "compiling '${GO_PACKAGES_TEST[@]}' with tags '$tags'"
            test_compile_only "$(go list "${GO_PACKAGES_TEST[@]}")" -tags="$tags" || {
                info "" "failed to compile '${GO_PACKAGES_TEST[@]}' with tags '$tags'"
                err=1
            }
        done
        [[ ! $err ]] || die "" "FAILED"
    fi

    banner1 "golint"
    "$GOLINT_EXE" -set_exit_status "${GO_PACKAGES_ALL[@]}" || die "" "FAILED"
    ( cd "$HELM_TEST_DIR" && "$GOLINT_EXE" -set_exit_status "./..." ) || die "" "FAILED"

    banner1 "logcheck"
    "$LOGCHECK_EXE" -check-value \
        "${GO_PACKAGES_ALL[@]}" || die "" "FAILED"
    ( cd "$HELM_TEST_DIR" && "$LOGCHECK_EXE" -check-parameters \
        -check-value \
        "./..." ) || die "" "FAILED"

    banner1 "gofmt"
    gofmt -l -d "${GO_PACKAGES_ALL[@]%/...}" | tee fmt_diff.txt || die "" "FAILED"
    [[ -s fmt_diff.txt ]] && die "" "FAILED"

    gofmt -l -d "$HELM_TEST_DIR" | tee fmt_diff.txt || die "" "FAILED"
    [[ -s fmt_diff.txt ]] && die "" "FAILED"

    echo $'\n'"SUCCESS"
}

function parse_args() {
    while (( $# > 0 )); do
        case $1 in
            "-h" | "--help" )
                print_usage
                exit 0
                ;;
            "--full" )
                P_FULL=1
                ;;
            * )
                die "error: unknown option '$1'"
        esac
        shift
    done
}

function print_usage() {
    cat >&2 <<EOF

Usage

    $(get_script_name) OPTIONS

Options

    -h, --help
        Print help and quit without doing anything. The exit code is 0.

    --full
        Enable all checks.

Remarks

    Options can be specified in any order. If an option is specified
    multiple times, the last value will be used unless stated otherwise.

EOF
}

function get_script_name() {
    printf "%q" "$(basename "$BASH_SOURCE")"
}

function die() {
    info "$@"
    exit 1
}

function info() {
    for line in "$@"; do
        echo "$line" >&2
    done
}

function banner1() {
    echo $'\n'"===" "$@" $'\n'
}


function check_dependencies() {
    check_go
    check_golint_or_install
    check_logcheck_or_install
}

function check_go() {
    local expected_version
    expected_version=$(cat GOLANG_VERSION) || {
        die "error reading expected Go version from file GOLANG_VERSION"
    }
    [[ $expected_version ]] || die "error: no expected version found in file GOLANG_VERSION"
    GOLANG_VERSION=$expected_version

    local actual_version
    actual_version=$(go version | sed -E -e '2,$d; /^go version go[0]*[0-9]{1,4}\.[0]*[0-9]{1,4}(\.[0]*[0-9]{1,4})?([^0-9]|$)/!d; s/^go version go[0]*([0-9]+)\.[0]*([0-9]+)((\.)[0]*([0-9]+))?.*/\1.\2\4\5/')
    if [[ ! $actual_version ]]; then
        die "error: could not determine go version"
    fi
    if [[ $actual_version != "$expected_version" ]]; then
        die "error: expected Go version ${expected_version} but found ${actual_version}"
    fi
}

function check_golint_or_install() {
    local golint_install_url='golang.org/x/lint/golint'
    check_tool_or_install "golint" "${golint_install_url}"
    GOLINT_EXE=$(cd "$GOPATH_1" && GO111MODULE=auto go list -f '{{.Target}}' "${golint_install_url}" 2>/dev/null)
}

function check_logcheck_or_install() {
    local logcheck_install_url='sigs.k8s.io/logtools/logcheck'
    check_tool_or_install "logcheck" "${logcheck_install_url}"
    LOGCHECK_EXE=$(cd "$GOPATH_1" && GO111MODULE=auto go list -f '{{.Target}}' "${logcheck_install_url}" 2>/dev/null)
}

function check_tool_or_install(){
    local TOOL_NAME=$1
    local TOOL_INSTALL_URL=$2

    local rc=0
    local TOOL_EXE=$(which ${TOOL_NAME}) || rc=$?
    (( rc > 1 )) && die
    if [[ $rc == 1 || ! $TOOL_EXE ]]; then
        # don't run go list from current directory, because it would modify our go.mod file
        rc=0
        TOOL_EXE=$(cd "$GOPATH_1" && GO111MODULE=auto go list -f '{{.Target}}' "${TOOL_INSTALL_URL}" 2>/dev/null) || rc=$?
        if [[ $rc != 0 || ! $TOOL_EXE || ! -f $TOOL_EXE ]]; then
            echo "${TOOL_NAME} not found. Installing ${TOOL_NAME} into current GOPATH ..."
            # don't run go get/list from current directory, because it would modify our go.mod file
            ( cd "$GOPATH_1" && GO111MODULE=auto go get -u ${TOOL_INSTALL_URL} ) || die
            rc=0
            TOOL_EXE=$(cd "$GOPATH_1" && GO111MODULE=auto go list -f '{{.Target}}' ${TOOL_INSTALL_URL} 2>/dev/null) || rc=$?
            if [[ $rc != 0 || ! $TOOL_EXE || ! -f $TOOL_EXE ]]; then
                die "error: could not install ${TOOL_NAME}"
            fi
        fi
    fi
}

function test_compile_only() {
    local packages=$1 # newline-separated package import paths
    shift 1
    local build_args=("$@")

    local err=
    for pkg in $packages; do
        # compile test binary without running it
        go test "${build_args[@]}" -c "$pkg" -o /dev/null || err=1
    done

    [[ ! $err ]] || return 1
}


##################################################
# Main
##################################################

cd "$HERE"
main "$@"
