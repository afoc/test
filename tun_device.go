package main

import (
	"fmt"
	"log"
	"os"

	"github.com/songgao/water"
)

// checkRootPrivileges 检查是否具有root权限
func checkRootPrivileges() error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("需要root权限运行，请使用sudo")
	}
	return nil
}

// createTUNDevice 创建TUN设备（如果已存在则尝试其他名称）
func createTUNDevice(baseName string) (*water.Interface, error) {
	const maxAttempts = 10 // 最多尝试10个设备名称

	for i := 0; i < maxAttempts; i++ {
		// 构造设备名称：tun0, tun1, tun2, ...
		deviceName := baseName
		if i > 0 {
			deviceName = fmt.Sprintf("%s%d", baseName, i)
		}

		// 检查设备是否已存在
		if err := runCmdSilent("ip", "link", "show", deviceName); err == nil {
			// 设备已存在，尝试下一个名称
			log.Printf("TUN设备 %s 已存在，尝试下一个名称...", deviceName)
			continue
		}

		// 尝试创建设备
		config := water.Config{
			DeviceType: water.TUN,
		}
		config.Name = deviceName

		iface, err := water.New(config)
		if err != nil {
			// 创建失败，可能是设备刚好被创建了，尝试下一个
			log.Printf("创建TUN设备 %s 失败: %v，尝试下一个名称...", deviceName, err)
			continue
		}

		log.Printf("成功创建TUN设备: %s", iface.Name())
		return iface, nil
	}

	return nil, fmt.Errorf("无法创建TUN设备：已尝试 %s0-%s%d 全部失败", baseName, baseName, maxAttempts-1)
}

// configureTUNDevice 配置TUN设备IP地址
func configureTUNDevice(ifaceName string, ipAddr string, mtu int) error {
	// 先清理可能存在的IP地址配置
	_ = runCmdSilent("ip", "addr", "flush", "dev", ifaceName) // 忽略错误，因为设备可能是新创建的

	// 设置IP地址
	output, err := runCmdCombined("ip", "addr", "add", ipAddr, "dev", ifaceName)
	if err != nil {
		// 检查是否是因为地址已存在（某些情况下可能清理失败）
		if containsString(string(output), "File exists") || containsString(string(output), "RTNETLINK answers: File exists") {
			log.Printf("警告: IP地址 %s 已存在于 %s，尝试使用replace", ipAddr, ifaceName)
			// 尝试使用replace（注意：某些系统不支持replace，改用delete+add）
			_ = runCmdSilent("ip", "addr", "del", ipAddr, "dev", ifaceName)
			// 重试添加
			output, err = runCmdCombined("ip", "addr", "add", ipAddr, "dev", ifaceName)
			if err != nil {
				return fmt.Errorf("设置IP地址失败: %v, 输出: %s", err, string(output))
			}
		} else {
			return fmt.Errorf("设置IP地址失败: %v, 输出: %s", err, string(output))
		}
	}

	// 设置MTU（在启动前设置）
	output, err = runCmdCombined("ip", "link", "set", "dev", ifaceName, "mtu", fmt.Sprintf("%d", mtu))
	if err != nil {
		return fmt.Errorf("设置MTU失败: %v, 输出: %s", err, string(output))
	}

	// 启动接口
	output, err = runCmdCombined("ip", "link", "set", "dev", ifaceName, "up")
	if err != nil {
		return fmt.Errorf("启动接口失败: %v, 输出: %s", err, string(output))
	}

	log.Printf("配置TUN设备 %s: IP=%s, MTU=%d", ifaceName, ipAddr, mtu)
	return nil
}

// cleanupTUNDevice 清理TUN设备
func cleanupTUNDevice(ifaceName string) {
	// 先检查设备是否存在
	if err := runCmdSilent("ip", "link", "show", ifaceName); err != nil {
		// 设备不存在，无需清理
		log.Printf("TUN设备 %s 不存在，无需清理", ifaceName)
		return
	}

	// 关闭接口
	if output, err := runCmdCombined("ip", "link", "set", "dev", ifaceName, "down"); err != nil {
		log.Printf("警告: 关闭TUN设备 %s 失败: %v, 输出: %s", ifaceName, err, string(output))
	}

	// 删除接口
	if output, err := runCmdCombined("ip", "link", "delete", ifaceName); err != nil {
		log.Printf("警告: 删除TUN设备 %s 失败: %v, 输出: %s", ifaceName, err, string(output))
	} else {
		log.Printf("已清理TUN设备: %s", ifaceName)
	}
}

// enableIPForwarding 启用IP转发
func enableIPForwarding() error {
	output, err := runCmdCombined("sysctl", "-w", "net.ipv4.ip_forward=1")
	if err != nil {
		return fmt.Errorf("启用IP转发失败: %v, 输出: %s", err, string(output))
	}
	log.Println("已启用IP转发")
	return nil
}
