<h1 align="center">🚀 订阅检测转换工具</h1>

<p align="center">
	<a href="https://github.com/rongrong13/sub-test-max/releases"><img src="https://img.shields.io/github/v/release/rongrong13/sub-test-max?style=flat-square&include_prereleases&label=version" /></a>
	<a href="https://github.com/rongrong13/sub-test-max/releases"><img src="https://img.shields.io/github/downloads/rongrong13/sub-test-max/total.svg?style=flat-square" /></a>
  <a href="https://hub.docker.com/r/beck8/subs-check/tags"><img src="https://img.shields.io/docker/pulls/beck8/subs-check" /></a>
	<a href="https://github.com/rongrong13/sub-test-max/issues"><img src="https://img.shields.io/github/issues-raw/rongrong13/sub-test-max.svg?style=flat-square&label=issues" /></a>
	<a href="https://github.com/rongrong13/sub-test-max/graphs/contributors"><img src="https://img.shields.io/github/contributors/rongrong13/sub-test-max?style=flat-square" /></a>
	<a href="https://github.com/rongrong13/sub-test-max/blob/master/LICENSE"><img src="https://img.shields.io/github/license/rongrong13/sub-test-max?style=flat-square" /></a>
</p>

---

> **✨ 修复逻辑、简化操作、增加功能、节省内存、一键启动无需配置**

> **⚠️ 注意：** 功能更新频繁，请查看最新的[配置文件](https://github.com/rongrong13/sub-test-max/blob/master/config/config.example.yaml)以获取最新功能。  
> **⚠️ 注意：** 如果想要查看功能更新，可以参照 [示例配置提交历史](https://github.com/rongrong13/sub-test-max/commits/master/config/config.example.yaml),这里有变动说明有更功能/逻辑更新

> **🙏 鸣谢**：本项目的基座与核心功能来自原作者 [beck-8](https://github.com/beck-8) 的 [subs-check](https://github.com/beck-8/subs-check)（订阅检测转换工具）。在此深表感谢！

## ✨ 功能特性

- **🔗 订阅合并**
- **🔍 节点可用性检测**
- **🗑️ 节点去重**
- **⏱️ 节点测速**
- **🎬 流媒体平台解锁检测**
- **✏️ 节点重命名**
- **🔄 任意格式订阅转换**
- **🔔 支持100+通知渠道**
- **🖥️ WEB 控制面板**
- **⏰ 支持 Crontab 表达式**
- **🖥️ 多平台支持**

## 🎬 流媒体与 IP 风险检测

> 本项目内嵌整合了 **MediaUnlockTest**(https://github.com/HsukqiLee/MediaUnlockTest) 的检测逻辑作为流媒体解锁检测引擎;IP 风险检测改用 **ip-api.com** 的免费启发式方案(替代原先易被反爬拦截的 scamalytics)。

- **流媒体解锁检测**:逐节点用 Chrome 指纹(tls_client)探测 Netflix / Disney+ / YouTube Premium / OpenAI ChatGPT / Anthropic Claude / Google Gemini / Spotify 等 100+ 服务,结果只把解锁成功的服务标注到节点名上(失败不显示)。
- **IP 风险检测**:通过节点出口 IP 查询 ip-api.com(免费、无反爬),按 hosting / proxy / 云厂商 ASN / 移动 / 住宅等启发式规则打分(0-100%),并输出机房/住宅与代理/原生属性。
- **节点命名格式**(同 IP-Stream-Checker):原名·风险%·GM✓(sg)·NF✓(us)·GPT✓【emoji 机房|代理】
  例:🇭🇰 HK-01·61%·GM✓(sg)·NF✓(us)·GPT✓【🟠 机房|代理】
- platforms 配置项沿用原有平台名即可(netflix/disney/openai/gemini/claude/spotify/youtube/tiktok/iprisk),也支持直接填写 MediaUnlockTest 服务名(如 Amazon Prime Video)以启用更多服务。

## 🛠️ 部署与使用 
> 首次运行会在当前目录生成默认配置文件。

### 🪜 代理设置（可选）
<details>
  <summary>展开查看</summary>

如果拉取非Github订阅速度慢，可使用通用的 HTTP_PROXY HTTPS_PROXY 环境变量加快速度；此变量不会影响节点测试速度
```bash
# HTTP 代理示例
export HTTP_PROXY=http://username:password@192.168.1.1:7890
export HTTPS_PROXY=http://username:password@192.168.1.1:7890

# SOCKS5 代理示例
export HTTP_PROXY=socks5://username:password@192.168.1.1:7890
export HTTPS_PROXY=socks5://username:password@192.168.1.1:7890

# SOCKS5H 代理示例
export HTTP_PROXY=socks5h://username:password@192.168.1.1:7890
export HTTPS_PROXY=socks5h://username:password@192.168.1.1:7890
```
如果想加速github的链接，可使用网上公开的github proxy，或者使用下方自建测速地址处的worker.js自建加速
```
# Github Proxy，获取订阅使用，结尾要带的 /
# github-proxy: "https://ghfast.top/"
github-proxy: "https://custom-domain/raw/"
```

</details>

### 🌐 自建测速地址（可选）
<details>
  <summary>展开查看</summary>

> **⚠️ 注意：** 避免使用 Speedtest 或 Cloudflare 下载链接，因为部分节点会屏蔽测速网站。

1. 将 [worker.js](./doc/cloudflare/worker.js) 部署到 Cloudflare Workers。
2. 绑定自定义域名（避免被节点屏蔽）。
3. 在配置文件中设置 `speed-test-url` 为你的 Workers 地址：

```yaml
# 100MB
speed-test-url: https://custom-domain/speedtest?bytes=104857600
# 1GB
speed-test-url: https://custom-domain/speedtest?bytes=1073741824
```

</details>

### 🐳 Docker Compose 运行

> **⚠️ 注意：** 本项目暂未发布预构建镜像，请使用 Docker Compose **从源码构建**镜像后运行。

```bash
# 从当前目录的 Dockerfile 构建并启动
docker compose up -d --build
```

```yaml
version: "3"
services:
  subs-check:
    build: .
    image: subs-check:latest
    container_name: subs-check
    volumes:
      - ./config:/app/config
      - ./output:/app/output
    ports:
      - "8199:8199"
    environment:
      - TZ=Asia/Shanghai
      # - HTTP_PROXY=http://192.168.1.1:7890
      # - HTTPS_PROXY=http://192.168.1.1:7890
      # - API_KEY=subs-check
    restart: always
    network_mode: bridge
```
### 🖥️ 源码运行

```bash
go run . -f ./config/config.yaml
```

## 🔔 通知渠道配置（可选）
<details>
  <summary>展开查看</summary>

> **📦 支持 100+ 通知渠道**，通过 [Apprise](https://github.com/caronc/apprise) 发送通知。

### 🌐 Vercel 部署

1. 点击[**此处**](https://vercel.com/new/clone?repository-url=https://github.com/beck-8/apprise_vercel)部署 Apprise。
2. 部署后获取 API 链接，如 `https://testapprise-beck8s-projects.vercel.app/notify`。
3. 建议为 Vercel 项目设置自定义域名`diydomain.com`（国内访问 Vercel 可能受限）。

### 🐳 Docker 部署

> **⚠️ 注意：** 不支持 arm/v7。

```bash
# 基础运行
docker run --name apprise -p 8000:8000 --restart always -d caronc/apprise:latest

# 使用代理运行
docker run --name apprise \
  -p 8000:8000 \
  -e HTTP_PROXY=http://192.168.1.1:7890 \
  -e HTTPS_PROXY=http://192.168.1.1:7890 \
  --restart always \
  -d caronc/apprise:latest
```

### 📝 配置文件中配置通知

```yaml
# 填写搭建的apprise API server 地址
# https://notify.xxxx.us.kg/notify
apprise-api-server: "https://diydomain.com/notify"
# 填写通知目标
# 支持100+ 个通知渠道，详细格式请参照 https://github.com/caronc/apprise
recipient-url: 
  # telegram格式：tgram://{bot_token}/{chat_id}
  # - tgram://xxxxxx/-1002149239223
  # 钉钉格式：dingtalk://{Secret}@{ApiKey}
  # - dingtalk://xxxxxx@xxxxxxx
# 自定义通知标题
notify-title: "🔔 节点状态更新"
```
</details>

## 💾 保存方法配置

> **⚠️ 注意：** 选择保存方法时，请更改 `save-method` 配置。

- **本地保存**：保存到 `./output` 文件夹。
- **R2**：保存到 Cloudflare R2 [配置方法](./doc/r2.md)。
- **Gist**：保存到 GitHub Gist [配置方法](./doc/gist.md)。
- **WebDAV**：保存到 WebDAV 服务器 [配置方法](./doc/webdav.md)。
- **S3**：保存到 S3 对象存储。

## 📲 订阅使用方法

> **💡 提示：** 检测完成后会生成 `all.yaml` 保存到 output 目录，并由 8199 端口提供文件服务。

**🚀 通用订阅（Clash 格式）**
```bash
http://127.0.0.1:8199/sub/all.yaml
```

## 🌐 内置端口说明
> 检测完成后会在 output 目录生成 `all.yaml`；output 目录中的所有文件会被 8199 端口提供文件服务。

| 服务地址 | 格式说明 | 来源说明 |
|---|---|---|
| `http://127.0.0.1:8199/sub/all.yaml` | Clash 格式节点 | 由本工具直接生成 |

## 🗺️ 架构图
<details>
  <summary>展开查看</summary>

```mermaid
graph TD
    A[订阅链接] -->|获取订阅链接| B[subs-check]
    subgraph subs-check 处理流程
        B -->|转成 YAML 格式| B1[节点去重]
        B1 -->|去除冗余节点| B2[测活]
        B2 -->|节点可用| B3[流媒体+重命名]
        B2 -->|节点不可用| X[丢弃]
        B3 -->|filter 通过| B4[测速]
        B3 -->|filter 不通过| X[丢弃]
        B4 -->|测速达标| B5[生成 all.yaml]
        B4 -->|测速不达标| X[丢弃]
    end
    B5 -->|保存到 output 目录| C[output 目录]
    C -->|保存到各位置| H1[R2/Gist/WebDAV/S3]
    H1 -->|存储完成| H2[发送消息通知]
    C -->|文件服务| F[8199 端口: /sub/all.yaml]
    B -->|Web 管理| G[8199 端口: /admin]
``` 

</details>

## 🙏 鸣谢
[beck-8](https://github.com/beck-8)（本项目的基座来自作者的 subs-check）、[cmliu](https://github.com/cmliu)、[bestruirui](https://github.com/bestruirui/BestSub)、[1password](https://1password.com/)、[ipinfo.io](https://ipinfo.io/)

## ⭐ Star History

[![Stargazers over time](https://starchart.cc/rongrong13/sub-test-max.svg?variant=adaptive)](https://starchart.cc/rongrong13/sub-test-max)

## ⚖️ 免责声明

本工具仅供学习和研究使用，使用者应自行承担风险并遵守相关法律法规。
