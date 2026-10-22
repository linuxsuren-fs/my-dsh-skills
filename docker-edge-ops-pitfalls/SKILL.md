---
name: docker-edge-ops-pitfalls
description: 边缘机 docker 守护进程与容器出网运维陷阱：iptables:false 导致容器出不了公网的诊断与双路径修复、enable --now 不重启已运行服务、scratch 镜像 docker create 要求、Client/Server 版本错位、离线自包含物料的修复传播链、连通性探测分类法。
whenToUse: 容器无法访问公网而宿主机正常、docker 安装/升级后版本错位、制作或提取纯文件载体镜像、离线包内脚本修复不生效、部署后需要连通性自检分类排障时。
---

# 边缘机 docker 运维陷阱

## 1. 容器出不了公网（宿主机正常）——NAT 缺失

**诊断特征**：容器 ping 自身网关/宿主机通，ping 任何公网 IP 全部 100% 丢包；宿主机一切正常。首选检查：

1. `/etc/docker/daemon.json` 是否含 `"iptables": false`（dockerd 完全不写 NAT 规则）
2. `iptables -t nat -S POSTROUTING` 是否有各 bridge 网段的 MASQUERADE
3. `sysctl net.ipv4.ip_forward`、`iptables -S FORWARD` policy

机理：ip_forward=1 + FORWARD ACCEPT 时流量其实被转发出网卡，但**源 IP 是私有地址**，上游无回包 → 全超时。

**修复双路径**：

- **方案 A（推荐）**：JSON 合并方式从 daemon.json 移除该键（**保留其余全部配置**，禁止整文件覆盖；先备份）+ `systemctl restart docker`，让 dockerd 自动管理 NAT
- **方案 B（不重启）**：为各 bridge 网段补 `iptables -t nat -A POSTROUTING -s <网段> ! -o <网桥> -j MASQUERADE`；**网段/网桥名必须动态获取**（`docker network inspect`：Subnet + Options 网桥名，缺省 `br-<id前12>`/docker0），禁止硬编码 172.17/172.18；持久化用 systemd oneshot unit，**开机动态重算**（网络增删自维护）而非固化规则
- 幂等：加规则前 `-C` 查重；配置已是目标态则 no-op

## 2. `systemctl enable --now` 不会重启已运行的服务

替换二进制后（如离线物料装 docker）必须 `systemctl restart`——`--now` 只启动未运行的服务。否则出现 **Client（新二进制）/ Server（旧进程）版本错位**：`docker version` 两端版本不同即为未生效；安装脚本应自带版本一致性断言。

## 3. scratch 纯文件载体镜像的 docker create 要求

`FROM scratch` + COPY 的镜像若没有 ENTRYPOINT/CMD，`docker create` 被 daemon 拒绝（"no command specified"）。两种解法：镜像侧加哑 `ENTRYPOINT`（注明"仅为满足 create，不会执行"）；或提取侧 `docker create <img> true` 末位追加兜底命令（对存量镜像也生效）。

## 4. 离线自包含物料的修复传播链

打包工具在**打包时**从镜像提取脚本固化进包——修脚本必须走完：合并代码 → CI 重推镜像 → **重新打包** → 才到得了目标机。旧包永远带旧脚本；只更新工具二进制无效。装好的机器手动 `systemctl restart docker` 可立即生效（二进制已就位）。

## 5. 部署后连通性自检分类法（拒绝笼统报错）

临时容器接入业务网络（探测镜像用**包内已加载镜像**，离线环境无需拉取），依次探测并分类：

| 结果组合 | 结论 |
|---|---|
| 网关不通 | 网桥/子网配置异常（非 NAT 问题） |
| 网关通 + 公网 IP 不通 | NAT 缺失（查 iptables:false） |
| 公网通 + 业务端点不通 | 对端防火墙/白名单，或端点未监听 |
| 公网通 + 域名不通 | DNS 问题（resolv.conf/内网 DNS） |

## 6. 速查表

| 现象 | 根因 |
|---|---|
| 容器连任何公网超时、宿主机正常 | daemon.json iptables:false → 无 MASQUERADE |
| 装了 27 的物料、Server 还是 29 | enable --now 没重启旧 dockerd |
| docker create 报 no command specified | scratch 镜像无 ENTRYPOINT/CMD |
| 修了安装脚本、装机还是旧行为 | 包内脚本是打包时固化的，要重推镜像+重新打包 |
| 方案 B 重启后规则又没了 | 持久化必须开机动态重算，不能固化当时网段 |
