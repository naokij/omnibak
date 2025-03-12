#!/usr/bin/env python
# -*- coding: utf-8 -*-

import os
import sys
import subprocess
import time
import logging
from optparse import OptionParser

# 配置日志
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s',
    stream=sys.stdout
)
logger = logging.getLogger('OmniBakLite')

class OmniBakLite:
    def __init__(self, config):
        self.config = config
        self.backup_dir = "/tmp/omnibaklite_backups"
        self.timestamp = time.strftime("%Y%m%d%H%M%S")
        
        # 创建备份目录
        if not os.path.exists(self.backup_dir):
            os.makedirs(self.backup_dir)

    def backup_mysql(self):
        """备份MySQL数据库"""
        if not self.config.get('mysql', {}).get('enabled', False):
            return

        logger.info("开始备份MySQL数据库...")
        backup_file = os.path.join(self.backup_dir, "mysql_%s.sql.gz" % self.timestamp)
        
        try:
            # 转义密码中的特殊字符
            escaped_password = self.config['mysql']['password'].replace("'", "'\\''")
            
            # 测试MySQL连接
            test_cmd = "mysql -h %s -P %s -u %s -p'%s' -e 'SELECT 1'" % (
                self.config['mysql']['host'],
                str(self.config['mysql']['port']),
                self.config['mysql']['user'],
                escaped_password
            )
            retcode = subprocess.call(test_cmd, shell=True)
            if retcode != 0:
                raise Exception("无法连接到MySQL服务器，请检查配置")

            # 执行备份
            retcode = subprocess.call(' '.join([
                'mysqldump',
                '-h', self.config['mysql']['host'],
                '-P', str(self.config['mysql']['port']),
                '-u', self.config['mysql']['user'],
                "-p'%s'" % escaped_password,
                '--all-databases',
                '|', 'gzip', '>', backup_file
            ]), shell=True)
            if retcode != 0:
                raise Exception("MySQL备份失败，返回码: %d" % retcode)
            logger.info("MySQL备份成功: %s" % backup_file)
        except Exception, e:
            logger.error(str(e))

    def backup_files(self):
        """备份文件"""
        if not self.config.get('files', {}).get('enabled', False):
            return

        logger.info("开始备份文件...")
        for path in self.config['files']['paths']:
            src, dest = path.split(':')
            if not os.path.exists(src):
                logger.error("备份路径不存在: %s" % src)
                continue
                
            backup_file = os.path.join(self.backup_dir, "%s_%s.tar.gz" % (dest, self.timestamp))
            
            try:
                retcode = subprocess.call(' '.join([
                    'tar', '-czf', 
                    backup_file,
                    '-C', os.path.dirname(src),
                    os.path.basename(src)
                ]), shell=True)
                if retcode != 0:
                    raise Exception("文件备份失败，返回码: %d" % retcode)
                logger.info("文件备份成功: %s" % backup_file)
            except Exception, e:
                logger.error(str(e))

    def upload_to_webdav(self):
        """上传到WebDAV"""
        if not self.config.get('webdav', {}).get('enabled', False):
            return

        logger.info("开始上传到WebDAV...")
        # 测试WebDAV连接
        test_cmd = 'curl -u %s:%s -X PROPFIND %s' % (
            self.config['webdav']['user'],
            self.config['webdav']['password'],
            self.config['webdav']['url']
        )
        retcode = subprocess.call(test_cmd, shell=True)
        if retcode != 0:
            logger.error("无法连接到WebDAV服务器，请检查配置")
            return

        for root, _, files in os.walk(self.backup_dir):
            for file in files:
                file_path = os.path.join(root, file)
                try:
                    retcode = subprocess.call(' '.join([
                        'curl',
                        '-u', "%s:%s" % (self.config['webdav']['user'], self.config['webdav']['password']),
                        '-T', file_path,
                        "%s/%s" % (self.config['webdav']['url'], file)
                    ]), shell=True)
                    if retcode != 0:
                        raise Exception("上传失败，返回码: %d" % retcode)
                    logger.info("上传成功: %s" % file)
                except Exception, e:
                    logger.error(str(e))
    
    def cleanup_old_backups(self):
        """清理旧备份文件"""
        # 获取保留天数，默认为7天
        retention_days = self.config.get('retention', {}).get('days', 7)
        # 确保retention_days是整数
        try:
            retention_days = int(retention_days)
        except (ValueError, TypeError):
            logger.warning("保留天数配置无效，使用默认值7天")
            retention_days = 7
            
        logger.info("开始清理超过 %d 天的旧备份..." % retention_days)
        
        # 计算截止时间
        cutoff_time = time.time() - (retention_days * 86400)  # 86400秒 = 1天
        
        # 清理本地备份
        if os.path.exists(self.backup_dir):
            for root, _, files in os.walk(self.backup_dir):
                for file in files:
                    file_path = os.path.join(root, file)
                    file_mtime = os.path.getmtime(file_path)
                    if file_mtime < cutoff_time:
                        try:
                            os.remove(file_path)
                            logger.info("已删除旧备份文件: %s" % file_path)
                        except Exception, e:
                            logger.error("删除文件失败: %s, 错误: %s" % (file_path, str(e)))

def parse_config(config_file):
    """解析配置文件"""
    config = {
        'mysql': {'enabled': False},
        'files': {'enabled': False, 'paths': []},
        'webdav': {'enabled': False},
        'retention': {'days': 7}  # 默认保留7天
    }
    
    try:
        f = open(config_file, 'r')
        try:
            current_section = None
            current_subsection = None
            for line in f:
                line = line.strip()
                if not line or line.startswith('#'):
                    continue
                
                # 处理section
                if line.endswith(':'):
                    section_name = line[:-1].strip()
                    logger.debug("进入section: %s" % section_name)
                    
                    # 检查是否为顶级section
                    if section_name in config:
                        current_section = section_name
                        current_subsection = None
                    # 检查是否为子section
                    elif current_section and section_name == 'paths' and current_section == 'files':
                        current_subsection = 'paths'
                    else:
                        current_section = section_name
                        if current_section not in config:
                            config[current_section] = {}
                    continue
                
                # 处理键值对
                if ':' in line and not line.startswith('-'):
                    key, value = line.split(':', 1)
                    key = key.strip()
                    value = value.strip()
                    logger.debug("处理键值对: %s = %s" % (key, value))
                    
                    # 处理布尔值
                    if value.lower() in ('true', 'false'):
                        value = value.lower() == 'true'
                    # 处理数字
                    elif value.isdigit():
                        value = int(value)
                    # 尝试转换可能包含注释的数字
                    elif '#' in value:
                        num_part = value.split('#')[0].strip()
                        if num_part.isdigit():
                            value = int(num_part)
                    config[current_section][key] = value
                # 处理列表项
                elif line.startswith('-'):
                    list_item = line[1:].strip()
                    logger.debug("处理列表项: %s" % list_item)
                    
                    # 处理files.paths下的列表项
                    if current_section == 'files' and current_subsection == 'paths':
                        if ':' in list_item:
                            src, dest = list_item.split(':', 1)
                            config['files']['paths'].append("%s:%s" % (src.strip(), dest.strip()))
                        else:
                            config['files']['paths'].append(list_item)
                    # 处理files下的paths列表项（直接在files下的缩进列表）
                    elif current_section == 'files' and not current_subsection:
                        if ':' in list_item:
                            src, dest = list_item.split(':', 1)
                            config['files']['paths'].append("%s:%s" % (src.strip(), dest.strip()))
                        else:
                            config['files']['paths'].append(list_item)
                    # 处理其他section下的列表项
                    elif current_section:
                        if 'paths' not in config[current_section]:
                            config[current_section]['paths'] = []
                        if ':' in list_item:
                            src, dest = list_item.split(':', 1)
                            config[current_section]['paths'].append("%s:%s" % (src.strip(), dest.strip()))
                        else:
                            config[current_section]['paths'].append(list_item)
        finally:
            f.close()
    except IOError, e:
        logger.error("无法读取配置文件: %s" % str(e))
        sys.exit(1)
    except Exception, e:
        logger.error("解析配置文件时发生错误: %s" % str(e))
        sys.exit(1)
    
    # 确保retention.days是整数
    if 'retention' in config and 'days' in config['retention']:
        try:
            config['retention']['days'] = int(config['retention']['days'])
        except (ValueError, TypeError):
            logger.warning("保留天数配置无效，使用默认值7天")
            config['retention']['days'] = 7
    
    # 打印解析后的配置
    logger.debug("解析后的配置: %s" % config)
    return config

def main():
    parser = OptionParser(usage="usage: %prog -c CONFIG_FILE")
    parser.add_option("-c", "--config", dest="config_file",
                      help="配置文件路径", metavar="FILE")
    
    (options, args) = parser.parse_args()
    
    if not options.config_file:
        parser.error("必须指定配置文件")
    
    # 解析配置文件
    config = parse_config(options.config_file)
    
    # 检查必要配置项
    required_sections = ['mysql', 'files', 'webdav']
    for section in required_sections:
        if section not in config:
            logger.error("配置文件中缺少必要部分: %s" % section)
            sys.exit(1)
    
    # 检查files.paths
    if 'paths' not in config['files'] or not config['files']['paths']:
        logger.error("配置文件中缺少files.paths或paths为空")
        sys.exit(1)
    
    bak = OmniBakLite(config)
    bak.backup_mysql()
    bak.backup_files()
    bak.upload_to_webdav()
    bak.cleanup_old_backups()  # 清理旧备份

if __name__ == "__main__":
    main() 