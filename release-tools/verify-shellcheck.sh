#!/usr/bin/env bash

# Copyright 2018 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit
set -o nounset
set -o pipefail

# The csi-release-tools directory.
TOOLS="$(dirname "${BASH_SOURCE[0]}")"
. "${TOOLS}/util.sh"

# Directory to check. Default is the parent of the tools themselves.
ROOT="${1:-${TOOLS}/..}"

# required version for this script, if not installed on the host we will
# use the official docker image instead. keep this in sync with SHELLCHECK_IMAGE
SHELLCHECK_VERSION="0.6.0"
# upstream shellcheck latest stable image as of January 10th, 2019
SHELLCHECK_IMAGE="koalaman/shellcheck-alpine:v0.6.0@sha256:7d4d712a2686da99d37580b4e2f45eb658b74e4b01caf67c1099adc294b96b52"

# fixed name for the shellcheck docker container so we can reliably clean it up
SHELLCHECK_CONTAINER="k8s-shellcheck"

# disabled lints
disabled=(
  # this lint disallows non-constant source, which we use extensively without
  # any known bugs
  1090
  # this lint prefers command -v to which, they are not the same
  2230
)
# comma separate for passing to shellcheck
join_by() {
  local IFS="$1";
  shift;
  echo "$*";
}
SHELLCHECK_DISABLED="$(join_by , "${disabled[@]}")"
readonly SHELLCHECK_DISABLED

# creates the shellcheck container for later use
create_container () {
  # TODO(bentheelder): this is a performance hack, we create the container with
  # a sleep MAX_INT32 so that it is effectively paused.
  # We then repeatedly exec to it to run each shellcheck, and later rm it when
  # we're done.
  # This is incredibly much faster than creating a container for each shellcheck
  # call ...
  docker run --name "${SHELLCHECK_CONTAINER}" -d --rm -v "${ROOT}:${ROOT}" -w "${ROOT}" --entrypoint="sleep" "${SHELLCHECK_IMAGE}" 2147483647
}
# removes the shellcheck container
remove_container () {
  docker rm -f "${SHELLCHECK_CONTAINER}" &> /dev/null || true
}

# ensure we're linting the source tree
cd "${ROOT}"

# find all shell scripts excluding ./_*, ./.git/*, ./vendor*,
# and anything git-ignored
all_shell_scripts=()
while IFS=$'\n' read -r script;
  do git check-ignore -q "$script" || all_shell_scripts+=("$script");
done < <(find . -name "*.sh" \
  -not \( \
    -path ./_\*      -o \
    -path ./.git\*   -o \
    -path ./vendor\*    \
  \))

# detect if the host machine has the required shellcheck version installed
# if so, we will use that instead.
HAVE_SHELLCHECK=false
if which shellcheck &>/dev/null; then
  detected_version="$(shellcheck --version | grep 'version: .*')"
  if [[ "${detected_version}" = "version: ${SHELLCHECK_VERSION}" ]]; then
    HAVE_SHELLCHECK=true
  fi
fi

# tell the user which we've selected and possibly set up the container
if ${HAVE_SHELLCHECK}; then
  echo "Using host shellcheck ${SHELLCHECK_VERSION} binary."
else
  echo "Using shellcheck ${SHELLCHECK_VERSION} docker image."
  # remove any previous container, ensure we will attempt to cleanup on exit,
  # and create the container
  remove_container
  kube::util::trap_add 'remove_container' EXIT
  if ! output="$(create_container 2>&1)"; then
      {
        echo "Failed to create shellcheck container with output: "
        echo ""
        echo "${output}"
      } >&2
      exit 1
  fi
fi

# lint each script, tracking failures
errors=()
for f in "${all_shell_scripts[@]}"; do
  set +o errexit
  if ${HAVE_SHELLCHECK}; then
    failedLint=$(shellcheck --exclude="${SHELLCHECK_DISABLED}" "${f}")
  else
    failedLint=$(docker exec -t ${SHELLCHECK_CONTAINER} \
                 shellcheck --exclude="${SHELLCHECK_DISABLED}" "${f}")
  fi
  set -o errexit
  if [[ -n "${failedLint}" ]]; then
      errors+=( "${failedLint}" )
  fi
done

# Check to be sure all the packages that should pass lint are.
if [ ${#errors[@]} -eq 0 ]; then
  echo 'Congratulations! All shell files are passing lint.'
else
  {
    echo "Errors from shellcheck:"
    for err in "${errors[@]}"; do
      echo "$err"
    done
    echo
    echo 'Please review the above warnings. You can test via "./hack/verify-shellcheck"'
    echo 'If the above warnings do not make sense, you can exempt them from shellcheck'
    echo 'checking by adding the "shellcheck disable" directive'
    echo '(https://github.com/koalaman/shellcheck/wiki/Directive#disable).'
    echo
  } >&2
  false
fi