# Esurfing-go-webui

> 基于 [Esurfing-go](https://github.com/DreamwareN/Esurfing-go) 的带 Web 管理界面的天翼校园网认证客户端

## 简介

Esurfing-go-webui 在原版 Esurfing-go 的基础上，增加了一个基于浏览器的 Web 管理界面，支持通过网页远程管理多个网络接口的认证登录，实时查看日志和连接状态，适用于需要管理多网卡、多账号校园网认证的场景。

## 区别

本项目在原版 [Esurfing-go](https://github.com/DreamwareN/Esurfing-go) 的基础上新增了以下功能：

- **Web 管理界面** - 基于浏览器的可视化管理面板
- **RESTful API** - 完整的 HTTP API 接口，便于自动化集成
- **实时日志流 (SSE)** - 通过 Server-Sent Events 实时推送运行日志
- **多网卡独立管理** - 每个网络接口独立配置和运行
- **启用/禁用单个接口** - 灵活控制各接口的连接状态
- **手动登录/登出** - 支持手动触发指定接口的认证操作
- **系统网卡自动发现** - 自动检测本机可用网络接口
- **全局设置持久化** - Web 端口、访问模式等全局配置持久化保存
- **Deb / Opkg 包** - 提供 systemd 服务文件和 OpenWrt init 脚本，开箱即用

## 特性

- 基于浏览器的 Web 管理面板，响应式设计
- 支持同时管理多个网络接口的认证会话
- 每个接口独立配置：用户名、密码、网卡绑定、DNS、检查间隔等
- 启用/禁用、手动连接/断开单个接口
- 服务启动时自动连接已启用的接口
- 实时日志流 (Server-Sent Events)
- 自动检测本机网络接口
- 全局设置：Web 端口、访问模式、默认最大重试次数
- 配置自动持久化到 JSON 文件
- 支持 Linux / Windows / macOS / OpenWrt
- 提供 systemd 服务文件和 OpenWrt init 脚本
- 单二进制文件，前端资源内嵌，零外部依赖

## 快速开始

### 1. 获取程序

**方式一：下载预编译二进制**

从 [Releases](https://github.com/DreamwareN/Esurfing-go-webui/releases) 页面下载对应平台的二进制文件。

**方式二：从源码编译**

```bash
git clone https://github.com/DreamwareN/Esurfing-go-webui.git
cd Esurfing-go-webui
go build -trimpath -ldflags="-s -w" -o esurfing-go-webui .
```

要求 Go 1.25.3 或更高版本。

### 2. 运行

```bash
# 使用默认配置 (端口 8080)
./esurfing-go-webui

# 指定配置文件
./esurfing-go-webui -c /path/to/config.json

# 指定 Web 端口
./esurfing-go-webui -p 9090
```

启动后在浏览器中打开 `http://<设备IP>:8080` 即可访问管理界面。

## CLI 参数

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `-c` | 配置文件路径 | `esurfing_data/config.json` |
| `-p` | Web 服务端口 (覆盖配置文件中的设置) | 使用配置文件中的值，未设置则为 `8080` |

## 配置文件

配置文件为 JSON 格式，结构如下：

```json
{
  "interfaces": [
    {
      "interface": "wan0",
      "enabled": true,
      "username": "10001234",
      "password": "12345678",
      "check_interval": 10000,
      "retry_interval": 10000,
      "bind_interface": "eth0",
      "dns_address": "119.29.29.29:53",
      "max_retries": 5
    },
    {
      "interface": "wan1",
      "enabled": false,
      "username": "10005678",
      "password": "87654321",
      "check_interval": 10000,
      "retry_interval": 10000,
      "bind_interface": "eth1",
      "dns_address": "",
      "max_retries": 5
    }
  ],
  "settings": {
    "web_port": 8080,
    "default_max_retries": 5,
    "access_mode": "all"
  }
}
```

### 接口配置字段

| 字段 | 说明 |
|------|------|
| `interface` | 接口名称标识 |
| `enabled` | 是否在服务启动时自动连接 |
| `username` | 认证用户名 |
| `password` | 认证密码 |
| `check_interval` | 网络状态检查间隔 (毫秒)，默认 `10000` |
| `retry_interval` | 登录失败重试间隔 (毫秒)，默认 `10000`，负值表示不重试 |
| `bind_interface` | 绑定的网卡设备名称 (如 `eth0`、`enp0s1`、`wan0`)，留空使用系统默认 |
| `dns_address` | 自定义 DNS 服务器地址 (需带端口号，如 `119.29.29.29:53`)，一般留空即可 |
| `max_retries` | 最大重试次数 |

### 全局设置字段

| 字段 | 说明 | 可选值 |
|------|------|--------|
| `web_port` | Web 服务端口 | 正整数，默认 `8080` |
| `default_max_retries` | 新建接口时的默认最大重试次数 | 正整数，默认 `5` |
| `access_mode` | 访问控制模式 | `all` (允许所有)、`lan` (仅局域网)、`localhost` (仅本地) |

## Web 管理界面

### 列表面板

- 以卡片形式展示所有已配置的网络接口
- 每个接口显示：名称、状态、用户 IP、上次登录时间、心跳信息、错误计数
- 状态类型：`online` (在线) / `offline` (离线) / `auth` (认证中) / `disabled` (已禁用)
- 支持的操作：启用/禁用、手动连接/断开、编辑、删除
- 列表每 15 秒自动刷新

### 更多设置

- 查看和修改全局设置 (Web 端口、最大重试次数、访问模式)
- 实时日志查看，支持 SSE 推送

## API 接口

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/interfaces` | 获取所有接口列表 |
| `POST` | `/api/interfaces` | 添加新接口 |
| `PUT` | `/api/interfaces/{name}` | 更新接口配置 |
| `DELETE` | `/api/interfaces/{name}` | 删除接口 |
| `POST` | `/api/interfaces/{name}/enable` | 启用接口 |
| `DELETE` | `/api/interfaces/{name}/disable` | 禁用接口 |
| `POST` | `/api/interfaces/{name}/login` | 手动触发登录 |
| `DELETE` | `/api/interfaces/{name}/logout` | 手动登出 |
| `GET` | `/api/settings` | 获取全局设置 |
| `PUT` | `/api/settings` | 更新全局设置 |
| `GET` | `/api/logs` | 获取全部日志 |
| `GET` | `/api/logs/stream` | SSE 实时日志流 |
| `GET` | `/api/system/interfaces` | 获取本机网络接口列表 |

## 部署

### Linux

适用于 Debian / Ubuntu 等使用 systemd 的发行版：

```bash
# 安装 Deb 包 (从 Releases 下载)
sudo dpkg -i esurfing-go-webui_*_amd64.deb

# 安装后会自动注册并启动服务
# 查看服务状态
sudo systemctl status esurfing-go-webui

# 手动管理服务
sudo systemctl start esurfing-go-webui
sudo systemctl stop esurfing-go-webui
sudo systemctl restart esurfing-go-webui

# 查看日志
sudo journalctl -u esurfing-go-webui -f
```

Deb 包安装后文件位置：
- 二进制文件：`/usr/bin/esurfing-go-webui`
- 配置文件：`/etc/esurfing-webui/config.json`
- 数据目录：`/var/lib/esurfing-webui`
- 服务文件：`/lib/systemd/system/esurfing-go-webui.service`

### OpenWrt

```bash
# 安装 ipk 包 (从 Releases 下载)
opkg install esurfing-go-webui_*_*.ipk

# 服务会自动启动
# 手动管理服务
/etc/init.d/esurfing-go-webui start
/etc/init.d/esurfing-go-webui stop
/etc/init.d/esurfing-go-webui restart
```

### Windows / macOS

直接运行下载的二进制文件即可：

```cmd
:: Windows
esurfing-go-webui.exe -c config.json
```

```bash
# macOS
./esurfing-go-webui -c config.json
```

## 支持的平台

| 操作系统 | 架构 |
|----------|------|
| Linux | amd64, arm (v7), arm64, mips, mipsle, mips64, mips64le, riscv64 |
| Windows | amd64, arm64 |
| macOS | amd64, arm64 |

Deb 包支持：amd64, armhf, arm64
Opkg 包支持：x86_64, arm_cortex-a9, aarch64, mips_24kc, mipsel_24kc

## 项目结构

```
Esurfing-go-webui/
├── main.go            # 程序入口，HTTP 服务器，前端嵌入，生命周期管理
├── client.go          # 认证客户端核心逻辑，网络检查与心跳维持
├── auth.go            # 认证流程：获取学校信息、Ticket、登录
├── cipher.go          # 加密算法实现 (AES/3DES/SM4/ZUC/XTEA)
├── request.go         # HTTP 请求构造，自定义请求头与校验
├── xml.go             # XML 报文生成与解析
├── utils.go           # 工具函数：网卡 IP 获取、DNS 解析、随机数生成
├── api.go             # RESTful API 路由与处理函数
├── manager.go         # 接口管理器：客户端生命周期、配置持久化
├── loghub.go          # 日志中心：存储与 SSE 广播
├── interfaces.go      # 系统网络接口发现
├── go.mod             # Go 模块依赖
├── web/
│   └── index.html     # 单页前端界面 (HTML/CSS/JS 一体)
├── data/
│   └── config.json    # 示例配置文件
├── packaging/
│   ├── deb/
│   │   └── esurfing-go-webui.service   # systemd 服务文件
│   └── opkg/
│       └── esurfing-go-webui           # OpenWrt init 脚本
└── .github/
    └── workflows/
        └── build.yml   # CI/CD 自动构建与打包
```

## 认证流程

```
网络检测 (HTTP 204 检查)
    │
    ├─ 204 → 网络正常，等待下次检查
    │
    └─ 302 → 捕获重定向，进入认证流程
                │
                ├─ 1. GetSchoolInfo()    获取学校/区域信息
                ├─ 2. GetEConfig()       获取认证服务器配置
                ├─ 3. GetUserAndAcIP()   解析用户 IP 与 AC IP
                ├─ 4. GetAlgoId()        获取加密算法 ID
                ├─ 5. NewCipher()        选择加密算法
                ├─ 6. GetTicket()        获取认证票据
                └─ 7. Login()            执行登录
                        │
                        └─ 登录成功 → 启动心跳保活
```

## 依赖

| 依赖 | 用途 |
|------|------|
| [github.com/emmansun/gmsm](https://github.com/emmansun/gmsm) | SM4 / ZUC 国密算法实现 |
| [github.com/google/uuid](https://github.com/google/uuid) | 生成 Client-ID |

## DNS 说明

当系统使用 DNS-over-HTTPS (DoH) 时，未认证状态下 DoH 无法工作，会导致认证所需域名解析失败。此时需要在接口配置中手动指定 `dns_address`，一般填写 DHCP 获取的 DNS 地址即可（需带端口号，如 `119.29.29.29:53`）。

## 致谢

- [Esurfing-go](https://github.com/DreamwareN/Esurfing-go) - 原版天翼校园网认证客户端
- [Rsplwe/ESurfingDialer](https://github.com/nicai1900/ESurfingDialer) - 原始参考实现

## 许可证

[Apache License 2.0](LICENSE)
