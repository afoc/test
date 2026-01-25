# TLS-VPN 编译说明

## 编译结果

已成功编译以下版本：

### ✅ 编译成功的二进制文件

| 平台 | 架构 | 文件名 | 大小 | 说明 |
|------|------|--------|------|------|
| Linux | x86_64 | `tls-vpn` | 12 MB | Linux 64位版本 |
| Windows | x86_64 | `tls-vpn.exe` | 13 MB | Windows 64位版本（推荐）|
| Windows | x86 | `tls-vpn-x86.exe` | 12 MB | Windows 32位版本（兼容性）|

### 编译环境

- **Go 版本**: 1.25.6
- **编译平台**: Linux (linux/amd64)
- **编译日期**: 2026-01-25
- **编译方式**: 标准编译（包含调试信息）

## 快速使用

### Linux

```bash
# 运行帮助
./bin/tls-vpn --help

# 启动管理界面（需要 root 权限）
sudo ./bin/tls-vpn

# 仅启动后台服务
sudo ./bin/tls-vpn --service

# 查看服务状态
./bin/tls-vpn --status
```

### Windows

```cmd
# 以管理员身份运行 PowerShell 或 CMD

# 运行帮助
bin\tls-vpn.exe --help

# 启动管理界面
bin\tls-vpn.exe

# 仅启动后台服务
bin\tls-vpn.exe --service

# 查看服务状态
bin\tls-vpn.exe --status
```

## 项目结构

```
tls-vpn/
├── source/          # 源代码目录
│   ├── *.go        # Go 源代码文件
│   ├── tokens/     # Token 存储目录
│   ├── go.mod      # Go 模块定义
│   └── go.sum      # 依赖版本锁定
├── bin/            # 编译输出目录（不提交到git）
│   ├── tls-vpn     # Linux 二进制文件
│   └── tls-vpn.exe # Windows 二进制文件
├── BUILD.md        # 编译说明（本文件）
├── README.md       # 项目文档
├── IMPLEMENTATION.md  # 实现细节
└── 其他文档...
```

## 编译命令参考

如果需要重新编译，可以使用以下命令：

### Linux 版本
```bash
cd source
go build -o ../bin/tls-vpn
```

### Windows 64位版本（交叉编译）
```bash
cd source
GOOS=windows GOARCH=amd64 go build -o ../bin/tls-vpn.exe
```

### Windows 32位版本（交叉编译）
```bash
cd source
GOOS=windows GOARCH=386 go build -o ../bin/tls-vpn-x86.exe
```

### 优化编译（减小体积）
```bash
# Linux
cd source
go build -ldflags="-s -w" -o ../bin/tls-vpn

# Windows 64位
cd source
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o ../bin/tls-vpn.exe

# 使用 upx 进一步压缩（可选）
cd ../bin
upx --best --lzma tls-vpn
upx --best --lzma tls-vpn.exe
```

参数说明：
- `-ldflags="-s -w"`: 去除调试信息和符号表，减小文件大小约 30%
- `-v`: 显示详细编译过程
- `-o`: 指定输出文件名

## 系统要求

### Linux
- 内核 2.6+ (推荐 4.0+)
- root 权限或 CAP_NET_ADMIN 能力
- iproute2 工具集
- iptables (如需 NAT 功能)

### Windows
- Windows 7+ (推荐 Windows 10/11)
- 管理员权限
- Wintun 驱动（程序会自动加载）

## 依赖项

项目使用 Go Modules 管理依赖，主要依赖：

```go
require (
    github.com/gdamore/tcell/v2 v2.8.1       // TUI 终端控制
    github.com/rivo/tview v0.0.0-20250120200925-37dc67bfedf9  // TUI 框架
    golang.org/x/term v0.28.0                 // 终端交互
    golang.zx2c4.com/wireguard/tun v0.0.0-20230325221338-052af4a8072b // TUN 设备
)
```

## 文件说明

编译后的二进制文件是独立可执行文件，包含所有必要的代码。

运行时会创建的目录和文件：
- `./certs/` - 证书存储目录
- `./tokens/` - Token 存储目录
- `./config.json` - 配置文件
- `/var/log/tls-vpn.log` (Linux) - 日志文件
- `/var/run/tlsvpn.pid` (Linux) - PID 文件
- `/var/run/vpn_control.sock` (Linux) - IPC 控制套接字

## 故障排查

### 编译失败

**问题**: `cannot find package`
```bash
# 解决方案：清理模块缓存并重新下载
cd source
go clean -modcache
go mod download
go build -v -o ../bin/tls-vpn
```

**问题**: `undefined: water.New` (Linux)
```bash
# 解决方案：安装 TUN 设备依赖
sudo apt-get install -y linux-headers-$(uname -r)
```

### 运行失败

**问题**: `permission denied`
```bash
# Linux: 需要 root 权限
sudo ./bin/tls-vpn

# 或者赋予 CAP_NET_ADMIN 能力
sudo setcap cap_net_admin=eip ./bin/tls-vpn
```

**问题**: Windows 提示缺少 Wintun 驱动
```
下载 wintun.dll：https://www.wintun.net/
放到 bin/ 目录（与 tls-vpn.exe 同目录）
```

## 打包发布

### Linux 发行包
```bash
# 创建发行包
mkdir -p tls-vpn-linux-amd64
cp bin/tls-vpn tls-vpn-linux-amd64/
cp README.md tls-vpn-linux-amd64/
tar czf tls-vpn-linux-amd64.tar.gz tls-vpn-linux-amd64/
```

### Windows 发行包
```bash
# 创建发行包
mkdir -p tls-vpn-windows-amd64
cp bin/tls-vpn.exe tls-vpn-windows-amd64/
cp README.md tls-vpn-windows-amd64/
# 下载 wintun.dll 并放入目录
zip -r tls-vpn-windows-amd64.zip tls-vpn-windows-amd64/
```

## 版本信息

- **项目**: TLS-VPN 系统
- **版本**: v2.0
- **Go 最低版本要求**: 1.21+
- **许可证**: MIT License

## 更多信息

详细使用说明请参考：
- [README.md](README.md) - 完整用户手册
- [IMPLEMENTATION.md](IMPLEMENTATION.md) - 技术实现细节
- [CONFIG.md](CONFIG.md) - 配置指南（如果存在）

---

编译日期: 2026-01-25
编译器: Go 1.25.6
