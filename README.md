# warp

Go 版 Cloudflare Warp Endpoint 优选工具，用于批量探测 Cloudflare Warp 的 WireGuard Endpoint，筛选出延迟最低、丢包率最低的 `IP:端口`。

功能对齐 [CloudflareWarpSpeedTest](https://github.com/peanut996/CloudflareWarpSpeedTest) v1.3.2.2：内置 IPv4/IPv6 公网网段、固定端口列表、CIDR 展开、`-full` 全端口扫描、`-q` 快速模式（随机取 1000 条）、真实 WireGuard 握手确认、按丢包率→延迟排序、CSV 导出与控制台打印。

## 构建

```sh
go build -o warp .
./warp -h
./warp -v
```

## GitHub Actions 与版本号

分支推送和 Pull Request 会运行 `go vet ./...` 和 `go test ./...`。推送 `v主版本.次版本.补丁版本` 格式的 tag 后，CI 在检查通过后构建 Linux 和 macOS 的 `amd64`、`arm64` 压缩包，可在对应 Actions 运行的 **Artifacts** 下载。

```sh
git tag -a v1.0.0 -m "v1.0.0"
git push origin v1.0.0
```

CI 将 tag 注入二进制，因此该构建的 `./warp -v` 输出 `v1.0.0`；以后只需推送新 tag，无需修改 `main.go`。直接用 `go build` 构建时仍使用源码中的默认版本。

## 获取 Cloudflare IP 清单（ip.txt）

工具内置 `-update-ip`，可从 Cloudflare 官方端点 `https://www.cloudflare.com/ips-v4` / `ips-v6` 下载当前公开的 IP 段并合并写入文件（IPv4 在前、IPv6 在后，每行一个 CIDR），该文件可直接作为 `-f` 的输入：

```sh
./warp -update-ip ip.txt
./warp -f ip.txt -q=false   # 对官方全量段扫描
```

注意：官方列表是 Cloudflare **所有服务**（CDN/WARP/Zero Trust 等）的全量 anycast 段，并非“WARP 专用”清单；要用它筛 WARP 端点，需配合本工具的端口扫描 + WireGuard 握手确认。

默认运行会扫描内置的 8 个 IPv4 /24 网段（`162.159.192/193/195/204`、`188.114.96/97/98/99`）与 54 个固定端口，共 110592 条 `IP:端口`，`-q` 开启时随机取 1000 条。`-ipv6` 切换为内置 IPv6 网段（`2606:4700:d0::/48`、`2606:4700:d1::/48`）。

## 参数

| 参数 | 默认 | 说明 |
| --- | --- | --- |
| `-n` | 200 | 并发握手线程数（最大 1000） |
| `-t` | 10 | 单个 IP 的延迟测试次数 |
| `-q` | true | 快速模式，随机扫描 1000 条地址；`-q=false` 关闭 |
| `-ipv6` | false | 仅扫描内置 IPv6 网段 |
| `-tl` | 300 | 平均延迟上限（毫秒） |
| `-tll` | 0 | 平均延迟下限（毫秒） |
| `-tlr` | 1 | 丢包率上限（0.00~1.00，0 表示只要丢包就过滤） |
| `-p` | 10 | 展示结果条数，0 表示不展示 |
| `-f` | | IP 网段清单文件（每行一个，支持 CIDR） |
| `-ip` | | 逗号分隔的 IP/CIDR，例如 `1.1.1.1,2.2.2.2/24,2606:4700::/32` |
| `-o` | result.csv | 结果输出文件，空值 `-o ""` 不写文件 |
| `-pri` | | 自定义 WireGuard 私钥（Base64） |
| `-pub` | WARP 公钥 | 自定义对端公钥（Base64） |
| `-reserved` | | 自定义 reserved 字段，格式 `[0, 0, 0]`，需配合 `-pri` |
| `-full` | false | 扫描全部 1~10000 端口（而不是内置 54 端口） |
| `-update-ip` | | 下载 Cloudflare 官方 IP 段到指定文件并退出（生成 `ip.txt`） |
| `-h` / `-v` | | 帮助 / 版本 |

## 输出

- CSV 表头为 `IP:Port,Loss,Latency`；`Loss` 是握手失败比例，`Latency` 是成功握手平均耗时（毫秒）。
- 结果先按丢包率升序、再按延迟升序；`-tl`/`-tll`/`-tlr` 在排序后过滤，遇到第一个超出上限的结果即截断。
- 控制台默认打印前 `-p` 条；`-p 0` 只写 CSV 不打印表格。

## 用 wgcf 生成 WireGuard 配置并设置优选 IP

本工具测出的最优 `IP:端口`，最终要写进 WARP 的 WireGuard 配置里才能用。下面用 [wgcf](https://github.com/ViRb3/wgcf) 生成官方配置，再把优选结果替换进去。

### 1. 安装 wgcf

从 wgcf 的 GitHub Releases 下载对应平台二进制，解压得到 `wgcf` 可执行文件（也可用 `brew install wgcf` 等包管理器安装）。

### 2. 注册账号并生成配置

```sh
./wgcf register    # 首次运行注册一个新 WARP 账号，生成 wgcf-account.toml
                   # 若提示接受条款，加 --accept-tos
./wgcf generate    # 根据账号生成 wgcf-profile.conf（WireGuard 配置）
```

生成的 `wgcf-profile.conf` 结构如下（关键字段）：

```ini
[Interface]
PrivateKey = <你的私钥>
Address = 172.16.0.2/32, 2606:4700:110:xxxx::/128
DNS = 1.1.1.1, 1.0.0.1, 2606:4700:4700::1111, 2606:4700:4700::1001
MTU = 1280

[Peer]
PublicKey = bmXOC+F1FxEMF9dyiK2H5/1SUtzH0JuVo51h2wPfgyo=
AllowedIPs = 0.0.0.0/0, ::/0
Endpoint = engage.cloudflareclient.com:2408
```

注意：`[Peer] PublicKey` 就是 WARP 官方公钥，也是本工具内置的默认 `-pub`。所以本工具默认测的就是这套配置对应的端点，结果可直接使用。

### 3. 测出优选 IP:端口

```sh
./warp            # 默认快速模式，输出 result.csv（按丢包率→延迟排序）
head -2 result.csv  # 第 1 行是表头，第 2 行就是当前最优的 IP:端口
```

### 4. 把优选结果写入配置

把 `Endpoint` 从默认域名改成测出来的 `IP:端口`：

```sh
best=$(awk -F, 'NR==2{print $1}' result.csv)

# macOS
sed -i '' "s|^Endpoint = .*|Endpoint = $best|" wgcf-profile.conf
# Linux
sed -i "s|^Endpoint = .*|Endpoint = $best|" wgcf-profile.conf
```

改完后 `Endpoint` 形如：

```ini
Endpoint = 162.159.195.222:5279
```

### 5. 加载使用

- **Linux / 命令行**：`wg-quick up wgcf-profile.conf`（关闭用 `wg-quick down wgcf-profile.conf`）
- **图形客户端**：把改好的 `wgcf-profile.conf` 导入 WireGuard 官方客户端、TunSafe、Wgcf 相关工具等任意支持 WireGuard 配置的客户端即可。

> 提示：优选结果是针对“当前网络环境”测出来的，网络变化后需要重跑 `./warp` 重新优选。

## 测试

```sh
go test ./...
```

单元测试覆盖 IP/CIDR 展开、排序与过滤、CSV 格式、握手包结构与回环握手测量；集成测试仅在本机回环发送数据，不访问公网端点。

## 致谢

本项目包含源自 [CloudflareWarpSpeedTest](https://github.com/peanut996/CloudflareWarpSpeedTest) 的代码，感谢原作者 [peanut996](https://github.com/peanut996) 和上游贡献者。本项目沿用上游的 [GPL v3 许可证](LICENSE)。
