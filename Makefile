default: local

VERSION=v1.0.1

BIN = datakit-operator
ENTRY = main.go
BUILD_DIR = dist
CERT_DIR = self-certification
ARCH_AMD64 = amd64
ARCH_ARM64 = arm64
IMAGE_ARCH_AMD64 = linux/amd64
IMAGE_ARCH_ARM64 = linux/arm64
#UNAME_M:=$(shell uname -m | sed -e s/x86_64/x86_64/ -e s/aarch64.\*/arm64/)

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
	
	@sed -e "s/{{VERSION}}/$(VERSION)/g" -e "s/{{CABUNDLE}}/`cat $(CERT_DIR)/tls.crt | base64 | tr -d "\n"`/g" datakit-operator.yaml.template > datakit-operator.yaml 
	
endef

define image
	sudo docker buildx build --platform $(1) \
		-t $(2)/datakit-operator/datakit-operator:$(VERSION) . --push
	sudo docker buildx build --platform $(1) \
		-t $(2)/datakit-operator/datakit-operator:latest . --push
endef

define upload
	@bash upload.sh $(1) $(2) $(3) $(4) $(5)
endef

local:
	$(call build,$(ARCH_ARM64),$(ARCH_AMD64))

pub_image:
	$(call image,$(IMAGE_ARCH_ARM64),pubrepo.jiagouyun.com)
	$(call image,$(IMAGE_ARCH_AMD64),pubrepo.jiagouyun.com)
	$(call upload,$(PRODU_OSS_HOST),$(PRODU_OSS_BUCKET),$(PRODU_OSS_ACCESS_KEY),$(PRODU_OSS_SECRET_KEY),$(VERSION))

pub_testing_image:
	$(call image,$(IMAGE_ARCH_ARM64),registry.jiagouyun.com)
	$(call image,$(IMAGE_ARCH_AMD64),registry.jiagouyun.com)
	$(call upload,$(LOCAL_OSS_HOST),$(LOCAL_OSS_BUCKET),$(LOCAL_OSS_ACCESS_KEY),$(LOCAL_OSS_SECRET_KEY),$(VERSION))

lint:
	#TODO

all_test:
	#TODO

clean:
	@rm -rf $(BUILD_DIR)/*
