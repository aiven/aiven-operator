SHELL := /bin/bash
ROOT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))

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
OPERATOR_SDK_DL_URL=https://github.com/operator-framework/operator-sdk/releases/download/v1.11.0
$(OPERATOR_SDK):
	mkdir -p $(TOOLS_BIN_DIR)
	bash -c "set +x; \
		$(eval TMPDIR=$(shell mktemp -d)) \
		trap 'rm -rf $(TMPDIR)' EXIT \
		&& cd $(TMPDIR) \
		&& wget ${OPERATOR_SDK_DL_URL}/operator-sdk_$(GOOS)_$(GOARCH) \
		&& wget ${OPERATOR_SDK_DL_URL}/checksums.txt \
		&& wget ${OPERATOR_SDK_DL_URL}/checksums.txt.asc \
		&& gpg --keyserver keyserver.ubuntu.com --recv-keys 052996E2A20B5C7E \
		&& gpg -u 'Operator SDK (release) <cncf-operator-sdk@cncf.io>' --verify $(TMPDIR)/checksums.txt.asc \
		&& grep operator-sdk_${GOOS}_${GOARCH} checksums.txt | sha256sum -c - \
		&& chmod 750 operator-sdk_$(GOOS)_$(GOARCH) \
		&& mv operator-sdk_$(GOOS)_$(GOARCH) $(ROOT_DIR)/${OPERATOR_SDK}"

ENVTEST_K8S_VERSION=1.22.0
ENVTEST_TOOLS_DIR=$(TOOLS_BIN_DIR)/k8s/$(ENVTEST_K8S_VERSION)-$(GOOS)-$(GOARCH)
ENVTEST_TOOLS_ETCD=$(ENVTEST_TOOLS_DIR)/etcd
ENVTEST_TOOLS_KUBE_APISERVER=$(ENVTEST_TOOLS_DIR)/kube-apiserver
ENVTEST_TOOLS_KUBECTL=$(ENVTEST_TOOLS_DIR)/kubectl
ENVTEST_TOOLS= $(ENVTEST_TOOLS_ETCD) $(ENVTEST_TOOLS_KUBE_APISERVER) $(ENVTEST_TOOLS_KUBECTL)

$(ENVTEST_TOOLS) &: $(SETUP_ENVTEST)
	$(SETUP_ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(TOOLS_BIN_DIR)
	chmod 750 -R $(ENVTEST_TOOLS_DIR)

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

.PHONY: clean-dist
clean-dist:
	rm -rf $(DIST_DIR)

.PHONY: clean-tools
clean-tools:
	rm -rf $(TOOLS_BIN_DIR)

.PHONY: clean-all
clean-all: clean-dist clean-tools

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
	$(GO) run hack/genrefs/main.go
	$(HUGO) --minify -s docs


##@ Distribution

# Actions to generate distributions ( operatorhub, helm, etc. )

VERSION=$(shell git describe --tags $(shell git rev-list --tags --max-count=1))
SHORT_VERSION=$(shell echo $(VERSION) | tr -d v)
IMG=aivenoy/aiven-operator:$(VERSION)

DIST_DIR=dist

$(DIST_DIR):
	mkdir -p $(DIST_DIR)

DIST_CONFIG_DIR=$(DIST_DIR)/config

$(DIST_CONFIG_DIR): $(DIST_DIR) config
	cp -r config $(DIST_CONFIG_DIR)

# Build operatorhub bundle

DIST_DIR_BUNDLE=$(DIST_DIR)/bundle

$(DIST_DIR_BUNDLE): $(DIST_DIR)
	mkdir -p $(DIST_DIR_BUNDLE)

DIST_BUNDLE=$(DIST_DIR_BUNDLE)/bundle
DIST_BUNDLE_DOCKERFILE=$(DIST_DIR_BUNDLE)/bundle.Dockerfile

$(DIST_BUNDLE) $(DIST_BUNDLE_DOCKERFILE) &: manifests $(DIST_DIR_BUNDLE) $(DIST_PROJECT) $(DIST_CONFIG_DIR) $(KUSTOMIZE) $(OPERATOR_SDK)
	cd $(DIST_CONFIG_DIR)/manager && $(abspath $(KUSTOMIZE)) edit set image controller=$(IMG)
	$(abspath $(KUSTOMIZE)) build $(DIST_CONFIG_DIR)/operatorhub/manifests | $(abspath $(OPERATOR_SDK)) generate bundle --package aiven-operator --version $(SHORT_VERSION)
	mv bundle $(DIST_BUNDLE)
	mv bundle.Dockerfile $(DIST_BUNDLE_DOCKERFILE)
	$(OPERATOR_SDK) bundle validate $(DIST_BUNDLE)

.PHONY: bundle
bundle: clean-dist $(DIST_BUNDLE) $(DIST_BUNDLE_DOCKERFILE) ## Generate bundle manifests and metadata, then validate generated files.

.PHONY: bundle-docker-build
bundle-docker-build: bundle
	@[ "${BUNDLE_IMG}" ] || ( echo ">> variable BUNDLE_IMG is not set"; exit 1 )
	docker build -f $(DIST_BUNDLE_DOCKERFILE) $(DIST_DIR_BUNDLE) -t $(BUNDLE_IMG)

.PHONY: bundle-docker-push
bundle-docker-push: bundle-docker-build
	docker push $(BUNDLE_IMG)

.PHONY: bundle-scorecard
bundle-scorecard: bundle-docker-push $(OPERATOR_SDK) ## Run scorecard tests against the bundle distribution
	$(OPERATOR_SDK) scorecard $(BUNDLE_IMG) --config $(DIST_DIR_BUNDLE)/bundle/tests/scorecard/config.yaml -w 120s

.PHONY: bundle-test-run
bundle-test-run: bundle-docker-push $(OPERATOR_SDK) ## Run the bundle against your cluster ( this will reinstall OLM, use on disposable clusters like KIND )
	$(OPERATOR_SDK) olm uninstall
	$(OPERATOR_SDK) olm install
	$(OPERATOR_SDK) run bundle $(BUNDLE_IMG)

install: manifests ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

uninstall: manifests ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

run: manifests generate install ## Run a controller from your host.
	$(GO) run ./main.go