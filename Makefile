# Image URL to use all building/pushing image targets
IMG_TAG ?= $(shell git rev-parse HEAD)
IMG ?= aivenoy/aiven-operator:${IMG_TAG}
ifeq ($(shell command -v podman 2> /dev/null),)
    CONTAINER_TOOL=docker
else
    CONTAINER_TOOL=podman
endif

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = "1.26.*"
KUBEBUILDER_ASSETS_CMD = '$(ENVTEST) use "$(ENVTEST_K8S_VERSION)" --bin-dir $(LOCALBIN) -p path'

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
charts: ## Updates helm charts, updates changelog in docs (removes placeholder header).
	go run ./generators/charts/... --version=$(version) --operator-charts ./charts/aiven-operator --crd-charts ./charts/aiven-operator-crds
	[ "$(version)" == "" ] || sed '/## \[/d' ./CHANGELOG.md > docs/docs/changelog.md

.PHONY: userconfigs
userconfigs: ## Run userconfigs generator.
	go generate ./...

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd:allowDangerousTypes=true webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: boilerplate
boilerplate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: generate
generate: userconfigs boilerplate imports manifests docs charts fmt ## Run all code generation targets.

.PHONY: fmt
fmt: ## Format code.
	go fmt ./...
	trunk fmt

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

# On MACOS requires gnu-sed. Run `brew info gnu-sed` and follow instructions to replace default sed.
.PHONY: imports
imports: ## Run goimports against code.
	find . -type f -name '*.go' -exec sed -zi 's/"\n\+\t"/"\n"/g' {} +
	goimports -local "github.com/aiven/aiven-operator" -w .

##@ Checks

.PHONY: check-avn-client
check-avn-client: ## Check if avn client is installed and user is authenticated.
	@if ! command -v avn >/dev/null 2>&1; then \
		(echo ">> avn command not found, please install https://github.com/aiven/aiven-client"; exit 1) \
	fi
	@if ! avn user info >/dev/null 2>&1; then \
		(echo ">> User not authenticated, please login first with 'avn user login'"; exit 1) \
	fi

.PHONY: check-env-vars
check-env-vars: ## Check if required environment variables are set.
		@[ "${AIVEN_TOKEN}" ] || (echo ">> variable AIVEN_TOKEN is not set"; exit 1)
		@[ "${AIVEN_PROJECT_NAME}" ] || (echo ">> variable AIVEN_PROJECT_NAME is not set"; exit 1)

##@ Tests

.PHONY: test-e2e
test-e2e: check-env-vars check-avn-client build ## Run end-to-end tests using kuttl (https://kuttl.dev).
	kubectl kuttl test --config test/e2e/kuttl-test.yaml

.PHONY: test-e2e-preinstalled
test-e2e-preinstalled: check-env-vars check-avn-client ## Run end-to-end tests using kuttl (https://kuttl.dev) with preinstalled operator ('make e2e-setup-kind' should be run before this target).
	kubectl kuttl test --config test/e2e/kuttl-test.preinstalled.yaml

test: envtest ## Run tests. To target a specific test, use 'run=TestName make test'.
	export KUBEBUILDER_ASSETS=$(shell eval ${KUBEBUILDER_ASSETS_CMD}); \
	go test ./tests/... -race -run=$(run) -v $(if $(run), -timeout 10m, -timeout 42m) -parallel 10 -cover -coverpkg=./controllers -covermode=atomic -coverprofile=coverage.out

##@ Build

.PHONY: build
build: generate fmt vet ## Build manager binary.
	go build -o bin/manager main.go

.PHONY: run
run: generate install fmt vet ## Run a controller from your host.
	go run ./main.go

.PHONY: docker-build
docker-build: test ## Build docker image with the manager.
	$(CONTAINER_TOOL) build -t ${IMG} .

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	$(CONTAINER_TOOL) push ${IMG}

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
docs: ## Generate CRDs api-reference.
	go run ./generators/docs/...

.PHONY: serve-docs
serve-docs: ## Run live preview.
	$(CONTAINER_TOOL) run --rm -it -p 8000:8000 -v ${PWD}/docs:/docs squidfunk/mkdocs-material

##@ Build dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)
TOOLS_DIR := hack/tools

## Tool Binaries
KUBECTL ?= kubectl
KUSTOMIZE ?= $(LOCALBIN)/kustomize-$(KUSTOMIZE_VERSION)
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen-$(CONTROLLER_TOOLS_VERSION)
ENVTEST ?= $(LOCALBIN)/setup-envtest-$(ENVTEST_VERSION)
GOLANGCI_LINT = $(LOCALBIN)/golangci-lint-$(GOLANGCI_LINT_VERSION)

## Tool Versions
KUSTOMIZE_VERSION ?= v4.5.7
CONTROLLER_TOOLS_VERSION ?= v0.9.2
ENVTEST_VERSION ?= release-0.16
GOLANGCI_LINT_VERSION ?= v1.54.2

.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	$(call go-install-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v4,$(KUSTOMIZE_VERSION))

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	$(call go-install-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen,$(CONTROLLER_TOOLS_VERSION))

# KUBEBUILDER_ASSETS is installed in this target so that it can be used by e.g. IDE test integrations.
.PHONY: envtest
envtest: $(ENVTEST) ## Download setup-envtest locally if necessary.
$(ENVTEST): $(LOCALBIN)
	$(call go-install-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest,$(ENVTEST_VERSION)) ;\
	echo -e ">>Installing kubebuilder assets to path:"; \
	eval $(KUBEBUILDER_ASSETS_CMD); \
	echo -e "\n"

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/cmd/golangci-lint,${GOLANGCI_LINT_VERSION})

ENABLE_WEBHOOKS ?= true
CERT_MANAGER_TAG ?= v1.11.0
OPERATOR_IMAGE_TAG ?= $(shell git rev-parse HEAD)

# Podman requires specific image name
OPERATOR_IMAGE_NAME ?= localhost/operator
ifneq ($(CONTAINER_TOOL), podman)
	OPERATOR_IMAGE_NAME = operator
endif

##@ Test setup and cleanup

# Clean previous installations and delete resources
CLEANUP_TARGETS := aiven-operator-crds aiven-operator aiven-token cert-manager
CLEANUP_NAMESPACES := cert-manager
CLEANUP_SECRETS := aiven-token
.PHONY: cleanup
cleanup: ## Cleanup resources created by e2e-setup-kind.
	$(foreach target,$(CLEANUP_TARGETS),helm uninstall $(target) || true;)
	$(foreach namespace,$(CLEANUP_NAMESPACES),kubectl delete namespace $(namespace) || true;)
	$(foreach secret,$(CLEANUP_SECRETS),kubectl delete secret $(secret) || true;)

SETUP_PREREQUISITES = jq base64 kcat helm kind $(CONTAINER_TOOL) avn
.PHONY: e2e-setup-kind
e2e-setup-kind: check-env-vars ## Setup kind cluster and install operator.
# Validates prerequisites
	$(foreach bin,$(SETUP_PREREQUISITES),\
        $(if $(shell command -v $(bin) 2> /dev/null),,$(error Please install `$(bin)` first)))

# Check that kind cluster is running. Keep the --image argument in sync with developer-docs.md
	@kubectl config view -o jsonpath='{.contexts[*].name}' | grep -q kind-kind || \
	 (echo ">> Kind cluster not found. Please create it using 'kind create cluster --image kindest/node:v1.26.6 --wait 5m'"; exit 1)

	$(MAKE) cleanup

# Installs cert manager if webhooks enabled
# We use helm here instead of "kubectl apply", because it waits pods up and running
# Otherwise, operator will fail webhook check
ifeq ($(ENABLE_WEBHOOKS), true)
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
	$(CONTAINER_TOOL) build -t ${OPERATOR_IMAGE_NAME}:${OPERATOR_IMAGE_TAG} .
ifeq ($(CONTAINER_TOOL), podman)
	podman image save --format oci-archive ${OPERATOR_IMAGE_NAME}:${OPERATOR_IMAGE_TAG} -o /tmp/operator-image.tar
	kind load image-archive /tmp/operator-image.tar
else
	kind load docker-image $(OPERATOR_IMAGE_NAME):$(OPERATOR_IMAGE_TAG)
endif

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
		--set webhooks.enabled=${ENABLE_WEBHOOKS} \
		aiven-operator charts/aiven-operator

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary (ideally with version)
# $2 - package url which can be installed
# $3 - specific version of package
define go-install-tool
	@[ -f $(1) ] || { \
		set -e; \
		package=$(2)@$(3) ;\
		echo "Downloading $${package}" ;\
		GOBIN=$(LOCALBIN) go install $${package} ;\
		mv "$$(echo "$(1)" | sed "s/-$(3)$$//")" $(1) ;\
	}
endef
