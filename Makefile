VERSION := 0.1.0

BIN := $(CURDIR)/bin
NODE_BIN = $(CURDIR)/node_modules/.bin

# 依赖版本
# https://github.com/protocolbuffers/protobuf
PROTOC_VERSION := 35.1
# https://github.com/protocolbuffers/protobuf-go
PROTOC_GEN_GO_VERSION := 1.36.12

# 获取不同环境的配置
ifeq ($(OS), Windows_NT)
	PROTOC_ZIP := protoc-$(PROTOC_VERSION)-win64.zip
	PROTOC_GEN_GO_ZIP := protoc-gen-go.v$(PROTOC_GEN_GO_VERSION).windows.amd64.zip
else
	OS := $(shell uname)
	ifeq ($(OS), Linux)
		PROTOC_ZIP := protoc-$(PROTOC_VERSION)-linux-x86_64.zip
		PROTOC_GEN_GO_ZIP := protoc-gen-go.v$(PROTOC_GEN_GO_VERSION).linux.amd64.tar.gz
	else ifeq ($(OS), Darwin)
		PROTOC_ZIP := protoc-$(PROTOC_VERSION)-osx-aarch_64.zip
		PROTOC_GEN_GO_ZIP := protoc-gen-go.v$(PROTOC_GEN_GO_VERSION).darwin.arm64.tar.gz
	endif
endif


# 构建脚手架生成器二进制
build-dev:
	@echo "Building protoc-gen-scaffold (version: dev)"
	@go build -ldflags "-X go.icytown.com/scaffold/internal/generator.injectedVersion=dev" \
		-o $(BIN)/protoc-gen-scaffold ./cmd/protoc-gen-scaffold
	@echo "Protoc-gen-scaffold build success"


# 生成脚手架的 API IDL
gen-api: $(BIN)/protoc $(BIN)/protoc-gen-go
	@echo "Generate scaffold api source code"
	@$(BIN)/protoc \
		--go_out=. \
		--go_opt=module=go.icytown.com/scaffold \
		--plugin=protoc-gen-go=$(BIN)/protoc-gen-go \
		-I. api/api.proto
	@echo "Generate scaffold api source code success"


# 生成 example 演示代码
gen-example: build-dev $(BIN)/protoc $(NODE_BIN)/prettier
	@echo "Generate example source code"
	@mkdir -p ./example/dev/gen
	@mkdir -p ./example/test/gen
	@mkdir -p ./example/go-server/gen
	@mkdir -p ./example/ts-client/gen
	@cd ./example/dev && $(BIN)/protoc-gen-scaffold -protoc=$(BIN)/protoc -out=./gen -lang=go -kind=server \
		test.proto shared.proto other/other.proto
	@cd ./example/dev && $(BIN)/protoc-gen-scaffold -protoc=$(BIN)/protoc -out=./gen -lang=typescript -kind=client \
		-format=prettier -prettier=$(NODE_BIN)/prettier test.proto shared.proto other/other.proto
	@cd ./example/test && $(BIN)/protoc-gen-scaffold -protoc=$(BIN)/protoc -out=./gen -lang=go -kind=server user.proto
	@cd ./example && $(BIN)/protoc-gen-scaffold -protoc=$(BIN)/protoc -out=./go-server/gen -lang=go -kind=server user.proto
	@cd ./example && $(BIN)/protoc-gen-scaffold -protoc=$(BIN)/protoc -out=./ts-client/gen -lang=typescript -kind=client \
		 -format=prettier -prettier=$(NODE_BIN)/prettier user.proto
	@echo "Generate example source code success"


# 检查 assert 目录一致性并 lint 主模块、mux 模块和 example 模块
lint:
	@echo "Checking assert consistency"
	@diff -ru internal/assert go/mux/internal/assert
	@echo "Linting scaffold"
	@golangci-lint run ./...
	@echo "Linting mux"
	@cd go/mux && golangci-lint run ./...
	@echo "Linting example"
	@cd example && golangci-lint run ./...
	@echo "Lint success"


# 测试主模块、mux 模块和 example 模块
test:
	@echo "Testing scaffold"
	@go test -race ./...
	@echo "Testing mux"
	@cd go/mux && go test -race ./...
	@echo "Testing example"
	@cd example && go test -race ./...
	@echo "Test success"


# 安装 protoc
$(BIN)/protoc: Makefile
	@if [ "$(PROTOC_ZIP)" = "" ]; then \
		echo "Error(protoc): unsupported operating system $(OS)"; \
		exit 1; \
	fi; \
	echo "Checking for installed protoc version"; \
	if [ -x "$(BIN)/protoc" ]; then \
		INSTALLED_PROTOC_VERSION=$$($(BIN)/protoc --version | grep -o "[0-9]*\.[0-9]*"); \
		if [ "$(PROTOC_VERSION)" = "$$INSTALLED_PROTOC_VERSION" ]; then \
			echo "Protoc $(PROTOC_VERSION) already installed"; \
			exit 0; \
		fi; \
	fi; \
	echo "Installing protoc $(PROTOC_VERSION) for $(OS)"; \
	mkdir -p $(@D); \
	curl -fL -o $(BIN)/$(PROTOC_ZIP) https://github.com/protocolbuffers/protobuf/releases/download/v$(PROTOC_VERSION)/$(PROTOC_ZIP); \
	if [ ! -f "$(BIN)/$(PROTOC_ZIP)" ]; then \
		echo "Download protoc $(PROTOC_VERSION) fail"; \
		exit 1; \
	fi; \
	echo "Unzip protoc $(PROTOC_VERSION)"; \
	unzip -oj $(BIN)/$(PROTOC_ZIP) -d $(BIN) "bin/protoc*"; \
	unzip -o $(BIN)/$(PROTOC_ZIP) -d $(BIN) "include/**/*"; \
	echo "Protoc $(PROTOC_VERSION) install success"


# 安装 protoc-gen-go
$(BIN)/protoc-gen-go: Makefile
	@if [ "$(PROTOC_GEN_GO_ZIP)" = "" ]; then \
		echo "Error(protoc-gen-go): unsupported operating system $(OS)"; \
		exit 1; \
	fi; \
	echo "Checking for installed protoc-gen-go version"; \
	if [ -x "$(BIN)/protoc-gen-go" ]; then \
		INSTALLED_PROTOC_GEN_GO_VERSION=$$($(BIN)/protoc-gen-go --version | grep -o "[0-9]*\.[0-9]*\.[0-9]*"); \
		if [ "$(PROTOC_GEN_GO_VERSION)" = "$$INSTALLED_PROTOC_GEN_GO_VERSION" ]; then \
			echo "Protoc-gen-go $(PROTOC_GEN_GO_VERSION) already installed"; \
			exit 0; \
		fi; \
	fi; \
	echo "Installing protoc-gen-go $(PROTOC_VERSION) for $(OS)"; \
	mkdir -p $(@D); \
	curl -fL -o $(BIN)/$(PROTOC_GEN_GO_ZIP) https://github.com/protocolbuffers/protobuf-go/releases/download/v$(PROTOC_GEN_GO_VERSION)/$(PROTOC_GEN_GO_ZIP); \
	if [ ! -f "$(BIN)/$(PROTOC_GEN_GO_ZIP)" ]; then \
		echo "Download protoc-gen-go $(PROTOC_GEN_GO_VERSION) fail"; \
		exit 1; \
	fi; \
	echo "Unzip protoc-gen-go $(PROTOC_GEN_GO_VERSION)"; \
	if echo "$(PROTOC_GEN_GO_ZIP)" | grep -q "\.tar\.gz$$"; then \
		tar -xzf $(BIN)/$(PROTOC_GEN_GO_ZIP) -C $(BIN); \
	else \
		unzip -o $(BIN)/$(PROTOC_GEN_GO_ZIP) -d $(BIN) "protoc-gen-go*"; \
	fi; \
	if [ -f "$(BIN)/protoc-gen-go.exe" ]; then \
		mv $(BIN)/protoc-gen-go.exe $(BIN)/protoc-gen-go; \
	fi; \
	echo "Protoc-gen-go $(PROTOC_GEN_GO_VERSION) install success"


# 安装 TypeScript 代码格式化依赖
$(NODE_BIN)/prettier: package.json pnpm-lock.yaml
	@echo "Installing Node.js dependencies"
	@pnpm install --frozen-lockfile
	@echo "Node.js dependencies install success"
