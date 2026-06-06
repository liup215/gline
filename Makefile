# gline Makefile
# 支持本地构建和跨平台编译

# 变量定义
BINARY_NAME := gline
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S' 2>/dev/null || echo "unknown")

# 构建参数
LDFLAGS_BASE := -X github.com/liup215/gline/internal/version.Version=$(VERSION) \
	-X github.com/liup215/gline/internal/version.Commit=$(COMMIT) \
	-X github.com/liup215/gline/internal/version.BuildTime=$(BUILD_TIME) \
	-s -w

ifeq ($(shell go env GOOS),windows)
LDFLAGS := -ldflags "$(LDFLAGS_BASE) -H=windowsgui"
else
LDFLAGS := -ldflags "$(LDFLAGS_BASE)"
endif

# 目标平台
PLATFORMS := darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64

# 默认目标
.DEFAULT_GOAL := build

# 本地构建
.PHONY: build
build:
	@echo "Building $(BINARY_NAME) $(VERSION)..."
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/gline
	@echo "Build complete: bin/$(BINARY_NAME)"

# 运行测试
.PHONY: test
test:
	@echo "Running tests..."
	go test -v ./...

# 清理构建产物
.PHONY: clean
clean:
	@echo "Cleaning..."
	rm -rf bin/ dist/
	@echo "Clean complete"

# 安装到系统
.PHONY: install
install:
	@echo "Installing $(BINARY_NAME)..."
	go install $(LDFLAGS) ./cmd/gline
	@echo "Install complete"

# 交叉编译所有平台
.PHONY: build-all
build-all: clean
	@echo "Building for all platforms..."
	@mkdir -p bin
	@for platform in $(PLATFORMS); do \
		GOOS=$$(echo $$platform | cut -d/ -f1); \
		GOARCH=$$(echo $$platform | cut -d/ -f2); \
		OUTPUT=$(BINARY_NAME)-$$GOOS-$$GOARCH; \
		if [ "$$GOOS" = "windows" ]; then OUTPUT=$$OUTPUT.exe; fi; \
		echo "  Building $$OUTPUT..."; \
		GOOS=$$GOOS GOARCH=$$GOARCH go build $(LDFLAGS) -o bin/$$OUTPUT ./cmd/gline; \
	done
	@echo "All builds complete"

# 打包发布产物
.PHONY: package
package: build-all
	@echo "Packaging..."
	@mkdir -p dist
	@for platform in $(PLATFORMS); do \
		GOOS=$$(echo $$platform | cut -d/ -f1); \
		GOARCH=$$(echo $$platform | cut -d/ -f2); \
		BINARY=$(BINARY_NAME)-$$GOOS-$$GOARCH; \
		if [ "$$GOOS" = "windows" ]; then \
			(cd bin && zip ../dist/$$BINARY.zip $$BINARY.exe); \
		else \
			(cd bin && tar -czf ../dist/$$BINARY.tar.gz $$BINARY); \
		fi; \
	done
	@echo "Packaging complete"

# 生成校验和
.PHONY: checksum
checksum: package
	@echo "Generating checksums..."
	@cd dist && sha256sum * > checksums.txt
	@echo "Checksums generated"

# 开发模式运行
.PHONY: run
run:
	go run ./cmd/gline

# 格式化代码
.PHONY: fmt
fmt:
	go fmt ./...

# 运行 linter
.PHONY: lint
lint:
	golangci-lint run ./...

# 下载依赖
.PHONY: deps
deps:
	go mod download
	go mod tidy

# 显示版本信息
.PHONY: version
version:
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT)"
	@echo "Build Time: $(BUILD_TIME)"

# 帮助信息
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build      - Build for current platform"
	@echo "  build-all  - Cross-compile for all platforms"
	@echo "  package    - Package all binaries"
	@echo "  checksum   - Generate SHA256 checksums"
	@echo "  test       - Run tests"
	@echo "  clean      - Clean build artifacts"
	@echo "  install    - Install to GOPATH/bin"
	@echo "  run        - Run in development mode"
	@echo "  fmt        - Format Go code"
	@echo "  lint       - Run linter"
	@echo "  deps       - Download and tidy dependencies"
	@echo "  version    - Show version info"
	@echo "  help       - Show this help"
