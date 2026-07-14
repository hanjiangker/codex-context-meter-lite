# Codex Context Meter Lite

一个轻量、只读、无需修改 Codex 的 Windows 上下文用量悬浮条。<br>
*A lightweight, read-only Windows overlay for Codex context usage, with no Codex modification required.*

Codex Context Meter Lite 直接读取 Codex 已生成的本地 session JSONL，在 Codex 窗口边缘显示当前上下文压力和 token 明细。它是单文件 portable 应用，不需要 Codex++、插件、Node.js、Python 或后台服务。<br>
*Codex Context Meter Lite reads the local session JSONL files already produced by Codex and shows current context pressure and token details at the edge of the Codex window. It is a single-file portable app and requires no Codex++, plugin, Node.js, Python, or background service.*

## 界面预览 / Interface Preview

| 折叠状态 / Collapsed | 展开状态 / Expanded |
| --- | --- |
| ![Codex Context Meter Lite collapsed overlay](docs/images/overlay-collapsed.png) | ![Codex Context Meter Lite expanded overlay](docs/images/overlay-expanded.png) |

## 主要功能 / Key Features

- **实时上下文占用 / Live context usage:** 每 500 ms 检查一次当前 session，使用 `last_token_usage.total_tokens / model_context_window` 计算 Used/Left 百分比。 / Checks the active session every 500 ms and calculates Used/Left as `last_token_usage.total_tokens / model_context_window`.
- **连续压力进度条 / Continuous pressure bar:** 已用上下文不超过 75% 时为绿色，超过 75% 至 85% 为橙黄色，超过 85% 为红色。 / Green through 75% used, amber above 75% through 85%, and red above 85%.
- **紧凑与详情视图 / Compact and detailed views:** 默认显示 28 px 紧凑条；鼠标悬停时展开，分别显示本轮和整个 session 的 Total、Input、Cache、Output 与 Reasoning token。 / Shows a 28 px compact strip by default; hover to expand Turn and Session totals for Total, Input, Cache, Output, and Reasoning tokens.
- **跟随 Codex 窗口 / Codex window tracking:** 浮窗不抢焦点，使用应用包身份和受控路径规则识别 Codex，支持多窗口前台切换，并跟随窗口移动、缩放、最小化、DPI 和显示器切换。 / The non-activating overlay identifies Codex using package identity and controlled path rules, follows the active Codex window when multiple windows are open, and tracks movement, resizing, minimization, DPI, and monitor changes.
- **四角锚定 / Four-corner anchoring:** 可拖动到 Codex 窗口任意角落。顶部锚点向下展开，底部锚点向上展开。 / Drag it to any Codex window corner. Top anchors expand downward and bottom anchors expand upward.
- **自动或固定任务 / Automatic or pinned task:** 默认选择最新包含有效 `token_count` 的 session，也可从托盘菜单固定到指定任务。任务名称来自 `session_index.jsonl`。 / Selects the latest session containing a valid `token_count` event by default, or lets you pin a task from the tray menu. Task names come from `session_index.jsonl`.
- **容错增量读取 / Resilient incremental reading:** 支持 JSONL 半行写入、损坏行、文件截断、轮转和 context compaction，不读取对话正文。 / Handles partial JSONL writes, malformed lines, truncation, rotation, and context compaction without reading conversation text.
- **托盘控制 / Tray controls:** 支持显示/隐藏、Used/Left 切换、深色/浅色主题、80%/100%/120% 缩放、session 选择和可选的用户级开机启动。 / Controls visibility, Used/Left mode, dark/light theme, 80%/100%/120% scale, session selection, and optional per-user startup.
- **单实例与便携运行 / Single-instance portable operation:** 同一用户会话只运行一个正式实例，无安装器、服务或计划任务。 / Runs one production instance per user session with no installer, service, or scheduled task.

## 系统要求 / Requirements

- Windows 10 或 Windows 11，64 位。 / Windows 10 or Windows 11, 64-bit.
- 已安装并至少运行过一次 Windows 版 Codex Desktop。程序优先使用 `OpenAI.Codex` 包身份识别官方 MSIX，也支持受控的企业重打包路径和显式 EXE 覆盖。 / Codex Desktop for Windows must be installed and launched at least once. The app prefers the `OpenAI.Codex` package identity for official MSIX builds and also supports controlled enterprise paths and an explicit EXE override.
- Codex session 数据默认位于 `%USERPROFILE%\.codex`；如设置了 `CODEX_HOME`，程序会优先使用该目录，命令行参数的优先级最高。 / Codex session data defaults to `%USERPROFILE%\.codex`; when `CODEX_HOME` is set, that directory is preferred, while command-line overrides have the highest priority.

## 使用方法 / Usage

1. 从发布 ZIP 解压 `codex-context-meter-lite.exe` 到任意可写目录。无需安装。<br>
   *Extract `codex-context-meter-lite.exe` from the release ZIP into any writable directory. No installation is required.*
2. 双击 EXE 启动。程序会驻留在 Windows 通知区域，并自动读取最新的有效 Codex session。<br>
   *Double-click the EXE. The app stays in the Windows notification area and automatically reads the latest valid Codex session.*
3. 切换到 Codex 主窗口。紧凑悬浮条会出现在窗口边缘；Codex 最小化或失去前台焦点时，悬浮条会自动隐藏。<br>
   *Switch to the Codex main window. The compact overlay appears at its edge and hides automatically when Codex is minimized or no longer in the foreground.*
4. 将鼠标移到悬浮条上查看详情；按住左键拖动可改变锚点和偏移。<br>
   *Hover over the overlay for details; hold the left mouse button and drag to change its anchor and offset.*
5. 右键单击悬浮条或通知区域图标打开设置菜单。双击通知区域图标可快速显示或隐藏悬浮条。<br>
   *Right-click the overlay or tray icon to open settings. Double-click the tray icon to quickly show or hide the overlay.*
6. 退出时选择托盘菜单中的 `Quit`。直接关闭 Codex 不会退出本程序。<br>
   *Choose `Quit` from the tray menu to exit. Closing Codex does not exit this app.*

## 托盘菜单 / Tray Menu

| Menu item | 中文说明 | English description |
| --- | --- | --- |
| `Codex window: detected / not detected` | 只读显示 Codex 窗口是否识别成功 | Read-only Codex window detection status |
| `Session data: detected / not detected` | 只读显示是否已读取有效 token session | Read-only valid token-session status |
| `Show overlay` | 显示或隐藏悬浮条 | Show or hide the overlay |
| `Show context used` | 勾选时显示 Used；取消时显示 Left | Show Used when checked; show Left when unchecked |
| `Light theme` | 在浅色与深色主题之间切换 | Switch between light and dark themes |
| `Scale 80% / 100% / 120%` | 调整悬浮条尺寸 | Change the overlay scale |
| `Session: Auto (latest active)` | 自动跟随最新有效 session | Follow the latest valid session automatically |
| Session list | 固定到所选任务，再次选择 Auto 可解除固定 | Pin the selected task; choose Auto to unpin |
| `Start with Windows` | 写入或移除当前用户的开机启动项 | Add or remove the current user's startup entry |
| `Quit` | 退出程序并移除托盘图标 | Exit the app and remove the tray icon |

## 数据与隐私边界 / Data and Privacy Boundary

程序默认只读取以下 Codex 文件：<br>
*By default, the app only reads these Codex files:*

```text
%USERPROFILE%\.codex\sessions\**\*.jsonl
%USERPROFILE%\.codex\session_index.jsonl
```

设置 `CODEX_HOME` 或命令行路径覆盖时，程序会读取所选目录下对应的 `sessions` 与 `session_index.jsonl`，访问范围不变。<br>
*When `CODEX_HOME` or command-line path overrides are used, the app reads the corresponding `sessions` and `session_index.jsonl` under the selected location; the access boundary remains unchanged.*

程序不会修改 `.codex` 目录、Codex 安装包或 `app.asar`，也不会使用 CDP、DOM 注入、React Fiber、网络 API、认证文件、密钥或 provider token。它只解析结构化的 `session_meta` 和 `token_count` 记录，不解析消息正文。<br>
*The app does not modify the `.codex` directory, Codex installation, or `app.asar`. It does not use CDP, DOM injection, React Fiber, network APIs, authentication files, secrets, or provider tokens. It parses only structured `session_meta` and `token_count` records, not message content.*

只有在用户更改设置后，程序才会创建自身配置：<br>
*The app creates its own configuration only after the user changes a setting:*

```text
%LOCALAPPDATA%\CodexContextMeterLite\config.json
```

启用 `Start with Windows` 时，程序仅写入当前用户的注册表项 `HKCU\Software\Microsoft\Windows\CurrentVersion\Run\CodexContextMeterLite`；关闭该选项会移除此项。<br>
*When `Start with Windows` is enabled, the app writes only `HKCU\Software\Microsoft\Windows\CurrentVersion\Run\CodexContextMeterLite`; disabling the option removes that entry.*

## 指标说明 / Metric Reference

| Metric | 中文说明 | English description |
| --- | --- | --- |
| `Used` | 当前上下文已用比例与 token 数 | Current context usage percentage and tokens |
| `Left` | 当前上下文剩余比例与 token 数 | Remaining context percentage and tokens |
| `Turn` | 最新一次 `last_token_usage` 记录 | Latest `last_token_usage` record |
| `Session` | 当前 session 累计 `total_token_usage` | Cumulative `total_token_usage` for the session |
| `Input` | 输入 token | Input tokens |
| `Cache` | 命中的缓存输入 token | Cached input tokens |
| `Output` | 输出 token | Output tokens |
| `Reasoning` | 推理输出 token | Reasoning output tokens |

`Session` 是累计消耗，不等于模型上下文窗口占用；上下文压力始终以最新 `last_token_usage.total_tokens` 为准。<br>
*`Session` is cumulative consumption, not model context-window occupancy; context pressure always uses the latest `last_token_usage.total_tokens`.*

## 故障排查 / Troubleshooting

如果托盘图标存在但 Codex 窗口没有悬浮条，先打开托盘菜单查看 `Codex window` 与 `Session data` 状态。<br>
*If the tray icon is present but no overlay appears on Codex, first open the tray menu and check the `Codex window` and `Session data` status lines.*

```powershell
# Verify that the standalone UI can be displayed without Codex window detection
.\codex-context-meter-lite.exe --demo

# Inspect the Codex process and package identity without reading conversation data
Get-Process -Name ChatGPT,Codex -ErrorAction SilentlyContinue |
  Select-Object Id, ProcessName, Path, MainWindowTitle, MainWindowHandle

Get-AppxPackage |
  Where-Object Name -Match '^OpenAI\.(Codex|ChatGPT)$' |
  Select-Object Name, PackageFamilyName, Version, Architecture, InstallLocation
```

`--demo` 能显示但 `Codex window: not detected` 通常表示安装渠道或进程识别不匹配；`Codex window: detected` 但 `Session data: not detected` 时，应检查 `CODEX_HOME` 或 session 路径。上述命令不读取 JSONL 内容、对话正文、token 或认证信息。<br>
*If `--demo` works but the menu says `Codex window: not detected`, the installation channel or process identity is usually unmatched. If the window is detected but session data is not, check `CODEX_HOME` or the session paths. These commands do not read JSONL contents, conversation text, tokens, or authentication data.*

## 命令行与开发 / CLI and Development

```powershell
# Show the version
.\codex-context-meter-lite.exe --version

# Use custom Codex data paths
.\codex-context-meter-lite.exe --sessions "D:\path\to\sessions" `
  --session-index "D:\path\to\session_index.jsonl"

# Bind an enterprise/repackaged Codex executable explicitly
.\codex-context-meter-lite.exe --codex-exe "D:\Apps\Codex\ChatGPT.exe"

# Open a standalone UI with synthetic data
.\codex-context-meter-lite.exe --demo
```

构建需要 Go 1.26 或更高版本。<br>
*Building requires Go 1.26 or later.*

```powershell
go test ./...
go test -race ./...
go vet ./...
.\scripts\build.ps1
```

发布脚本生成单 EXE portable ZIP，不会安装任何依赖。当前构建目标为 `dist\codex-context-meter-lite-windows-amd64-v0.1.0.zip`。<br>
*The release script creates a portable single-EXE ZIP and installs no dependencies. The current build target is `dist\codex-context-meter-lite-windows-amd64-v0.1.0.zip`.*

## 卸载 / Uninstall

1. 从托盘菜单选择 `Quit`。 / Choose `Quit` from the tray menu.
2. 如果启用了 `Start with Windows`，建议先在托盘菜单中取消勾选。 / If `Start with Windows` is enabled, disable it from the tray menu first.
3. 删除解压出的程序目录。若需同时清除偏好设置，可手动删除 `%LOCALAPPDATA%\CodexContextMeterLite`。 / Delete the extracted app directory. To remove preferences as well, manually delete `%LOCALAPPDATA%\CodexContextMeterLite`.

## 引用 / References

本项目在设计与实现评估阶段参考了以下开源项目：<br>
*This project referenced the following open-source projects during its design and implementation evaluation:*

- [Codex Context Used Meter](https://github.com/Minghou-Lei/codex-context-used-meter)
- [Codex Monitor](https://github.com/KevinKE93/Codex-Monitor)

## 许可证 / License

本项目使用 MIT License。第三方项目与 Go 依赖声明见 [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)。<br>
*This project is licensed under the MIT License. See [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md) for third-party projects and Go dependency notices.*
