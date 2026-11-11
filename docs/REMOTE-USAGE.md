# 在其它机器上使用 kube-explorer（Tailscale / Mac mini 操作手册）

> 目标读者：想在**另一台机器**上使用本项目的 Line A / Line B，典型场景是同一 Tailscale tailnet 里的 `nickmac-mini`。
> 本文的地址、证书、命令均在 2026-11-12 的真实环境实测过。IP 若变动，请用第 1 节的自查命令重新确认。
> 通用参数与更完整的说明见 [使用说明](./USAGE.md)，版本矩阵见 [README](../README.md)。

---

## 0. 先弄清三件事

1. **kube-explorer 不是集群，是"集群浏览器"**。它必须拿到一份能连到某个 Kubernetes 集群的 kubeconfig，才能展示节点 / Pod / 资源。它自己不存数据、不需要数据库。
2. **没有 macOS 原生二进制**。发布产物只有 `linux/amd64`、`linux/arm64` 的静态二进制和 OCI 镜像。在 Mac 上要用**容器运行时**（Docker Desktop / Rancher Desktop / nerdctl / OrbStack）跑镜像；Linux 主机可以直接跑二进制。
3. **按集群的 Kubernetes 版本选构建线**，两条线的 client-go / steve 不同，混用会失败：

| 构建线 | 目标集群 | Harbor 镜像 tag | GitHub 二进制 |
| ------ | -------- | --------------- | ------------- |
| **Line A** `v0.6.0-1.31` | K8s **1.31 – 1.33**（K3s / RKE2） | `100.116.110.97:8443/library/kube-explorer:v0.6.0-1.31` | [release v0.6.0-1.31](https://github.com/Nick11Show/kube-explorer/releases/tag/v0.6.0-1.31) |
| **Line B** `v0.6.1-1.34` | K8s **1.34 – 1.36**（K3s / RKE2） | `100.116.110.97:8443/library/kube-explorer:v0.6.1-1.34` | [release v0.6.1-1.34](https://github.com/Nick11Show/kube-explorer/releases/tag/v0.6.1-1.34) |

两个 tag 都是**已合成的多架构 manifest**（`linux/amd64` + `linux/arm64`），容器运行时会自动挑平台：Apple Silicon Mac mini → arm64，Intel → amd64，不用手选。部分版本另有 `-amd64` / `-arm64` 后缀的单架构 tag，以 Harbor 仓库实际列表为准。

---

## 1. 你的 tailnet 现状（实测）

| 设备 | Tailscale IP | 角色 | 已经跑着什么 |
| ---- | ------------ | ---- | ------------ |
| `nick-mbp`（这台 Mac Pro） | `100.101.50.23` | Rancher Desktop + K3s `v1.32.12+k3s1` | Line A，`http://127.0.0.1:8989`，**仅本机可达** |
| `nickmac-mini` | `100.101.242.95` | 目标 Mac mini | 无（22 端口未开，命令需在本机终端执行） |
| `sledlab01` | `100.116.110.97` | 单节点 RKE2 `v1.34.7+rke2r1` + Harbor | Line B UI `http://100.116.110.97:8989`（**tailnet 内可达**）、Harbor `https://100.116.110.97:8443` |

在 Mac mini 上先跑这几条做体检：

```bash
tailscale status                       # 应能看到三台设备、同一账号
tailscale ping sledlab01               # 实测 18–104ms 可达
uname -m                               # arm64 = Apple Silicon；x86_64 = Intel
curl --noproxy '*' -sI http://100.116.110.97:8989/ | head -3   # 期望 302 -> /dashboard/
```

> ⚠️ **代理坑（实测）**：这台 Mac Pro 设了 `HTTP_PROXY/HTTPS_PROXY=http://127.0.0.1:64847`。
> 直接 `curl http://100.116.110.97:8989/...` 会走代理并**超时**，加 `--noproxy '*'` 立刻返回 302。
> 若 Mac mini 也配了代理，请在 shell 里设置 `export NO_PROXY=100.64.0.0/10,localhost,127.0.0.1`（Tailscale 网段是 `100.64.0.0/10`），或在浏览器代理设置里排除该网段，否则浏览器也打不开内网 IP。

---

## 2. 方案 A：零安装，直接打开已部署好的 UI（最快，推荐先用这个）

`nickmac-mini` 上**什么都不用装**，浏览器打开：

```
http://100.116.110.97:8989
```

根路径会 302 跳到 `/dashboard/`。这个实例是部署在 `sledlab01` 的 `kube-explorer` 命名空间里的 Line B（`hostNetwork: true`），管理的是 **RKE2 v1.34.7 集群**，由 Deployment 保证重启后自动恢复。

实测（从 Mac Pro 走 tailnet）：

```bash
curl --noproxy '*' -s -o /dev/null -w '%{http_code}\n' http://100.116.110.97:8989/            # 302
curl --noproxy '*' -s -o /dev/null -w '%{http_code}\n' http://100.116.110.97:8989/dashboard/  # 200
curl --noproxy '*' -s http://100.116.110.97:8989/v1/nodes | python3 -c \
  'import sys,json; d=json.load(sys.stdin); print("count=",d["count"], [x["id"] for x in d["data"]])'
# count= 1 ['sledlab01.suse.site']
```

**只要你想管理的是这台 RKE2 集群，方案 A 就够了**，不用碰 Harbor、证书、kubeconfig。

> 打不开时按顺序查：① `tailscale status` 里 `sledlab01` 是否在线；② Tailscale ACL 是否允许两台设备互访；③ 在 sledlab01 上 `kubectl -n kube-explorer get pods` 是否为 Running。

---

## 3. 方案 B：在 Mac mini 本机跑容器，连接远程 RKE2（推荐，可管理任意集群）

适合：想在 Mac mini 上有一个**独立的** kube-explorer，或者以后要连别的集群。整套命令已在 Mac Pro 上用同一镜像实测通过（容器内 `/v1/nodes` 读出 `sledlab01.suse.site`，`/v1/namespaces` 78KB、`/v1/services` 190KB 全 200）。

### 3.1 处理自签 Harbor（必做）

Harbor 的证书是 `CN=harbor.rancher.com`、由私有 `cattle-ca` 签发，**SAN 里只有 DNS 名、没有 IP SAN**；而且 registry 返回的认证 realm 也是 `https://harbor.rancher.com:8443/service/token`。所以直接用 IP 拉取必然报 `x509: cannot validate certificate for 100.116.110.97 because it doesn't contain any IP SANs`。

`library` 项目是公开的，**匿名拉取即可，不需要 `docker login`**（实测：401 → 取匿名 token → 200）。

**第 1 步：给域名加一条 hosts 映射**（幂等，可随时删掉该行回滚）：

```bash
sudo sh -c 'grep -q "100.116.110.97 harbor.rancher.com" /etc/hosts || printf "\n# kube-explorer private lab\n100.116.110.97 harbor.rancher.com sledlab01\n" >> /etc/hosts'
sudo dscacheutil -flushcache; sudo killall -HUP mDNSResponder
```

这一行同时解决两件事：Harbor 的 token realm 域名能解析，以及 apiserver 证书 SAN 里的 `sledlab01` 能解析（见 3.2）。

**第 2 步：让容器运行时跳过自签证书校验**。任选其一：

- **Docker Desktop**：Settings → Docker Engine，在 JSON 里加 `"insecure-registries": ["100.116.110.97:8443", "harbor.rancher.com:8443"]`，Apply & Restart。
- **Rancher Desktop / nerdctl**：命令行可直接带参数：
  ```bash
  nerdctl --insecure-registry pull 100.116.110.97:8443/library/kube-explorer:v0.6.1-1.34
  ```
  或在 VM 内的 `/etc/containerd/certs.d/100.116.110.97:8443/hosts.toml` 写 `skip_verify = true`（RKE2 节点上就是这么修的，无需重启 containerd）。
- **不想跳过校验的更严格做法**：把 CA 导入系统信任，然后统一用域名 `harbor.rancher.com:8443` 拉取：
  ```bash
  scp root@100.116.110.97:/data/harbor/cert/cacerts.pem /tmp/harbor-ca.pem
  sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain /tmp/harbor-ca.pem
  ```
  注意 Docker Desktop 的 daemon 跑在 Linux VM 里，可能仍需要在 daemon 侧配置 insecure registry；Rancher Desktop 同理。

验证拉取（Apple Silicon 会自动选 arm64）：

```bash
docker pull 100.116.110.97:8443/library/kube-explorer:v0.6.1-1.34
# 或：nerdctl --insecure-registry pull ...
```

### 3.2 准备 kubeconfig

从 RKE2 节点取管理员 kubeconfig（**这等价于集群管理员凭据**）：

```bash
mkdir -p ~/.kube
ssh root@100.116.110.97 'cat /etc/rancher/rke2/rke2.yaml' > ~/.kube/rke2-remote.yaml
chmod 600 ~/.kube/rke2-remote.yaml
```

原始文件的 `server` 是 `https://127.0.0.1:6443`，在 Mac mini 上必须改。两种改法：

| 写法 | server | 是否需要 `--insecure-skip-tls-verify` | 说明 |
| ---- | ------ | ------------------------------------- | ---- |
| **① 推荐：保留 TLS 校验** | `https://sledlab01:6443` | **不需要** | apiserver 证书 SAN 含 DNS `sledlab01`，配合 3.1 的 hosts 映射；实测容器日志 0 条证书错误 |
| ② 简单：跳过校验 | `https://100.116.110.97:6443` | **需要** | apiserver 证书 SAN **不含** tailnet IP，用 IP 必然校验失败 |

改成写法 ①（只用 Python 标准库，macOS 自带）：

```bash
python3 -c 'import os; p=os.path.expanduser("~/.kube/rke2-remote.yaml"); s=open(p).read(); assert "127.0.0.1:6443" in s, "server 已被改过？"; open(p,"w").write(s.replace("https://127.0.0.1:6443","https://sledlab01:6443")); print("ok")'
grep -n "server:" ~/.kube/rke2-remote.yaml        # 应显示 https://sledlab01:6443
```

若没有 Python，用 `sed -i '' 's|https://127.0.0.1:6443|https://sledlab01:6443|' ~/.kube/rke2-remote.yaml`（macOS 的 `sed -i` 需要那个空引号）。

验证 TLS 链路 —— 从 kubeconfig 里取出 CA，返回 **401 是正确结果**（代表 TLS 握手和证书校验都通过，只是没带客户端凭据）：

```bash
python3 -c 'import base64,os,re; s=open(os.path.expanduser("~/.kube/rke2-remote.yaml")).read(); open("/tmp/rke2-ca.crt","wb").write(base64.b64decode(re.search(r"certificate-authority-data:\s*(\S+)",s).group(1)))'
curl --noproxy '*' -s -o /dev/null -w '%{http_code}\n' --cacert /tmp/rke2-ca.crt https://sledlab01:6443/version
```

> 对比：把 URL 换成 `https://100.116.110.97:6443` 会直接 TLS 失败（IP 不在证书 SAN 里），这正是写法 ② 必须加 `--insecure-skip-tls-verify` 的原因。

> Mac mini 若还不能免密 `ssh root@100.116.110.97`：先从已能登录的 Mac Pro 用安全通道（AirDrop / `scp` / 加密传输）把 `rke2-remote.yaml` 拷过去，或把 Mac mini 的公钥追加到 sledlab01 的 `~/.ssh/authorized_keys`。**不要用明文聊天工具传这个文件。**

### 3.3 启动容器

```bash
docker run -d --name kube-explorer \
  -p 8989:8989 \
  --add-host sledlab01:100.116.110.97 \
  -v "$HOME/.kube/rke2-remote.yaml:/etc/kubeconfig:ro" \
  100.116.110.97:8443/library/kube-explorer:v0.6.1-1.34 \
  --kubeconfig /etc/kubeconfig \
  --http-listen-port=8989 --https-listen-port=0
```

要点：

- 镜像 `ENTRYPOINT` 就是 kube-explorer，参数直接跟在镜像名后面。
- `--add-host sledlab01:...` 让**容器内**也能解析 apiserver 的 DNS SAN（不依赖宿主机 hosts，实测有效）。用写法 ② 时这行可省略，但要加 `--insecure-skip-tls-verify`。
- **容器内无法自动探测 kubeconfig**（容器里没有 `/etc/rancher/rke2/rke2.yaml`），必须显式 `--kubeconfig`。
- 用写法 ② 的命令差异：kubeconfig 里 server 改成 `https://100.116.110.97:6443`，并在参数里加 `--insecure-skip-tls-verify`。
- 端口冲突时把 `-p` 改成 `-p 18989:8989`，浏览器相应换端口。
- Rancher Desktop / nerdctl 用户把 `docker` 换成 `nerdctl --namespace k8s.io`（参数一致，实测通过）。

### 3.4 验证

```bash
docker logs kube-explorer | tail -20
curl --noproxy '*' -s http://127.0.0.1:8989/v1/nodes \
  | python3 -c 'import sys,json; d=json.load(sys.stdin); print("count=",d["count"], [x["id"] for x in d["data"]])'
open http://127.0.0.1:8989          # macOS 打开浏览器，根路径跳转到 /dashboard/
```

> 冷启动第一次请求 `/v1/nodes` 可能返回 **404**（informer 还在同步），等 3–10 秒再请求即 200，这是正常现象，实测第 2 次就恢复。

停止 / 删除：`docker rm -f kube-explorer`。

---

## 4. 方案 C：Mac mini 自己也有集群时

如果 Mac mini 上装了 Rancher Desktop / k3s，**最省事的是把 kube-explorer 直接部署进那个集群**，而不是在宿主机跑容器：

```bash
kubectl apply -f deploy/kubectl/namespace.yaml \
              -f deploy/kubectl/serviceAccount.yaml \
              -f deploy/kubectl/service.yaml \
              -f deploy/kubectl/deployment.yaml
kubectl -n kube-explorer rollout status deployment/kube-explorer
kubectl -n kube-explorer port-forward svc/kube-explorer 8989:8989
# 浏览器打开 http://localhost:8989
```

部署前按需改三处：

1. **镜像 tag**：按 Mac mini 集群的 K8s 版本选 `v0.6.0-1.31`（1.31–1.33）或 `v0.6.1-1.34`（1.34–1.36）。仓库 Line B 分支的清单默认是后者。
2. **hostPath**：Line B 清单挂的是 `/etc/rancher/rke2/rke2.yaml`（RKE2）；K3s 节点要改成 `/etc/rancher/k3s/k3s.yaml`，`mountPath` 一起改。若节点上的 containerd 不信任自签 Harbor，先按 3.1 的思路配置 insecure registry，或把镜像预先 `load` 进节点并设 `imagePullPolicy: IfNotPresent`。
3. **命名空间**：清单用自己的 `kube-explorer` 命名空间，不会动现有对象；`clusterRoleBinding.yaml` 是集群级对象，不需要时可以不 apply（挂载节点 kubeconfig 已足够）。

> 如果坚持"宿主机容器 + 本地 kubeconfig"，注意 `~/.kube/config` 里的 `server: https://127.0.0.1:6443` 在**容器内指向容器自己**，必然连不上。要把 server 改成 `https://host.docker.internal:6443` 并加 `--add-host host.docker.internal:host-gateway`，通常还要配 `--insecure-skip-tls-verify`（证书 SAN 一般不含这个域名）。这条是按容器网络通用原理给出的做法，本次环境未实测该组合。

---

## 5. 方案 D：从 Mac mini 管理这台 Mac Pro 上的 K3s（Line A）

**实测结论**：Mac Pro 上的 Line A 只监听在 `127.0.0.1:8989`（Rancher Desktop 的端口转发），从 tailnet IP `100.101.50.23:8989` 访问会超时；而 `sledlab01` 的 8989 因为是 `hostNetwork` Pod，tailnet 内可直接访问。所以 Mac mini **不能**直接打开 `http://100.101.50.23:8989`。

另外，Mac Pro 的 k3s apiserver 在 Lima VM 内（`https://127.0.0.1:6443`），Mac mini 拿不到可达的 kubeconfig。因此**推荐转发 UI，而不是在 Mac mini 上跑 Line A 去连 Mac Pro 的集群**。

可行的三条路：

**① 临时端口转发（实测通过，无需安装任何软件）** — 在 **Mac Pro** 上执行：

```bash
python3 - <<'PY'
import socket, threading
def pipe(a, b):
    try:
        while True:
            d = a.recv(65536)
            if not d: break
            b.sendall(d)
    except Exception: pass
    finally:
        try: b.shutdown(socket.SHUT_WR)
        except Exception: pass
s = socket.socket(); s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
s.bind(("0.0.0.0", 18989)); s.listen(20)
print("forwarding 0.0.0.0:18989 -> 127.0.0.1:8989 (Ctrl-C to stop)")
while True:
    c, _ = s.accept()
    u = socket.create_connection(("127.0.0.1", 8989))
    threading.Thread(target=pipe, args=(c, u), daemon=True).start()
    threading.Thread(target=pipe, args=(u, c), daemon=True).start()
PY
```

然后 Mac mini 打开 `http://100.101.50.23:18989`。实测该地址返回 302（正常）。用完 `Ctrl-C` 即恢复，无残留。等价的更短写法是装 `socat`：`socat TCP-LISTEN:18989,bind=100.101.50.23,fork,reuseaddr TCP:127.0.0.1:8989`。

**② SSH 隧道** — 需先在 Mac Pro 的「系统设置 → 通用 → 共享 → 远程登录」打开 SSH（当前 22 端口未开），然后在 Mac mini 上：

```bash
ssh -N -L 8989:127.0.0.1:8989 <MacPro用户名>@100.101.50.23
# 浏览器打开 http://localhost:8989
```

**③ Tailscale Serve** — 官方语法是 `tailscale serve --bg 8989`，之后访问 `https://nick-mbp.tail6fe5e6.ts.net`。但**本 tailnet 当前未启用 Serve**（实测提示 `Serve is not enabled on your tailnet`），需先在 Tailscale 管理控制台开启，本次未启用、也未留下任何配置。

---

## 6. Linux 主机：直接跑二进制

```bash
TAG=v0.6.1-1.34        # 或 v0.6.0-1.31
ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
curl -fL -o kube-explorer \
  "https://github.com/Nick11Show/kube-explorer/releases/download/${TAG}/kube-explorer-linux-${ARCH}"
chmod +x kube-explorer
./kube-explorer --version

# 在 RKE2 / K3s 节点上：不传 --kubeconfig 会自动探测
sudo ./kube-explorer --http-listen-port=8989 --https-listen-port=0
# 在普通 Linux 主机上：显式指向 kubeconfig
./kube-explorer --kubeconfig ~/.kube/config --http-listen-port=8989 --https-listen-port=0
```

二进制内嵌了 Dashboard 与 api-ui，**不需要任何外部 UI 文件或后端服务**。连远程集群时同样适用第 3.2 节的两种 kubeconfig 写法。

---

## 7. 常用参数与验证命令

| 参数 | 用途 |
| ---- | ---- |
| `--kubeconfig <path>` | 显式指定 kubeconfig（容器内必填） |
| `--http-listen-port=8989 --https-listen-port=0` | 只开 HTTP，关闭 HTTPS |
| `--insecure-skip-tls-verify` | kubeconfig 用 IP 作 server 时跳过 apiserver 证书校验 |
| `--token <token>` | 用 Bearer token 覆盖 kubeconfig 里的用户凭据 |
| `--enable-header-auth` | **仅 Line B**：启用 `Impersonate-User/Group` 请求头身份透传 |

```bash
# 一键体检（把 U 换成你的实例地址）
U=http://100.116.110.97:8989
for p in / /dashboard/ /v1/nodes /v1/namespaces /v1/pods /v1/services; do
  printf "%-20s " "$p"; curl --noproxy '*' -s --max-time 30 -o /dev/null -w '%{http_code}\n' "$U$p"
done
```

---

## 8. 故障排查

| 现象 | 原因 | 处理 |
| ---- | ---- | ---- |
| `curl` 访问 `100.x` 超时，但 `nc -z` 说端口开 | 走了 `HTTP_PROXY` | 加 `--noproxy '*'`，或 `export NO_PROXY=100.64.0.0/10,localhost,127.0.0.1`；浏览器也要排除代理 |
| `x509: cannot validate certificate for 100.116.110.97 ... no IP SANs` | Harbor 自签证书只有 DNS SAN | 按 3.1 配 insecure registry，或导入 CA 后用 `harbor.rancher.com:8443` 域名拉取 |
| `unauthorized` / token 请求卡住 | Harbor 的认证 realm 是 `https://harbor.rancher.com:8443/...` | `/etc/hosts` 加 `100.116.110.97 harbor.rancher.com`；匿名拉取本身是允许的，不必 `docker login` |
| 连 apiserver 报 `x509` / `connection refused` | 证书 SAN 不含 tailnet IP；或 `127.0.0.1` 指向容器自身 | 用 `https://sledlab01:6443` + hosts/`--add-host`（推荐），或加 `--insecure-skip-tls-verify` |
| 首次 `/v1/nodes` 返回 404 | informer 冷启动尚未同步 | 等 3–10 秒重试，实测第 2 次即 200 |
| Line B 的 `/v1/namespaces`、`/v1/pods` **无限挂起** | client-go v0.36 默认 `WatchListClient` 在 K8s 1.34 拿不到 initial-events-end bookmark | 用 `v0.6.1-1.34` 及之后的版本（已在代码里默认关闭并加单测）；旧镜像可临时设环境变量 `KUBE_FEATURE_WatchListClient=false` |
| 浏览器打开是空白 / 404 | 访问了根路径或缓存了旧版静态资源 | 直接打开 `/dashboard/`，强制刷新；确认 `/dashboard/js/*`、`/api-ui/ui.min.js` 返回 200 |
| Mac mini 打不开 sledlab01 的 UI | Tailscale ACL / 节点离线 / Pod 未 Running | `tailscale status`、`tailscale ping sledlab01`、在节点上 `kubectl -n kube-explorer get pods` |

---

## 9. 安全提醒（重要）

- **kube-explorer 的 HTTP 端口默认没有任何鉴权**，而且 steve 支持写操作。任何能访问 8989 的人都能以 kubeconfig 的身份操作集群。请只在 tailnet 内使用，用 Tailscale ACL 收紧可访问范围，**不要**用 `tailscale funnel` 暴露到公网，也不要把 8989 映射到路由器。
- `rke2-remote.yaml` 是**集群管理员凭据**：`chmod 600`、不要提交到 git、不要放进共享目录。需要最小权限时，可创建只读 ServiceAccount 并用 `--token`（会新增集群对象，按需评估）。
- Harbor 的 `library` 项目是公开只读的，匿名拉取足够；不要把管理员账号写进任何脚本或文档。
- 用完随手关：`docker rm -f kube-explorer`；方案 D 的转发脚本 `Ctrl-C` 即停。
- 方案 3.1 的 hosts 映射与 insecure registry 只影响本机容器运行时，需要回滚时删掉 `/etc/hosts` 那一行、去掉 Docker Engine 的 `insecure-registries` 即可。

---

## 10. 本次实测记录（便于复现核对）

- RKE2 `v1.34.7+rke2r1` 单节点，最终镜像 UI/API 全绿：`/`、`/dashboard/`、JS、favicon、api-ui 均 200；`/v1/nodes|namespaces|pods|services|configmaps` 均 200，读出 **1 节点 / 41 namespace / 1063 pods / 61 service**。
- 容器内挂载远程 kubeconfig 的两种写法都实测通过：IP + `--insecure-skip-tls-verify`、以及 `sledlab01` DNS SAN + kubeconfig 自带 CA（**不加** insecure 参数，日志 0 条证书错误）。
- Harbor 两个正式 tag `v0.6.0-1.31`（digest `sha256:e60d21ee…`）、`v0.6.1-1.34`（digest `sha256:bf488b56…`）都是 `manifest.list.v2`，各含 `linux/amd64` + `linux/arm64`。
- `tailscale serve` 在本 tailnet 未启用；Mac Pro 的 `100.101.50.23:8989` 直连不通，但 `0.0.0.0:18989 → 127.0.0.1:8989` 转发后 tailnet 内返回 302。
