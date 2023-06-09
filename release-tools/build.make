# Copyright 2019 The Kubernetes Authors.
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

.PHONY: build-% build container-% container push-% push clean test

# A space-separated list of all commands in the repository, must be
# set in main Makefile of a repository.
# CMDS=

# This is the default. It can be overridden in the main Makefile after
# including build.make.
REGISTRY_NAME?=quay.io/k8scsi

# Can be set to -mod=vendor to ensure that the "vendor" directory is used.
GOFLAGS_VENDOR=

# Revision that gets built into each binary via the main.version
# string. Uses the `git describe` output based on the most recent
# version tag with a short revision suffix or, if nothing has been
# tagged yet, just the revision.
#
# Beware that tags may also be missing in shallow clones as done by
# some CI systems (like TravisCI, which pulls only 50 commits).
REV=$(shell git describe --long --tags --match='v*' --dirty 2>/dev/null || git rev-list -n1 HEAD)

# A space-separated list of image tags under which the current build is to be pushed.
# Determined dynamically.
IMAGE_TAGS=

# A "canary" image gets built if the current commit is the head of the remote "master" branch.
# That branch does not exist when building some other branch in TravisCI.
IMAGE_TAGS+=$(shell if [ "$$(git rev-list -n1 HEAD)" = "$$(git rev-list -n1 origin/master 2>/dev/null)" ]; then echo "canary"; fi)

# A "X.Y.Z-canary" image gets built if the current commit is the head of a "origin/release-X.Y.Z" branch.
# The actual suffix does not matter, only the "release-" prefix is checked.
IMAGE_TAGS+=$(shell git branch -r --points-at=HEAD | grep 'origin/release-' | grep -v -e ' -> ' | sed -e 's;.*/release-\(.*\);\1-canary;')

# A release image "vX.Y.Z" gets built if there is a tag of that format for the current commit.
# --abbrev=0 suppresses long format, only showing the closest tag.
IMAGE_TAGS+=$(shell tagged="$$(git describe --tags --match='v*' --abbrev=0)"; if [ "$$tagged" ] && [ "$$(git rev-list -n1 HEAD)" = "$$(git rev-list -n1 $$tagged)" ]; then echo $$tagged; fi)

# Images are named after the command contained in them.
IMAGE_NAME=$(REGISTRY_NAME)/$*

TESTARGS ?= -race -coverprofile=coverage.out
ifdef V
# Adding "-alsologtostderr" assumes that all test binaries contain glog. This is not guaranteed.
TESTARGS += -v -args -alsologtostderr -v 5
else
TESTARGS +=
endif

# Specific packages can be excluded from each of the tests below by setting the *_FILTER_CMD variables
# to something like "| grep -v 'github.com/kubernetes-csi/project/pkg/foobar'". See usage below.

# BUILD_PLATFORMS contains a set of <os> <arch> <suffix> triplets,
# separated by semicolon. An empty variable or empty entry (= just a
# semicolon) builds for the default platform of the current Go
# toolchain.
BUILD_PLATFORMS ?= linux

# Add go ldflags using LDFLAGS at the time of compilation.
IMPORTPATH_LDFLAGS = -X main.version=$(REV)
EXT_LDFLAGS = -extldflags "-static"
LDFLAGS =
FULL_LDFLAGS = $(LDFLAGS) $(IMPORTPATH_LDFLAGS) $(EXT_LDFLAGS)
# This builds each command (= the sub-directories of ./cmd) for the target platform(s)
# defined by BUILD_PLATFORMS.
$(CMDS:%=build-%): build-%: check-go-version-go
	mkdir -p bin
	echo '$(BUILD_PLATFORMS)' | tr ';' '\n' | while read -r os arch suffix; do \
		if ! (set -x; CGO_ENABLED=0 GOOS="$$os" GOARCH="$$arch" go build $(GOFLAGS_VENDOR) -a -ldflags '$(FULL_LDFLAGS)' -o "./bin/$*$$suffix" ./cmd/$*); then \
			echo "Building $* for GOOS=$$os GOARCH=$$arch failed, see error(s) above."; \
			exit 1; \
		fi; \
	done

$(CMDS:%=container-%): container-%:
	docker build -t $*:latest -f $(shell if [ -e ./build/$*/Dockerfile ]; then echo ./build/$*/Dockerfile; else echo Dockerfile; fi) --label revision=$(REV) .

$(CMDS:%=push-%): push-%: container-%
	set -ex; \
	push_image () { \
		docker tag $*:latest $(IMAGE_NAME):$$tag; \
		docker push $(IMAGE_NAME):$$tag; \
	}; \
	for tag in $(IMAGE_TAGS); do \
		if [ "$$tag" = "canary" ] || echo "$$tag" | grep -q -e '-canary$$'; then \
			: "creating or overwriting canary image"; \
			push_image; \
		elif docker pull $(IMAGE_NAME):$$tag 2>&1 | tee /dev/stderr | grep -q "manifest for $(IMAGE_NAME):$$tag not found"; then \
			: "creating release image"; \
			push_image; \
		else \
			: "release image $(IMAGE_NAME):$$tag already exists, skipping push"; \
		fi; \
	done

build: $(CMDS:%=build-%)
container: $(CMDS:%=container-%)
push: $(CMDS:%=push-%)

# Additional parameters are needed when pushing to a local registry,
# see https://github.com/docker/buildx/issues/94.
# However, that then runs into https://github.com/docker/cli/issues/2396.
#
# What works for local testing is:
# make push-multiarch PULL_BASE_REF=master REGISTRY_NAME=<your account on dockerhub.io> BUILD_PLATFORMS="linux amd64; windows amd64 .exe; linux ppc64le -ppc64le; linux s390x -s390x"
DOCKER_BUILDX_CREATE_ARGS ?=

# This target builds a multiarch image for one command using Moby BuildKit builder toolkit.
# Docker Buildx is included in Docker 19.03.
#
# ./cmd/<command>/Dockerfile[.Windows] is used if found, otherwise Dockerfile[.Windows].
# It is currently optional: if no such file exists, Windows images are not included,
# even when Windows is listed in BUILD_PLATFORMS. That way, projects can test that
# Windows binaries can be built before adding a Dockerfile for it.
#
# BUILD_PLATFORMS determines which individual images are included in the multiarch image.
# PULL_BASE_REF must be set to 'master', 'release-x.y', or a tag name, and determines
# the tag for the resulting multiarch image.
$(CMDS:%=push-multiarch-%): push-multiarch-%: check-pull-base-ref build-%
	set -ex; \
	DOCKER_CLI_EXPERIMENTAL=enabled; \
	export DOCKER_CLI_EXPERIMENTAL; \
	docker buildx create $(DOCKER_BUILDX_CREATE_ARGS) --use --name multiarchimage-buildertest; \
	trap "docker buildx rm multiarchimage-buildertest" EXIT; \
	dockerfile_linux=$$(if [ -e ./cmd/$*/Dockerfile ]; then echo ./cmd/$*/Dockerfile; else echo Dockerfile; fi); \
	dockerfile_windows=$$(if [ -e ./cmd/$*/Dockerfile.Windows ]; then echo ./cmd/$*/Dockerfile.Windows; else echo Dockerfile.Windows; fi); \
	if [ '$(BUILD_PLATFORMS)' ]; then build_platforms='$(BUILD_PLATFORMS)'; else build_platforms="linux amd64"; fi; \
	if ! [ -f "$$dockerfile_windows" ]; then \
		build_platforms="$$(echo "$$build_platforms" | sed -e 's/windows *[^ ]* *.exe//g' -e 's/; *;/;/g')"; \
	fi; \
	pushMultiArch () { \
		tag=$$1; \
		echo "$$build_platforms" | tr ';' '\n' | while read -r os arch suffix; do \
			docker buildx build --push \
				--tag $(IMAGE_NAME):$$arch-$$os-$$tag \
				--platform=$$os/$$arch \
				--file $$(eval echo \$${dockerfile_$$os}) \
				--build-arg binary=./bin/$*$$suffix \
				--build-arg ARCH=$$arch \
				--label revision=$(REV) \
				.; \
		done; \
		images=$$(echo "$$build_platforms" | tr ';' '\n' | while read -r os arch suffix; do echo $(IMAGE_NAME):$$arch-$$os-$$tag; done); \
		docker manifest create --amend $(IMAGE_NAME):$$tag $$images; \
		docker manifest push -p $(IMAGE_NAME):$$tag; \
	}; \
	if [ $(PULL_BASE_REF) = "master" ]; then \
			: "creating or overwriting canary image"; \
			pushMultiArch canary; \
	elif echo $(PULL_BASE_REF) | grep -q -e 'release-*' ; then \
			: "creating or overwriting canary image for release branch"; \
			release_canary_tag=$$(echo $(PULL_BASE_REF) | cut -f2 -d '-')-canary; \
			pushMultiArch $$release_canary_tag; \
	elif docker pull $(IMAGE_NAME):$(PULL_BASE_REF) 2>&1 | tee /dev/stderr | grep -q "manifest for $(IMAGE_NAME):$(PULL_BASE_REF) not found"; then \
			: "creating release image"; \
			pushMultiArch $(PULL_BASE_REF); \
	else \
			: "ERROR: release image $(IMAGE_NAME):$(PULL_BASE_REF) already exists: a new tag is required!"; \
			exit 1; \
	fi

.PHONY: check-pull-base-ref
check-pull-base-ref:
	if ! [ "$(PULL_BASE_REF)" ]; then \
		echo >&2 "ERROR: PULL_BASE_REF must be set to 'master', 'release-x.y', or a tag name."; \
		exit 1; \
	fi

.PHONY: push-multiarch
push-multiarch: $(CMDS:%=push-multiarch-%)

clean:
	-rm -rf bin

test: check-go-version-go

.PHONY: test-go
test: test-go
test-go:
	@ echo; echo "### $@:"
	go test $(GOFLAGS_VENDOR) `go list $(GOFLAGS_VENDOR) ./... | grep -v -e 'vendor' -e '/test/e2e$$' $(TEST_GO_FILTER_CMD)` $(TESTARGS); \
    go tool cover -func coverage.out | tail -n 1 | awk '{ print "Total coverage: " $$3 }'

.PHONY: test-vet
test: test-vet
test-vet:
	@ echo; echo "### $@:"
	go vet $(GOFLAGS_VENDOR) `go list $(GOFLAGS_VENDOR) ./... | grep -v vendor $(TEST_VET_FILTER_CMD)`

.PHONY: test-fmt
test: test-fmt
test-fmt:
	@ echo; echo "### $@:"
	files=$$(find . -name '*.go' | grep -v './vendor' $(TEST_FMT_FILTER_CMD)); \
	if [ $$(gofmt -d $$files | wc -l) -ne 0 ]; then \
		echo "formatting errors:"; \
		gofmt -d $$files; \
		false; \
	fi

# Targets in the makefile can depend on check-go-version-<path to go binary>
# to trigger a warning if the x.y version of that binary does not match
# what the project uses. Make ensures that this is only checked once per
# invocation.
.PHONY: check-go-version-%
check-go-version-%:
	./release-tools/verify-go-version.sh "$*"
