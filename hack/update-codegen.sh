#!/bin/bash
set -u -o pipefail

# set from args
unset VERIFY

HERE=$(cd "$(dirname "$BASH_SOURCE")" && pwd) || exit 1


function die() {
    if [[ -n "$*" ]]; then
        echo "$@" >&2
    fi
    exit 1
}

function read_args() {
    trap handle_error ERR

    until [[ -z ${1+x} ]]
    do
        case "$1" in
            "-h" | "--help" )
                print_usage
                exit 0
                ;;
            "--verify" )
                VERIFY=1
                ;;
            * )
                echo "error: invalid command line option '$1'" >&2
                exit 1
        esac
        shift
    done
}

function print_usage() {
    echo "usage:"
    echo ""
    echo "   $(basename $BASH_SOURCE) [OPTIONS]"
    echo ""
    echo "When run without any options, all existing generated code will be regenerated."
    echo ""
    echo "OPTIONS"
    echo ""
    echo "   -h, --help"
    echo "      Print usage help and exit. No other operations will be performed."
    echo ""
    echo "   --verify"
    echo "      Verifies that the generated code is up-to-date. The existing generated"
    echo "      code will not be touched."
    echo ""
}

function is_verify_mode() {
    [[ -n ${VERIFY:-} ]]
}

function generate_mocks() {
    local pkg="$1" interfaces="$2" dest="$3"
    local destPkgName
    if [[ -z $pkg ]]; then
        die "generate_mocks: package must be specified"
    fi
    if [[ -z $interfaces ]]; then
        die "generate_mocks: interface list must be specified"
    fi
    if [[ -z $dest ]]; then
        die "generate_mocks: destination must be specified"
    fi
    echo "## ${ACTION} mocks for package '$pkg' ###############"

    destPkgName=$(basename "$(dirname "$dest")") || die
    set -x
    "$GOPATH_1/bin/mockgen" \
        -copyright_file="${PROJECT_ROOT}/hack/boilerplate.go.txt" \
        -destination="${MOCK_ROOT}/${dest}" \
        -package="$destPkgName" \
        "$pkg" \
        "$interfaces" \
        || die "mock generation for '$pkg' failed"
    { set +x; } 2>/dev/null
    if is_verify_mode; then
        set -x
        diff -Naupr "${GEN_DIR}/${dest}" "${PROJECT_ROOT}/${dest}" || die "Regeneration required for mocks of '$pkg'"
        { set +x; } 2>/dev/null
    fi
    echo
}

function checkGoVersion() {
    local expectedVersion=`cat $HERE/../GOLANG_VERSION`
    [[ ! -z $expectedVersion ]] || die
    go version | grep "${expectedVersion}"
    local result=$?
    if [[ $result != 0 ]]; then
        die "error: Expected Go version ${expectedVersion} but was: $(go version)"
    fi
}

#
# main
#

source "$HERE/.setpaths"

read_args "$@"

# Check and prepare build enviroment
export GOPATH=`go env GOPATH`
if [[ -z $GOPATH ]]; then
    die "GOPATH not set"
fi
GOPATH_1=${GOPATH%%:*}  # the first entry of the GOPATH

checkGoVersion

# prepare code generator
"$HERE/bootstrap-codegen.sh" || die "failed to bootstrap code generator"
[[ -f $CODEGEN_PKG/generate-groups.sh ]] \
    || die "\$CODEGEN_PKG ('$CODEGEN_PKG'): file 'generate-groups.sh' does not exist"
[[ -x $CODEGEN_PKG/generate-groups.sh ]] \
    || die "\$CODEGEN_PKG ('$CODEGEN_PKG'): file 'generate-groups.sh' is not executable"

# prepare mockgen
MOCKGEN_EXE="$GOPATH_1/bin/mockgen"
if [[ ! -x $MOCKGEN_EXE ]]; then
    echo "Installing mockgen"
    ( cd "$GOPATH_1" && GO111MODULE=on go get github.com/golang/mock/mockgen@v1.4.3 ) || die "Installation of mockgen failed"
fi
[[ -f $MOCKGEN_EXE ]] || die "'$MOCKGEN_EXE' does not exist"
[[ -x $MOCKGEN_EXE ]] || die "'$MOCKGEN_EXE' is not executable"

GEN_DIR="$PROJECT_ROOT/gen"

if is_verify_mode; then
    MOCK_ROOT=${GEN_DIR}
    ACTION="Verify"
else
    MOCK_ROOT=${PROJECT_ROOT}
    ACTION="Generate"
fi

echo
echo "PROJECT_ROOT: $PROJECT_ROOT"
echo "GEN_DIR:      $GEN_DIR"
echo "MOCK_ROOT:    $MOCK_ROOT"
echo "CODEGEN_PKG:  $CODEGEN_PKG"
echo "GOPATH:       $GOPATH_1"
echo "VERIFY:       $(if is_verify_mode; then echo "true"; else echo "false"; fi)"
echo "GO version:   $(go version)"

echo
generate_mocks \
    "github.com/SAP/stewardci-core/pkg/run" \
    "Run,Manager" \
    "pkg/run/mocks/mocks.go"

echo "${ACTION} successful"
