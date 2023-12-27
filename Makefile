# VERSION defines the project version for the bundle.
# Update this value when you upgrade the version of your project.
# To re-generate a bundle for another specific version without changing the standard setup, you can:
# - use the VERSION as arg of the bundle target (e.g make bundle VERSION=0.0.2)
# - use environment variables to overwrite this value (e.g export VERSION=0.0.2)
VERSION ?= 0.0.1

# CHANNELS define the bundle channels used in the bundle.
# Add a new line here if you would like to change its default config. (E.g CHANNELS = "candidate,fast,stable")
# To re-generate a bundle for other specific channels without changing the standard setup, you can:
# - use the CHANNELS as arg of the bundle target (e.g make bundle CHANNELS=candidate,fast,stable)
# - use environment variables to overwrite this value (e.g export CHANNELS="candidate,fast,stable")
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif

# DEFAULT_CHANNEL defines the default channel used in the bundle.
# Add a new line here if you would like to change its default config. (E.g DEFAULT_CHANNEL = "stable")
# To re-generate a bundle for any other default channel without changing the default setup, you can:
# - use the DEFAULT_CHANNEL as arg of the bundle target (e.g make bundle DEFAULT_CHANNEL=stable)
# - use environment variables to overwrite this value (e.g export DEFAULT_CHANNEL="stable")
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# IMAGE_TAG_BASE defines the docker.io namespace and part of the image name for remote images.
# This variable is used to construct full image tags for bundle and catalog images.
#
# For example, running 'make bundle-build bundle-push catalog-build catalog-push' will build and push both
# aiven.io/kubernetes-operator-bundle:$VERSION and aiven.io/k8s-exp-catalog:$VERSION.
IMAGE_TAG_BASE ?= aiven.io/kubernetes-operator

# BUNDLE_IMG defines the image:tag used for the bundle.
# You can use it as an arg. (E.g make bundle-build BUNDLE_IMG=<some-registry>/<project-name-bundle>:<tag>)
BUNDLE_IMG ?= $(IMAGE_TAG_BASE)-bundle:v$(VERSION)

# BUNDLE_GEN_FLAGS are the flags passed to the operator-sdk generate bundle command
BUNDLE_GEN_FLAGS ?= -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)

# USE_IMAGE_DIGESTS defines if images are resolved via tags or digests
# You can enable this value if you would like to use SHA Based Digests
# To enable set flag to true
USE_IMAGE_DIGESTS ?= false
ifeq ($(USE_IMAGE_DIGESTS), true)
	BUNDLE_GEN_FLAGS += --use-image-digests
endif

# Image URL to use all building/pushing image targets
IMG ?= aivenoy/aiven-operator:${IMG_TAG}
IMG_TAG ?= $(shell git rev-parse HEAD)
# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.24.2

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: charts
charts: ## Updates helm charts, updates changelog in docs (removes placeholder header)
	go run ./generators/charts/... --version=$(version) --operator-charts ./charts/aiven-operator --crd-charts ./charts/aiven-operator-crds
	[ "$(version)" == "" ] || sed '/## \[/d' ./CHANGELOG.md > docs/docs/changelog.md

.PHONY: userconfigs
userconfigs:
	go generate ./...

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd:allowDangerousTypes=true webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: boilerplate
boilerplate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: generate
generate: userconfigs boilerplate imports manifests docs charts fmt

.PHONY: fmt
fmt:
	go fmt ./...
	trunk fmt

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test-e2e
test-e2e: build ## Run end-to-end tests using kuttl (https://kuttl.dev/)
	@[ "${AIVEN_TOKEN}" ] || ( echo ">> variable AIVEN_TOKEN is not set"; exit 1 )
	@[ "${AIVEN_PROJECT_NAME}" ] || ( echo ">> variable AIVEN_PROJECT_NAME is not set"; exit 1 )
	kubectl kuttl test --config test/e2e/kuttl-test.yaml

.PHONY: test-e2e-preinstalled
test-e2e-preinstalled:
	kubectl kuttl test --config test/e2e/kuttl-test.preinstalled.yaml

test: envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" \
	go test ./tests/... -race -run=$(run) -v -timeout 42m -parallel 10 -cover -coverpkg=./controllers -covermode=atomic -coverprofile=coverage.out

##@ Build

.PHONY: build
build: generate fmt vet ## Build manager binary.
	go build -o bin/manager main.go

.PHONY: run
run: generate install fmt vet ## Run a controller from your host.
	go run ./main.go

.PHONY: docker-build
docker-build: test ## Build docker image with the manager.
	docker build -t ${IMG} .

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	docker push ${IMG}

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy
deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/default | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

##@ Docs

.PHONY: docs
docs: ## Generate CRDS api-reference
	go run ./generators/docs/...

.PHONY: serve-docs
serve-docs: ## Run live preview.
	docker run --rm -it -p 8000:8000 -v ${PWD}/docs:/docs squidfunk/mkdocs-material

##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)
TOOLS_DIR := hack/tools

## Tool Binaries
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest
GOLANGCILINT=$(LOCALBIN)/golangci-lint
GEN_CRD_API_REF_DOCS=$(LOCALBIN)/gen-crd-api-reference-docs
GOIMPORTS ?= $(LOCALBIN)/goimports


## Tool Versions
KUSTOMIZE_VERSION ?= v4.2.0
CONTROLLER_TOOLS_VERSION ?= v0.9.2

KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"
.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	test -s $(LOCALBIN)/kustomize || { curl -s $(KUSTOMIZE_INSTALL_SCRIPT) | bash -s -- $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN); }

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

.PHONY: bundle
bundle: manifests kustomize ## Generate bundle manifests and metadata, then validate generated files.
	operator-sdk generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	$(KUSTOMIZE) build config/manifests | operator-sdk generate bundle $(BUNDLE_GEN_FLAGS)
	operator-sdk bundle validate ./bundle

.PHONY: bundle-build
bundle-build: ## Build the bundle image.
	docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

.PHONY: bundle-push
bundle-push: ## Push the bundle image.
	$(MAKE) docker-push IMG=$(BUNDLE_IMG)

.PHONY: opm
OPM = ./bin/opm
opm: ## Download opm locally if necessary.
ifeq (,$(wildcard $(OPM)))
ifeq (,$(shell which opm 2>/dev/null))
	@{ \
	set -e ;\
	mkdir -p $(dir $(OPM)) ;\
	OS=$(shell go env GOOS) && ARCH=$(shell go env GOARCH) && \
	curl -sSLo $(OPM) https://github.com/operator-framework/operator-registry/releases/download/v1.23.0/$${OS}-$${ARCH}-opm ;\
	chmod +x $(OPM) ;\
	}
else
OPM = $(shell which opm)
endif
endif

# A comma-separated list of bundle images (e.g. make catalog-build BUNDLE_IMGS=example.com/operator-bundle:v0.1.0,example.com/operator-bundle:v0.2.0).
# These images MUST exist in a registry and be pull-able.
BUNDLE_IMGS ?= $(BUNDLE_IMG)

# The image tag given to the resulting catalog image (e.g. make catalog-build CATALOG_IMG=example.com/operator-catalog:v0.2.0).
CATALOG_IMG ?= $(IMAGE_TAG_BASE)-catalog:v$(VERSION)

# Set CATALOG_BASE_IMG to an existing catalog image tag to add $BUNDLE_IMGS to that image.
ifneq ($(origin CATALOG_BASE_IMG), undefined)
FROM_INDEX_OPT := --from-index $(CATALOG_BASE_IMG)
endif

# Build a catalog image by adding bundle images to an empty catalog using the operator package manager tool, 'opm'.
# This recipe invokes 'opm' in 'semver' bundle add mode. For more information on add modes, see:
# https://github.com/operator-framework/community-operators/blob/7f1438c/docs/packaging-operator.md#updating-your-existing-operator
.PHONY: catalog-build
catalog-build: opm ## Build a catalog image.
	$(OPM) index add --container-tool docker --mode semver --tag $(CATALOG_IMG) --bundles $(BUNDLE_IMGS) $(FROM_INDEX_OPT)

# Push the catalog image.
.PHONY: catalog-push
catalog-push: ## Push a catalog image.
	$(MAKE) docker-push IMG=$(CATALOG_IMG)

.PHONY: build-manifests
build-manifests: manifests kustomize
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	mkdir -p releases
	$(KUSTOMIZE) build config/default > releases/aiven-operator-${IMG_TAG}.yaml

.PHONY: e2e-setup-kind
WEBHOOKS_ENABLED ?= true
CERT_MANAGER_TAG ?= v1.11.0
OPERATOR_IMAGE_TAG ?= $(shell git rev-parse HEAD)

PODMAN = podman
# Podman requires specific image name
OPERATOR_IMAGE_NAME ?= localhost/operator
ifeq (, $(shell which podman))
	PODMAN = docker
	OPERATOR_IMAGE_NAME = operator
endif

SETUP_PREREQUISITES = jq base64 kcat helm kind $(PODMAN) avn
e2e-setup-kind:
	# Validates prerequisites
	$(foreach bin,$(SETUP_PREREQUISITES),\
        $(if $(shell command -v $(bin) 2> /dev/null),,$(error Please install `$(bin)` first)))
	@[ "${AIVEN_TOKEN}" ] || ( echo ">> variable AIVEN_TOKEN is not set"; exit 1 )
	@[ "${AIVEN_PROJECT_NAME}" ] || ( echo ">> variable AIVEN_PROJECT_NAME is not set"; exit 1 )

	# Cleans previous installation. Ignores errors
	helm uninstall aiven-operator-crds || true
	helm uninstall aiven-operator || true
	kubectl delete secret aiven-token || true

	# Cleanups cert-manager
	helm uninstall cert-manager || true
	kubectl delete namespace cert-manager || true

	# Installs cert manager if webhooks enabled
	# We use helm here instead of "kubectl apply", because it waits pods up and running
	# Otherwise, operator will fail webhook check
ifeq ($(WEBHOOKS_ENABLED), true)
	helm repo add jetstack https://charts.jetstack.io
	helm repo update
	helm install \
      cert-manager jetstack/cert-manager \
      --namespace cert-manager \
      --create-namespace \
      --version $(CERT_MANAGER_TAG) \
      --set installCRDs=true
endif

	# Builds the operator
	$(PODMAN) build -t ${OPERATOR_IMAGE_NAME}:${OPERATOR_IMAGE_TAG} .
	kind load docker-image $(OPERATOR_IMAGE_NAME):$(OPERATOR_IMAGE_TAG)

	# Installs operators charts
	kubectl create secret generic aiven-token --from-literal=token=$(AIVEN_TOKEN)
	helm install aiven-operator-crds charts/aiven-operator-crds
	helm install \
		--set defaultTokenSecret.name=aiven-token \
		--set defaultTokenSecret.key=token \
		--set leaderElect=true \
		--set image.repository="${OPERATOR_IMAGE_NAME}" \
		--set image.tag=$(OPERATOR_IMAGE_TAG) \
		--set image.pullPolicy="Never" \
		--set resources.requests.cpu=0 \
		--set resources.requests.memory=0 \
		--set webhooks.enabled=${WEBHOOKS_ENABLED} \
		aiven-operator charts/aiven-operator

.PHONY: goimports
goimports: $(GOIMPORTS) ## Download goimports locally if necessary.
$(GOIMPORTS): $(LOCALBIN)
	test -s $(LOCALBIN)/goimports || GOBIN=$(LOCALBIN) go install golang.org/x/tools/cmd/goimports@latest

# On MACOS requires gnu-sed. Run `brew info gnu-sed` and follow instructions to replace default sed.
.PHONY: imports
imports: goimports
	find . -type f -name '*.go' -exec sed -zi 's/"\n\+\t"/"\n"/g' {} +
	$(GOIMPORTS) -local "github.com/aiven/aiven-operator" -w .
