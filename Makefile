# All source code excluding any third party code and excluding the testbed.
# This is the code that we want to run tests for and lint, staticcheck, etc.
ALL_SRC := $(shell find . -name '*.go' \
							-not -path './testbed/*' \
							-not -path '*/third_party/*' \
							-not -path '*/internal/data/opentelemetry-proto/*' \
							-not -path '*/internal/data/opentelemetry-proto-gen/*' \
							-not -path './.circleci/scripts/reportgenerator/*' \
							-not -path './examples/demo/app/*' \
							-type f | sort)

# ALL_PKGS is the list of all packages where ALL_SRC files reside.
ALL_PKGS := $(shell go list $(sort $(dir $(ALL_SRC))))

# All source code and documents. Used in spell check.
ALL_DOC := $(shell find . \( -name "*.md" -o -name "*.yaml" \) \
                                -type f | sort)

GOTEST_OPT?= -v -race -timeout 180s
GO_ACC=go-acc
GOTEST=go test
GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)
ADDLICENSE= addlicense
MISSPELL=misspell -error
MISSPELL_CORRECTION=misspell -w
LINT=golangci-lint
IMPI=impi
GOSEC=gosec
STATIC_CHECK=staticcheck
# BUILD_TYPE should be one of (dev, release).
BUILD_TYPE?=release


GIT_SHA=$(shell git rev-parse --short HEAD)
BUILD_INFO_IMPORT_PATH=github.com/o11y/opentelemetry-collector-o11y/internal/version
BUILD_X1=-X $(BUILD_INFO_IMPORT_PATH).GitHash=$(GIT_SHA)
ifdef VERSION
BUILD_X2=-X $(BUILD_INFO_IMPORT_PATH).Version=$(VERSION)
endif
BUILD_X3=-X $(BUILD_INFO_IMPORT_PATH).BuildType=$(BUILD_TYPE)
BUILD_INFO=-ldflags "${BUILD_X1} ${BUILD_X2} ${BUILD_X3}"

RUN_CONFIG?=local/config.yaml

CONTRIB_PATH=$(CURDIR)/../opentelemetry-collector-contrib

all-srcs:
	@echo $(ALL_SRC) | tr ' ' '\n' | sort

all-pkgs:
	@echo $(ALL_PKGS) | tr ' ' '\n' | sort
all-env:
	@printenv

.DEFAULT_GOAL := all

.PHONY: all
all: checklicense impi lint misspell test otelcol

.PHONY: betatest

.PHONY: testbed-loadtest
testbed-loadtest: otelcol
	cd ./testbed/tests && ./runtests.sh

.PHONY: testbed-correctness
testbed-correctness: otelcol
	cd ./testbed/correctness && ./runtests.sh

.PHONY: testbed-list-loadtest
testbed-list-loadtest:
	TESTBED_CONFIG=local.yaml $(GOTEST) -v ./testbed/tests --test.list '.*'| grep "^Test"

.PHONY: testbed-list-correctness
testbed-list-correctness:
	TESTBED_CONFIG=inprocess.yaml $(GOTEST) -v ./testbed/correctness --test.list '.*'| grep "^Test"

.PHONY: test
test:
	echo $(ALL_PKGS) | xargs -n 10 $(GOTEST) $(GOTEST_OPT)

.PHONY: benchmark
benchmark:
	$(GOTEST) -bench=. -run=notests $(ALL_PKGS)

.PHONY: test-with-cover
test-with-cover:
	@echo Verifying that all packages have test files to count in coverage
	@internal/buildscripts/check-test-files.sh $(subst github.com/o11y/opentelemetry-collector-o11y/,./,$(ALL_PKGS))
	@echo pre-compiling tests
	@time go test -i $(ALL_PKGS)
	$(GO_ACC) $(ALL_PKGS)
	go tool cover -html=coverage.txt -o coverage.html

.PHONY: addlicense
addlicense:
	$(ADDLICENSE) -y "" -c 'The OpenTelemetry Authors' $(ALL_SRC)

.PHONY: checklicense
checklicense:
	@ADDLICENSEOUT=`$(ADDLICENSE) -check $(ALL_SRC) 2>&1`; \
		if [ "$$ADDLICENSEOUT" ]; then \
			echo "$(ADDLICENSE) FAILED => add License errors:\n"; \
			echo "$$ADDLICENSEOUT\n"; \
			echo "Use 'make addlicense' to fix this."; \
			exit 1; \
		else \
			echo "Check License finished successfully"; \
		fi

.PHONY: misspell
misspell:
	$(MISSPELL) $(ALL_DOC)

.PHONY: misspell-correction
misspell-correction:
	$(MISSPELL_CORRECTION) $(ALL_DOC)

.PHONY: lint-gosec
lint-gosec:
	# TODO: Consider to use gosec from golangci-lint
	$(GOSEC) -quiet -exclude=G104 $(ALL_PKGS)

.PHONY: lint-static-check
lint-static-check:
	@STATIC_CHECK_OUT=`$(STATIC_CHECK) $(ALL_PKGS) 2>&1`; \
		if [ "$$STATIC_CHECK_OUT" ]; then \
			echo "$(STATIC_CHECK) FAILED => static check errors:\n"; \
			echo "$$STATIC_CHECK_OUT\n"; \
			exit 1; \
		else \
			echo "Static check finished successfully"; \
		fi

.PHONY: lint
lint: lint-static-check
	$(LINT) run

.PHONY: impi
impi:
	@$(IMPI) --local github.com/o11y/opentelemetry-collector-o11y --scheme stdThirdPartyLocal --skip internal/data/opentelemetry-proto ./...

.PHONY: fmt
fmt:
	gofmt  -w -s ./
	goimports -w  -local github.com/o11y/opentelemetry-collector-o11y ./

.PHONY: install-tools
install-tools:
	go install github.com/client9/misspell/cmd/misspell
	go install github.com/golangci/golangci-lint/cmd/golangci-lint
	go install github.com/google/addlicense
	go install github.com/jstemmer/go-junit-report
	go install github.com/mjibson/esc
	go install github.com/ory/go-acc
	go install github.com/pavius/impi/cmd/impi
	go install github.com/securego/gosec/cmd/gosec
	go install github.com/tcnksm/ghr
	go install honnef.co/go/tools/cmd/staticcheck

.PHONY: otelcol
otelcol:
	GO111MODULE=on CGO_ENABLED=0 go build -o ./bin/otelcol_$(GOOS)_$(GOARCH)$(EXTENSION) $(BUILD_INFO) ./cmd/otelcol

.PHONY: run
run:
	GO111MODULE=on go run --race ./cmd/otelcol/... --config ${RUN_CONFIG}

.PHONY: docker-component # Not intended to be used directly
docker-component: check-component
	GOOS=linux $(MAKE) $(COMPONENT)
	cp ./bin/$(COMPONENT)_linux_amd64 ./cmd/$(COMPONENT)/$(COMPONENT)
	docker build -t $(COMPONENT) ./cmd/$(COMPONENT)/
	rm ./cmd/$(COMPONENT)/$(COMPONENT)

.PHONY: check-component
check-component:
ifndef COMPONENT
	$(error COMPONENT variable was not defined)
endif

.PHONY: add-tag
add-tag:
	@[ "${TAG}" ] || ( echo ">> env var TAG is not set"; exit 1 )
	@echo "Adding tag ${TAG}"
	@git tag -a ${TAG} -s -m "Version ${TAG}"

.PHONY: delete-tag
delete-tag:
	@[ "${TAG}" ] || ( echo ">> env var TAG is not set"; exit 1 )
	@echo "Deleting tag ${TAG}"
	@git tag -d ${TAG}

.PHONY: docker-otelcol
docker-otelcol:
	COMPONENT=otelcol $(MAKE) docker-component

.PHONY: binaries
binaries: otelcol

.PHONY: binaries-all-sys
binaries-all-sys: binaries-darwin_amd64 binaries-linux_amd64 binaries-linux_arm64 binaries-windows_amd64

.PHONY: binaries-darwin_amd64
binaries-darwin_amd64:
	GOOS=darwin  GOARCH=amd64 $(MAKE) binaries

.PHONY: binaries-linux_amd64
binaries-linux_amd64:
	GOOS=linux   GOARCH=amd64 $(MAKE) binaries

.PHONY: binaries-linux_arm64
binaries-linux_arm64:
	GOOS=linux   GOARCH=arm64 $(MAKE) binaries

.PHONY: binaries-windows_amd64
binaries-windows_amd64:
	GOOS=windows GOARCH=amd64 EXTENSION=.exe $(MAKE) binaries

.PHONY: deb-rpm-package
%-package: ARCH ?= amd64
%-package:
	$(MAKE) binaries-linux_$(ARCH)
	docker build -t otelcol-fpm internal/buildscripts/packaging/fpm
	docker run --rm -v $(CURDIR):/repo -e PACKAGE=$* -e VERSION=$(VERSION) -e ARCH=$(ARCH) otelcol-fpm

# Definitions for ProtoBuf generation.

# The source directory for OTLP ProtoBufs.
OPENTELEMETRY_PROTO_SRC_DIR=internal/data/opentelemetry-proto

# Find all .proto files.
OPENTELEMETRY_PROTO_FILES := $(subst $(OPENTELEMETRY_PROTO_SRC_DIR)/,,$(wildcard $(OPENTELEMETRY_PROTO_SRC_DIR)/opentelemetry/proto/*/v1/*.proto $(OPENTELEMETRY_PROTO_SRC_DIR)/opentelemetry/proto/collector/*/v1/*.proto))

# Target directory to write generated files to.
PROTO_TARGET_GEN_DIR=internal/data/opentelemetry-proto-gen

# Go package name to use for generated files.
PROTO_PACKAGE=github.com/o11y/opentelemetry-collector-o11y/$(PROTO_TARGET_GEN_DIR)

# Intermediate directory used during generation.
PROTO_INTERMEDIATE_DIR=internal/data/tempprotodir

# Function to execute a command. Note the empty line before endef to make sure each command
# gets executed separately instead of concatenated with previous one.
# Accepts command to execute as first parameter.
define exec-command
$(1)

endef

# Generate OTLP Protobuf Go files. This will place generated files in PROTO_TARGET_GEN_DIR.
genproto:
	git submodule update --init
	# Call a sub-make to ensure OPENTELEMETRY_PROTO_FILES is populated after the submodule
	# files are present.
	$(MAKE) genproto_sub
	$(MAKE) fmt

genproto_sub:
	@echo Generating code for the following files:
	@$(foreach file,$(OPENTELEMETRY_PROTO_FILES),$(call exec-command,echo $(file)))

	@echo Copy .proto file to intermediate directory.
	mkdir -p $(PROTO_INTERMEDIATE_DIR)
	cp -R $(OPENTELEMETRY_PROTO_SRC_DIR)/* $(PROTO_INTERMEDIATE_DIR)

	@echo Modify them in the intermediate directory.
	$(foreach file,$(OPENTELEMETRY_PROTO_FILES),$(call exec-command,sed 's+github.com/open-telemetry/opentelemetry-proto/gen/go/+github.com/o11y/opentelemetry-collector-o11y/internal/data/opentelemetry-proto-gen/+g' $(OPENTELEMETRY_PROTO_SRC_DIR)/$(file) > $(PROTO_INTERMEDIATE_DIR)/$(file)))

	@echo Generate Go code from .proto files in intermediate directory.
	$(foreach file,$(OPENTELEMETRY_PROTO_FILES),$(call exec-command,cd $(PROTO_INTERMEDIATE_DIR) && protoc --gogofaster_out=plugins=grpc:./ -I./ $(file)))

	@echo Generate gRPC gateway code.
	cd $(PROTO_INTERMEDIATE_DIR) && protoc --grpc-gateway_out=logtostderr=true,grpc_api_configuration=opentelemetry/proto/collector/trace/v1/trace_service_http.yaml:./ opentelemetry/proto/collector/trace/v1/trace_service.proto
	cd $(PROTO_INTERMEDIATE_DIR) && protoc --grpc-gateway_out=logtostderr=true,grpc_api_configuration=opentelemetry/proto/collector/metrics/v1/metrics_service_http.yaml:./ opentelemetry/proto/collector/metrics/v1/metrics_service.proto
	cd $(PROTO_INTERMEDIATE_DIR) && protoc --grpc-gateway_out=logtostderr=true,grpc_api_configuration=opentelemetry/proto/collector/logs/v1/logs_service_http.yaml:./ opentelemetry/proto/collector/logs/v1/logs_service.proto

	@echo Move generated code to target directory.
	mkdir -p $(PROTO_TARGET_GEN_DIR)
	cp -R $(PROTO_INTERMEDIATE_DIR)/$(PROTO_PACKAGE)/* $(PROTO_TARGET_GEN_DIR)/

	@echo Delete intermediate directory.
	@rm -rf $(PROTO_INTERMEDIATE_DIR)

	@rm -rf $(OPENTELEMETRY_PROTO_SRC_DIR)/*
	@rm -rf $(OPENTELEMETRY_PROTO_SRC_DIR)/.* > /dev/null 2>&1 || true

# Generate structs, functions and tests for pdata package. Must be used after any changes
# to proto and after running `make genproto`
genpdata:
	go run cmd/pdatagen/main.go

# Checks that the HEAD of the contrib repo checked out in CONTRIB_PATH compiles
# against the current version of this repo.
.PHONY: check-contrib
check-contrib:
	@echo Setting contrib at $(CONTRIB_PATH) to use this core checkout
	make -C $(CONTRIB_PATH) for-all CMD="go mod edit -replace github.com/o11y/opentelemetry-collector-o11y=$(CURDIR)"
	make -C $(CONTRIB_PATH) test
	@echo Restoring contrib to no longer use this core checkout
	make -C $(CONTRIB_PATH) for-all CMD="go mod edit -dropreplace github.com/o11y/opentelemetry-collector-o11y"
