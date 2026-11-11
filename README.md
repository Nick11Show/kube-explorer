# kube-explorer

kube-explorer is a portable, almost stateless explorer for Kubernetes with no external dependency. It integrates the Rancher **steve** framework and its dashboard, and ships as a single static binary with the UI embedded.

本仓库是按 Kubernetes 版本区间**分线维护**的构建版本：不同 Kubernetes 大版本使用的 steve / client-go 差异较大，因此拆成两条独立构建线，各自锁定依赖栈并单独发版。

> 上游项目：[cnrancher/kube-explorer](https://github.com/cnrancher/kube-explorer)
>
> **使用者文档**：
> - [使用说明](./docs/USAGE.md) — 获取方式、连接集群、部署、访问 UI、常见问题。
> - [远程 / Tailscale 使用手册](./docs/REMOTE-USAGE.md) — 从另一台机器（如 Mac mini）使用、私有 Harbor 与自签证书、跨机器连接远程集群、端口转发与排障。
>
> 本 README 侧重版本矩阵、构建与发布。

![](docs/assets/kube-explorer-record.gif)

## 版本与支持矩阵

| 构建线 | 分支 | 目标 K8s 版本 | 依赖栈 | 发版 tag | 状态 |
| ------ | ---- | ------------- | ------ | -------- | ---- |
| 上游 v0.4 | — | >= 1.25 <= 1.28 | Rancher v2.8.x | — | 上游历史版本 |
| 上游 v0.5 | `main` 基线 | >= 1.27 <= 1.30 | Rancher v2.9.x | — | 上游历史版本 |
| **Line A** | `k8s-1.31-1.33` | **1.31 – 1.33**（RKE2 / K3s） | Rancher v2.12.x：steve v0.6.40、client-go v0.33.4、Go 1.24.x | `v0.6.0-1.31` | **已发布** |
| **Line B** | `k8s-1.34-1.36` | **1.34 – 1.36**（RKE2 / K3s） | Rancher v2.15.2：steve v0.9.23、client-go v0.36.4、Go 1.26.x | `v0.6.1-1.34` | **已发布** |

**请按集群的 Kubernetes 版本选择对应的构建线与 tag**，跨大版本混用可能因 client-go 与 apiserver 不兼容而失败。

- Line A 已在单节点 K3s `v1.32.12+k3s1`（linux/arm64）完成端到端验证：节点 / Pod / Namespace 查询、Dashboard 与静态资源均正常。
- Line B 已在单节点 RKE2 `v1.34.7+rke2r1`（linux/amd64）完成端到端验证：节点 / Pod / Namespace 查询、Rancher v2.15.2 Dashboard 与静态资源均正常；并真机验证了 `/etc/rancher/rke2/rke2.yaml` 自动探测分支。

> **Line B 兼容性修复（必读）**：client-go v0.36.4 默认开启的 `WatchListClient`（K8s 1.35+ 起默认 true）依赖 apiserver 发送 initial-events-end bookmark。在 **K8s 1.34** 上该 bookmark 不到达，informer 缓存永远无法同步，`/v1/namespaces`、`/v1/pods` 等列表接口会**无限挂起**。本分支在程序启动时自动设置 `KUBE_FEATURE_WatchListClient=false`（除非用户已显式设置该变量），改用经典分页 LIST，覆盖 1.34–1.36 全区间。1.35+ 集群可显式设置 `KUBE_FEATURE_WatchListClient=true` 重新启用流式 LIST。

## 快速开始

### 下载

从 [Releases](https://github.com/Nick11Show/kube-explorer/releases) 下载对应架构的静态二进制：

- `kube-explorer-linux-amd64`
- `kube-explorer-linux-arm64`

两个产物都内嵌了 Dashboard 与 api-ui，运行时**不需要任何外部 UI 文件或后端服务**。

### 运行

默认监听 HTTPS `9443` / HTTP `9080`。常用做法是只开 HTTP 并指定端口（关闭 HTTPS）：

```bash
./kube-explorer --kubeconfig=/path/to/kubeconfig \
  --http-listen-port=8989 --https-listen-port=0
```

浏览器打开 `http://<host>:8989`（根路径会跳转到 `/dashboard/`）。完整参数与更多用法见 [使用说明](./docs/USAGE.md)。

### kubeconfig 自动探测

不显式指定 `--kubeconfig` 时，按以下优先级自动选择：

1. `--kubeconfig` 显式指定
2. `KUBECONFIG` 环境变量
3. `/etc/rancher/rke2/rke2.yaml`（RKE2 节点）
4. `/etc/rancher/k3s/k3s.yaml`（K3s 节点）
5. 以上都没有时，回落到 client-go 默认规则（在 Pod 内运行时使用 ServiceAccount 的 in-cluster 配置）

因此在 RKE2 / K3s 节点上直接运行二进制即可，无需传参。

### 使用 Token 认证

`--token` 用 Bearer Token 访问 kube-apiserver，配合 `--kubeconfig` 提供 apiserver 地址与 CA 即可，会覆盖 kubeconfig 中的用户凭据：

```bash
./kube-explorer --kubeconfig=/path/to/kubeconfig --token=<bearer-token> \
  --http-listen-port=8989 --https-listen-port=0
```

### 其他参数

| 参数 | 说明 |
| ---- | ---- |
| `--insecure-skip-tls-verify` | 跳过 apiserver TLS 校验 |
| `--system-default-registry` | shell Pod 使用的系统默认镜像仓库 |
| `--pod-image` | shell Pod 镜像，默认 `rancher/shell:v0.2.1-rc.7` |
| `--bind-address` | 自定义监听地址（`unix://`、`tcp://` 等） |
| `--apiui-version` | api-ui 版本（隐藏参数，默认 1.1.11） |

Line B（steve v0.9 栈）在以上参数之外还接受：`--enable-metrics`、`--metrics-listen-addr`、`--metrics-update-interval-seconds`、`--enable-pprof`、`--pprof-listen-addr`、`--enable-header-auth`。其中 **`--enable-header-auth` 此前虽被 flag 解析但不会生效**（kube-explorer 用自己的 `ToServer` 构建 server），本分支已补齐：启用后优先采用请求的 `Impersonate-User` / `Impersonate-Group` 头，未携带时回落到 admin 身份。该开关会绕过认证，请只在受信任网络中使用。

## 部署到 Kubernetes

`deploy/kubectl/` 提供了可直接 apply 的清单（Namespace / Deployment / Service / ServiceAccount / ClusterRoleBinding）。Line B 使用**独立命名空间 `kube-explorer`**，部署只新增对象，不改动集群里已有的任何工作负载：

```bash
kubectl apply -f deploy/kubectl/namespace.yaml \
              -f deploy/kubectl/serviceAccount.yaml \
              -f deploy/kubectl/clusterRoleBinding.yaml \
              -f deploy/kubectl/service.yaml \
              -f deploy/kubectl/deployment.yaml
```

部署前请按需修改 `deployment.yaml`：

- `spec.template.spec.containers[].image`：改成**你自己可访问的镜像仓库**（清单中当前是一个私有 registry 示例地址，仅用于本地集群验证）。
- `volumeMounts` / `volumes` 的 kubeconfig 路径：Line B 清单默认是 RKE2 的 `/etc/rancher/rke2/rke2.yaml`；K3s 节点请两处一起改成 `/etc/rancher/k3s/k3s.yaml`。
- `imagePullPolicy`：内网 registry 不可从节点访问时，可预先把镜像 load 进节点 containerd 并设为 `IfNotPresent`。

设计要点：

- `hostNetwork: true` + hostPath 挂载 RKE2 的 `rke2.yaml`，使 Pod 内能访问 kubeconfig 中的 `https://127.0.0.1:6443`，同时把 8989 暴露在节点上；程序自动探测该路径，无需传 `--kubeconfig`。
- 因为已挂载节点 kubeconfig（本身具备管理员凭据），`clusterRoleBinding.yaml` 在验证场景下并非必需；若希望最小权限 footprint，可以不 apply 它。
- `strategy: Recreate`：单节点 hostNetwork 场景下避免新旧 Pod 抢占 8989 端口导致滚动更新死锁。
- `tolerations`：容忍 `node-role.kubernetes.io/control-plane`（RKE2/K3s）、`node-role.kubernetes.io/controlplane`（RKE1 旧键）与 `cattle.io/os` 污点，保证可调度到默认打了污点的 control-plane 节点。

## 容器镜像

**镜像只发布到自建私有 registry（例如 Harbor），不推送到任何公共 registry。** CI 只产出静态二进制并附到 GitHub release；镜像由运维在本机从这两个二进制构建后推送到私有 registry。这样 GitHub-hosted runner 无需访问内网，也不需要任何公共仓库的写权限。

运行时镜像基于 `FROM scratch`：二进制为 `CGO_ENABLED=0` 全静态链接，镜像内只有 `/usr/bin/kube-explorer`，无系统包依赖、构建过程无需联网安装基础软件。

```bash
# 构建目录内放好两个 release 二进制；Dockerfile 直接复制仓库的 package/Dockerfile
BUILD=/path/to/build-dir
REG=<your-registry>/library/kube-explorer
TAG=v0.6.1-1.34
cp package/Dockerfile "$BUILD/"

nerdctl build --platform linux/amd64 -f "$BUILD/Dockerfile" -t "$REG:$TAG-amd64" "$BUILD"
nerdctl build --platform linux/arm64 -f "$BUILD/Dockerfile" -t "$REG:$TAG-arm64" "$BUILD"
nerdctl push --insecure-registry "$REG:$TAG-amd64"
nerdctl push --insecure-registry "$REG:$TAG-arm64"

# 合成 linux/amd64 + linux/arm64 多架构 manifest（子镜像需已在 registry 中）
nerdctl manifest create "$REG:$TAG" "$REG:$TAG-amd64" "$REG:$TAG-arm64"
nerdctl manifest push --insecure-registry "$REG:$TAG"
```

自签名证书的 Harbor 需先给 containerd 配置 insecure registry（`manifest create` 不读命令行的 `--insecure-registry`）。RKE2 节点还有一个额外坑：证书通常只有 DNS SAN、没有 IP SAN，用 IP 形式访问 registry 即使信任 CA 也会报 `x509: cannot validate certificate for <IP>`，必须对该 host 单独 `skip_verify`；完整步骤见 [使用说明](./docs/USAGE.md#7-容器镜像与私有-registry)。

## 从源码构建

前置：Go 工具链（版本由 `go.mod` 决定，Line B 为 1.26.x）。

**关键：UI 资产不入库**（`.gitignore` 排除 `internal/ui/ui/`），构建带内嵌 UI 的二进制前必须先下载资产，且必须加 `-tags embed`：

```bash
# 1) 下载 Dashboard (Rancher v2.15.2 embedded build) + api-ui (1.1.11)
bash scripts/download

# 2) 构建（当前架构）
CGO_ENABLED=0 go build -tags embed -o dist/kube-explorer .

# 3) 交叉编译双架构
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags embed \
  -ldflags "-s -w -extldflags -static" -o dist/kube-explorer-linux-amd64 .
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -tags embed \
  -ldflags "-s -w -extldflags -static" -o dist/kube-explorer-linux-arm64 .
```

`scripts/download` 从 `releases.rancher.com` 获取官方资产（原上游使用的 pandaria S3 源已失效），可用 `CATTLE_DASHBOARD_UI_VERSION` / `CATTLE_API_UI_VERSION` 覆盖版本。

> 不加 `-tags embed` 构建出的二进制不含 Dashboard 静态资源；该 tag 是内嵌 UI 的开关，不要省略。

本地调试（外部 UI 目录）：

```bash
make dev
./bin/kube-explorer --debug --ui-path /opt/ui/dist/ \
  --http-listen-port=9898 --https-listen-port=0
```

`CROSS=tag make` 走 goreleaser 生成全平台产物（含 UPX 压缩）。

## CI / 发布流程

| Workflow | 触发 | 内容 |
| -------- | ---- | ---- |
| `.github/workflows/pr.yaml` | Pull Request | gofmt / go vet / go test + `linux/amd64`、`linux/arm64` 双架构交叉编译 |
| `.github/workflows/push.yaml` | 分支 push、按线 tag push | 下载 UI 资产 → 双架构静态二进制（含架构校验）→ tag 时创建 draft GitHub release 并附上两个二进制；**不构建/推送镜像** |

- Go 版本通过 `actions/setup-go` 的 `go-version-file: go.mod` 自动选择，因此同一个 workflow 可服务 Go 1.24（Line A）与 Go 1.26（Line B）两条构建线。
- tag 规则：Line A `v*.*.*-1.31`，Line B `v*.*.*-1.34`。
- 分支 push 只构建不建 release；只有 tag push 会创建 draft release 并附上两个架构的二进制。

## 功能

- 指定系统默认 registry：`--system-default-registry`
- 指定 shell 镜像：`--pod-image`
- 反代 / Ingress 部署：
  - [Ingress + 域名](./deploy/kubectl/README.md)
  - [Ingress + 域名及路径前缀](./deploy/kubectl/path-prefix/Readme.md)
  - 通过 Ingress 做 Basic Auth：[nginx](./deploy/kubectl/nginx-auth/README.md)、[traefik-v1](./deploy/kubectl/traefik-v1-auth/README.md)、[traefik-v2](./deploy/kubectl/traefik-v2-auth/README.md)
- RKE2 / K3s kubeconfig 自动探测、`--token` 认证、control-plane 污点容忍、Dashboard + api-ui 内嵌进静态二进制
- （Line B 新增）Rancher v2.15.2 Dashboard 内嵌、`--enable-header-auth` 真正生效、RKE2 独立命名空间部署清单
- （Line B 新增）自动关闭 `WatchListClient` 特性门控，修复 client-go v0.36 在 K8s 1.34 上列表接口挂起的问题

## 相关项目

- [kube-explorer-ui](https://github.com/cnrancher/kube-explorer-ui)
- [autok3s](https://github.com/cnrancher/autok3s)
- [api-ui](https://github.com/rancher/api-ui)
