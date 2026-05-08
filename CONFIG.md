# Gline 配置指南

## 配置方式

Gline 支持三种配置方式（优先级从高到低）：

1. **工作区配置** (`.gline/config.yaml`) - 当前目录
2. **全局配置** (`~/.gline/config.yaml`) - 用户主目录
3. **环境变量** (`GLINE_*`) - 最低优先级

## 配置 Provider

### 方式 1：使用命令行

```bash
# 设置默认 Provider (anthropic 或 openai)
gline config set provider.default openai

# 设置 OpenAI API Key
gline config set provider.openai.api_key sk-xxxxxxxxxx

# 设置 OpenAI 模型
gline config set provider.openai.model gpt-4

# 设置自定义 OpenAI 端点 (如 OpenRouter、DashScope、Ollama)
gline config set provider.openai.base_url https://openrouter.ai/api/v1

# 设置 Anthropic API Key
gline config set provider.anthropic.api_key sk-ant-xxxxxxxxxx

# 设置 Anthropic 模型
gline config set provider.anthropic.model claude-3-5-sonnet-20241022
```

### 方式 2：使用环境变量

```bash
# Windows (CMD)
set GLINE_PROVIDER=openai
set GLINE_OPENAI_API_KEY=sk-xxxxxxxxxx
set GLINE_OPENAI_MODEL=gpt-4

# Windows (PowerShell)
$env:GLINE_PROVIDER="openai"
$env:GLINE_OPENAI_API_KEY="sk-xxxxxxxxxx"
$env:GLINE_OPENAI_MODEL="gpt-4"

# Linux/Mac
export GLINE_PROVIDER=openai
export GLINE_OPENAI_API_KEY=sk-xxxxxxxxxx
export GLINE_OPENAI_MODEL=gpt-4
export GLINE_ANTHROPIC_API_KEY=sk-ant-xxxxxxxxxx
```

### 方式 3：直接编辑配置文件

编辑 `~/.gline/config.yaml` (Windows: `%USERPROFILE%\.gline\config.yaml`):

```yaml
# LLM Provider Settings
provider:
  # 默认 Provider: anthropic 或 openai
  default: openai
  
  # Anthropic (Claude) 设置
  anthropic:
    api_key: "sk-ant-xxxxxxxxxx"
    model: claude-3-5-sonnet-20241022
  
  # OpenAI 设置
  # 支持 OpenAI 官方 API、OpenRouter、DashScope、Ollama 等
  openai:
    api_key: "sk-xxxxxxxxxx"
    model: gpt-4
    # 自定义端点 (可选)
    base_url: ""

# UI 设置
ui:
  theme: default
  animations: true

# 日志设置
log:
  level: info
  file: ""
```

## 支持的 Provider

### OpenAI

```bash
# 官方 OpenAI
gline config set provider.default openai
gline config set provider.openai.api_key sk-xxxxxxxxxx
gline config set provider.openai.model gpt-4

# OpenRouter (支持多种模型)
gline config set provider.openai.base_url https://openrouter.ai/api/v1
gline config set provider.openai.api_key sk-or-v1-xxxxxxxxxx
gline config set provider.openai.model anthropic/claude-3.5-sonnet

# DashScope (阿里云)
gline config set provider.openai.base_url https://dashscope.aliyuncs.com/compatible-mode/v1
gline config set provider.openai.api_key sk-xxxxxxxxxx
gline config set provider.openai.model qwen-max

# Ollama (本地模型)
gline config set provider.openai.base_url http://localhost:11434/v1
gline config set provider.openai.api_key ollama
gline config set provider.openai.model llama2
```

### Anthropic

```bash
# Claude 官方 API
gline config set provider.default anthropic
gline config set provider.anthropic.api_key sk-ant-xxxxxxxxxx
gline config set provider.anthropic.model claude-3-5-sonnet-20241022
```

可用模型：
- `claude-3-opus-20240229`
- `claude-3-5-sonnet-20241022`
- `claude-3-sonnet-20240229`
- `claude-3-haiku-20240307`

## 查看配置

```bash
# 查看所有配置
gline config list

# 查看特定配置项
gline config get provider.default
gline config get provider.openai.model

# 查看配置文件路径
gline config path
```

## 快速开始

### 使用 OpenAI

```bash
# 设置 API Key
gline config set provider.openai.api_key sk-xxxxxxxxxx

# 开始使用
gline chat
```

### 使用 Anthropic

```bash
# 设置默认 Provider 和 API Key
gline config set provider.default anthropic
gline config set provider.anthropic.api_key sk-ant-xxxxxxxxxx

# 开始使用
gline chat
```

### 使用 OpenRouter (推荐，支持多种模型)

```bash
# 设置 OpenRouter
gline config set provider.default openai
gline config set provider.openai.base_url https://openrouter.ai/api/v1
gline config set provider.openai.api_key sk-or-v1-xxxxxxxxxx
gline config set provider.openai.model anthropic/claude-3.5-sonnet

# 开始使用
gline chat
```

## 故障排除

### API Key 未配置

如果看到错误：`API key not configured`，请检查：

1. 是否设置了正确的 Provider
2. API Key 是否正确
3. 环境变量或配置文件是否生效

### 检查当前配置

```bash
gline config list
```

### 重置配置

删除配置文件后重新运行：

```bash
# Windows
rmdir /s /q %USERPROFILE%\.gline

# Linux/Mac
rm -rf ~/.gline
```

然后运行任意 `gline` 命令会重新生成默认配置。
