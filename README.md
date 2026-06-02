# IP-Updater Service

自动化动态公网IP更新服务，为没有固定公网IP的服务器提供DNS记录和配置文件自动更新功能。

## 功能特性

- ✅ **多种IP检测方式**：优先使用API端点，支持Web端点作为备选；可自动从纯文本/JSON/HTML 响应中提取并校验公网IP
- ✅ **多DNS服务商支持**：阿里云、腾讯云、华为云、Cloudflare、GoDaddy
- ✅ **配置文件更新**：支持JSON、YAML、TOML、INI格式文件的IP地址更新
- ✅ **混合更新模式**：DNS和文件更新可同时使用，按配置顺序执行
- ✅ **失败重试机制**：可配置重试间隔和次数，支持无限重试
- ✅ **灵活运行模式**：支持单次运行（默认）和守护进程模式（-daemon）
- ✅ **日志管理**：多级别日志记录，文件轮转支持
- ✅ **安全加密**：API密钥自动加密存储
- ✅ **配置备份**：文件更新前自动备份

## 项目结构

```
ip_updater/
├── cmd/ip_updater/          # 主程序入口
│   └── main.go
├── internal/                # 内部包
│   ├── config/              # 配置管理
│   ├── crypto/              # 加密功能
│   ├── detector/            # IP检测
│   ├── logger/              # 日志管理
│   └── updater/             # 更新器
├── pkg/                     # 公共包
│   ├── dns/                 # DNS服务商接口
│   └── fileupdate/          # 文件更新功能
├── examples/                # 配置示例
│   ├── aliyun-config.conf
│   ├── tencent-config.conf
│   ├── huawei-config.conf
│   ├── cloudflare-config.conf
│   ├── godaddy-config.conf
│   ├── file-update-config.conf
│   ├── sample-files/        # 示例配置文件
│   └── README.md
├── build.sh                 # 编译脚本
├── go.mod
└── README.md
```

## 快速开始

### 1. 编译

```bash
# 在Linux环境下运行
chmod +x build.sh
./build.sh
```

编译完成后，在`build/`目录下会生成：
- `ip_updater` - 主程序
- `ip_updater.service` - systemd服务文件
- `install.sh` - 安装脚本
- `uninstall.sh` - 卸载脚本
- `README.md` - 部署说明

### 2. 安装

```bash
cd build/
sudo ./install.sh
```

### 3. 配置

选择合适的配置示例并复制到配置目录：

```bash
# 阿里云DNS示例
sudo cp examples/aliyun-config.conf /etc/ip_updater/config.conf

# 编辑配置文件
sudo nano /etc/ip_updater/config.conf
```

### 4. 运行方式

#### 单次运行（默认）
```bash
# 执行一次IP检测和更新后退出
./ip_updater -config /etc/ip_updater/config.conf
```

#### 守护进程模式
```bash
# 持续运行，定期检测IP变化
./ip_updater -daemon -config /etc/ip_updater/config.conf
```

#### systemd服务（推荐用于守护进程模式）
```bash
sudo systemctl enable ip_updater
sudo systemctl start ip_updater
```

## 配置说明

### 基础配置

```toml
# IP检测间隔（秒）
check_interval = 300

[ip_detection]
timeout = 30
# 优先尝试 api_endpoints（从上到下，第一个成功即停止）
api_endpoints = [
    "https://myip.ipip.net",
    "https://ddns.oray.com/checkip",
    "https://vv.video.qq.com/checktime?otype=json",
]
# api_endpoints 全部失败后，再依次尝试 web_endpoints 作为兜底
web_endpoints = [
    "https://whois.pconline.com.cn/ipJson.jsp?json=true",
    "https://g3.letv.com/r?format=1",
]

[retry]
interval = 60        # 重试间隔
max_retries = -1     # 最大重试次数（-1表示无限）

[logging]
level = "info"
file_path = "/var/log/ip_updater/ip_updater.log"
```

> **IP检测说明**
> - **探测顺序**：先按顺序遍历 `api_endpoints`，任一端点成功即返回、后续不再请求；全部失败后才依次尝试 `web_endpoints` 兜底。
> - **响应解析**：无需端点返回纯 IP，程序会用正则从响应体（纯文本 / JSON / HTML 均可）中提取第一个合法 IPv4，并做 0–255 取值范围校验，解析失败的端点会自动跳过。
> - **默认源均为中国大陆可直连服务**。如服务器走代理出口，请确保所选端点返回的是你期望写入 DNS 的那个公网 IP。

### DNS更新配置

```toml
[[dns_updater]]
name = "aliyun-main"
provider = "aliyun"
access_key = "your_access_key_id"
secret_key = "your_access_key_secret"
domain = "example.com"

[[dns_updater.record]]
name = "www"
type = "A"
ttl = 600
```

### 文件更新配置

```toml
[[file_updater]]
name = "nginx-config"
file_path = "/etc/nginx/config.json"
format = "json"
key_path = "server/public_ip"  # JSON路径
backup = true
```

## 支持的路径格式

- **JSON**: `server/public_ip` → `{"server": {"public_ip": "1.2.3.4"}}`
- **YAML**: `services/webapp/environment/EXTERNAL_IP`
- **TOML**: `network/external_address` → `[network] external_address = "1.2.3.4"`
- **INI**: `server/bind_ip` → `[server] bind_ip = 1.2.3.4`

## 监控和管理

### 查看服务状态
```bash
sudo systemctl status ip_updater
```

### 查看日志
```bash
# 实时日志
sudo journalctl -u ip_updater -f

# 日志文件
sudo tail -f /var/log/ip_updater/ip_updater.log
```

### 重启服务
```bash
sudo systemctl restart ip_updater
```

## 安全特性

1. **API密钥加密**：所有敏感信息在配置文件中自动加密存储
2. **文件权限**：配置文件建议设置为600权限
3. **备份机制**：文件更新前自动创建备份
4. **错误处理**：完善的错误处理和重试机制

## DNS服务商支持状态

| 服务商 | 状态 | 说明 |
|--------|------|------|
| 阿里云 | ✅ 已实现 | 完整的API实现，支持阿里云DNS |
| 腾讯云 | ✅ 已实现 | 完整的DNSPod API实现，支持腾讯云DNS |
| 华为云 | ✅ 已实现 | 完整的华为云DNS API实现 |
| Cloudflare | ✅ 已实现 | 完整的Cloudflare API v4实现 |
| GoDaddy | ✅ 已实现 | 完整的GoDaddy API实现 |

## 开发说明

### 添加新的DNS服务商

1. 在`pkg/dns/`下创建新的provider文件
2. 实现`Provider`接口
3. 在`providers.go`的`CreateProvider`函数中添加支持
4. 更新配置示例

### 扩展文件格式支持

1. 在`pkg/fileupdate/fileupdate.go`中添加新格式的处理方法
2. 实现相应的验证和更新逻辑
3. 添加配置示例

## 故障排除

### 常见问题

1. **服务启动失败**
   - 检查配置文件语法
   - 确认日志目录权限
   - 查看系统日志：`journalctl -u ip_updater`

2. **IP检测失败**
   - 检查网络连接
   - 验证API端点可访问性
   - 调整超时时间

3. **DNS更新失败**
   - 验证API密钥正确性
   - 检查域名和记录配置
   - 查看详细错误日志

4. **文件更新失败**
   - 检查文件权限
   - 验证文件格式和路径
   - 确认备份目录可写

## 版本信息

- 版本：1.2.3
- Go版本要求：1.21+
- 目标系统：Linux Debian/Ubuntu
- 架构：AMD64

## 许可证

本项目按需求开发，请根据您的使用场景确定许可证。