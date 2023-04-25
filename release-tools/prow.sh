#! /bin/bash

RELEASE_TOOLS_ROOT="$(realpath "$(dirname "${BASH_SOURCE[0]}")")"
REPO_DIR="$(pwd)"

# Sets the default value for a variable if not set already and logs the value.
# Any variable set this way is usually something that a repo's .prow.sh
# or the job can set.
configvar () {
    # Ignore: Word is of the form "A"B"C" (B indicated). Did you mean "ABC" or "A\"B\"C"?
    # shellcheck disable=SC2140
    eval : \$\{"$1":="\$2"\}
    eval echo "\$3:" "$1=\${$1}"
}

# Prints the value of a variable + version suffix, falling back to variable + "LATEST".
get_versioned_variable () {
    local var="$1"
    local version="$2"
    local value

    eval value="\${${var}_${version}}"
    if ! [ "$value" ]; then
        eval value="\${${var}_LATEST}"
    fi
    echo "$value"
}

# This takes a version string like CSI_PROW_KUBERNETES_VERSION and
# maps it to the corresponding git tag, branch or commit.
version_to_git () {
    version="$1"
    shift
    case "$version" in
        latest|master) echo "master";;
        release-*) echo "$version";;
        *) echo "v$version";;
    esac
}

# the list of windows versions was matched from:
# - https://hub.docker.com/_/microsoft-windows-nanoserver
# - https://hub.docker.com/_/microsoft-windows-servercore
configvar CSI_PROW_BUILD_PLATFORMS "linux amd64; linux ppc64le -ppc64le; linux s390x -s390x; linux arm64 -arm64; windows amd64 .exe nanoserver:1809 servercore:ltsc2019; windows amd64 .exe nanoserver:1909 servercore:1909; windows amd64 .exe nanoserver:2004 servercore:2004; windows amd64 .exe nanoserver:20H2 servercore:20H2" "Go target platforms (= GOOS + GOARCH) and file suffix of the resulting binaries"

# If we have a vendor directory, then use it. We must be careful to only
# use this for "make" invocations inside the project's repo itself because
# setting it globally can break other go usages (like "go get <some command>"
# which is disabled with GOFLAGS=-mod=vendor).
configvar GOFLAGS_VENDOR "$( [ -d vendor ] && echo '-mod=vendor' )" "Go flags for using the vendor directory"

configvar CSI_PROW_GO_VERSION_BUILD "1.16" "Go version for building the component" # depends on component's source code
configvar CSI_PROW_GO_VERSION_E2E "" "override Go version for building the Kubernetes E2E test suite" # normally doesn't need to be set, see install_e2e
configvar CSI_PROW_GO_VERSION_SANITY "${CSI_PROW_GO_VERSION_BUILD}" "Go version for building the csi-sanity test suite" # depends on CSI_PROW_SANITY settings below
configvar CSI_PROW_GO_VERSION_KIND "${CSI_PROW_GO_VERSION_BUILD}" "Go version for building 'kind'" # depends on CSI_PROW_KIND_VERSION below
configvar CSI_PROW_GO_VERSION_GINKGO "${CSI_PROW_GO_VERSION_BUILD}" "Go version for building ginkgo" # depends on CSI_PROW_GINKGO_VERSION below