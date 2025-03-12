# OmniBak - 全能备份工具

[![Go Report Card](https://goreportcard.com/badge/github.com/naokij/omnibak)](https://goreportcard.com/report/github.com/naokij/omnibak)
![License](https://img.shields.io/badge/license-MIT-blue)

OmniBak 是一个用 Go 编写的统一备份解决方案，支持：

✅ MySQL 数据库备份  
✅ Docker 容器及数据卷备份  
✅ 文件/目录备份  
☁️ WebDAV 云存储上传  
⚡ 自动清理旧备份

## 功能特性

- 单配置文件管理所有备份任务
- 支持增量备份（通过 rclone）
- 日志记录和错误追踪
- 最小化系统依赖
- 跨平台支持（Linux/macOS）

## 快速开始

### 前提条件
- Go 1.21+
- rclone (配置好 WebDAV)
- Docker (如需备份容器)
- MySQL Client (如需备份数据库)

### 配置 rclone WebDAV

OmniBak 使用 rclone 上传备份文件到 WebDAV 存储。按照以下步骤配置：

1. **安装 rclone**（如果尚未安装）：
   ```bash
   # Debian/Ubuntu
   sudo apt install rclone
   
   # CentOS/RHEL
   sudo yum install rclone
   
   # macOS
   brew install rclone
   ```

2. **配置 WebDAV 远程存储**：
   ```bash
   rclone config
   ```
   
   按照交互提示：
   - 选择 `n` 创建新的远程存储
   - 名称：输入 `mywebdav`（与配置文件中 `webdav.remote` 对应）
   - 类型：选择 `webdav`
   - URL：输入您的 WebDAV 服务器地址（例如 `https://dav.example.com/remote.php/webdav/`）
   - 供应商：选择相应的 WebDAV 供应商（如 Nextcloud、Owncloud 等）
   - 用户名：输入 WebDAV 账户用户名
   - 密码：输入 WebDAV 账户密码
   - 高级配置：通常选择默认值
   - 确认配置：`y`

3. **测试 WebDAV 连接**：
   ```bash
   rclone lsd mywebdav:
   ```
   如果配置正确，将显示 WebDAV 根目录下的文件夹列表。

### 安装

#### 方式一：下载预编译的二进制文件（最简单）

您可以从[GitHub Releases](https://github.com/naokij/omnibak/releases)下载最新的预编译二进制文件：

- [Linux (amd64)](https://github.com/naokij/omnibak/releases/latest/download/omnibak-linux-amd64.tar.gz)
- [Linux (arm64)](https://github.com/naokij/omnibak/releases/latest/download/omnibak-linux-arm64.tar.gz)
- [Linux (x86 32位)](https://github.com/naokij/omnibak/releases/latest/download/omnibak-linux-386.tar.gz)
- [macOS (Intel)](https://github.com/naokij/omnibak/releases/latest/download/omnibak-darwin-amd64.tar.gz)
- [macOS (Apple Silicon)](https://github.com/naokij/omnibak/releases/latest/download/omnibak-darwin-arm64.tar.gz)

下载后解压并移动到系统PATH中：
```bash
# 例如 Linux/macOS
tar -xzvf omnibak-linux-amd64.tar.gz
sudo mv omnibak-linux-amd64 /usr/local/bin/omnibak
chmod +x /usr/local/bin/omnibak
```

#### 方式二：通过Go安装（推荐开发者使用）
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

### 配置示例
```yaml
# config.yaml
webdav:
  remote: "mywebdav"
  path: "backups"
  retention_days: 7
  rclone_config: ""    # 可选: 明确指定rclone配置文件路径

mysql:
  enabled: true
  host: "localhost"
  port: 3306
  user: "root"
  password: "secure_password"
  databases: ["all"]

docker:
  enabled: true
  containers: ["all"]
  backup_compose: true
  compose_paths: 
    - "/opt/apps/*/docker-compose.yml"
  backup_volumes: true

files:
  enabled: true
  paths:
    - "/etc/nginx:nginx_config"
    - "/var/www:web_content"

logging:
  level: "info"
  file: "/var/log/omnibak.log"
```

### 使用
```bash
# 运行备份
omnibak -c config.yaml

# 查看帮助
omnibak -h

# 定时任务示例（每天2AM）
0 2 * * * /usr/local/bin/omnibak -c /etc/omnibak/config.yaml >> /var/log/omnibak.log 2>&1
```

### 在Cron环境下使用（v0.1.4新增）

当通过cron计划任务运行omnibak时，可能会遇到rclone无法找到配置文件的问题。这是因为cron运行环境与用户登录环境不同。有两种解决方法：

#### 方法一：在配置文件中指定rclone配置路径（推荐）

```yaml
webdav:
  remote: "mywebdav"
  path: "backups"
  retention_days: 7
  rclone_config: "/home/用户名/.config/rclone/rclone.conf"  # 添加此行，使用绝对路径
```

#### 方法二：在crontab中设置HOME环境变量

```
# 在crontab中添加
HOME=/home/用户名
0 2 * * * /usr/local/bin/omnibak -c /etc/omnibak/config.yaml >> /var/log/omnibak.log 2>&1
```

#### 诊断和故障排除

从v0.1.4版本开始，omnibak包含自动环境诊断功能，会在启动时记录系统信息、环境变量和rclone配置状态。诊断信息会记录到日志中，便于排查问题。

您还可以使用包含的测试脚本来模拟cron环境：

```bash
# 下载测试脚本
wget https://raw.githubusercontent.com/naokij/omnibak/main/test-cron.sh
chmod +x test-cron.sh

# 编辑脚本中的配置文件路径
nano test-cron.sh

# 运行测试
./test-cron.sh
```

## 许可证
MIT License