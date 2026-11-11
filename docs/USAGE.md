# kube-explorer 使用说明

面向**使用这个工具的人**（而不是构建/发布它的人）。构建、发版、CI 的说明见仓库 [README](../README.md)。

> 适用版本：本文的命令与参数以 **Line B `v0.6.1-1.34`**（Kubernetes 1.34–1.36，Rancher v2.15.2 栈）为准。仓库按 Kubernetes 版本区间分线维护，默认分支可能仍是上游基线代码；请以对应 release 的产物和 `./kube-explorer --help` 实际输出为准。Line B 已在 RKE2 `v1.34.7+rke2r1` 真集群上完成验证。

## 这是什么

kube-explorer 是一个**单文件、几乎无状态**的 Kubernetes 浏览器：内嵌 Rancher Dashboard UI 与 api-ui，基于 steve 框架，用 kubeconfig 直连 kube-apiserver。不需要 Rancher Server、不需要数据库、不需要额外的前端服务。

典型用途：在 RKE2 / K3s 集群上快速查看与管理集群资源，而不部署一整套 Rancher。

- 一个静态二进制（内嵌 UI）或一个很小的容器镜像即可运行
- 支持 `linux/amd64` 与 `linux/arm64`
- 启动后打开浏览器即可使用，根路径 `/` 会跳转到 `/dashboard/`

> **安全前提**：kube-explorer 默认**没有登录认证**，任何能访问其端口的人都会以你所提供 kubeconfig 的权限操作集群。请只在内网/本机使用，或按[安全建议](#9-安全建议)加认证与网络限制。

---

## 1. 选版本、选架构

**先确认集群版本和机器架构**，再选产物：

```bash
kubectl get nodes                 # 看 VERSION 列，例如 v1.32.12+k3s1
uname -m                          # x86_64 → amd64；aarch64 / arm64 → arm64
```

| 构建线 | 适用 Kubernetes | 发版 tag | 状态 |
| ------ | --------------- | -------- | ---- |
| Line A | 1.31 – 1.33（RKE2 / K3s） | `v0.6.0-1.31` | 已发布 |
| **Line B** | **1.34 – 1.36**（RKE2 / K3s） | `v0.6.1-1.34` | 本地构建 + RKE2 1.34.7 真集群验证完成，待发布 |

选错线的典型后果是 client-go 与 apiserver 版本偏差过大导致连接/资源发现异常，**请按集群版本选对应 tag**。下载后可自查：

```bash
./kube-explorer --version        # 例如 kube-explorer version v0.6.1-1.34 (abcdef0)
```

---

## 2. 获取

### 方式 A：下载预编译二进制（推荐）

从 [Releases](https://github.com/Nick11Show/kube-explorer/releases) 下载与架构匹配的文件，两个产物都已内嵌 Dashboard 与 api-ui，运行时**不依赖任何外部 UI 文件**：

```bash
# 以 Line B / amd64 为例
curl -fL -o kube-explorer \
  https://github.com/Nick11Show/kube-explorer/releases/download/v0.6.1-1.34/kube-explorer-linux-amd64
chmod +x kube-explorer
./kube-explorer --version
```

其它 tag / 架构把 URL 中的 tag 与 `amd64` 换成对应值即可。

### 方式 B：从私有 registry 拉镜像

本项目**不向任何公共 registry 推送镜像**，镜像只发布在自建 registry（例如 Harbor）。地址与凭据请向运维获取：

```bash
nerdctl pull <your-registry>/library/kube-explorer:v0.6.1-1.34
# 或 docker pull <your-registry>/library/kube-explorer:v0.6.1-1.34
```

镜像是 `linux/amd64 + linux/arm64` 多架构 manifest，拉取时会按本机架构自动选择。运行时镜像基于 `FROM scratch`，只有一个静态二进制。

### 方式 C：从源码构建

见 [README「从源码构建」](../README.md#从源码构建)。注意 UI 资产不入库，构建前必须执行 `bash scripts/download`，且必须带 `-tags embed`。

---

## 3. 快速开始

默认监听 **HTTPS 9443 / HTTP 9080**。常用做法是只开 HTTP、显式指定端口并关闭 HTTPS：

### 场景 1：在 RKE2 / K3s 节点上直接运行（无需传参）

程序会自动探测节点上的 kubeconfig（见[下一节](#4-kubeconfig-自动探测)）。RKE2 的 `rke2.yaml` 与 K3s 的 `k3s.yaml` 通常只有 root 可读：

```bash
sudo ./kube-explorer --http-listen-port=8989 --https-listen-port=0
```

### 场景 2：用本机 kubeconfig 连接任意集群

```bash
./kube-explorer --kubeconfig ~/.kube/config \
  --http-listen-port=8989 --https-listen-port=0
```

### 场景 3：用 Bearer Token 认证

`--token` 会用该 token 作为访问 apiserver 的唯一凭据，apiserver 地址与 CA 仍由 `--kubeconfig` 提供（会覆盖 kubeconfig 里的用户凭据）：

```bash
./kube-explorer --kubeconfig /path/to/kubeconfig --token <bearer-token> \
  --http-listen-port=8989 --https-listen-port=0
```

### 场景 4：用 HTTP 头做身份代理（仅 Line B，默认关闭）

`--enable-header-auth` 让前置网关通过 `Impersonate-User` / `Impersonate-Group` 头指定身份；未带这些头时按 admin 处理。**启用后 kube-explorer 自身不再做认证**，必须部署在已经完成鉴权的可信网关后面：

```bash
./kube-explorer --kubeconfig /path/to/kubeconfig \
  --enable-header-auth --http-listen-port=8989 --https-listen-port=0
curl -H 'Impersonate-User: alice' -H 'Impersonate-Group: developers' \
  http://localhost:8989/v1/namespaces
```

> 该 flag 在 steve v0.9 才出现；早期 Line B 代码虽然能解析这个参数却不会生效，本版本已修复（在自定义 `ToServer` 中补上了认证中间件）。

打开浏览器访问 `http://<host>:8989`（`http://<host>:8989/dashboard/` 是实际 UI 路径）。

---

## 4. kubeconfig 自动探测

不显式指定 `--kubeconfig` 时，按以下优先级选择：

1. `--kubeconfig` 显式指定
2. `KUBECONFIG` 环境变量
3. `/etc/rancher/rke2/rke2.yaml`（RKE2 节点）
4. `/etc/rancher/k3s/k3s.yaml`（K3s 节点）
5. 以上都没有 → 回落 client-go 默认规则（在 Pod 内运行时使用 ServiceAccount 的 in-cluster 配置）

因此在 RKE2 / K3s 节点上直接运行即可。也可以用 `--context` 指定 kubeconfig 中的 context。

---

## 5. 全部参数

以下为 **Line B（`v0.6.1-1.34`，steve v0.9.23）** 实测 `--help` 的完整参数：

| 参数 | 默认值 | 说明 |
| ---- | ------ | ---- |
| `--kubeconfig` | `$KUBECONFIG` | kubeconfig 路径；不传则自动探测 |
| `--context` | `$CONTEXT` | kubeconfig 中的 context 名 |
| `--http-listen-port` | `9080` | HTTP 端口；设 `0` 关闭 HTTP |
| `--https-listen-port` | `9443` | HTTPS 端口；设 `0` 关闭 HTTPS |
| `--ui-path` | 空 | 使用外部 UI 目录（不推荐；正式产物已内嵌 UI） |
| `--token` | 空 | Bearer token，覆盖 kubeconfig 用户凭据 |
| `--insecure-skip-tls-verify` | `false` | 跳过 apiserver TLS 校验（仅测试用） |
| `--system-default-registry` | 空 | shell Pod 使用的系统默认镜像仓库 |
| `--pod-image` | `rancher/shell:v0.2.1-rc.7` | shell Pod 镜像 |
| `--bind-address` | 空 | 自定义监听地址，支持 `unix://`、`tcp://`、`namedpipe:` |
| `--webhook-auth` | `false` | 启用 webhook 认证（`$WEBHOOK_AUTH`） |
| `--webhook-kubeconfig` | `$WEBHOOK_KUBECONFIG` | webhook 认证用 kubeconfig |
| `--webhook-url` | `$WEBHOOK_URL` | webhook 认证服务地址 |
| `--webhook-cache-ttl` | `0` | webhook 认证缓存 TTL（`$WEBHOOK_CACHE_TTL`） |
| `--debug` | `false` | 开启调试 |
| `--debug-level` | `1` | 调试级别（Line A 为 `7`） |
| `--sql-cache` | `false` | 启用 SQL 缓存 |
| `--enable-metrics` | `false` | steve v0.9 新增 |
| `--metrics-listen-addr` | `localhost:6080` | steve v0.9 新增 |
| `--metrics-update-interval-seconds` | `0` | steve v0.9 新增 |
| `--enable-pprof` | `false` | steve v0.9 新增 |
| `--pprof-listen-addr` | `localhost:6060` | steve v0.9 新增 |
| `--enable-header-auth` | `false` | 允许 `Impersonate-*` 头，见场景 4 |
| `--apiui-version` | `1.1.11` | api-ui 版本（隐藏参数） |
| `--help` / `--version` | — | 帮助 / 版本 |

以 `./kube-explorer --help` 实际输出为准；不同小版本的 steve 可能增减参数。

### Line B 的环境变量默认值

程序启动时会把 `KUBE_FEATURE_WatchListClient` 默认设为 `false`（仅当你没有显式设置时才生效）。原因：client-go v0.36 默认开启的流式 LIST 在 **K8s 1.34** apiserver 上收不到结束 bookmark，informer 永不就绪，列表接口会一直挂起；关闭后回落为普通分页 LIST，1.34–1.36 都能用。若你的集群是 1.35+ 且想启用该优化：

```bash
KUBE_FEATURE_WatchListClient=true ./kube-explorer --kubeconfig ... \
  --http-listen-port=8989 --https-listen-port=0
```

---

## 6. 部署到 Kubernetes 集群

`deploy/kubectl/` 提供了可直接 apply 的清单（Line B 使用**独立命名空间 `kube-explorer`**，只新增对象、不改动集群现有工作负载）：

- `namespace.yaml` — 独立命名空间 `kube-explorer`
- `serviceAccount.yaml` — ServiceAccount
- `clusterRoleBinding.yaml` — 绑定集群读取/操作权限（可选，见下）
- `service.yaml` — ClusterIP Service，端口 `8989`
- `deployment.yaml` — Deployment，启动参数 `--https-listen-port=0 --http-listen-port=8989`

```bash
kubectl apply -f deploy/kubectl/namespace.yaml \
              -f deploy/kubectl/serviceAccount.yaml \
              -f deploy/kubectl/clusterRoleBinding.yaml \
              -f deploy/kubectl/service.yaml \
              -f deploy/kubectl/deployment.yaml

kubectl -n kube-explorer rollout status deployment/kube-explorer
kubectl -n kube-explorer get pods -l app=kube-explorer -o wide
```

回滚只需删除这些新增对象：`kubectl delete ns kube-explorer` 再 `kubectl delete clusterrolebinding kube-explorer`。

### 部署前必须改的地方

1. **镜像地址**：`deployment.yaml` 里的 `image` 当前是本项目验证环境的**私有 registry 示例地址**，其它环境无法拉取，必须改成你自己可访问的 registry（见[下一节](#7-容器镜像与私有-registry)）。
2. **kubeconfig 路径**：Line B 清单挂载的是 RKE2 的 `/etc/rancher/rke2/rke2.yaml`。K3s 节点请把 `volumeMounts[].mountPath` 与 `volumes[].hostPath.path` **两处一起**改为 `/etc/rancher/k3s/k3s.yaml`。
3. **镜像可拉取性**：若节点访问不到 registry，可先把镜像导入节点 containerd，并保持 `imagePullPolicy: IfNotPresent`；自签名证书的节点配置见第 7 节。
4. **权限（可选）**：因为清单挂载了节点 kubeconfig（本身已具备管理员凭据），`clusterRoleBinding.yaml` 在验证场景下并非必需。若希望不新增任何集群级 RBAC 对象，可以不 apply 它。

### 清单的几个设计点（改的时候别删）

- `hostNetwork: true`：让 Pod 能访问 kubeconfig 里写的 `https://127.0.0.1:6443`，同时把 8989 暴露在节点上。
- `strategy: Recreate`：单节点 hostNetwork 场景下，避免新旧 Pod 抢占 8989 端口导致滚动更新死锁。
- `tolerations`：容忍 `node-role.kubernetes.io/control-plane`（RKE2/K3s）、旧版 `node-role.kubernetes.io/controlplane` 和 `cattle.io/os` 污点，确保能调度到默认打了污点的 control-plane 节点。
- `readinessProbe`：`tcpSocket: 8989`。

### 访问 UI

- hostNetwork 部署：直接 `http://<节点IP>:8989`
- 通过 Service：`kubectl -n kube-explorer port-forward svc/kube-explorer 8989:8989`，然后 `http://localhost:8989`
- 通过 Ingress / 路径前缀 / Basic Auth：见 [deploy/kubectl/README.md](../deploy/kubectl/README.md) 及 `nginx-auth/`、`traefik-v1-auth/`、`traefik-v2-auth/`、`path-prefix/` 子目录

---

## 7. 容器镜像与私有 registry

**发布策略**：CI 只产出 `linux/amd64`、`linux/arm64` 两个静态二进制并附到 GitHub release；**镜像在本机构建后推送到自建私有 registry**，不推送到公共 registry。这样 CI 无需访问内网，也不需要公共仓库写权限。

### 从私有 registry 使用

```bash
# 自签名证书的 Harbor 需要 --insecure-registry
nerdctl login <your-registry> -u <user> -p <password> --insecure-registry
nerdctl pull <your-registry>/library/kube-explorer:v0.6.1-1.34
```

### 自行构建并推送多架构镜像

准备一个构建目录，放入从 release 下载的二进制（`kube-explorer-linux-amd64`、`kube-explorer-linux-arm64`），再把仓库的 `package/Dockerfile` 复制进去：

```bash
BUILD=/path/to/build-dir
mkdir -p "$BUILD"
cp kube-explorer-linux-amd64 kube-explorer-linux-arm64 "$BUILD/"
cp package/Dockerfile "$BUILD/"

REG=<your-registry>/library/kube-explorer
TAG=v0.6.1-1.34

# 分别构建两个单架构镜像（Dockerfile 按 TARGETOS/TARGETARCH 选二进制）
nerdctl build --platform linux/amd64 -f "$BUILD/Dockerfile" -t "$REG:$TAG-amd64" "$BUILD"
nerdctl build --platform linux/arm64 -f "$BUILD/Dockerfile" -t "$REG:$TAG-arm64" "$BUILD"
nerdctl push --insecure-registry "$REG:$TAG-amd64"
nerdctl push --insecure-registry "$REG:$TAG-arm64"

# 合成多架构 manifest（子镜像需已存在于 registry 中）
nerdctl manifest create "$REG:$TAG" "$REG:$TAG-amd64" "$REG:$TAG-arm64"
nerdctl manifest push --insecure-registry "$REG:$TAG"
```

> **自签名证书注意**：`nerdctl manifest create` 不会读取命令行的 `--insecure-registry`，它需要向 registry 拉取子镜像的 manifest。若使用自签名证书的 Harbor，需先在 containerd/nerdctl 配置里把该 host 标为 insecure，例如：
>
> ```bash
> # /etc/nerdctl/nerdctl.toml
> insecure_registry = true
>
> # /etc/containerd/certs.d/<your-registry>/hosts.toml
> [host."https://<your-registry>"]
>   insecure = true
>   skip_verify = true
> ```
>
> 在 Rancher Desktop 环境里，这些文件位于 lima VM 内（用 `rdctl shell sudo ...` 写入）。

**RKE2 节点上的坑（实测）**：Harbor 自签证书一般只有 DNS SAN（例如 `CN=harbor.rancher.com`），用 **IP 形式**访问时即使把 CA 加入系统信任库也会失败：`x509: cannot validate certificate for 100.116.110.97 because it doesn't contain any IP SANs`。解决方法是对该 host 单独关闭校验，且**不需要重启 RKE2/containerd、不影响已有工作负载**：

```bash
# 1) 立即生效（containerd 按 hosts.toml 逐次读取，无需重启）
sudo mkdir -p /var/lib/rancher/rke2/agent/etc/containerd/certs.d/<registry-ip>:8443
sudo tee /var/lib/rancher/rke2/agent/etc/containerd/certs.d/<registry-ip>:8443/hosts.toml <<'EOF'
server = "https://<registry-ip>:8443/v2"
capabilities = ["pull", "resolve", "push"]
skip_verify = true

[host."https://<registry-ip>:8443"]
  capabilities = ["pull", "resolve", "push"]
  skip_verify = true
EOF

# 2) 持久化：在 /etc/rancher/rke2/registries.yaml 的 configs 下追加（不影响其它 host）
#   "<registry-ip>:8443":
#     tls:
#       insecure_skip_verify: true
# 先备份原文件；RKE2 下次重启时会据此重新生成 certs.d。
```

验证：`crictl pull <registry-ip>:8443/library/kube-explorer:v0.6.1-1.34` 能成功即可。另一个更简单的替代方案是始终使用证书里带 DNS SAN 的域名（如 `harbor.rancher.com:8443`）作为镜像地址。

完成后用 `nerdctl manifest inspect "$REG:$TAG"` 或 registry 的 API 确认包含 `linux/amd64` 与 `linux/arm64` 两个平台。

---

## 8. 界面与接口自检

浏览器打开 `http://<host>:8989/` 会 302 跳转到 `/dashboard/`。以下为实测可用的自检端点（steve API）：

```bash
curl -s http://localhost:8989/v1/nodes       | jq '.count'
curl -s http://localhost:8989/v1/pods        | jq '.count'
curl -s http://localhost:8989/v1/namespaces  | jq '.count'
curl -s -o /dev/null -w '%{http_code}\n' http://localhost:8989/dashboard/
curl -s -o /dev/null -w '%{http_code}\n' http://localhost:8989/favicon.ico
```

`/dashboard/`、静态 JS 资源、`favicon.ico`、api-ui 均返回 200 说明内嵌 UI 完整。Dashboard 内可浏览集群的节点、Pod、命名空间、工作负载等资源；具体可用的编辑/终端能力取决于所连集群与 kubeconfig 权限，以实际 UI 为准。

---

## 9. 安全建议

- **默认无登录认证**：不要用 kubeconfig 里的高权限凭据把它直接暴露到公网。
- 优先只在 `localhost` / 内网监听，或用 Ingress + Basic Auth（见 `deploy/kubectl/nginx-auth/`、`traefik-v2-auth/`）。
- 需要认证时可启用 `--webhook-auth` 并配合 `--webhook-url` / `--webhook-kubeconfig`；集群内部署时给 ServiceAccount 最小必要权限。
- 容器镜像只放私有 registry；不要使用来源不明的镜像。
- `--insecure-skip-tls-verify` 仅用于排障，不要长期使用。

---

## 10. 常见问题

| 现象 | 原因与处理 |
| ---- | ---------- |
| `dial tcp 127.0.0.1:6443: connect refused` | Pod 内访问不到 kubeconfig 写死的 apiserver 地址。用 `hostNetwork: true`，或改用 server 地址指向集群真实可达地址的 kubeconfig；RKE2/K3s 挂载路径见第 6 节。 |
| UI 404 / 白屏、静态资源 404 | 二进制不是 `-tags embed` 构建，或误设了 `--ui-path` 指向不存在的目录。用 release 里的正式产物，检查 `/dashboard/`、`favicon.ico` 是否 200。 |
| `exec format error` | 架构选错。用 `uname -m` 重新确认并下载对应二进制。 |
| 端口被占用 / 更新后一直 Pending | 默认 9443 与 9080；单节点 hostNetwork 下滚动更新会抢 8989 端口。保持 `strategy: Recreate`，必要时先删掉旧 Pod。 |
| API 返回 403 / 资源列表为空 | kubeconfig 用户或 ServiceAccount 权限不足；集群内部署时检查 `clusterRoleBinding.yaml`。 |
| apiserver 证书校验失败 | 测试环境可临时加 `--insecure-skip-tls-verify`；正式环境应配置正确的 CA。 |
| 连接新版本集群异常 | 检查是否用错构建线；按第 1 节的支持矩阵选择 tag。 |
| Line B 在 K8s 1.34 上 `/v1/namespaces`、`/v1/pods` 一直不返回，但单个对象和 `?limit=1` 正常 | client-go `WatchListClient` 与 1.34 apiserver 的 initial-events bookmark 不兼容，informer 缓存未同步。本版本已默认关闭该特性；若被外部显式设为 `true`，删除 `KUBE_FEATURE_WatchListClient` 或设为 `false` 后重启。日志里会有成片的 `Warning: event bookmark expired`。 |
| 拉取镜像时报 TLS 错误 | 自签名 registry：见第 7 节的 insecure registry 配置与 `--insecure-registry`。 |
| `x509: cannot validate certificate for <IP> because it doesn't contain any IP SANs` | registry 证书只有 DNS SAN。改用证书中包含的域名，或按第 7 节为该 IP host 配 `skip_verify = true`（RKE2 节点无需重启服务）。 |

排障时可加 `--debug --debug-level 7`，并确认服务端版本：`./kube-explorer --version`。

---

## 相关链接

- [README（版本矩阵 / 构建 / CI）](../README.md)
- [Releases 下载页](https://github.com/Nick11Show/kube-explorer/releases)
- [部署清单与 Ingress 示例](../deploy/kubectl/README.md)
- [上游项目 cnrancher/kube-explorer](https://github.com/cnrancher/kube-explorer)
