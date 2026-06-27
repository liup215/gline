# 版本检测功能开发文档

## 功能概述

实现软件自动检测新版本的功能，当有新版本发布时提示用户下载更新。

## 实现内容

### 1. 后端模块 (internal/version/)

#### 1.1 types.go - 类型定义
- `GitHubRelease` - GitHub API 发布响应结构
- `Asset` - 可下载资源
- `UpdateCheckResult` - 更新检查结果
- `CheckerConfig` - 检查器配置
- `CachedResult` - 缓存结果
- `VersionInfo` - 语义化版本信息
- `UpdateInfo` - UI 格式化的更新信息
- `CheckerState` - 检查器状态

#### 1.2 checker.go - 核心检查器
- `Checker` 结构体 - 版本检查器
- `CheckNow()` - 同步检查
- `CheckAsync()` - 异步检查
- `ParseVersion()` - 语义化版本解析
- `GreaterThan()` - 版本比较
- `findDownloadURL()` - 平台特定下载链接
- 缓存管理 (load/save)

#### 1.3 service.go - 服务层
- `Service` 结构体 - 版本检查服务
- `CheckForUpdates()` - 检查更新
- `CheckForUpdatesAsync()` - 异步检查
- 与配置系统集成

### 2. 配置集成 (internal/config/)

#### 2.1 UpdateConfig 结构
```go
type UpdateConfig struct {
    Enabled           bool   // 启用自动检查
    CheckInterval     string // 检查间隔 (如 "24h")
    IncludePrerelease bool   // 包含预发布版本
    LastChecked       string // 上次检查时间
}
```

#### 2.2 默认配置
```yaml
update:
  enabled: true
  check_interval: "24h"
  include_prerelease: false
  last_checked: ""
```

### 3. 前端组件

#### 3.1 useVersionCheck.ts - Hook
- 检查更新逻辑
- 自动检查（启动后5秒延迟）
- 定期检查和缓存
- 忽略状态管理

#### 3.2 UpdateNotification.tsx - 通知组件
- 顶部固定通知栏
- 版本信息展示
- 下载和忽略按钮
- 主题适配

#### 3.3 UpdatesTab.tsx - 设置标签页
- 当前/最新版本显示
- 手动检查按钮
- 下载更新按钮
- 发布说明展示

### 4. CI/CD 集成

#### 4.1 GitHub Actions 修改
- 在构建时注入版本号
- 使用 tag 名称作为版本号
- 支持 dev 版本（非 tag 构建）

```yaml
# 版本提取逻辑
if [[ "${{ github.ref }}" == refs/tags/* ]]; then
  version="${{ github.ref_name }}"
else
  version=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
fi
```

#### 4.2 构建脚本
- `build-all.ps1` 和 `build-all.sh` 已支持版本注入
- 使用 `-ldflags -X` 注入版本信息

## 工作流程

```
应用启动
    ↓
加载配置 (UpdateConfig)
    ↓
创建 VersionService
    ↓
加载缓存 (update_cache.json)
    ↓
检查是否应该检查更新
    ↓
是 → 异步调用 GitHub API
    ↓
有新版本? → 显示通知
    ↓
用户点击下载 → 打开浏览器
```

## 缓存机制

- 缓存文件: `~/.gline/update_cache.json`
- 默认检查间隔: 24小时
- 避免频繁 API 调用（GitHub API 限制）

## 语义化版本比较

支持标准语义化版本:
- `v1.0.0` > `v0.9.0`
- `v1.2.0` > `v1.1.9`
- `v1.0.0` > `v1.0.0-beta` (稳定版 > 预发布版)

## 平台检测

自动检测平台下载对应版本:
- Windows (amd64) → `gline-windows-amd64.exe`
- macOS (arm64) → `gline-darwin-arm64`
- Linux (amd64) → `gline-linux-amd64`

## 待完成

- [ ] 在 App.tsx 中集成版本检查
- [ ] 在 SettingsPanel.tsx 中添加 UpdatesTab
- [ ] 测试版本检测流程
- [ ] 添加更多单元测试

## 文件列表

### 新增文件
- `internal/version/checker.go`
- `internal/version/types.go`
- `internal/version/service.go`
- `frontend/src/components/UpdateNotification.tsx`
- `frontend/src/components/settings/UpdatesTab.tsx`

### 修改文件
- `internal/config/config.go` - 添加 UpdateConfig
- `.github/workflows/build.yml` - 版本注入
- `frontend/src/App.tsx` - 集成版本检查
- `frontend/src/components/SettingsPanel.tsx` - 添加 UpdatesTab

## 技术要点

1. **线程安全**: checker 使用 mutex 保护状态
2. **错误处理**: 网络错误不影响应用启动
3. **可配置**: 用户可禁用自动检查
4. **缓存**: 避免频繁 API 调用
5. **平台适配**: 自动检测下载链接
