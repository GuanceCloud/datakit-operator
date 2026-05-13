default: local

VERSION=v1.8.6

BIN           = datakit-operator
ENTRY         = ./cmd/main.go
BUILD_DIR     = dist
DOCKERFILE_DIR= dockerfiles
ARCH_AMD64    = amd64
ARCH_ARM64    = arm64
IMAGE_ARCHS   = linux/arm64,linux/amd64
GOLINT_BINARY = golangci-lint
# UNAME_S     = $(shell uname -s)
# UNAME_M     = $(shell uname -m | sed -e s/x86_64/x86_64/ -e s/aarch64.\*/arm64/)

SUPPORTED_GOLINT_VERSION         = 1.46.2
SUPPORTED_GOLINT_VERSION_ANOTHER = v1.46.2

# Make them evaluate(expand) only once
DATE                   := $(shell date -u +'%Y-%m-%d %H:%M:%S')
GOVERSION              := $(shell go version)
COMMIT                 := $(shell git rev-parse --short HEAD)
GIT_BRANCH             := $(shell git rev-parse --abbrev-ref HEAD)
GOLINT_VERSION         := $(shell $(GOLINT_BINARY) --version | cut -c 27- | cut -d' ' -f1)
GOLINT_VERSION_ERR_MSG := golangci-lint version($(GOLINT_VERSION)) is not supported, please use version $(SUPPORTED_GOLINT_VERSION)

# Generate 'pkg/git' package
define GIT_INFO
package git

// nolint
const (
	BuildAt string = "$(DATE)"
	Version string = "$(VERSION)"
	Golang  string = "$(GOVERSION)"
	Commit  string = "$(COMMIT)"
	Branch  string = "$(GIT_BRANCH)"
)
endef
export GIT_INFO

define build
	@rm -rf $(BUILD_DIR)/*

	@echo "======= $(BIN) $(1) ======="
	@GO111MODULE=off CGO_ENABLED=0 GOOS=linux GOARCH=$(1) go build -o $(BUILD_DIR)/$(1)/$(BIN) $(ENTRY)
	@echo "[OK] $(BUILD_DIR)/$(1)/$(BIN)"
	@echo "======= $(BIN) $(2) ======="
	@GO111MODULE=off CGO_ENABLED=0 GOOS=linux GOARCH=$(2) go build -o $(BUILD_DIR)/$(2)/$(BIN) $(ENTRY)
	@echo "[OK] $(BUILD_DIR)/$(2)/$(BIN)"

	@echo "----"
	@tree -Csh -L 3 $(BUILD_DIR)

endef

define build_image
	sudo docker buildx build --platform $(1) -t $(2)/datakit-operator:$(VERSION) -f $(DOCKERFILE_DIR)/Dockerfile . --push
	sudo docker buildx build --platform $(1) -t $(2)/datakit-operator:latest -f $(DOCKERFILE_DIR)/Dockerfile . --push
endef

define build_uos_image
	sudo docker buildx build --platform $(1) -t $(2)/datakit-operator:$(VERSION) -f $(DOCKERFILE_DIR)/Dockerfile.uos . --push
	sudo docker buildx build --platform $(1) -t $(2)/datakit-operator:latest -f $(DOCKERFILE_DIR)/Dockerfile.uos . --push
endef

define build_k8s_charts
	@helm repo ls
	@echo `echo $(VERSION) | cut -d'-' -f1`
	@bash brand.sh $(1) $(VERSION)

	@helm package charts/datakit-operator --version `echo $(VERSION) | cut -d'-' -f1` --app-version `echo $(VERSION)`
	@helm cm-push --force datakit-operator-`echo $(VERSION) | cut -d'-' -f1`.tgz $(2)
	@rm -f datakit-operator-`echo $(VERSION) | cut -d'-' -f1`.tgz
endef

define clean_charts
	@rm -f datakit-operator.yaml
	@cat /dev/null > charts/datakit-operator/README.md
	@cat /dev/null > charts/datakit-operator/Chart.yaml
	@cat /dev/null > charts/datakit-operator/values.yaml
endef

define check_golint_version
	@case $(GOLINT_VERSION) in \
	$(SUPPORTED_GOLINT_VERSION)) \
	;; \
	$(SUPPORTED_GOLINT_VERSION_ANOTHER)) \
	;; \
	*) \
		echo '$(GOLINT_VERSION_ERR_MSG)'; \
		exit 1; \
	esac;
endef

define upload
	@bash upload.sh $(1) $(2) $(3) $(4) $(5)
endef

local: deps
	$(call build,$(ARCH_ARM64),$(ARCH_AMD64))

pub_image:
	$(call clean_charts)
	$(call build_image,$(IMAGE_ARCHS),pubrepo.guance.com/datakit-operator)
	@helm repo add datakit-operator-chart-cn https://pubrepo.guance.com/chartrepo/datakit-operator || true
	@helm repo update
	$(call build_k8s_charts,guance,'datakit-operator-chart-cn')
	$(call upload,$(PRODUCTION_OSS_HOST),$(PRODUCTION_OSS_BUCKET),$(PRODUCTION_OSS_ACCESS_KEY),$(PRODUCTION_OSS_SECRET_KEY),$(VERSION))

	$(call clean_charts)
	$(call build_image,$(IMAGE_ARCHS),pubrepo.guance.com/truewatch)
	@helm repo add datakit-operator-chart-intl https://pubrepo.truewatch.com/chartrepo/truewatch || true
	@helm repo update
	$(call build_k8s_charts,truewatch,'datakit-operator-chart-intl')
	$(call upload,$(TRUEWATCH_PRODUCTION_OSS_HOST),$(TRUEWATCH_PRODUCTION_OSS_BUCKET),$(TRUEWATCH_PRODUCTION_OSS_ACCESS_KEY),$(TRUEWATCH_PRODUCTION_OSS_SECRET_KEY),$(VERSION))


pub_testing_image:
	$(call clean_charts)
	$(call build_image,$(IMAGE_ARCHS),registry.jiagouyun.com/datakit-operator)
	$(call build_k8s_charts,testing,'datakit-operator-testing')
	$(call upload,$(LOCAL_OSS_HOST),$(LOCAL_OSS_BUCKET),$(LOCAL_OSS_ACCESS_KEY),$(LOCAL_OSS_SECRET_KEY),$(VERSION))

pub_uos_image:
	$(call build_uos_image,$(IMAGE_ARCHS),pubrepo.guance.com/uos-dataflux)

lint: deps test
	@bash scripts/check_copyright.sh
	$(GOLINT_BINARY) run --allow-parallel-runners;
	@if [ $$? != 0 ]; then \
		exit -1; \
	fi

deps: prepare gofmt

# ignore files under vendor/.git/git
gofmt:
	@GO111MODULE=off gofmt -w -l $(shell find . -type f -name '*.go'| grep -v "/vendor/\|/.git/\|/git/\|.*_y.go\|packed-packr.go")

prepare:
	@mkdir -p pkg/git
	@echo "$$GIT_INFO" > pkg/git/git.go

test: deps
	@echo "======= Running Go tests recursively... ======="
	@go test ./... | grep -v "no test files" || (echo "Tests failed!" && exit 1)

clean:
	@rm -rf $(BUILD_DIR)/*
	$(call clean_charts)
