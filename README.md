# TLS VPN 系统

一个基于TLS 1.3的安全VPN系统，使用Go语言结构化开发，支持TUN设备、动态路由、NAT转发以及基于Token的证书申请。

## ✨ 特性

### 核心功能
- 🔒 **TLS 1.3加密** - 使用最新的TLS协议确保通信安全
- 🎯 **双向认证** - 服务器和客户端互相验证证书
- 🌐 **TUN设备支持** - 基于Linux TUN/TAP实现三层VPN
- 🔄 **自动重连** - 客户端断线自动重连
- 💓 **心跳保活** - 自动检测连接状态
- 📊 **会话管理** - 支持多客户端并发连接

### 路由功能
- **全流量模式** - 所有流量通过VPN（redirect-gateway）
- **分流模式** - 仅特定路由通过VPN（split-tunnel）
- **动态路由推送** - 服务器向客户端推送路由配置
- **DNS劫持** - 可选的DNS重定向功能
- **NAT支持** - 服务器端自动配置NAT和IP转发

### 📁 模块化架构
- **界面与逻辑分离** - `vpn_tui.go` 仅保留 TUI 交互，其他功能拆分到各自文件（`config.go`, `cert_manager.go`, `vpn_server.go`, `vpn_client.go` 等）
- **Token/证书独立** - `token_manager.go`, `token_crypto.go`, `cert_api_server.go` 支撑安全的证书申请服务
- **运维辅助** - `route_manager.go`, `iptables_nat.go`, `tun_device.go`, `daemon.go` 等文件封装复杂系统命令
- **测试与工具链友好** - `utils.go` 提供命令执行、PID 与格式化工具，便于 `go test`/`go build`

### 🆕 最新改进（2026-01）
- ✅ **交互式菜单系统** - 数字/快捷键界面，无需记忆命令
- ✅ **流量统计** - 实时显示每个客户端的流量
- ✅ **客户端管理** - 查看在线客户端、踢出功能
- ✅ **路由模式配置** - NAT模式、分流模式一键切换
- ✅ **快速向导** - 服务端部署向导、客户端设置向导
- ✅ **动态TUN设备名称** - 自动选择可用设备（tun0-tun9）
- ✅ **Token文件支持** - 从文件读取Token，无需手动输入

---

## 📦 安装

### 系统要求
- Linux（内核 2.6+）
- Go 1.20+（推荐使用最新稳定版）
- root 权限或 `CAP_NET_ADMIN`

### 依赖
```bash
# Ubuntu/Debian
sudo apt-get install iproute2 iptables

# CentOS/RHEL
sudo yum install iproute iptables
```

依赖 managed by Go modules，`go mod tidy` 会自动拉取 `github.com/songgao/water` 等。

### 编译
```bash
# 进入项目根目录
cd /config/tls-vpn

# 拉取依赖
go mod tidy

# 生成可执行文件
go build -o vpn_tui .
```

---

## 🚀 快速开始

### 1. 构建并运行交互式菜单（推荐）⭐

```bash
# 编译（如上一步）
sudo ./vpn_tui
```

交互式菜单提供：
- 📊 服务端管理（启动/停止/配置/在线客户端/流量）
- 🔗 客户端管理（连接/断开/配置）
- 📜 证书与Token管理（初始化CA、生成CSR、申请证书、读取Token）
- ⚙️ 配置管理（保存/加载/自动保存至 `config.json`）
- 🎯 快速向导（向导式部署服务端或配置客户端）

### 2. 常用命令行选项

```bash
sudo ./vpn_tui --daemon server   # 后台以服务端模式运行
sudo ./vpn_tui --daemon client   # 后台以客户端模式运行
sudo ./vpn_tui --status          # 显示守护进程状态
sudo ./vpn_tui --stop            # 停止后台服务
```

### 3. 证书同步与客户端配置

第一次运行会在当前目录生成 `./certs/`（CA、证书、私钥）。将以下文件分发到客户端：

```bash
scp certs/ca.pem client-machine:~/vpn/certs/
scp certs/client.pem client-machine:~/vpn/certs/
scp certs/client-key.pem client-machine:~/vpn/certs/
```

编辑客户端的 `config.json`：

```json
{
  "server_address": "YOUR_SERVER_IP",
  "server_port": 8080,
  ...
}
```

然后在客户端运行：

```bash
sudo ./vpn_tui
```

---

## 📋 配置说明

### 服务器配置示例
```json
{
  "server_address": "0.0.0.0",
  "server_port": 8080,
  "network": "10.8.0.0/24",
  "server_ip": "10.8.0.1/24",
  "client_ip_start": 2,
  "client_ip_end": 254,
  "mtu": 1500,
  "keep_alive_timeout_sec": 90,
  "max_connections": 100,
  "session_timeout_sec": 300,
  "enable_nat": true,
  "nat_interface": "eth0",
  "dns_servers": ["8.8.8.8", "8.8.4.4"],
  "route_mode": "full",
  "redirect_gateway": true,
  "redirect_dns": true
}
```

### 客户端配置示例
```json
{
  "server_address": "vpn.example.com",
  "server_port": 8080,
  "network": "10.8.0.0/24",
  "mtu": 1500,
  "keep_alive_timeout_sec": 90,
  "reconnect_delay_sec": 5,
  "route_mode": "split",
  "push_routes": [
    "192.168.100.0/24",
    "10.10.0.0/16"
  ],
  "dns_servers": ["8.8.8.8"]
}
```

### 配置参数说明

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `server_address` | 服务器地址 | localhost |
| `server_port` | 服务器端口 | 8080 |
| `network` | VPN网络CIDR | 10.8.0.0/24 |
| `server_ip` | 服务器VPN IP | 10.8.0.1/24 |
| `mtu` | 最大传输单元 | 1500 |
| `enable_nat` | 启用NAT（仅服务器） | true |
| `route_mode` | 路由模式（full/split） | split |
| `redirect_gateway` | 重定向默认网关 | false |
| `redirect_dns` | 劫持DNS | false |

---

## 🔧 高级功能

### 动态TUN设备名称

程序会自动选择可用的TUN设备名称：

```
TUN设备 tun0 已存在，尝试下一个名称...
成功创建TUN设备: tun1
服务器使用TUN设备: tun1
```

这意味着：
- ✅ 不会影响已存在的VPN连接
- ✅ 支持多个VPN实例同时运行
- ✅ 异常退出后无需手动清理即可重启

### 路由模式

#### 全流量模式（Full Tunnel）
所有流量通过VPN：
```json
{
  "route_mode": "full",
  "redirect_gateway": true,
  "exclude_routes": ["192.168.1.0/24"]  // 排除本地网络
}
```

#### 分流模式（Split Tunnel）
仅特定路由通过VPN：
```json
{
  "route_mode": "split",
  "push_routes": [
    "192.168.100.0/24",
    "10.10.0.0/16"
  ]
}
```

### NAT配置

服务器端自动配置NAT转发：
```json
{
  "enable_nat": true,
  "nat_interface": "eth0"  // 留空自动检测
}
```

程序会自动：
- 启用IP转发
- 配置iptables MASQUERADE规则
- 添加FORWARD规则
- 退出时清理所有规则

---

## 🛠️ 运维管理

### 查看状态
```bash
# 查看进程
ps aux | grep vpn

# 查看TUN设备
ip link show | grep tun

# 查看路由
ip route show

# 查看NAT规则
sudo iptables -t nat -L -n -v
```

### 停止服务
```bash
# 优雅停止（推荐）
sudo kill -SIGTERM $(cat /var/run/tlsvpn.pid)

# 或直接按 Ctrl+C
```

### 手动清理
```bash
# 删除TUN设备
sudo ip link delete tun1

# 删除PID文件
sudo rm /var/run/tlsvpn*.pid

# 清理NAT规则
sudo iptables -t nat -F
sudo iptables -F FORWARD
```

### systemd服务

创建 `/etc/systemd/system/tlsvpn.service`:
```ini
[Unit]
Description=TLS VPN Server
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/vpn server
WorkingDirectory=/etc/tlsvpn
Restart=on-failure
User=root

[Install]
WantedBy=multi-user.target
```

启用服务：
```bash
sudo systemctl enable tlsvpn
sudo systemctl start tlsvpn
sudo systemctl status tlsvpn
```

---

## 🐛 故障排查

### 常见问题

#### 1. 启动失败："程序已在运行"
```bash
# 检查进程
ps aux | grep vpn

# 如果没有运行，删除PID文件
sudo rm /var/run/tlsvpn.pid
```

#### 2. 连接超时
```bash
# 检查防火墙
sudo iptables -L -n | grep 8080

# 检查服务器监听
sudo netstat -tlnp | grep 8080

# 测试网络连通性
ping SERVER_IP
telnet SERVER_IP 8080
```

#### 3. TUN设备创建失败
```bash
# 确认有root权限
id

# 检查设备占用
ip link show | grep tun

# 手动清理（如果需要）
for i in {0..9}; do sudo ip link delete tun$i 2>/dev/null; done
```

#### 4. 路由不生效
```bash
# 查看路由表
ip route show

# 检查TUN设备状态
ip link show tun1
ip addr show tun1

# 检查IP转发
cat /proc/sys/net/ipv4/ip_forward  # 应该是1
```

#### 5. NAT不工作
```bash
# 检查NAT规则
sudo iptables -t nat -L -n -v

# 检查FORWARD规则
sudo iptables -L FORWARD -n -v
```

---

## 📚 文档

### 核心文档
- **README.md** - 本文档（快速开始和概览）
- **快速参考.md** - 常用命令和配置速查
- **完整改进总结.md** - 最新改进的详细说明

### 技术文档
- **TUN接口问题修复说明.md** - 第一版改进方案
- **动态TUN设备名称-修复说明.md** - 动态设备功能详解

### 测试
- **test_dynamic_tun.sh** - TUN设备功能自动化测试

---

## 🔒 安全性

### 加密
- **TLS 1.3** - 使用最新的TLS协议
- **双向认证** - 服务器和客户端互相验证
- **RSA 4096** - 强加密密钥长度

### 证书管理
- 自动生成CA和证书（首次运行）
- 证书有效期10年（CA）/ 1年（服务器和客户端）
- 支持手动替换证书

### 防护机制
- **序列号验证** - 防止重放攻击
- **CRC32校验** - 检测数据篡改
- **会话超时** - 自动清理过期连接
- **连接数限制** - 防止资源耗尽

---

## 🧪 测试

### 运行自动化测试
```bash
sudo ./test_dynamic_tun.sh
```

### 手动测试

#### 测试服务器
```bash
# 1. 启动服务器
sudo ./vpn server &

# 2. 检查监听
sudo netstat -tlnp | grep 8080

# 3. 测试TLS连接
openssl s_client -connect localhost:8080 -tls1_3
```

#### 测试客户端
```bash
# 1. 启动客户端
sudo ./vpn client &

# 2. 测试连通性
ping 10.8.0.1

# 3. 测试路由
traceroute -n 8.8.8.8
```

---

## 📊 性能

### 基准测试

环境：
- CPU: Intel i5-8250U
- 内存: 8GB
- 网络: 1Gbps

结果：
- 吞吐量: ~400 Mbps
- 延迟: +2-3 ms
- CPU占用: ~15%（单核）
- 内存占用: ~20MB

### 优化建议
1. 调整MTU大小（根据网络环境）
2. 使用多核CPU（未来改进）
3. 启用硬件加速（需要硬件支持）

---

## 🗺️ 未来计划

### 短期（v1.1）
- [ ] 支持配置热重载
- [ ] 添加Web管理界面
- [ ] 实现Prometheus指标
- [ ] 支持多CA证书

### 中期（v1.5）
- [ ] 支持IPv6
- [ ] 实现TAP模式（二层VPN）
- [ ] 添加带宽限制功能
- [ ] 支持用户认证

### 长期（v2.0）
- [ ] 支持QUIC协议
- [ ] 实现P2P模式
- [ ] 跨平台支持（Windows, macOS）
- [ ] 移动端支持（iOS, Android）

---

## 🤝 贡献

欢迎提交Issue和Pull Request！

### 开发环境设置
```bash
# 克隆仓库
git clone <repository-url>
cd tls-vpn

# 安装依赖
go mod download

# 运行测试
go test -v ./...

# 编译
go build -o vpn "TLS VPN 系统.go"
```

---

## 📄 许可证

本项目采用 MIT 许可证。详见 LICENSE 文件。

---

## 🙏 致谢

- [songgao/water](https://github.com/songgao/water) - TUN/TAP设备支持
- Go标准库 - TLS实现

---

## 📞 联系方式

- Issues: [GitHub Issues]
- Email: [your-email]
- 文档: 查看 `docs/` 目录

---

**版本**: v1.0  
**最后更新**: 2026-01-16  
**状态**: 生产可用 ✅
