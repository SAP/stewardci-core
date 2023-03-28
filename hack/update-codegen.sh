#!/bin/bash
set -u -o pipefail

MOCKGEN_VERSION=v1.6.0

# set from args
unset \
    VERIFY \
    GENERATE_CLIENTS \
    GENERATE_MOCKS \
#

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
            "--gen-clients" )
                GENERATE_CLIENTS=1
                ;;
            "--gen-mocks" )
                GENERATE_MOCKS=1
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

    if [[ ! ( ${GENERATE_MOCKS-} || ${GENERATE_CLIENTS-} ) ]]; then
        GENERATE_MOCKS=1
        GENERATE_CLIENTS=1
    fi
}

function print_usage() {
    echo "usage:"
    echo ""
    echo "   $(basename $BASH_SOURCE) [OPTIONS]"
    echo ""
    echo "When run without any options, all types of code will be (re)generated."
    echo ""
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
    echo "   --gen-clients"
    echo "      Enable generation of clients. This is enabled implicitly if no --gen-*"
    echo "      option is specified."
    echo ""
    echo "   --gen-mocks"
    echo "      Enable generation of mocks. This is enabled implicitly if no --gen-*"
    echo "      option is specified."
    echo ""
}

function is_verify_mode() {
    [[ ${VERIFY-} ]]
}

function is_generate_clients() {
    [[ ${GENERATE_CLIENTS-} ]]
}

function is_generate_mocks() {
    [[ ${GENERATE_MOCKS-} ]]
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
if is_generate_clients; then
    "$HERE/bootstrap-codegen.sh" || die "failed to bootstrap code generator"
    [[ -f $CODEGEN_PKG/generate-groups.sh ]] \
        || die "\$CODEGEN_PKG ('$CODEGEN_PKG'): file 'generate-groups.sh' does not exist"
    [[ -x $CODEGEN_PKG/generate-groups.sh ]] \
        || die "\$CODEGEN_PKG ('$CODEGEN_PKG'): file 'generate-groups.sh' is not executable"
fi

# prepare mockgen
if is_generate_mocks; then
    MOCKGEN_EXE="$GOPATH_1/bin/mockgen"
    if [[ ! -x $MOCKGEN_EXE ]]; then
        echo "Installing mockgen"
        ( cd "$GOPATH_1" && go install "github.com/golang/mock/mockgen@$MOCKGEN_VERSION" ) || die "Installation of mockgen failed"
    fi
    [[ -f $MOCKGEN_EXE ]] || die "'$MOCKGEN_EXE' does not exist"
    [[ -x $MOCKGEN_EXE ]] || die "'$MOCKGEN_EXE' is not executable"
fi

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
echo "Generators:   $(is_generate_clients && echo "clients") $(is_generate_mocks && echo "mocks")"
echo "GO version:   $(go version)"

echo
echo "## Cleanup old generated stuff ####################"

set -x
rm -rf \
    "${GEN_DIR}" \
    || die "Cleanup failed"
{ set +x; } 2>/dev/null

if is_generate_clients; then
    set -x
    rm -rf \
        "${GOPATH_1}/bin/"{client-gen,deepcopy-gen,defaulter-gen,informer-gen,lister-gen} \
        || die "Cleanup failed"
    { set +x; } 2>/dev/null
fi

if ! is_verify_mode; then
    if is_generate_clients; then
        set -x
        rm -rf \
            "${PROJECT_ROOT}/pkg/client" \
            "${PROJECT_ROOT}/pkg/tektonclient" \
            "${PROJECT_ROOT}/pkg/apis/steward/v1alpha1/zz_generated.deepcopy.go" \
            || die "Cleanup failed"
        { set +x; } 2>/dev/null
    fi
    if is_generate_mocks; then
        set -x
        rm -rf \
            "${PROJECT_ROOT}/pkg/k8s/mocks/mocks.go" \
            "${PROJECT_ROOT}/pkg/k8s/mocks/client-go/corev1/mocks.go" \
            "${PROJECT_ROOT}/pkg/k8s/secrets/mocks/mocks.go" \
            "${PROJECT_ROOT}/pkg/runctl/run/mocks/mocks.go" \
            || die "Cleanup failed"
        { set +x; } 2>/dev/null
    fi
fi

echo
echo "## Generate #######################################"

if is_generate_clients; then
    set -x
    "${CODEGEN_PKG}/generate-groups.sh" \
        all \
        github.com/SAP/stewardci-core/pkg/client \
        github.com/SAP/stewardci-core/pkg/apis \
        steward:v1alpha1 \
        --go-header-file "${PROJECT_ROOT}/hack/boilerplate.go.txt" \
        --output-base "${GEN_DIR}" \
        || die "Code generation failed"
    "${CODEGEN_PKG}/generate-groups.sh" \
        "client,informer,lister" \
        github.com/SAP/stewardci-core/pkg/tektonclient \
        github.com/tektoncd/pipeline/pkg/apis \
        pipeline:v1beta1 \
        --go-header-file "${PROJECT_ROOT}/hack/boilerplate.go.txt" \
        --output-base "${GEN_DIR}" \
        || die "Code generation failed"
    { set +x; } 2>/dev/null

    echo
    if is_verify_mode; then
        echo "## Verifying generated sources ####################"
        set -x
        diff -Naupr ${GEN_DIR}/github.com/SAP/stewardci-core/pkg/client/ ${PROJECT_ROOT}/pkg/client/ || die "Regeneration required for clients"
        diff -Naupr ${GEN_DIR}/github.com/SAP/stewardci-core/pkg/tektonclient/ ${PROJECT_ROOT}/pkg/tektonclient/ || die "Regeneration required for tektonclients"
        diff -Naupr ${GEN_DIR}/github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1/zz_generated.deepcopy.go ${PROJECT_ROOT}/pkg/apis/steward/v1alpha1/zz_generated.deepcopy.go || die "Regeneration required for apis"
        { set +x; } 2>/dev/null
    else
        echo "## Move generated files ###########################"
        set -x
        mv "${GEN_DIR}/github.com/SAP/stewardci-core/pkg/client" "${PROJECT_ROOT}/pkg/" || die "Moving generated clients failed"
        mv "${GEN_DIR}/github.com/SAP/stewardci-core/pkg/tektonclient" "${PROJECT_ROOT}/pkg/" || die "Moving generated tektonclients failed"
        cp -r "${GEN_DIR}/github.com/SAP/stewardci-core/pkg/apis" "${PROJECT_ROOT}/pkg/" || die "Copying generated apis failed"
        rm -rf "${GEN_DIR}/github.com" || die "Cleanup gen dir failed"
        { set +x; } 2>/dev/null
    fi
fi

if is_generate_mocks; then
    echo
    generate_mocks \
        "github.com/SAP/stewardci-core/pkg/k8s" \
        "ClientFactory,NamespaceManager,PipelineRun,PipelineRunFetcher,TenantFetcher" \
        "pkg/k8s/mocks/mocks.go"
    generate_mocks \
        "k8s.io/client-go/kubernetes/typed/core/v1" \
        "CoreV1Interface,ConfigMapInterface" \
        "pkg/k8s/mocks/client-go/corev1/mocks.go"
    generate_mocks \
        "github.com/SAP/stewardci-core/pkg/k8s/secrets" \
        "SecretHelper,SecretProvider" \
        "pkg/k8s/secrets/mocks/mocks.go"
    generate_mocks \
        "github.com/SAP/stewardci-core/pkg/runctl/run" \
        "Run,Manager,SecretManager" \
        "pkg/runctl/run/mocks/mocks.go"
    generate_mocks \
        "github.com/SAP/stewardci-core/pkg/runctl/metrics" \
        "CounterMetric,PipelineRunsMetric,StateItemsMetric,ResultsMetric" \
        "pkg/runctl/metrics/testing/mocks.go"
fi

echo "${ACTION} successful"
