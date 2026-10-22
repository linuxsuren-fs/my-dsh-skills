---
name: windows-repo-tooling-pitfalls
description: Windows 仓库工具链陷阱：无 BOM 中文 ps1 被 PowerShell 5.1 按 GBK 误读、go:embed 前端产物必须入库的 gitignore 例外、全局文本替换式迁移的误伤面排查、WSL bash 与 git-bash 的 PATH 之争、DNS 污染时段的外网依赖应对。
whenToUse: 在 Windows 上开发/构建跨平台工具、写含中文的 PowerShell 脚本、仓库内嵌前端 dist 产物、做模块路径/域名迁移替换、或本机验证因 bash 环境错乱/外网不可达失败时。
---

# Windows 仓库工具链陷阱

## 1. 无 BOM 的中文 .ps1 → PowerShell 5.1 按 GBK 误读

- 现象：语法明明完整的脚本报"缺少右}"且位置离奇（报错点在好几个语句块之后）
- 根因：UTF-8 无 BOM + 含中文 → PS 5.1 按系统 ANSI(GBK) 解码，中文误读破坏字符串/语法
- 修复：文件加 UTF-8 BOM；**无副作用复现验证**：`[System.Management.Automation.Language.Parser]::ParseFile(...)` 比 `powershell -File` 安全（不执行）
- 预防：仓库含非 ASCII 的 .ps1 一律带 BOM；同类问题存在于一切按 BOM 探测编码的工具链

## 2. go:embed 的前端 dist 产物必须入库

- `//go:embed web/dist` 依赖工作树文件——**必须随仓库分发**，否则干净检出构建出的二进制缺 UI 资源
- 陷阱：仓库级 `.gitignore` 的 `dist/`（为打包输出设的）会误伤它——新 hash 的 JS/CSS 被**静默忽略**，M/D 状态看不出缺失，直到干净构建才炸
- 修复：`.gitignore` 加否定例外（先豁免目录再豁免内容 `!deploy-tool/web/dist/` + `!**`），并把新产物显式提交
- 自检：构建产物更新后 `git status` 里必须能看到新 hash 文件；看不到=被 ignore 了

## 3. 全局文本替换式迁移的误伤面

模块路径/域名迁移用全局 sed 后，**非 import/非配置位置的字符串**最易被打坏（实测：容器名、临时文件路径、跳过清单键、日志程序名全中招）。迁移后必做：

```
git grep -n '<新路径>' -- '*.go' | 剔除 import 与 go.mod，逐个核对
```

典型坏味道：拼进命令/文件名/容器名的字符串含 `/`、`:`——docker 容器名只允许 `[a-zA-Z0-9][a-zA-Z0-9_.-]`。

## 4. WSL bash 与 git-bash 的 PATH 之争

- Windows 上 `bash` 解析优先级：PATH 里 WSL(`system32\bash.exe`) 常排前——**WSL bash 内没有 docker**（除非开 Docker Desktop WSL 集成），本机走 bash 的测试全挂，报"docker: command not found"实为进错了 bash
- 应对：本机验证前把 git-bash 目录前置 PATH（`$env:PATH='C:\Program Files\Git\bin;'+$env:PATH`），新进程生效
- 判别法：报错带 "could not be found in this WSL 2 distro" 即中了此坑

## 5. DNS 污染时段的外网依赖应对

docker.io 解析到假 IP（如 Twitter/Facebook 段）时：直连 pull 全超时。应对优先级：registry 镜像副本（manifest 级直拷见 jenkins-pipeline-dev）> 换可用网络/时段重试 > 构建可注入 `--build-arg` 指向内网副本的设计（让构建不依赖外网是终极解）。

## 6. 速查表

| 现象 | 根因 |
|---|---|
| ps1 报缺少右}、位置离奇 | 无 BOM 中文被 5.1 按 GBK 误读 |
| 干净检出构建的工具有 UI 缺资源 | dist 新产物被 gitignore 静默忽略，未入库 |
| 容器名/路径报非法字符 | 全局替换把模块路径拼进了命令字符串 |
| 本机 bash 调 docker 报 command not found | 进了 WSL bash（无 docker），PATH 前置 git-bash |
| docker pull 解析到奇怪 IP 超时 | DNS 污染，用镜像副本/imagetools 直拷 |
