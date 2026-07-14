# Codex Context Meter Lite

[English](README.md) | [简体中文](README.zh-CN.md)

一个轻量、只读、无需修改 Codex 的 Windows 上下文用量悬浮条。

Codex Context Meter Lite 直接读取 Codex 已生成的本地 session JSONL，在 Codex 窗口边缘显示当前上下文压力和 token 明细。它是单文件 portable 应用，不需要 Codex++、插件、Node.js、Python 或后台服务。

## 界面预览

| 折叠状态 | 展开状态 |
| --- | --- |
| ![Codex Context Meter Lite 折叠状态](docs/images/overlay-collapsed.png) | ![Codex Context Meter Lite 展开状态](docs/images/overlay-expanded.png) |

## 主要功能

- **实时上下文占用：** 每 500 ms 检查一次当前 session，使用 `last_token_usage.total_tokens / model_context_window` 计算 Used/Left 百分比。
- **连续压力进度条：** 已用上下文不超过 75% 时为绿色，超过 75% 至 85% 为橙黄色，超过 85% 为红色。
- **紧凑与详情视图：** 默认显示 28 px 紧凑条；鼠标悬停时展开，分别显示本轮和整个 session 的 Total、Input、Cache、Output 与 Reasoning token。
- **跟随 Codex 窗口：** 浮窗不抢焦点，使用应用包身份和受控路径规则识别 Codex，支持多窗口前台切换，并跟随窗口移动、缩放、最小化、DPI 和显示器切换。
- **四角锚定：** 可拖动到 Codex 窗口任意角落。顶部锚点向下展开，底部锚点向上展开。
- **自动或固定任务：** 默认选择最新包含有效 `token_count` 的 session，也可从托盘菜单固定到指定任务。任务名称来自 `session_index.jsonl`。
- **容错增量读取：** 支持 JSONL 半行写入、损坏行、文件截断、轮转和 context compaction，不读取对话正文。
- **托盘控制：** 支持显示/隐藏、Used/Left 切换、深色/浅色主题、80%/100%/120% 缩放、session 选择和可选的用户级开机启动。
- **单实例与便携运行：** 同一用户会话只运行一个正式实例，无安装器、服务或计划任务。

## 系统要求

- Windows 10 或 Windows 11，64 位。
- 已安装并至少运行过一次 Windows 版 Codex Desktop。程序优先使用 `OpenAI.Codex` 包身份识别官方 MSIX，也支持受控的企业重打包路径和显式 EXE 覆盖。
- Codex session 数据默认位于 `%USERPROFILE%\.codex`；如设置了 `CODEX_HOME`，程序会优先使用该目录，命令行参数的优先级最高。

## 使用方法

1. 从发布 ZIP 解压 `codex-context-meter-lite.exe` 到任意可写目录。无需安装。
2. 双击 EXE 启动。程序会驻留在 Windows 通知区域，并自动读取最新的有效 Codex session。
3. 切换到 Codex 主窗口。紧凑悬浮条会出现在窗口边缘；Codex 最小化或失去前台焦点时，悬浮条会自动隐藏。
4. 将鼠标移到悬浮条上查看详情；按住左键拖动可改变锚点和偏移。
5. 右键单击悬浮条或通知区域图标打开设置菜单。双击通知区域图标可快速显示或隐藏悬浮条。
6. 退出时选择托盘菜单中的 `Quit`。直接关闭 Codex 不会退出本程序。

## 托盘菜单

| 菜单项 | 说明 |
| --- | --- |
| `Codex window: detected / not detected` | 只读显示 Codex 窗口是否识别成功 |
| `Session data: detected / not detected` | 只读显示是否已读取有效 token session |
| `Show overlay` | 显示或隐藏悬浮条 |
| `Show context used` | 勾选时显示 Used；取消时显示 Left |
| `Light theme` | 在浅色与深色主题之间切换 |
| `Scale 80% / 100% / 120%` | 调整悬浮条尺寸 |
| `Session: Auto (latest active)` | 自动跟随最新有效 session |
| Session list | 固定到所选任务，再次选择 Auto 可解除固定 |
| `Start with Windows` | 写入或移除当前用户的开机启动项 |
| `Quit` | 退出程序并移除托盘图标 |

## 数据与隐私边界

程序默认只读取以下 Codex 文件：

```text
%USERPROFILE%\.codex\sessions\**\*.jsonl
%USERPROFILE%\.codex\session_index.jsonl
```

设置 `CODEX_HOME` 或命令行路径覆盖时，程序会读取所选目录下对应的 `sessions` 与 `session_index.jsonl`，访问范围不变。

程序不会修改 `.codex` 目录、Codex 安装包或 `app.asar`，也不会使用 CDP、DOM 注入、React Fiber、网络 API、认证文件、密钥或 provider token。它只解析结构化的 `session_meta` 和 `token_count` 记录，不解析消息正文。

只有在用户更改设置后，程序才会创建自身配置：

```text
%LOCALAPPDATA%\CodexContextMeterLite\config.json
```

启用 `Start with Windows` 时，程序仅写入当前用户的注册表项 `HKCU\Software\Microsoft\Windows\CurrentVersion\Run\CodexContextMeterLite`；关闭该选项会移除此项。

## 指标说明

| 指标 | 说明 |
| --- | --- |
| `Used` | 当前上下文已用比例与 token 数 |
| `Left` | 当前上下文剩余比例与 token 数 |
| `Turn` | 最新一次 `last_token_usage` 记录 |
| `Session` | 当前 session 累计 `total_token_usage` |
| `Input` | 输入 token |
| `Cache` | 命中的缓存输入 token |
| `Output` | 输出 token |
| `Reasoning` | 推理输出 token |

`Session` 是累计消耗，不等于模型上下文窗口占用；上下文压力始终以最新 `last_token_usage.total_tokens` 为准。

## 故障排查

如果托盘图标存在但 Codex 窗口没有悬浮条，先打开托盘菜单查看 `Codex window` 与 `Session data` 状态。

```powershell
# 在不依赖 Codex 窗口识别的情况下验证独立 UI
.\codex-context-meter-lite.exe --demo

# 检查 Codex 进程和包身份，不读取对话数据
Get-Process -Name ChatGPT,Codex -ErrorAction SilentlyContinue |
  Select-Object Id, ProcessName, Path, MainWindowTitle, MainWindowHandle

Get-AppxPackage |
  Where-Object Name -Match '^OpenAI\.(Codex|ChatGPT)$' |
  Select-Object Name, PackageFamilyName, Version, Architecture, InstallLocation
```

`--demo` 能显示但 `Codex window: not detected` 通常表示安装渠道或进程识别不匹配；`Codex window: detected` 但 `Session data: not detected` 时，应检查 `CODEX_HOME` 或 session 路径。上述命令不读取 JSONL 内容、对话正文、token 或认证信息。

## 命令行与开发

```powershell
# 显示版本
.\codex-context-meter-lite.exe --version

# 使用自定义 Codex 数据路径
.\codex-context-meter-lite.exe --sessions "D:\path\to\sessions" `
  --session-index "D:\path\to\session_index.jsonl"

# 显式绑定企业重打包的 Codex 可执行文件
.\codex-context-meter-lite.exe --codex-exe "D:\Apps\Codex\ChatGPT.exe"

# 用合成数据打开独立 UI
.\codex-context-meter-lite.exe --demo
```

构建需要 Go 1.26 或更高版本。

```powershell
go test ./...
go test -race ./...
go vet ./...
.\scripts\build.ps1
```

发布脚本生成单 EXE portable ZIP，不会安装任何依赖。当前构建目标为 `dist\codex-context-meter-lite-windows-amd64-v0.1.0.zip`。

## 卸载

1. 从托盘菜单选择 `Quit`。
2. 如果启用了 `Start with Windows`，建议先在托盘菜单中取消勾选。
3. 删除解压出的程序目录。若需同时清除偏好设置，可手动删除 `%LOCALAPPDATA%\CodexContextMeterLite`。

## 引用

本项目在设计与实现评估阶段参考了以下开源项目：

- [Codex Context Used Meter](https://github.com/Minghou-Lei/codex-context-used-meter)
- [Codex Monitor](https://github.com/KevinKE93/Codex-Monitor)

## 许可证

本项目使用 MIT License。第三方项目与 Go 依赖声明见 [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)。
