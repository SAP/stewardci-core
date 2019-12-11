#!/bin/bash
set -u -o pipefail

function die() {
    if [[ -n "$*" ]]; then
        echo "$@" >&2
    fi
    exit 1
}

function banner1() {
    echo $'\n'"===" "$@" $'\n'
}

if ! which go &>/dev/null; then
    die "error: go not found"$'\n\n'"Install Go and add its bin directory to your PATH!"
fi

GOPATH=${GOPATH:-$(go env GOPATH)} || die "error: could not determine GOPATH"
export GOPATH
GOPATH_1=${GOPATH%%:*}  # the first entry of the GOPATH

if [[ -z $GOPATH_1 ]]; then
    die "error: GOPATH not set"
fi

GOLINT_EXE=$(which golint); rc=$?
[[ $rc > 1 ]] && die
if [[ $rc == 1 || -z $GOLINT_EXE ]]; then
    # don't run go list from current directory, because it would modify our go.mod file
    GOLINT_EXE=$(cd "$GOPATH_1" && go list -f '{{.Target}}' 'golang.org/x/lint/golint' 2>/dev/null); rc=$?
    if [[ $rc != 0 || -z $GOLINT_EXE || ! -f $GOLINT_EXE ]]; then
        echo "golint not found. Installing golint into current GOPATH ..."
        # don't run go get/list from current directory, because it would modify our go.mod file
        ( cd "$GOPATH_1" && go get -u 'golang.org/x/lint/golint' ) || die
        GOLINT_EXE=$(cd "$GOPATH_1" && go list -f '{{.Target}}' 'golang.org/x/lint/golint' 2>/dev/null); rc=$?
        if [[ $rc != 0 || -z $GOLINT_EXE || ! -f $GOLINT_EXE ]]; then
            die "error: could not install golint"
        fi
    fi
fi
unset rc


banner1 "Settings"
echo "GOPATH=$GOPATH"
echo "GOLINT=$GOLINT_EXE"

banner1 "go build"
go build ./... || die

banner1 "go test"
go test -coverprofile coverage.txt ./... || die
go tool cover -html=coverage.txt -o coverage.html || die

banner1 "golint"
"$GOLINT_EXE" -set_exit_status ./pkg/... ./cmd/... ./test/... || die

banner1 "gofmt"
gofmt -l ./pkg/ ./cmd/ ./test/ || die
gofmt -d ./pkg/ ./cmd/ ./test/ > fmt_diff.txt || die
[[ -s fmt_diff.txt ]] && die "gofmt failed, see fmt_diff.txt"

echo $'\n'"SUCCESS"
