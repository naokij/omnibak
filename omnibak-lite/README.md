# OmniBakLite

OmniBakLite是一个轻量级的备份工具，专为CentOS 5等旧版Linux系统设计，无需额外依赖，使用Python 2.4+即可运行。

## 功能特点

- MySQL数据库备份
- 文件系统备份
- WebDAV远程存储
- 自动清理旧备份
- 无需额外依赖，兼容Python 2.4+

## 安装

### 自动安装

使用安装脚本进行安装：

```bash
chmod +x install.sh
./install.sh
```

### 手动安装

1. 复制主程序到系统目录：

```bash
cp omnibak_lite.py /usr/local/bin/omnibaklite
chmod +x /usr/local/bin/omnibaklite
```

2. 创建配置目录：

```bash
mkdir -p /etc/omnibaklite
```

3. 复制示例配置文件：

```bash
cp config.yaml.example /etc/omnibaklite/config.yaml
```

## 配置

编辑配置文件 `/etc/omnibaklite/config.yaml`：

```yaml
mysql:
  enabled: true
  host: localhost
  port: 3306
  user: root
  password: your_password

files:
  enabled: true
  paths:
    - /path/to/backup:backup_name
    - /another/path:another_name

webdav:
  enabled: true
  url: http://your-webdav-server.com/path
  user: webdav_user
  password: webdav_password

retention:
  days: 7  # 保留备份的天数
```

### 配置说明

#### MySQL备份

- `enabled`: 是否启用MySQL备份
- `host`: MySQL服务器地址
- `port`: MySQL服务器端口
- `user`: MySQL用户名
- `password`: MySQL密码

#### 文件备份

- `enabled`: 是否启用文件备份
- `paths`: 要备份的文件路径列表，格式为 `源路径:备份名称`

#### WebDAV上传

- `enabled`: 是否启用WebDAV上传
- `url`: WebDAV服务器URL
- `user`: WebDAV用户名
- `password`: WebDAV密码

#### 保留策略

- `days`: 保留备份的天数，超过此天数的备份将被自动删除

## 使用方法

### 手动运行

```bash
omnibaklite -c /etc/omnibaklite/config.yaml
```

### 设置定时任务

编辑crontab：

```bash
crontab -e
```

添加定时任务（每天凌晨2点运行）：

```
0 2 * * * /usr/local/bin/omnibaklite -c /etc/omnibaklite/config.yaml
```

## 日志

默认情况下，日志会输出到标准输出。如需保存日志，可以重定向输出：

```bash
omnibaklite -c /etc/omnibaklite/config.yaml > /var/log/omnibaklite.log 2>&1
```

## 故障排除

### 调试模式

如需更详细的日志，可以修改脚本中的日志级别：

```python
logging.basicConfig(
    level=logging.DEBUG,  # 改为DEBUG级别
    format='%(asctime)s - %(levelname)s - %(message)s',
    stream=sys.stdout
)
```

### 常见问题

1. **MySQL连接失败**：检查MySQL配置和凭据
2. **文件备份失败**：检查文件路径和权限
3. **WebDAV上传失败**：检查WebDAV配置和网络连接

## 许可证

MIT License 