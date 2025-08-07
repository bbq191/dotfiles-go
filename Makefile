#!/usr/bin/make -f

# 变量定义
BINARY_NAME=dotfiles
BUILD_DIR=bin
MAIN_PATH=cmd/dotfiles/main.go

# 默认目标
.PHONY: build
build:
	@echo "🔨 编译项目..."
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "✅ 编译完成: $(BUILD_DIR)/$(BINARY_NAME)"

# 优化编译
.PHONY: build-release
build-release:
	@echo "🚀 优化编译..."
	go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "✅ 优化编译完成"

# 清理
.PHONY: clean
clean:
	@echo "🧹 清理文件..."
	rm -f $(BUILD_DIR)/$(BINARY_NAME)
	@echo "✅ 清理完成"

# 测试
.PHONY: test
test: build
	@echo "🧪 测试程序..."
	./$(BUILD_DIR)/$(BINARY_NAME) --version

# 安装依赖
.PHONY: deps
deps:
	@echo "📦 安装依赖..."
	go mod tidy
	@echo "✅ 依赖安装完成"

# 帮助信息
.PHONY: help
help:
	@echo "可用命令:"
	@echo "  make build        - 编译项目"
	@echo "  make build-release - 优化编译"
	@echo "  make test         - 编译并测试"
	@echo "  make clean        - 清理编译文件"
	@echo "  make deps         - 安装依赖"