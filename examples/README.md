# IP-Updater 配置文件示例

本目录包含了IP-Updater的各种配置示例，帮助您快速上手不同DNS服务商和文件更新场景。

## DNS服务商配置示例

### 阿里云 (aliyun-config.conf)
```bash
# 复制并修改配置
cp examples/aliyun-config.conf /etc/ip_updater/config.conf
```
**配置要点：**
- 需要阿里云Access Key ID和Secret
- 支持多个域名和记录
- TTL建议设置为300-3600秒

### 腾讯云 (tencent-config.conf)
```bash
cp examples/tencent-config.conf /etc/ip_updater/config.conf
```
**配置要点：**
- 需要腾讯云SecretId和SecretKey
- 控制台获取API密钥：https://console.cloud.tencent.com/cam/capi

### 华为云 (huawei-config.conf)
```bash
cp examples/huawei-config.conf /etc/ip_updater/config.conf
```
**配置要点：**
- 需要华为云Access Key和Secret Access Key
- 支持华为云DNS服务

### Cloudflare (cloudflare-config.conf)
```bash
cp examples/cloudflare-config.conf /etc/ip_updater/config.conf
```
**配置要点：**
- 推荐使用API Token（比Global API Key更安全）
- TTL设置为1表示自动（Cloudflare代理模式）

### GoDaddy (godaddy-config.conf)
```bash
cp examples/godaddy-config.conf /etc/ip_updater/config.conf
```
**配置要点：**
- 需要GoDaddy API Key和Secret
- 开发者中心获取：https://developer.godaddy.com/keys

## 文件更新配置示例

### 配置文件更新 (file-update-config.conf)
此配置展示如何自动更新各种格式的配置文件中的IP地址。

**支持的文件格式：**
- **JSON**: 使用路径如 \`server/public_ip\`
- **YAML**: 使用路径如 \`services/webapp/environment/EXTERNAL_IP\`
- **TOML**: 使用路径如 \`network/external_address\`
- **INI**: 使用路径如 \`server/bind_ip\`

### 路径格式说明

#### JSON路径示例
```json
{
  "server": {
    "public_ip": "1.2.3.4"  // 路径: server/public_ip
  }
}
```

#### YAML路径示例
```yaml
services:
  webapp:
    environment:
      EXTERNAL_IP: "1.2.3.4"  # 路径: services/webapp/environment/EXTERNAL_IP
```

#### TOML路径示例
```toml
[network]
external_address = "1.2.3.4"  # 路径: network/external_address
```

#### INI路径示例
```ini
[server]
bind_ip = 1.2.3.4  ; 路径: server/bind_ip
```

## 示例文件

`sample-files/` 目录包含了各种格式的示例配置文件，展示了如何在实际项目中组织配置结构。

## 安全注意事项

1. **API密钥加密**: 配置中的敏感信息会自动加密存储
2. **文件权限**: 确保配置文件权限设置正确（建议600）
3. **备份**: 启用文件更新的backup选项以防意外

## 快速开始

1. 选择合适的配置示例
2. 复制到 `/etc/ip_updater/config.conf`
3. 填入您的API密钥和域名信息
4. 启动服务：`systemctl start ip-updater`

## 混合配置

您可以在同一个配置文件中混合使用DNS更新和文件更新。**两种模式可以同时使用，严格按照配置文件中的先后顺序执行**：

### 执行顺序
1. **DNS更新**：按配置文件中 `[[dns_updater]]` 块的顺序执行
2. **文件更新**：DNS更新完成后，按 `[[file_updater]]` 块的顺序执行

### 混合配置示例 (mixed-config.conf)
```bash
cp examples/mixed-config.conf /etc/ip_updater/config.conf
```

此配置展示了一个完整的混合更新场景：
- 先更新主域名DNS（阿里云）
- 再更新备用域名DNS（Cloudflare）
- 然后依次更新：Nginx配置 → 应用配置 → Docker配置 → 系统服务配置

### 执行流程
```
IP变化检测 → DNS更新（按顺序） → 文件更新（按顺序） → 记录日志
```

这样可以确保DNS解析和本地服务配置保持同步更新。