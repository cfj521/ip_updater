# IP-Updater 部署指南

## 编译完成

✅ 已在Windows环境下成功交叉编译出Linux版本的可执行文件
✅ 已实现PRD第4条的所有功能改进：
  - 默认更新周期改为10分钟
  - 更新前原值与新值对比（相同则跳过）
  - IP格式检测和掩码保留功能
  - 优化文件操作减少锁死时间（适合NFS环境）

✅ DNS记录查询与比对功能完全实现：
  - 所有5个DNS提供商都支持GetRecord方法
  - 自动查询当前DNS记录值进行对比
  - 避免不必要的DNS更新操作

✅ 重要问题修复：
  - 移除首次运行检测和自动服务创建功能（避免权限问题）
  - 替换所有IP检测端点为中国大陆可访问服务
  - 实现DNS和文件更新分离定时器功能

## 部署包内容

- `ip_updater` - Linux AMD64可执行文件（约7MB）
- `ip_updater.service` - systemd服务配置文件
- `install.sh` - 自动安装脚本
- `uninstall.sh` - 卸载脚本
- `README.md` - 详细部署说明

## 快速部署

1. 将整个build目录复制到Linux服务器
2. 在服务器上执行：
   ```bash
   sudo ./install.sh
   ```
3. 编辑配置文件：
   ```bash
   sudo vi /etc/ip_updater/config.conf
   ```
4. 启动服务：
   ```bash
   sudo systemctl enable ip_updater
   sudo systemctl start ip_updater
   ```

## 功能说明

### 新增改进功能
- **智能更新**：更新前会检查当前值，相同则跳过更新
- **DNS记录比对**：自动查询DNS当前记录值并进行比较
- **分离定时器**：DNS检查60分钟，文件检查10分钟（可独立配置）
- **中国大陆优化**：使用IPIP.NET、花生壳、3322等国内IP检测服务
- **IP格式验证**：自动检测IP格式，不合法时记录警告
- **掩码保留**：自动保留CIDR掩码（如/24）
- **原子写入**：使用临时文件+重命名，减少文件锁死时间
- **简化部署**：移除运行时服务创建，通过安装脚本统一处理

### 支持的DNS提供商（全部支持记录查询对比）
- 阿里云 DNS
- 腾讯云 DNS
- 华为云 DNS
- Cloudflare DNS
- GoDaddy DNS

### 使用的IP检测服务（中国大陆优化）
- IPIP.NET (https://myip.ipip.net)
- 花生壳 (https://ddns.oray.com/checkip)
- 3322动态域名 (https://ip.3322.net)
- ip.cn (https://ip.cn/api/index?ip&type=0)

### 支持的配置文件格式
- JSON (.json)
- YAML (.yaml/.yml)
- TOML (.toml)
- INI (.ini)

## 配置说明

新版本支持分离的检查间隔配置：

```toml
# DNS更新检查间隔 (seconds, default: 3600 = 60 minutes)
dns_check_interval = 3600

# 文件更新检查间隔 (seconds, default: 600 = 10 minutes)
file_check_interval = 600
```

## 注意事项

- 可执行文件已针对Linux AMD64编译
- 服务以root权限运行
- 移除了首次运行检测，避免交互问题
- 使用中国大陆可访问的IP检测服务，网络稳定性更好
- DNS和文件更新现在使用独立的定时器
- 日志文件位于：`/var/log/ip_updater/ip_updater.log`
- 配置文件位于：`/etc/ip_updater/config.conf`