# OmniBak - 全能备份工具

[![Go Report Card](https://goreportcard.com/badge/github.com/naokij/omnibak)](https://goreportcard.com/report/github.com/naokij/omnibak)
![License](https://img.shields.io/badge/license-MIT-blue)

OmniBak 是一个用 Go 编写的统一备份解决方案，支持多种备份类型和云存储上传。

## 目录

- [功能特性](#功能特性)
- [快速开始](#快速开始)
  - [前提条件](#前提条件)
  - [安装](#安装)
  - [配置](#配置)
  - [使用](#使用)
- [WebDAV 配置](#webdav-配置)
- [定时任务](#定时任务)
  - [Cron 环境配置](#cron-环境配置)
  - [故障排除](#故障排除)
- [许可证](#许可证)

## 功能特性

✅ **MySQL 数据库备份**：自动压缩的数据库备份  
✅ **Docker 容器及数据卷备份**：保存容器配置和数据  
✅ **文件/目录备份**：灵活指定需要备份的路径  
☁️ **WebDAV 云存储上传**：通过 rclone 支持多种云存储  
⚡ **自动清理旧备份**：基于保留策略自动管理备份存储空间  

- 单配置文件管理所有备份任务
- 日志记录和错误追踪
- 最小化系统依赖
- 跨平台支持（Linux/macOS）

## 快速开始

### 前提条件

- Go 1.21+
- rclone (用于 WebDAV 云存储)
- Docker (如需备份容器)
- MySQL Client (如需备份数据库)

### 安装

#### 方式一：下载预编译的二进制文件（推荐）

从 [GitHub Releases](https://github.com/naokij/omnibak/releases) 下载最新版本：

- [Linux (amd64)](https://github.com/naokij/omnibak/releases/latest/download/omnibak-linux-amd64.tar.gz)
- [Linux (arm64)](https://github.com/naokij/omnibak/releases/latest/download/omnibak-linux-arm64.tar.gz)
- [Linux (x86 32位)](https://github.com/naokij/omnibak/releases/latest/download/omnibak-linux-386.tar.gz)
- [macOS (Intel)](https://github.com/naokij/omnibak/releases/latest/download/omnibak-darwin-amd64.tar.gz)
- [macOS (Apple Silicon)](https://github.com/naokij/omnibak/releases/latest/download/omnibak-darwin-arm64.tar.gz)

安装步骤：
```bash
# 解压并安装到系统路径
tar -xzvf omnibak-linux-amd64.tar.gz
sudo mv omnibak-linux-amd64 /usr/local/bin/omnibak
chmod +x /usr/local/bin/omnibak
```

#### 方式二：通过 Go 安装（开发者）

```bash
go install github.com/naokij/omnibak@latest
```

#### 方式三：从源码编译

```bash
# 克隆仓库
git clone https://github.com/naokij/omnibak.git
cd omnibak

# 编译
go build -o omnibak

# 初始化配置
cp config.example.yaml config.yaml
```

### 配置

创建配置文件 `config.yaml`，示例如下：

```yaml
# WebDAV配置
webdav:
  remote: "mywebdav"        # rclone中配置的WebDAV远程名称
  path: "backups"           # 远程备份目录
  retention_days: 7         # 保留天数
  rclone_config: ""         # 可选: 指定rclone配置文件路径，解决cron环境问题

# MySQL备份配置
mysql:
  enabled: true             # 是否启用MySQL备份
  host: "localhost"         # MySQL主机地址
  port: 3306                # MySQL端口
  user: "root"              # MySQL用户名
  password: "password"      # MySQL密码
  databases:                # 要备份的数据库列表
    - "all"                 # 使用"all"备份所有数据库

# Docker备份配置
docker:
  enabled: true             # 是否启用Docker备份
  containers:               # 要备份的容器列表
    - "all"                 # 使用"all"备份所有容器
  backup_compose: true      # 是否备份docker-compose文件
  compose_paths:            # docker-compose文件路径（支持glob模式）
    - "/opt/apps/*/docker-compose.yml"
  backup_volumes: true      # 是否备份数据卷

# 文件备份配置
files:
  enabled: true             # 是否启用文件备份
  paths:                    # 要备份的文件/目录列表
    - "/etc/nginx:nginx_config"  # 格式：源路径:备份名称
    - "/var/www:web_content"

# 日志配置
logging:
  level: "info"             # 日志级别：debug, info, warn, error
  file: "/var/log/omnibak.log"  # 日志文件路径（同时会输出到标准输出）
```

### 使用

基本命令：

```bash
# 运行备份
omnibak -c config.yaml

# 查看帮助
omnibak -h
```

## WebDAV 配置

OmniBak 使用 rclone 上传备份文件到 WebDAV 存储。

### 1. 安装 rclone

```bash
# Debian/Ubuntu
sudo apt install rclone

# CentOS/RHEL
sudo yum install rclone

# macOS
brew install rclone
```

### 2. 配置 WebDAV 远程存储

```bash
rclone config
```

按照交互提示操作：
- 选择 `n` 创建新的远程存储
- 名称：输入 `mywebdav`（与配置文件中的 `remote` 对应）
- 类型：选择 `webdav`
- URL：输入您的 WebDAV 服务器地址
- 供应商：选择相应的 WebDAV 供应商（如 Nextcloud）
- 用户名和密码：输入您的账户信息
- 完成配置并确认

### 3. 测试连接

```bash
rclone lsd mywebdav:
```

## 定时任务

### 设置 Cron 定时任务

添加到 crontab：

```bash
# 每天凌晨2点运行（移除了重定向，使用程序内部日志）
0 2 * * * /usr/local/bin/omnibak -c /etc/omnibak/config.yaml
```

### Cron 环境配置

在 cron 环境下运行时，可能会遇到 rclone 无法找到配置文件的问题，有两种解决方案：

#### 方案一：指定 rclone 配置文件路径（推荐）

在 `config.yaml` 中添加：

```yaml
webdav:
  # ... 其他配置 ...
  rclone_config: "/home/用户名/.config/rclone/rclone.conf"  # 使用绝对路径
```

#### 方案二：设置 HOME 环境变量

在 crontab 中设置：

```
HOME=/home/用户名
0 2 * * * /usr/local/bin/omnibak -c /etc/omnibak/config.yaml
```

### 故障排除

OmniBak v0.1.4+ 包含自动环境诊断功能，会在启动时记录系统信息到日志。

可使用测试脚本模拟 cron 环境：

```bash
# 下载并运行测试脚本
wget https://raw.githubusercontent.com/naokij/omnibak/main/test-cron.sh
chmod +x test-cron.sh
./test-cron.sh
```

## 许可证

MIT License