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
  local pkg="$1"
  local interfaces="$2"
  if [[ -z "$pkg" ]]; then
    exit 1
  fi
  if [[ -z "$interfaces" ]];then
    exit 1
  fi
  echo "## ${ACTION} mocks for package '$pkg' ###############"
  
  
  set -x
  "$GOPATH_1/bin/mockgen" \
    -copyright_file="${PROJECT_ROOT}/hack/boilerplate.go.txt" \
    -destination="${MOCK_ROOT}/pkg/${pkg}/mocks/mocks.go" \
    -package=mocks \
    github.com/SAP/stewardci-core/pkg/${pkg} \
    "${interfaces}" \
    || die "'$pkg' mock generation failed"
  { set +x; } 2>/dev/null
  if is_verify_mode; then
    set -x
    diff -Naupr ${GEN_DIR}/pkg/${pkg}/mocks/mocks.go ${PROJECT_ROOT}/pkg/${pkg}/mocks/mocks.go || die "Regeneration required for k8s/secrets mocks"
    { set +x; } 2>/dev/null
  fi
  echo
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
    ( cd "$GOPATH_1" && go get github.com/golang/mock/mockgen ) || die "Installation of mockgen failed"
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

echo
echo "## Cleanup old generated stuff ####################"
set -x
rm -rf \
    "${GEN_DIR}" \
    "${GOPATH_1}/bin/"{client-gen,deepcopy-gen,defaulter-gen,informer-gen,lister-gen} \
    || die "Cleanup failed"
{ set +x; } 2>/dev/null
if ! is_verify_mode; then
    set -x
    rm -rf \
        "${PROJECT_ROOT}/pkg/client" \
        "${PROJECT_ROOT}/pkg/tektonclient" \
        "${PROJECT_ROOT}/pkg/apis/steward/v1alpha1/zz_generated.deepcopy.go" \
        "${PROJECT_ROOT}/pkg/k8s/mocks/mocks.go" \
        "${PROJECT_ROOT}/pkg/k8s/secrets/mocks/mocks.go" \
        || die "Cleanup failed"
    { set +x; } 2>/dev/null
fi

echo
echo "## Generate #######################################"
set -x

"${CODEGEN_PKG}/generate-groups.sh" \
    all \
    github.com/SAP/stewardci-core/pkg/client \
    github.com/SAP/stewardci-core/pkg/apis \
    steward:v1alpha1 \
    --go-header-file "${PROJECT_ROOT}/hack/boilerplate.go.txt" \
    --output-base "${GEN_DIR}" \
    || die "Code generation failed"
{ set +x; } 2>/dev/null
set -x
"${CODEGEN_PKG}/generate-groups.sh" \
    "client,informer,lister" \
    github.com/SAP/stewardci-core/pkg/tektonclient \
    github.com/tektoncd/pipeline/pkg/apis \
    pipeline:v1alpha1 \
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

echo

generate_mocks k8s "ClientFactory,NamespaceManager,PipelineRun,PipelineRunFetcher,TenantFetcher"
generate_mocks "k8s/secrets"  "SecretHelper,SecretProvider"
generate_mocks run  "Run,Manager"

echo "${ACTION} successful"
