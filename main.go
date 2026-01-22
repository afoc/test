package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "-h", "--help", "help":
			printHelp()
			return

		case "--service":
			// 启动后台服务模式
			runService()
			return

		case "--stop":
			// 停止后台服务
			stopService()
			return

		case "--status":
			// 显示服务状态
			showStatus()
			return

		default:
			fmt.Printf("未知参数: %s\n", os.Args[1])
			fmt.Println("使用 --help 查看帮助")
			return
		}
	}

	// 默认：启动 TUI
	runTUI()
}

func printHelp() {
	fmt.Println("╔══════════════════════════════════════════════════╗")
	fmt.Println("║           TLS VPN 管理系统                        ║")
	fmt.Println("╚══════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("用法:")
	fmt.Println("  ./tls-vpn              进入交互式管理界面 (TUI)")
	fmt.Println("  ./tls-vpn --service    启动后台服务")
	fmt.Println("  ./tls-vpn --status     查看服务状态")
	fmt.Println("  ./tls-vpn --stop       停止后台服务")
	fmt.Println()
	fmt.Println("架构说明:")
	fmt.Println("  本程序采用服务/客户端架构：")
	fmt.Println("  1. 先运行 ./tls-vpn --service 启动后台服务")
	fmt.Println("  2. 再运行 ./tls-vpn 进入 TUI 管理界面")
	fmt.Println("  3. TUI 通过 Unix Socket 与后台服务通信")
	fmt.Println()
	fmt.Println("日志文件: /var/log/tls-vpn.log")
	fmt.Println("控制套接字: /var/run/vpn_control.sock")
}

// runService 运行后台服务
func runService() {
	// 设置日志文件（自动轮转：最大 10MB，保留 5 个备份）
	logWriter, err := NewRotatingFileWriter("/var/log/tls-vpn.log", 10, 5)
	if err != nil {
		log.Printf("警告: 无法打开日志文件: %v, 使用标准输出", err)
		logWriter = nil
	} else {
		defer logWriter.Close()
	}

	// 初始化服务端日志系统（同时输出到文件和内存缓冲区）
	logger := InitServiceLogger(logWriter)

	// 设置标准 log 包的输出到我们的日志系统
	log.SetOutput(logger)
	log.SetFlags(log.Ldate | log.Ltime)

	log.Println("========================================")
	log.Printf("TLS VPN 服务启动 (PID: %d)", os.Getpid())
	log.Println("========================================")

	// 创建服务实例
	service := NewVPNService()

	// 创建控制服务器
	controlServer := NewControlServer(service)
	if err := controlServer.Start(); err != nil {
		log.Fatalf("启动控制服务器失败: %v", err)
	}

	fmt.Println("TLS VPN 服务已启动")
	fmt.Println("控制套接字:", ControlSocketPath)
	fmt.Println("日志文件: /var/log/tls-vpn.log")
	fmt.Println()
	fmt.Println("使用 ./tls-vpn 进入管理界面")
	fmt.Println("使用 ./tls-vpn --stop 停止服务")

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 等待退出信号
	<-sigChan
	log.Println("收到退出信号，正在停止服务...")

	service.Cleanup()
	controlServer.Stop()

	log.Println("TLS VPN 服务已退出")
}

// stopService 停止后台服务
func stopService() {
	client := NewControlClient()
	if !client.IsServiceRunning() {
		fmt.Println("服务未运行")
		return
	}

	fmt.Println("正在停止服务...")
	resp, err := client.Shutdown()
	if err != nil {
		fmt.Printf("停止失败: %v\n", err)
		return
	}
	if resp.Success {
		fmt.Println("服务已停止")
	} else {
		fmt.Printf("停止失败: %s\n", resp.Error)
	}
}

// showStatus 显示服务状态
func showStatus() {
	client := NewControlClient()
	if !client.IsServiceRunning() {
		fmt.Println("服务状态: [未运行]")
		fmt.Println()
		fmt.Println("启动服务: ./tls-vpn --service")
		return
	}

	fmt.Println("服务状态: [运行中]")
	fmt.Println()

	// 获取服务端状态
	serverStatus, err := client.ServerStatus()
	if err == nil {
		if serverStatus.Running {
			fmt.Printf("VPN服务端: 运行中 (端口: %d, 客户端: %d)\n",
				serverStatus.Port, serverStatus.ClientCount)
			fmt.Printf("  TUN设备: %s\n", serverStatus.TUNDevice)
			fmt.Printf("  网段: %s\n", serverStatus.Network)
			fmt.Printf("  流量: ↑%s ↓%s\n",
				formatBytes(serverStatus.TotalSent), formatBytes(serverStatus.TotalRecv))
		} else {
			fmt.Println("VPN服务端: 未启动")
		}
	}

	// 获取客户端状态
	clientStatus, err := client.ClientStatus()
	if err == nil {
		if clientStatus.Connected {
			fmt.Printf("VPN客户端: 已连接 (IP: %s)\n", clientStatus.AssignedIP)
			fmt.Printf("  服务器: %s:%d\n", clientStatus.ServerAddress, clientStatus.ServerPort)
		} else {
			fmt.Println("VPN客户端: 未连接")
		}
	}

	// 获取配置
	cfg, err := client.ConfigGet()
	if err == nil {
		fmt.Println()
		fmt.Println("当前配置:")
		cfgJSON, _ := json.MarshalIndent(cfg, "  ", "  ")
		fmt.Printf("  %s\n", string(cfgJSON))
	}
}

// runTUI 运行 TUI 界面
func runTUI() {
	// 检查服务是否运行
	client := NewControlClient()
	if !client.IsServiceRunning() {
		fmt.Println("警告: 后台服务未运行")
		fmt.Println()
		fmt.Println("请先启动后台服务:")
		fmt.Println("  sudo ./tls-vpn --service")
		fmt.Println()
		fmt.Println("或在后台运行:")
		fmt.Println("  sudo ./tls-vpn --service &")
		fmt.Println()
		fmt.Print("是否仍然进入 TUI? (功能将受限) [y/N]: ")

		var answer string
		fmt.Scanln(&answer)
		if answer != "y" && answer != "Y" {
			return
		}
	}

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	app := NewTUIApp()

	go func() {
		<-sigChan
		app.Stop()
	}()

	if err := app.Run(); err != nil {
		fmt.Printf("TUI运行错误: %v\n", err)
		os.Exit(1)
	}
}
