---
name: jenkins-pipeline-dev
description: Jenkins 声明式流水线（Groovy/sh 混编）实战陷阱：sh 步骤变量转义三定律、私有镜像仓库登录前置、多架构 buildx 镜像与 manifest 直拷同步、构建矩阵文件化与推送后架构校验。
whenToUse: 编写或调试 Jenkinsfile（Groovy 脚本内嵌 shell）、CI 构建推送容器镜像（尤其多架构 amd64+arm64）、同步镜像仓库副本、或流水线报 MissingPropertyException/注释行执行失败时。
---

# Jenkins 流水线（Groovy/sh 混编）实战

## 1. sh 步骤变量转义三定律（全部实测踩坑）

Groovy 三引号字符串里的 shell 脚本，变量归属极易弄错：

| 规则 | 错误形态 | 后果 |
|---|---|---|
| ① shell 变量必须单反斜杠转义：`\${var}` | 写成 `${var}` 或 `\\${var}` | Groovy 运行时求值 → `MissingPropertyException`，并行分支全部即挂（`\\$` 是"字面反斜杠+仍然插值"，双反斜杠救不了） |
| ② shell 引用 shell 变量必须双引号 | `grep -q '${var}'`（单引号） | 变量不展开，校验恒失败/匹配恒空 |
| ③ 注释必须用 `#` | `// 注释` | sh 默认 `-xe`，`//` 被当命令执行直接失败 |

**要点**：Groovy 插值变量（分支名、镜像名）与 shell 循环变量混排时，逐个变量问自己"谁求值"；注释里也不能出现裸 `${}`（注释同样被 Groovy 解析）。

## 2. 私有镜像仓库：登录是流水线第一步

- 私有仓库**拉取校验镜像同样要认证**——不要依赖 agent 上的残留 docker login 状态（会过期/被清理）
- 流水线开头 `withCredentials` 显式登录一次，后续拉取/推送全覆盖

## 3. 多架构镜像：buildx 与 manifest 直拷

- 构建：`docker buildx build --platform linux/amd64,linux/arm64 --provenance=false --sbom=0 -t <img> --push` 一步完成；多架构 manifest 无法落成本地镜像再推
- **同步镜像副本到私有仓库必须用 `docker buildx imagetools create -t <目标> <源>`**（manifest 级直拷，不拉层）——单机 `pull → retag → push` 只会推本机架构，arm64 静默丢失
- 基础镜像含 `RUN`（apt/go build 等）的跨架构构建需 agent 具备 binfmt/qemu；纯文件组装（scratch+COPY）不需要
- **推送后校验 manifest 含全部目标架构**（`docker manifest inspect | grep arch`），防静默退化单架构

## 4. 构建矩阵文件化

版本清单放独立文件（如 `build-matrix.env`，每行 `tag=基础镜像`），流水线纯 Groovy 读取后 `parallel` 分支构建；加/下线版本只改文件不动流水线。矩阵为空要显式报错防呆。

## 5. 速查表

| 现象 | 根因 |
|---|---|
| MissingPropertyException，分支即挂 | sh 脚本里的 shell 变量没转义成 `\${var}` |
| 架构校验恒失败 | grep 模式在单引号里，变量不展开 |
| 脚本第一行注释就失败 | `//` 注释被 `-xe` 当命令执行 |
| CI 拉基础镜像 401/not found | 私有仓库未登录（登录态不能靠残留） |
| arm64 包安装报 exec format error | 镜像副本同步用了单机 pull+push，arm64 变体丢了 |
| 构建成功但目标机缺变体 | 推完没校验 manifest，静默退化单架构 |
| apt 镜像源 https 证书不信任 | 网络对 443 有证书劫持——apt 源统一降级 http（GPG 签名保完整性） |
