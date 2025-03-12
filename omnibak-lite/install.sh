#!/bin/bash
# OmniBakLite 安装脚本

# 显示彩色输出
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}开始安装 OmniBakLite...${NC}"

# 检查是否为root用户
if [ "$(id -u)" != "0" ]; then
   echo -e "${RED}错误: 此脚本必须以root用户运行${NC}" 1>&2
   exit 1
fi

# 创建目录
echo -e "${YELLOW}创建必要的目录...${NC}"
mkdir -p /etc/omnibaklite
mkdir -p /tmp/omnibaklite_backups

# 复制主程序
echo -e "${YELLOW}安装主程序...${NC}"
cp omnibak_lite.py /usr/local/bin/omnibaklite
chmod +x /usr/local/bin/omnibaklite

# 创建配置文件
if [ ! -f "/etc/omnibaklite/config.yaml" ]; then
    echo -e "${YELLOW}创建配置文件...${NC}"
    cat > /etc/omnibaklite/config.yaml << EOF
# OmniBakLite 配置文件

mysql:
  enabled: true
  host: localhost
  port: 3306
  user: root
  password: your_password

files:
  enabled: true
  paths:
    - /var/www/html:web_content
    - /etc/nginx:nginx_config

webdav:
  enabled: false
  url: http://your-webdav-server.com/backup
  user: webdav_user
  password: webdav_password

retention:
  days: 7  # 保留备份的天数
EOF
    echo -e "${GREEN}配置文件已创建: /etc/omnibaklite/config.yaml${NC}"
    echo -e "${YELLOW}请编辑配置文件以适应您的环境${NC}"
else
    echo -e "${YELLOW}配置文件已存在，跳过创建${NC}"
fi

# 设置权限
echo -e "${YELLOW}设置权限...${NC}"
chmod 600 /etc/omnibaklite/config.yaml

# 创建cron任务
echo -e "${YELLOW}是否要创建定时任务? [y/N]${NC}"
read -r create_cron

if [[ "$create_cron" =~ ^[Yy]$ ]]; then
    echo -e "${YELLOW}创建定时任务...${NC}"
    # 检查是否已存在cron任务
    crontab -l | grep -q "omnibaklite" && {
        echo -e "${YELLOW}定时任务已存在，跳过创建${NC}"
    } || {
        # 添加到crontab
        (crontab -l 2>/dev/null; echo "0 2 * * * /usr/local/bin/omnibaklite -c /etc/omnibaklite/config.yaml > /var/log/omnibaklite.log 2>&1") | crontab -
        echo -e "${GREEN}定时任务已创建，每天凌晨2点运行${NC}"
    }
fi

echo -e "${GREEN}OmniBakLite 安装完成!${NC}"
echo -e "${YELLOW}请编辑 /etc/omnibaklite/config.yaml 配置文件${NC}"
echo -e "${YELLOW}运行命令: omnibaklite -c /etc/omnibaklite/config.yaml${NC}" 