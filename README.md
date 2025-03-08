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

### 安装

#### 方式一：直接安装（推荐）
```bash
go install github.com/naokij/omnibak@latest
```

#### 方式二：从源码编译
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

mysql:
  enabled: true
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
```

### 使用
```bash
# 运行备份
omnibak -c config.yaml

# 查看帮助
omnibak -h

# 定时任务示例（每天2AM）
0 2 * * * /usr/local/bin/omnibak -c /etc/omnibak/config.yaml >> /var/log/omnibak.log
```

## 许可证
MIT License