#!/bin/bash
# 测试omnibak在cron环境下的运行情况

# 设置基本环境变量
export PATH=/usr/local/sbin:/usr/local/bin:/usr/bin:/bin
export HOME=${HOME:-$(eval echo ~$(whoami))}

# 记录测试开始
echo "===== 开始测试 omnibak 在 cron 环境下的运行情况 ====="
echo "当前时间: $(date)"
echo "当前用户: $(whoami)"
echo "HOME目录: $HOME"
echo "当前目录: $(pwd)"

# 显示rclone信息
echo "rclone版本:"
rclone version 2>&1 | head -n 1

echo "rclone配置文件:"
for config_path in ~/.config/rclone/rclone.conf ~/.rclone.conf /etc/rclone/rclone.conf; do
  if [ -f "$config_path" ]; then
    echo "  - 找到: $config_path ($(ls -la $config_path))"
  fi
done

# 配置文件路径 - 根据实际情况修改
CONFIG_PATH="/etc/omnibak/config.yaml"
if [ ! -f "$CONFIG_PATH" ]; then
  echo "配置文件不存在: $CONFIG_PATH"
  # 尝试查找其他可能的配置文件
  POSSIBLE_CONFIGS=$(find /etc -name "config.yaml" -o -name "omnibak*.yaml" 2>/dev/null)
  if [ -n "$POSSIBLE_CONFIGS" ]; then
    echo "发现可能的配置文件:"
    echo "$POSSIBLE_CONFIGS"
    echo "请更新此脚本中的CONFIG_PATH变量"
  fi
  exit 1
fi

echo "使用配置文件: $CONFIG_PATH"

# 运行omnibak
echo "===== 执行 omnibak ====="
# 找到omnibak二进制文件
OMNIBAK_PATH=$(which omnibak 2>/dev/null)
if [ -z "$OMNIBAK_PATH" ]; then
  echo "未找到omnibak命令，请确保它已安装并在PATH中"
  exit 1
fi

echo "使用omnibak: $OMNIBAK_PATH"

# 创建临时日志文件
LOG_FILE="/tmp/omnibak-test-$(date +%Y%m%d%H%M%S).log"
echo "日志文件: $LOG_FILE"

# 执行omnibak并记录输出
$OMNIBAK_PATH -c "$CONFIG_PATH" > "$LOG_FILE" 2>&1
EXIT_CODE=$?

echo "omnibak退出代码: $EXIT_CODE"
echo "日志内容 (前100行):"
head -n 100 "$LOG_FILE"

# 完成
echo "===== 测试完成 ====="
echo "完整日志在: $LOG_FILE" 