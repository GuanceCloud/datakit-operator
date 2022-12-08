default: local

VERSION=v1.0.1

BIN = datakit-operator
ENTRY = main.go
BUILD_DIR = build
ARCH_AMD64 = amd64
ARCH_ARM64 = arm64
IMAGE_ARCH_AMD64 = linux/amd64
IMAGE_ARCH_ARM64 = linux/amd64
#UNAME_M:=$(shell uname -m | sed -e s/x86_64/x86_64/ -e s/aarch64.\*/arm64/)

define build
	@rm -rf $(BUILD_DIR)/*
	@bash tls.sh

	@echo "======= $(BIN) $(1) ======="
	@GO111MODULE=off CGO_ENABLED=0 GOOS=linux GOARCH=$(1) go build -o $(BUILD_DIR)/$(1)/$(BIN) -pkgdir $(ENTRY)
	@echo "[OK] $(BUILD_DIR)/$(1)/$(BIN)"
	@echo "======= $(BIN) $(2) ======="
	@GO111MODULE=off CGO_ENABLED=0 GOOS=linux GOARCH=$(2) go build -o $(BUILD_DIR)/$(2)/$(BIN) -pkgdir $(ENTRY)
	@echo "[OK] $(BUILD_DIR)/$(2)/$(BIN)"

	@echo "----"
	@tree -Csh -L 3 $(BUILD_DIR)
	
	@sed -e "s/{{VERSION}}/$(VERSION)/g" -e "s/{{CABUNDLE}}/`cat $(BUILD_DIR)/certs/tls.crt | base64 | tr -d "\n"`/g" datakit-operator.yaml.template > datakit-operator.yaml 
	
endef

define image
	sudo docker buildx build --platform $(1) \
		-t $(2)/datakit-operator/datakit-operator:$(VERSION) . --push
	sudo docker buildx build --platform $(1) \
		-t $(2)/datakit-operator/datakit-operator:latest . --push
endef

local:
	$(call build, $(ARCH_ARM64), $(ARCH_AMD64))

pub_image:
	$(call image, $(IMAGE_ARCH_ARM64), pubrepo.jiagouyun.com)
	$(call image, $(IMAGE_ARCH_AMD64), pubrepo.jiagouyun.com)

pub_testing_image:
	$(call image, $(IMAGE_ARCH_ARM64), register.jiagouyun.com)
	$(call image, $(IMAGE_ARCH_AMD64), register.jiagouyun.com)

lint:
	#TODO

all_test:
	#TODO

clean:
	@rm -rf $(BUILD_DIR)/*
