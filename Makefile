SHELL := /bin/bash

all: build

GO = go
GOOS = $(shell $(GO) env GOOS)
GOARCH = $(shell $(GO) env GOARCH)

TOOLS_DIR := hack/tools
TOOLS_BIN_DIR := $(TOOLS_DIR)/bin

HUGO=$(TOOLS_BIN_DIR)/hugo
CONTROLLER_GEN=$(TOOLS_BIN_DIR)/controller-gen
SETUP_ENVTEST=$(TOOLS_BIN_DIR)/setup-envtest
KUSTOMIZE=$(TOOLS_BIN_DIR)/kustomize
GINKGO=$(TOOLS_BIN_DIR)/ginkgo
GOLANGCILINT=$(TOOLS_BIN_DIR)/golangci-lint
GEN_CRD_API_REF_DOCS=$(TOOLS_BIN_DIR)/gen-crd-api-reference-docs
OPERATOR_SDK=$(TOOLS_BIN_DIR)/operator-sdk

$(HUGO): $(TOOLS_DIR)/go.mod ## Build hugo from tools folder.
	cd $(TOOLS_DIR) && $(GO) build -tags=tools,extended -o bin/hugo github.com/gohugoio/hugo

$(CONTROLLER_GEN): $(TOOLS_DIR)/go.mod ## Build controller-gen from tools folder.
	cd $(TOOLS_DIR) && $(GO) build -tags=tools -o bin/controller-gen sigs.k8s.io/controller-tools/cmd/controller-gen

$(SETUP_ENVTEST): $(TOOLS_DIR)/go.mod ## Build kustomize from tools folder.
	cd $(TOOLS_DIR) && $(GO) build -tags=tools -o bin/setup-envtest sigs.k8s.io/controller-runtime/tools/setup-envtest

$(KUSTOMIZE): $(TOOLS_DIR)/go.mod ## Build kustomize from tools folder.
	cd $(TOOLS_DIR) && $(GO) build -tags=tools -o bin/kustomize sigs.k8s.io/kustomize/kustomize/v3

$(GINKGO): $(TOOLS_DIR)/go.mod ## Build ginkgo from tools folder.
	cd $(TOOLS_DIR) && $(GO) build -tags=tools -o bin/ginkgo github.com/onsi/ginkgo/ginkgo

$(GOLANGCILINT): $(TOOLS_DIR)/go.mod ## Build golangci-lint from tools folder.
	cd $(TOOLS_DIR) && $(GO) build -tags=tools -o bin/golangci-lint github.com/golangci/golangci-lint/cmd/golangci-lint

$(GEN_CRD_API_REF_DOCS): $(TOOLS_DIR)/go.mod ## Build gen-crd-api-ref-docs from tools folder.
	cd $(TOOLS_DIR) && $(GO) build -tags=tools -o bin/gen-crd-api-ref-docs github.com/ahmetb/gen-crd-api-reference-docs

# I would like to also manage this in tools.go but there are issues with the grpc version
# TODO: Check again later if we can migrate this there
OPERATOR_SDK_DL_URL=https://github.com/operator-framework/operator-sdk/releases/download/v1.12.0
$(OPERATOR_SDK): ## Build operator-sdk from tools folder.
	curl -LO ${OPERATOR_SDK_DL_URL}/operator-sdk_${GOOS}_${GOARCH}
	gpg --keyserver keyserver.ubuntu.com --recv-keys 052996E2A20B5C7E
	curl -LO ${OPERATOR_SDK_DL_URL}/checksums.txt
	curl -LO ${OPERATOR_SDK_DL_URL}/checksums.txt.asc
	gpg -u "Operator SDK (release) <cncf-operator-sdk@cncf.io>" --verify checksums.txt.asc
	grep operator-sdk_${GOOS}_${GOARCH} checksums.txt | sha256sum -c -
	chmod +x operator-sdk_${GOOS}_${GOARCH} && sudo mv operator-sdk_${GOOS}_${GOARCH} ${OPERATOR_SDK}
	rm checksums.txt checksums.txt.asc

ENVTEST_K8S_VERSION=1.22.0
ENVTEST_TOOLS_DIR=$(TOOLS_BIN_DIR)/k8s/$(ENVTEST_K8S_VERSION)-$(GOOS)-$(GOARCH)
ENVTEST_TOOLS_ETCD=$(ENVTEST_TOOLS_DIR)/etcd
ENVTEST_TOOLS_KUBE_APISERVER=$(ENVTEST_TOOLS_DIR)/kube-apiserver
ENVTEST_TOOLS_KUBECTL=$(ENVTEST_TOOLS_DIR)/kubectl
ENVTEST_TOOLS= $(ENVTEST_TOOLS_ETCD) $(ENVTEST_TOOLS_KUBE_APISERVER) $(ENVTEST_TOOLS_KUBECTL)

$(ENVTEST_TOOLS): $(SETUP_ENVTEST)
	$(SETUP_ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(TOOLS_BIN_DIR)

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

# Actions to aid in development ( generation, building, testing )

.PHONY: build
build: generate ## Build manager binary.
	$(GO) build -o bin/manager main.go

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	$(GO) run ./main.go

CRD_OPTIONS = "crd:trivialVersions=true,preserveUnknownFields=false"

.PHONY: manifests
manifests: $(CONTROLLER_GEN) ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="api/..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: $(CONTROLLER_GEN) ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="api/..."

.PHONY: lint
lint: $(GOLANGCILINT) ## Run linter.
	$(GOLANGCILINT) run --verbose

.PHONY: test-acc
test-acc: $(GINKGO) $(ENVTEST_TOOLS) ## Run acceptance tests.
	KUBEBUILDER_CONTROLPLANE_START_TIMEOUT=120s \
	KUBEBUILDER_CONTROLPLANE_STOP_TIMEOUT=120s \
	KUBEBUILDER_ATTACH_CONTROL_PLANE_OUTPUT=true \
	KUBEBUILDER_ASSETS=$(abspath $(ENVTEST_TOOLS_DIR)) \
	$(GINKGO) \
		--nodes=4 \
		--race \
		--randomizeAllSpecs \
		--trace \
		--failFast \
		--test.count 1 \
		--progress \
		./controllers

.PHONY: test-e2e
test-e2e: manifests generate ## Run end-to-end tests using kuttl (https://kuttl.dev/)
	kubectl kuttl test --config test/e2e/kuttl-test.yaml

##@ Docs

.PHONY: serve-docs
serve-docs: $(HUGO) ## Run Hugo live preview.
	$(HUGO) serve docs -s docs

.PHONY: generate-docs
generate-docs: $(HUGO) $(GEN_CRD_API_REF_DOCS) ## Generate the documentation website locally.
	$(GO) generate hack/genrefs/gen.go
	$(HUGO) --minify -s docs


##@ Distribution

# Actions to generate distributions ( operatorhub, helm, etc. )

DIST_DIR=dist

$(DIST_DIR):
	mkdir $(DIST_DIR)

# The latest released version
VERSION=$(shell git describe --tags $(shell git rev-list --tags --max-count=1))
IMG=aivenoy/aiven-operator:$(VERSION)

bundle: $(DIST_DIR) $(KUSTOMIZE) $(OPERATOR_SDK) manifests ## Generate bundle manifests and metadata, then validate generated files.
	$(OPERATOR_SDK) generate kustomize manifests -q
	cd config/manager && $(abspath $(KUSTOMIZE)) edit set image controller=$(IMG)
	$(KUSTOMIZE) build config/manifests | $(OPERATOR_SDK) generate bundle -q --overwrite --version $(shell echo $(VERSION) | tr -d v)
	$(OPERATOR_SDK) bundle validate ./bundle
	mv bundle $(DIST_DIR)
	mv bundle.Dockerfile $(DIST_DIR)

