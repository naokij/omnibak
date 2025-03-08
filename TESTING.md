# OmniBak 测试文档

## 测试环境要求
- 隔离的测试环境（推荐使用 Docker）
- 测试用 MySQL 实例
- 测试用 Docker 容器
- 可访问的 WebDAV 服务

## 测试用例矩阵

### 基础功能测试
| 测试场景          | 测试步骤                                                                 | 预期结果                     |
|-------------------|--------------------------------------------------------------------------|------------------------------|
| 配置文件加载      | 1. 提供错误格式的配置文件<br>2. 提供不存在的文件路径                     | 程序应报错并退出             |
| 空运行测试        | `./omnibak -c config.yaml --dry-run`                                     | 显示备份计划但不执行实际操作 |

### MySQL 备份测试
1. **全库备份测试**
   ```bash
   mysql -e "CREATE DATABASE testdb; CREATE TABLE testdb.data (id INT); INSERT INTO testdb.data VALUES (1);"
   ./omnibak
   ```
   - 检查 `/tmp/backup/mysql/all_databases_*.sql` 是否包含 testdb

2. **指定数据库备份**
   ```yaml
   databases: ["testdb"]
   ```
   - 确认只有 testdb 的备份文件生成

### Docker 备份测试
1. **容器导出测试**
   ```bash
   docker run -d --name test-container alpine tail -f /dev/null
   ./omnibak
   ```
   - 检查 `/tmp/backup/docker` 下是否有 test-container 的 tar 和 json 文件

2. **数据卷备份**
   ```bash
   docker volume create test-vol
   docker run -v test-vol:/data --rm alpine sh -c "echo 'test' > /data/file"
   ./omnibak
   ```
   - 验证 `volumes/test-vol_*.tar.gz` 包含 file 文件

### 文件备份测试
1. **多目录备份**
   ```yaml
   paths:
     - "/tmp/src1:backup1"
     - "/tmp/src2:backup2"
   ```
   - 创建测试文件后验证备份包内容

### WebDAV 集成测试
1. **上传验证**
   ```bash
   rclone ls webdav:backups/$(date +%Y%m%d)
   ```
   - 确认远程存在备份文件

2. **清理策略测试**
   ```yaml
   retention_days: 1
   ```
   - 创建多个日期的测试备份，验证只保留最近1天

### 测试多配置文件
```bash
# 测试自定义配置路径
./omnibak -c /etc/omnibak/config.prod.yaml

# 测试默认配置
./omnibak  # 自动加载当前目录的 config.yaml
```

| 测试场景         | 预期结果                     |
|------------------|------------------------------|
| 不指定 -c 参数   | 使用当前目录的 config.yaml   |
| 指定不存在的路径 | 报错并显示帮助信息           |

## 压力测试
```bash
# 生成测试数据
mkdir -p /stress-test
dd if=/dev/urandom of=/stress-test/largefile bs=1M count=1024

# 运行备份
time ./omnibak -c config.stress.yaml
```
- 监控内存使用（应 < 100MB）
- 检查备份完整性

## 错误处理测试
| 错误类型          | 触发方式                          | 预期处理                     |
|-------------------|-----------------------------------|------------------------------|
| MySQL 连接失败    | 使用错误密码                      | 记录错误并跳过 MySQL 备份    |
| Docker 服务未启动 | 停止 Docker 服务                  | 记录错误并跳过 Docker 备份   |
| 磁盘空间不足      | 使用小容量临时目录                | 捕获错误并终止               |
| WebDAV 认证失败   | 配置错误的 rclone 凭证            | 上传失败并记录错误           |

## 性能指标
| 指标               | 预期值          |
|--------------------|-----------------|
| CPU 使用率         | < 15% (平均)   |
| 内存占用           | < 100 MB       |
| 10GB 文件备份时间  | < 5 分钟       |
| 恢复时间 (1GB 数据)| < 2 分钟       |
