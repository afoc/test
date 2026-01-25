# Windows 全流量代理修复说明

## 📋 修复内容

本次更新修复了Windows客户端的全流量代理功能，主要解决了以下问题：

### 1️⃣ **TAP设备以太网帧处理**
- **问题**：服务端报错 `write tun: invalid argument`
- **原因**：Windows使用TAP设备（Layer 2），需要以太网帧，而Linux使用TUN设备（Layer 3），只需IP包
- **修复**：
  - 新增 `tap_ethernet.go` - 以太网帧头处理
  - 新增 `vpn_client_windows.go` - Windows客户端TAP处理
  - 新增 `vpn_server_windows.go` - Windows服务端TAP处理
  - 数据发送：自动去除以太网帧头，只传输IP包
  - 数据接收：自动添加以太网帧头再写入TAP设备

### 2️⃣ **路由优先级问题**
- **问题**：VPN路由跃点数(26) > 默认路由跃点数(25)，导致流量不走VPN
- **修复**：将VPN路由跃点数改为 1（最高优先级）

### 3️⃣ **路由接口绑定**
- **问题**：路由绑定到物理网卡而不是TAP设备
- **修复**：改进接口索引查找逻辑，正确识别TAP设备

## 🎯 新增文件

```
tap_ethernet.go           # 以太网帧处理（仅Windows）
vpn_client_windows.go     # Windows客户端TAP设备处理
vpn_client_unix.go        # Unix/Linux客户端TUN设备处理
vpn_server_windows.go     # Windows服务端TAP设备处理
vpn_server_unix.go        # Unix/Linux服务端TUN设备处理
```

## 🔧 修改文件

```
route_manager_windows.go  # 改进接口索引查找，修复跃点数
vpn_client.go            # 使用平台特定的设备读写方法
vpn_server.go            # 使用平台特定的设备读写方法
```

## ✅ 测试步骤

### 1. 在Windows客户端测试连接

```cmd
# 1. 以管理员身份运行
tls-vpn.exe

# 2. 连接到服务器后检查路由表
route print -4

# 3. 期望看到的路由（跃点数为1）：
#    0.0.0.0        128.0.0.0         10.8.0.1     [TAP设备IP]      1
#    128.0.0.0      128.0.0.0         10.8.0.1     [TAP设备IP]      1
```

### 2. 验证流量走向

```cmd
# 测试访问外网
tracert 1.1.1.1

# 第一跳应该是VPN网关 (10.8.0.1)，不是物理网关
```

### 3. 检查服务端日志

服务端不应再出现 `write tun: invalid argument` 错误

## 📊 路由表对比

### ❌ 修复前（错误）
```
网络目标        网络掩码          网关       接口   跃点数
0.0.0.0          0.0.0.0    192.168.243.2  192.168.243.130     25  ← 默认（优先）
0.0.0.0        128.0.0.0         10.8.0.1  192.168.243.130     35  ← VPN（被忽略）
128.0.0.0      128.0.0.0         10.8.0.1  192.168.243.130     35  ← VPN（被忽略）
```
**问题**：跃点数35 > 25，流量走默认路由

### ✅ 修复后（正确）
```
网络目标        网络掩码          网关       接口   跃点数
0.0.0.0          0.0.0.0    192.168.243.2  192.168.243.130     25  ← 默认
0.0.0.0        128.0.0.0         10.8.0.1          10.8.0.2      1  ← VPN（优先）
128.0.0.0      128.0.0.0         10.8.0.1          10.8.0.2      1  ← VPN（优先）
4.217.179.206  255.255.255.255   192.168.243.2  192.168.243.130   1  ← 保护VPN连接
```
**正确**：跃点数1 < 25，流量走VPN路由

## 🔍 技术细节

### 以太网帧结构
```
+-------------------+-------------------+-------+------------------+
| 目标MAC (6字节)   | 源MAC (6字节)     | 类型  |   IP包数据       |
+-------------------+-------------------+-------+------------------+
                    14字节帧头                    
```

### 数据流向

#### 客户端发送（Windows TAP）
```
应用程序 → IP包 → TAP设备(添加帧头) → handleTUNRead(去除帧头) → VPN隧道
```

#### 客户端接收（Windows TAP）
```
VPN隧道 → IP包 → dataLoopWriteToTAP(添加帧头) → TAP设备 → 操作系统
```

#### 服务端处理（Windows TAP）
```
客户端1 → IP包 → handleClientDataWriteToTAP(添加帧头) → TAP设备 → 路由 → TAP设备 → handleTUNRead(去除帧头) → 客户端2
```

### 跃点数优先级

Windows路由选择规则：
1. **最长前缀匹配**：更具体的路由优先
2. **跃点数（Metric）**：数值越小优先级越高

```
跃点数 1   - VPN路由（最高优先级）
跃点数 25  - 默认物理网关
跃点数 281 - 本地链路
```

## 🐛 故障排查

### 问题1：流量仍不走VPN
**检查**：
```cmd
route print -4 | findstr "0.0.0.0"
```
**确认**：VPN路由的跃点数应该最小（1）

**解决**：
```cmd
# 手动删除旧路由
route delete 0.0.0.0 mask 128.0.0.0
route delete 128.0.0.0 mask 128.0.0.0

# 重新连接VPN
```

### 问题2：找不到TAP设备
**检查**：
```cmd
# 查看TAP设备
ipconfig | findstr /C:"本地连接" /C:"TAP"
```

**解决**：安装TAP-Windows驱动
- 下载OpenVPN：https://openvpn.net/community-downloads/
- 或单独安装TAP-Windows

### 问题3：无法访问服务端
**检查**：
```cmd
route print -4 | findstr "服务端IP"
```

**确认**：应该有一条直连路由保护VPN连接
```
4.217.179.206  255.255.255.255  192.168.243.2  ...  1
```

## 📝 配置示例

### 客户端配置（全流量模式）
```json
{
  "server_address": "your.vpn.server",
  "server_port": 8080,
  "route_mode": "full",
  "redirect_gateway": true,
  "redirect_dns": false,
  "exclude_routes": []
}
```

### 服务端配置
```json
{
  "network": "10.8.0.0/24",
  "server_ip": "10.8.0.1/24",
  "enable_nat": true,
  "route_mode": "full",
  "redirect_gateway": true
}
```

## 🎉 预期结果

连接成功后：

1. ✅ 路由表显示VPN路由优先级最高（metric=1）
2. ✅ `tracert 1.1.1.1` 第一跳是 10.8.0.1
3. ✅ 所有流量经过VPN隧道
4. ✅ 服务端无 "invalid argument" 错误
5. ✅ 可以正常访问互联网

## 🔄 编译命令

```bash
# Linux/Unix版本
go build -o tls-vpn

# Windows版本（交叉编译）
GOOS=windows GOARCH=amd64 go build -o tls-vpn.exe

# Windows版本（在Windows上编译）
go build -o tls-vpn.exe
```

## 📌 注意事项

1. **管理员权限**：Windows客户端必须以管理员身份运行
2. **TAP驱动**：需要安装TAP-Windows驱动
3. **防火墙**：确保防火墙允许VPN程序
4. **跃点数持久化**：路由在重启后会丢失，需重新连接VPN

## 🔗 相关资源

- TAP-Windows驱动：https://build.openvpn.net/downloads/releases/
- OpenVPN社区版：https://openvpn.net/community-downloads/
- Windows路由命令：https://learn.microsoft.com/zh-cn/windows-server/administration/windows-commands/route_ws2008
