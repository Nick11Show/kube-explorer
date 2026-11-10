# kube-explorer

kube-explorer is a portable, almost stateless explorer for Kubernetes with no external dependency. It integrates the Rancher **steve** framework and its dashboard, and ships as a single static binary with the UI embedded.

本仓库是按 Kubernetes 版本区间**分线维护**的构建版本：不同 Kubernetes 大版本使用的 steve / client-go 差异较大，因此拆成两条独立构建线，各自锁定依赖栈并单独发版。

> 上游项目：[cnrancher/kube-explorer](https://github.com/cnrancher/kube-explorer)

![](docs/assets/kube-explorer-record.gif)

## 版本与支持矩阵

| 构建线 | 分支 | 目标 K8s 版本 | 依赖栈 | 发版 tag | 状态 |
| ------ | ---- | ------------- | ------ | -------- | ---- |
| 上游 v0.4 | — | >= 1.25 <= 1.28 | Rancher v2.8.x | — | 上游历史版本 |
| 上游 v0.5 | `main` 基线 | >= 1.27 <= 1.30 | Rancher v2.9.x | — | 上游历史版本 |
| **Line A** | `k8s-1.31-1.33` | **1.31 – 1.33**（RKE2 / K3s） | Rancher v2.12.x：steve v0.6.40、client-go v0.33.4、Go 1.24.x | `v0.6.0-1.31` | **已发布** |
| Line B | `k8s-1.34-1.36` | 1.34 – 1.36（RKE2 / K3s） | Rancher v2.15.2：steve v0.9.23、client-go v0.36.4、Go 1.26.x | `v0.6.1-1.34`（规划） | 规划中，尚未发布 |

**请按集群的 Kubernetes 版本选择对应的构建线与 tag**，跨大版本混用可能因 client-go 与 apiserver 不兼容而失败。

Line A 已在单节点 K3s `v1.32.12+k3s1`（linux/arm64）完成端到端验证：节点 / Pod / Namespace 查询、Dashboard 与静态资源均正常。

## 快速开始

### 下载

从 [Releases](https://github.com/Nick11Show/kube-explorer/releases) 下载对应架构的静态二进制：

- `kube-explorer-linux-amd64`
- `kube-explorer-linux-arm64`

两个产物都内嵌了 Dashboard 与 api-ui，运行时**不需要任何外部 UI 文件或后端服务**。

### 运行

仅监听 HTTP（关闭 HTTPS）：

```bash
./kube-explorer --kubeconfig=/path/to/kubeconfig \
  --http-listen-port=8989 --https-listen-port=0
```

浏览器打开 `http://<host>:8989`。

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

## 部署到 Kubernetes

`deploy/kubectl/` 提供了可直接 apply 的清单（Deployment / Service / ServiceAccount / ClusterRoleBinding）：

```bash
kubectl apply -f deploy/kubectl/serviceAccount.yaml \
              -f deploy/kubectl/clusterRoleBinding.yaml \
              -f deploy/kubectl/service.yaml \
              -f deploy/kubectl/deployment.yaml
```

部署前请按需修改 `deployment.yaml`：

- `spec.template.spec.containers[].image`：改成**你自己可访问的镜像仓库**（清单中当前是一个私有 registry 示例地址，仅用于本地集群验证）。
- `imagePullPolicy`：内网 registry 不可从节点访问时，可预先把镜像 load 进节点 containerd 并设为 `IfNotPresent`。

设计要点：

- `hostNetwork: true` + hostPath 挂载 `/etc/rancher/k3s/k3s.yaml`，使 Pod 内能访问 kubeconfig 中的 `https://127.0.0.1:6443`，同时把 8989 暴露在节点上。
- `strategy: Recreate`：单节点 hostNetwork 场景下避免新旧 Pod 抢占 8989 端口导致滚动更新死锁。
- `tolerations`：容忍 `node-role.kubernetes.io/control-plane`（RKE2/K3s）、`node-role.kubernetes.io/controlplane`（RKE1 旧键）与 `cattle.io/os` 污点，保证可调度到默认打了污点的 control-plane 节点。

## 容器镜像

- CI 发版产物推送到 **ghcr.io**：`ghcr.io/<owner>/kube-explorer:<tag>`，为 `linux/amd64` + `linux/arm64` 多架构 manifest。
- 内网离线部署可把同一镜像 retag 后推到自建 registry（例如 Harbor），再在 `deployment.yaml` 中引用。
- 运行时镜像基于 `FROM scratch`：二进制为 `CGO_ENABLED=0` 全静态链接，镜像内只有 `/usr/bin/kube-explorer`，无系统包依赖、构建过程无需联网安装基础软件。

```bash
# 示例：把验证过的镜像推送到自建 registry
nerdctl tag ghcr.io/<owner>/kube-explorer:v0.6.0-1.31 \
  <your-registry>/library/kube-explorer:v0.6.0-1.31
nerdctl push <your-registry>/library/kube-explorer:v0.6.0-1.31
```

## 从源码构建

前置：Go 工具链（版本由 `go.mod` 决定，Line A 为 1.24.x）。

**关键：UI 资产不入库**（`.gitignore` 排除 `internal/ui/ui/`），构建带内嵌 UI 的二进制前必须先下载资产，且必须加 `-tags embed`：

```bash
# 1) 下载 Dashboard (Rancher v2.12.0 embedded build) + api-ui (1.1.11)
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
| `.github/workflows/push.yaml` | 分支 push、按线 tag push | 下载 UI 资产 → 双架构静态二进制 → 多架构镜像推 ghcr.io → tag 时创建 draft GitHub release |

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
- （Line A 新增）RKE2 / K3s kubeconfig 自动探测、`--token` 认证、control-plane 污点容忍、Dashboard + api-ui 内嵌进静态二进制

## 相关项目

- [kube-explorer-ui](https://github.com/cnrancher/kube-explorer-ui)
- [autok3s](https://github.com/cnrancher/autok3s)
- [api-ui](https://github.com/rancher/api-ui)
