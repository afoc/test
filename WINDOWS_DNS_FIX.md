# Windows DNS 问题修复说明

## 问题描述

Windows客户端在全流量代理模式下，无法访问国际网站（如 www.google.com），表现为：
- ✅ 可以访问国内网站（如 www.baidu.com）
- ❌ 无法访问国际网站（DNS解析失败或连接超时）
- ❌ ping www.google.com 解析到错误的IP地址

## 问题根源

**DNS设置在了错误的网络接口上**

### 原来的错误行为：
```
物理网卡 (Ethernet0):
  - DNS: 192.168.243.2 (本地DNS，无法解析国际域名)

VPN接口 (tls-vpn):
  - DNS: fec0:0:0:ffff::1%1 (错误的IPv6地址)
  - IP: 10.8.0.6
```

DNS请求通过物理网卡的DNS服务器（192.168.243.2），该DNS无法正确解析国际域名，导致访问失败。

### 修复后的正确行为：
```
VPN接口 (tls-vpn):
  - DNS: 8.8.8.8, 1.1.1.1 (Google公共DNS)
  - IP: 10.8.0.6
  - 所有流量通过VPN隧道
```

DNS请求通过VPN接口，使用8.8.8.8等公共DNS服务器，可以正确解析所有域名。

## 技术细节

### Windows DNS设置特点：
- Windows需要在**每个网络接口**上单独设置DNS
- 如果VPN接口没有设置DNS，系统会使用物理网卡的DNS
- 需要使用 `netsh interface ipv4 set dnsservers` 命令指定接口

### Linux DNS设置特点：
- Linux使用全局的 `/etc/resolv.conf` 文件
- 不需要区分网络接口

## 修复内容

### 1. route_manager_windows.go
```go
// 新增函数：为指定接口设置DNS
func (rm *RouteManager) SetDNSForInterface(dnsServers []string, vpnIface string) error {
    // 在VPN接口（而非物理网卡）上设置DNS服务器
    ifaceName := vpnIface  // 使用VPN接口名称
    
    // 使用 netsh 命令设置DNS
    netsh interface ipv4 set dnsservers <vpnIface> static 8.8.8.8 primary
    netsh interface ipv4 add dnsservers <vpnIface> 1.1.1.1 index=2
}
```

### 2. vpn_client.go
```go
// 在设置DNS时传入VPN接口名称
tunDeviceName := c.tunDevice.Name()  // 例如: "tls-vpn"
rm.SetDNSForInterface(c.config.DNSServers, tunDeviceName)
```

### 3. route_manager.go (Linux版本)
```go
// 添加相同的函数签名保持接口统一
// 但Linux实现中忽略接口参数，继续使用 /etc/resolv.conf
func (rm *RouteManager) SetDNSForInterface(dnsServers []string, vpnIface string) error {
    // Linux不需要指定接口，直接修改 /etc/resolv.conf
}
```

## 测试验证

修复后，在Windows客户端上执行以下测试：

### 1. 检查DNS配置
```cmd
ipconfig /all
```
应该看到VPN接口（tls-vpn）的DNS为：
```
未知适配器 tls-vpn:
   DNS 服务器  . . . . . . . . . . . : 8.8.8.8
                                       1.1.1.1
```

### 2. 测试DNS解析
```cmd
nslookup www.google.com
```
应该正确解析到Google的IP地址。

### 3. 测试国际网站访问
```cmd
curl www.google.com
ping www.google.com
curl https://www.google.com
```
应该能够正常访问。

## 编译和部署

### 编译Windows版本：
```bash
GOOS=windows GOARCH=amd64 go build -o tls-vpn.exe .
```

### 部署到Windows客户端：
1. 停止旧版本的VPN客户端
2. 替换 `tls-vpn.exe` 文件
3. 重新启动VPN客户端
4. 连接到VPN服务器
5. 验证DNS和网络访问

## 注意事项

1. **需要管理员权限**：修改网络接口DNS配置需要管理员权限
2. **不影响Linux客户端**：Linux版本继续使用原有逻辑，不受影响
3. **向后兼容**：保留了原有的 `SetDNS()` 函数，确保兼容性

## 日期
2026-01-25
