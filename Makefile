default: local

VERSION=v1.5.0

BIN           = datakit-operator
ENTRY         = main.go
BUILD_DIR     = dist
CERT_DIR      = self-certification
ARCH_AMD64    = amd64
ARCH_ARM64    = arm64
IMAGE_ARCHS   = "linux/arm64,linux/amd64"
GOLINT_BINARY = golangci-lint
# UNAME_M:=$(shell uname -m | sed -e s/x86_64/x86_64/ -e s/aarch64.\*/arm64/)

SUPPORTED_GOLINT_VERSION         = 1.46.2
SUPPORTED_GOLINT_VERSION_ANOTHER = v1.46.2

GOLINT_VERSION         := $(shell $(GOLINT_BINARY) --version | cut -c 27- | cut -d' ' -f1)
GOLINT_VERSION_ERR_MSG := golangci-lint version($(GOLINT_VERSION)) is not supported, please use version $(SUPPORTED_GOLINT_VERSION)

define build
	@rm -rf $(BUILD_DIR)/*

	@echo "======= $(BIN) $(1) ======="
	@GO111MODULE=off CGO_ENABLED=0 GOOS=linux GOARCH=$(1) go build -o $(BUILD_DIR)/$(1)/$(BIN) -pkgdir $(ENTRY)
	@echo "[OK] $(BUILD_DIR)/$(1)/$(BIN)"
	@echo "======= $(BIN) $(2) ======="
	@GO111MODULE=off CGO_ENABLED=0 GOOS=linux GOARCH=$(2) go build -o $(BUILD_DIR)/$(2)/$(BIN) -pkgdir $(ENTRY)
	@echo "[OK] $(BUILD_DIR)/$(2)/$(BIN)"

	@echo "----"
	@tree -Csh -L 3 $(BUILD_DIR)

endef

define image
	@sed -e "s/{{HUB}}/$(2)/g" \
		-e "s/{{VERSION}}/$(VERSION)/g" \
		-e "s/{{CABUNDLE}}/`cat $(CERT_DIR)/tls.crt | base64 | tr -d "\n"`/g" \
		datakit-operator.yaml.template > datakit-operator.yaml
	sudo docker buildx build --platform $(1) \
		-t $(2)/datakit-operator/datakit-operator:$(VERSION) . --push
	sudo docker buildx build --platform $(1) \
		-t $(2)/datakit-operator/datakit-operator:latest . --push
endef

define build_k8s_charts
	@helm repo ls
	@echo `echo $(VERSION) | cut -d'-' -f1`
	@sed -e "s,{{REPOSITORY}},$(2)/datakit-operator/datakit-operator,g" \
	     -e "s/{{CABUNDLE}}/`cat $(CERT_DIR)/tls.crt | base64 | tr -d "\n"`/g" \
	     charts/values.yaml > charts/datakit-operator/values.yaml
	@helm package charts/datakit-operator --version `echo $(VERSION) | cut -d'-' -f1` --app-version `echo $(VERSION)`
	@helm cm-push datakit-operator-`echo $(VERSION) | cut -d'-' -f1`.tgz $(1)
	@rm -f datakit-operator-`echo $(VERSION) | cut -d'-' -f1`.tgz
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

local:
	$(call build,$(ARCH_ARM64),$(ARCH_AMD64))

pub_image:
	$(call image,$(IMAGE_ARCHS),pubrepo.guance.com)
	$(call upload,$(PRODUCTION_OSS_HOST),$(PRODUCTION_OSS_BUCKET),$(PRODUCTION_OSS_ACCESS_KEY),$(PRODUCTION_OSS_SECRET_KEY),$(VERSION))
	$(call build_k8s_charts, 'datakit-operator', pubrepo.guance.com)

pub_testing_image:
	$(call image,$(IMAGE_ARCHS),registry.jiagouyun.com)
	$(call upload,$(LOCAL_OSS_HOST),$(LOCAL_OSS_BUCKET),$(LOCAL_OSS_ACCESS_KEY),$(LOCAL_OSS_SECRET_KEY),$(VERSION))
	$(call build_k8s_charts, 'datakit-operator-testing', registry.jiagouyun.com)

lint:
	$(GOLINT_BINARY) run --allow-parallel-runners;
	@if [ $$? != 0 ]; then \
		exit -1; \
	fi

all_test:
	#TODO

clean:
	@rm -rf $(BUILD_DIR)/*
