default: local

VERSION=v1.0.1

BIN = datakit-operator
ENTRY = main.go
BUILD_DIR = build
UNAME_M:=$(shell uname -m | sed -e s/x86_64/x86_64/ -e s/aarch64.\*/arm64/)

define build
	@rm -rf $(BUILD_DIR)/*
	@bash tls.sh

	@echo "======= $(BIN) $(1) ======="
	@GO111MODULE=off CGO_ENABLED=0 GOARCH=$(1) go build -o $(BUILD_DIR)/$(1)/$(BIN) -pkgdir $(ENTRY) 
	@tree -Csh -L 3 $(BUILD_DIR)
	
	@sed -e "s/{{VERSION}}/$(VERSION)/g" -e "s/{{CABUNDLE}}/`cat $(BUILD_DIR)/certs/tls.crt | base64 | tr -d "\n"`/g" datakit-operator.yaml.template > datakit-operator.yaml 
	
endef

define image
	sudo docker buildx build --platform $(1) \
		-t $(2)/datakit-operator/datakit-operator:$(VERSION) . 
endef

local:
	$(call build,$(UNAME_M))

image:
	$(call image,$(UNAME_M), pubrepo.jiagouyun.com)
