#!/usr/bin/env python
# -*- coding: utf-8 -*-

import os
import sys
import subprocess
import time
import logging
from optparse import OptionParser

# 设置默认编码为utf-8
reload(sys)
sys.setdefaultencoding('utf-8')

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
            return False

        upload_success = True
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
                    upload_success = False
        
        return upload_success
    
    def cleanup_temp_files(self):
        """清理临时备份文件"""
        logger.info("开始清理临时备份文件...")
        
        if os.path.exists(self.backup_dir):
            for root, _, files in os.walk(self.backup_dir):
                for file in files:
                    file_path = os.path.join(root, file)
                    try:
                        os.remove(file_path)
                        logger.info("已删除临时备份文件: %s" % file_path)
                    except Exception, e:
                        logger.error("删除临时文件失败: %s, 错误: %s" % (file_path, str(e)))
    
    def cleanup_old_webdav_backups(self):
        """清理WebDAV上的旧备份文件"""
        if not self.config.get('webdav', {}).get('enabled', False):
            return
            
        # 获取保留天数，默认为7天
        retention_days = self.config.get('retention', {}).get('days', 7)
        # 确保retention_days是整数
        try:
            retention_days = int(retention_days)
        except (ValueError, TypeError):
            logger.warning("保留天数配置无效，使用默认值7天")
            retention_days = 7
            
        logger.info("开始清理WebDAV上超过 %d 天的旧备份..." % retention_days)
        
        # 计算截止时间（YYYYMMDD格式）
        cutoff_date = time.strftime("%Y%m%d", time.localtime(time.time() - (retention_days * 86400)))
        
        # 获取WebDAV上的文件列表
        list_cmd = 'curl -u %s:%s -X PROPFIND %s' % (
            self.config['webdav']['user'],
            self.config['webdav']['password'],
            self.config['webdav']['url']
        )
        
        # 使用临时文件存储PROPFIND结果
        temp_file = os.path.join(self.backup_dir, "webdav_list.xml")
        
        try:
            os.system("%s > %s" % (list_cmd, temp_file))
            
            # 解析XML获取文件列表
            import xml.etree.ElementTree as ET
            if os.path.exists(temp_file):
                tree = ET.parse(temp_file)
                root = tree.getroot()
                
                # 查找所有文件名
                for href in root.findall(".//{DAV:}href"):
                    file_url = href.text
                    if file_url:
                        # 提取文件名
                        file_name = os.path.basename(file_url.rstrip('/'))
                        
                        # 检查文件名是否包含日期戳
                        if '_' in file_name:
                            try:
                                # 尝试提取日期部分（假设格式为name_YYYYMMDDHHMMSS.ext）
                                date_part = file_name.split('_')[1].split('.')[0][:8]  # 提取YYYYMMDD部分
                                
                                # 如果日期早于截止日期，则删除文件
                                if date_part < cutoff_date:
                                    delete_cmd = 'curl -u %s:%s -X DELETE %s/%s' % (
                                        self.config['webdav']['user'],
                                        self.config['webdav']['password'],
                                        self.config['webdav']['url'],
                                        file_name
                                    )
                                    retcode = subprocess.call(delete_cmd, shell=True)
                                    if retcode == 0:
                                        logger.info("已删除WebDAV上的旧备份文件: %s" % file_name)
                                    else:
                                        logger.error("删除WebDAV文件失败: %s, 返回码: %d" % (file_name, retcode))
                            except (IndexError, ValueError):
                                # 如果无法解析日期，则跳过
                                logger.warning("无法解析文件名中的日期: %s" % file_name)
        except Exception, e:
            logger.error("清理WebDAV文件时发生错误: %s" % str(e))
        finally:
            # 确保在所有情况下都删除临时文件
            if os.path.exists(temp_file):
                try:
                    os.remove(temp_file)
                    logger.debug("已删除临时文件: %s" % temp_file)
                except Exception, e:
                    logger.error("删除临时文件失败: %s, 错误: %s" % (temp_file, str(e)))

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
    
    # 上传到WebDAV，并根据上传结果决定是否清理
    upload_success = bak.upload_to_webdav()
    
    # 如果上传成功，清理临时文件和WebDAV上的旧文件
    if upload_success:
        logger.info("上传成功，开始清理...")
        bak.cleanup_temp_files()  # 清理临时文件
        bak.cleanup_old_webdav_backups()  # 根据保留策略清理WebDAV上的旧文件
        logger.info("===== 备份任务完成 =====")
        logger.info("备份已成功上传到WebDAV，临时文件已清理，旧备份已根据保留策略清理")
    else:
        logger.warning("上传失败，跳过清理步骤")
        logger.warning("===== 备份任务部分完成 =====")
        logger.warning("备份已创建但上传失败，临时文件已保留，请检查WebDAV配置后手动处理")

if __name__ == "__main__":
    main() 