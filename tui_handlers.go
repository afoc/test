package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

// ================ 服务端处理 ================

func handleServerStart(t *TUIApp) {
	t.addLog("正在启动服务端...")
	go func() {
		resp, err := t.client.ServerStart()
		if err != nil {
			t.addLog("[red]启动失败: %v", err)
			return
		}
		if resp.Success {
			t.addLog("[green]%s", resp.Message)
		} else {
			t.addLog("[red]%s", resp.Error)
		}
	}()
}

func handleServerStop(t *TUIApp) {
	t.addLog("正在停止服务端...")
	go func() {
		resp, err := t.client.ServerStop()
		if err != nil {
			t.addLog("[red]停止失败: %v", err)
			return
		}
		if resp.Success {
			t.addLog("[green]%s", resp.Message)
		} else {
			t.addLog("[red]%s", resp.Error)
		}
	}()
}

func handleShowClients(t *TUIApp) {
	clients, err := t.client.ServerClients()
	if err != nil {
		t.showInfoDialog("在线客户端", "获取失败: "+err.Error())
		return
	}

	if len(clients) == 0 {
		t.showInfoDialog("在线客户端", "当前没有客户端连接")
		return
	}

	var content strings.Builder
	content.WriteString(fmt.Sprintf("在线客户端数量: [green]%d[white]\n\n", len(clients)))
	content.WriteString("┌────┬──────────────┬──────────────┬──────────────┬──────────┐\n")
	content.WriteString("│ #  │ IP地址       │ 发送流量     │ 接收流量     │ 连接时间 │\n")
	content.WriteString("├────┼──────────────┼──────────────┼──────────────┼──────────┤\n")

	for i, c := range clients {
		content.WriteString(fmt.Sprintf("│ %2d │ %-12s │ %12s │ %12s │ %8s │\n",
			i+1, c.IP, formatBytes(c.BytesSent), formatBytes(c.BytesReceived), c.Duration))
	}
	content.WriteString("└────┴──────────────┴──────────────┴──────────────┴──────────┘")

	t.showInfoDialog("在线客户端", content.String())
}

func handleKickClient(t *TUIApp) {
	clients, err := t.client.ServerClients()
	if err != nil {
		t.addLog("[red]获取客户端列表失败: %v", err)
		return
	}
	if len(clients) == 0 {
		t.addLog("[yellow]当前没有客户端连接")
		return
	}

	defaultIP := clients[0].IP
	t.showInputDialog("输入要踢出的客户端IP", defaultIP, func(ip string) {
		if ip == "" {
			return
		}
		resp, err := t.client.ServerKick(ip)
		if err != nil {
			t.addLog("[red]踢出失败: %v", err)
		} else if resp.Success {
			t.addLog("[green]%s", resp.Message)
		} else {
			t.addLog("[red]%s", resp.Error)
		}
	})
}

func handleShowStats(t *TUIApp) {
	status, err := t.client.ServerStatus()
	if err != nil {
		t.showInfoDialog("流量统计", "获取失败: "+err.Error())
		return
	}

	if !status.Running {
		t.showInfoDialog("流量统计", "服务端未运行")
		return
	}

	clients, _ := t.client.ServerClients()

	var content strings.Builder
	content.WriteString(fmt.Sprintf("总发送流量: [green]%s[white]\n", formatBytes(status.TotalSent)))
	content.WriteString(fmt.Sprintf("总接收流量: [green]%s[white]\n", formatBytes(status.TotalRecv)))
	content.WriteString(fmt.Sprintf("在线客户端: [green]%d[white]\n\n", len(clients)))

	if len(clients) > 0 {
		content.WriteString("客户端流量详情:\n")
		for _, c := range clients {
			content.WriteString(fmt.Sprintf("  %s: ↑%s ↓%s\n",
				c.IP, formatBytes(c.BytesSent), formatBytes(c.BytesReceived)))
		}
	}

	t.showInfoDialog("流量统计", content.String())
}

// ================ 服务端设置处理 ================

func handleSetPort(t *TUIApp) {
	cfg, _ := t.client.ConfigGet()
	t.showInputDialog("监听端口", fmt.Sprintf("%d", cfg.ServerPort), func(value string) {
		if port, err := strconv.Atoi(value); err == nil && port > 0 && port < 65536 {
			resp, _ := t.client.ConfigUpdate("server_port", float64(port))
			if resp != nil && resp.Success {
				t.addLog("[green]端口已设置为: %d", port)
			} else {
				t.addLog("[red]设置失败")
			}
		} else {
			t.addLog("[red]无效的端口号")
		}
		t.showMenu("server_settings")
	})
}

func handleSetNetwork(t *TUIApp) {
	cfg, _ := t.client.ConfigGet()
	t.showInputDialog("VPN网段", cfg.Network, func(value string) {
		resp, _ := t.client.ConfigUpdate("network", value)
		if resp != nil && resp.Success {
			t.addLog("[green]VPN网段已设置为: %s", value)
		} else {
			t.addLog("[red]设置失败")
		}
		t.showMenu("server_settings")
	})
}

func handleSetMTU(t *TUIApp) {
	cfg, _ := t.client.ConfigGet()
	t.showInputDialog("MTU", fmt.Sprintf("%d", cfg.MTU), func(value string) {
		if mtu, err := strconv.Atoi(value); err == nil && mtu >= 576 && mtu <= 9000 {
			resp, _ := t.client.ConfigUpdate("mtu", float64(mtu))
			if resp != nil && resp.Success {
				t.addLog("[green]MTU已设置为: %d", mtu)
			} else {
				t.addLog("[red]设置失败")
			}
		} else {
			t.addLog("[red]MTU必须在576-9000之间")
		}
		t.showMenu("server_settings")
	})
}

func handleSetNATInterface(t *TUIApp) {
	cfg, _ := t.client.ConfigGet()
	t.showInputDialog("NAT出口网卡 (空=自动)", cfg.NATInterface, func(value string) {
		resp, _ := t.client.ConfigUpdate("nat_interface", value)
		if resp != nil && resp.Success {
			if value == "" {
				t.addLog("[green]将自动检测NAT出口网卡")
			} else {
				t.addLog("[green]NAT出口网卡已设置为: %s", value)
			}
		}
		t.showMenu("server_settings")
	})
}

func handleSetMaxConnections(t *TUIApp) {
	cfg, _ := t.client.ConfigGet()
	t.showInputDialog("最大连接数", fmt.Sprintf("%d", cfg.MaxConnections), func(value string) {
		if max, err := strconv.Atoi(value); err == nil && max > 0 && max <= 10000 {
			resp, _ := t.client.ConfigUpdate("max_connections", float64(max))
			if resp != nil && resp.Success {
				t.addLog("[green]最大连接数已设置为: %d", max)
			}
		} else {
			t.addLog("[red]无效的连接数")
		}
		t.showMenu("server_settings")
	})
}

func handleSetRouteModeFull(t *TUIApp) {
	t.client.ConfigUpdate("route_mode", "full")
	t.client.ConfigUpdate("enable_nat", true)
	t.client.ConfigUpdate("redirect_gateway", true)
	t.addLog("[green]已设置为NAT模式")
	t.showMenu("route_mode")
}

func handleSetRouteModeSplit(t *TUIApp) {
	t.client.ConfigUpdate("route_mode", "split")
	t.client.ConfigUpdate("enable_nat", true)
	t.client.ConfigUpdate("redirect_gateway", false)
	t.addLog("[green]已设置为分流模式")
	t.showMenu("route_mode")
}

func handleToggleNAT(t *TUIApp) {
	cfg, _ := t.client.ConfigGet()
	newValue := !cfg.EnableNAT
	t.client.ConfigUpdate("enable_nat", newValue)
	if newValue {
		t.addLog("[green]NAT已启用")
	} else {
		t.addLog("[yellow]NAT已禁用")
	}
	t.showMenu("route_mode")
}

func handleShowPushRoutes(t *TUIApp) {
	cfg, _ := t.client.ConfigGet()
	if len(cfg.PushRoutes) == 0 {
		t.showInfoDialog("推送路由", "暂无推送路由")
		return
	}
	var content strings.Builder
	content.WriteString("当前推送路由:\n\n")
	for i, route := range cfg.PushRoutes {
		content.WriteString(fmt.Sprintf("  %d. %s\n", i+1, route))
	}
	t.showInfoDialog("推送路由", content.String())
}

func handleAddPushRoute(t *TUIApp) {
	t.showInputDialog("输入网段 (如 192.168.1.0/24)", "", func(value string) {
		if value == "" {
			return
		}
		cfg, _ := t.client.ConfigGet()
		routes := append(cfg.PushRoutes, value)
		interfaceRoutes := make([]interface{}, len(routes))
		for i, r := range routes {
			interfaceRoutes[i] = r
		}
		resp, _ := t.client.ConfigUpdate("push_routes", interfaceRoutes)
		if resp != nil && resp.Success {
			t.addLog("[green]已添加路由: %s", value)
		} else {
			t.addLog("[red]添加失败")
		}
		t.showMenu("push_routes")
	})
}

func handleClearPushRoutes(t *TUIApp) {
	t.showConfirmDialog("确定清空所有推送路由?", func(confirmed bool) {
		if confirmed {
			t.client.ConfigUpdate("push_routes", []interface{}{})
			t.addLog("[green]已清空所有推送路由")
		}
		t.showMenu("push_routes")
	})
}

// ================ DNS设置处理 ================

func handleToggleDNS(t *TUIApp) {
	cfg, _ := t.client.ConfigGet()
	newValue := !cfg.RedirectDNS
	t.client.ConfigUpdate("redirect_dns", newValue)
	if newValue {
		t.addLog("[green]DNS劫持已启用 - 客户端将使用服务端推送的DNS")
	} else {
		t.addLog("[yellow]DNS劫持已禁用 - 客户端将使用本地DNS")
	}
	t.showMenu("dns_settings")
}

func handleSetDNSServers(t *TUIApp) {
	cfg, _ := t.client.ConfigGet()
	currentDNS := ""
	if len(cfg.DNSServers) > 0 {
		for i, dns := range cfg.DNSServers {
			if i > 0 {
				currentDNS += ","
			}
			currentDNS += dns
		}
	}
	t.showInputDialog("DNS服务器 (多个用逗号分隔，如 8.8.8.8,8.8.4.4)", currentDNS, func(value string) {
		if value == "" {
			t.addLog("[yellow]DNS服务器不能为空")
			t.showMenu("dns_settings")
			return
		}
		// 解析DNS服务器列表
		dnsServers := []interface{}{}
		for _, dns := range splitByComma(value) {
			dns = trimSpace(dns)
			if dns != "" {
				// 简单验证IP格式
				if net.ParseIP(dns) != nil {
					dnsServers = append(dnsServers, dns)
				} else {
					t.addLog("[red]无效的DNS服务器地址: %s", dns)
					t.showMenu("dns_settings")
					return
				}
			}
		}
		if len(dnsServers) == 0 {
			t.addLog("[yellow]至少需要一个有效的DNS服务器")
			t.showMenu("dns_settings")
			return
		}
		resp, _ := t.client.ConfigUpdate("dns_servers", dnsServers)
		if resp != nil && resp.Success {
			t.addLog("[green]DNS服务器已设置为: %v", dnsServers)
		} else {
			t.addLog("[red]设置DNS服务器失败")
		}
		t.showMenu("dns_settings")
	})
}

func handleShowDNSConfig(t *TUIApp) {
	cfg, _ := t.client.ConfigGet()
	var content strings.Builder
	content.WriteString("当前DNS配置:\n\n")
	
	// DNS劫持状态
	if cfg.RedirectDNS {
		content.WriteString("  DNS劫持:    [green]已启用[white]\n")
	} else {
		content.WriteString("  DNS劫持:    [yellow]已禁用[white]\n")
	}
	
	// DNS服务器列表
	content.WriteString("\n  DNS服务器:\n")
	if len(cfg.DNSServers) == 0 {
		content.WriteString("    (未配置)\n")
	} else {
		for i, dns := range cfg.DNSServers {
			content.WriteString(fmt.Sprintf("    %d. %s\n", i+1, dns))
		}
	}
	
	content.WriteString("\n说明:\n")
	content.WriteString("  · 启用DNS劫持后，客户端连接VPN时会自动使用上述DNS服务器\n")
	content.WriteString("  · 断开VPN后，客户端会恢复原有DNS配置\n")
	
	t.showInfoDialog("DNS配置", content.String())
}

// splitByComma 按逗号分割字符串
func splitByComma(s string) []string {
	result := []string{}
	current := ""
	for _, c := range s {
		if c == ',' {
			result = append(result, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

// ================ 证书处理 ================

func handleInitCA(t *TUIApp) {
	t.showConfirmDialog("确定初始化CA? (会覆盖现有证书)", func(confirmed bool) {
		if !confirmed {
			return
		}
		t.addLog("正在生成CA证书...")
		go func() {
			resp, err := t.client.CertInitCA()
			if err != nil {
				t.addLog("[red]生成失败: %v", err)
			} else if resp.Success {
				t.addLog("[green]CA证书已生成")
			} else {
				t.addLog("[red]%s", resp.Error)
			}
		}()
	})
}

func handleShowCertList(t *TUIApp) {
	list, err := t.client.CertList()
	if err != nil {
		t.showInfoDialog("证书列表", "获取失败: "+err.Error())
		return
	}

	var content strings.Builder
	content.WriteString(fmt.Sprintf("证书目录: %s\n\n", list.CertDir))

	for _, f := range list.Files {
		if f.Exists {
			content.WriteString(fmt.Sprintf("[green]✓[white] %-20s %10d bytes  %s\n",
				f.Name, f.Size, f.Modified.Format("2006-01-02 15:04")))
		} else {
			content.WriteString(fmt.Sprintf("[red]✗[white] %-20s (不存在)\n", f.Name))
		}
	}

	t.showInfoDialog("证书列表", content.String())
}

func handleShowSignedClients(t *TUIApp) {
	resp, err := t.client.Call(ActionCertClients, nil)
	if err != nil {
		t.showInfoDialog("已签发客户端", "获取失败: "+err.Error())
		return
	}

	var result SignedClientsResponse
	if resp.Data != nil {
		parseJSON(resp.Data, &result)
	}

	if len(result.Clients) == 0 {
		t.showInfoDialog("已签发客户端证书", "(暂无已签发的客户端证书)")
		return
	}

	var content strings.Builder
	content.WriteString("客户端证书列表:\n\n")
	for i, c := range result.Clients {
		content.WriteString(fmt.Sprintf("%d. %-30s %s\n", i+1, c.Name, c.Modified.Format("2006-01-02 15:04")))
	}

	t.showInfoDialog("已签发客户端证书", content.String())
}

// ================ Token 处理 ================

func handleGenToken(t *TUIApp) {
	t.showInputDialog("客户端名称", "", func(clientName string) {
		if clientName == "" {
			t.addLog("[red]客户端名称不能为空")
			return
		}
		t.showInputDialog("有效期(小时)", "24", func(durationStr string) {
			duration, _ := strconv.Atoi(durationStr)
			if duration <= 0 {
				duration = 24
			}

			go func() {
				token, err := t.client.TokenGenerate(clientName, duration)
				if err != nil {
					t.addLog("[red]生成Token失败: %v", err)
					return
				}

				t.addLog("[green]========== Token生成成功 ==========")
				t.addLog("Token ID:  %s", token.TokenID)
				t.addLog("Token Key: %s", token.TokenKey)
				t.addLog("客户端:    %s", token.ClientName)
				t.addLog("过期时间:  %s", token.ExpiresAt.Format("2006-01-02 15:04:05"))
				t.addLog("保存路径:  %s", token.FilePath)
				t.addLog("[green]===================================")
			}()
		})
	})
}

func handleShowTokenList(t *TUIApp) {
	tokens, err := t.client.TokenList()
	if err != nil {
		t.showInfoDialog("Token列表", "获取失败: "+err.Error())
		return
	}

	if len(tokens) == 0 {
		t.showInfoDialog("Token列表", "暂无Token")
		return
	}

	var content strings.Builder
	content.WriteString(fmt.Sprintf("Token数量: [green]%d[white]\n\n", len(tokens)))
	content.WriteString("┌────┬────────────────────────────┬────────────┬──────────┬──────────┐\n")
	content.WriteString("│ #  │ Token ID                   │ 客户端     │ 状态     │ 过期时间 │\n")
	content.WriteString("├────┼────────────────────────────┼────────────┼──────────┼──────────┤\n")

	for i, tk := range tokens {
		status := "[green]有效"
		if tk.Status == "used" {
			status = "[gray]已使用"
		} else if tk.Status == "expired" {
			status = "[red]已过期"
		}

		tokenID := tk.ID
		if len(tokenID) > 26 {
			tokenID = tokenID[:26]
		}
		clientName := tk.ClientName
		if len(clientName) > 10 {
			clientName = clientName[:10]
		}

		content.WriteString(fmt.Sprintf("│ %2d │ %-26s │ %-10s │ %-8s[white] │ %s │\n",
			i+1, tokenID, clientName, status, tk.ExpiresAt.Format("01-02 15:04")))
	}
	content.WriteString("└────┴────────────────────────────┴────────────┴──────────┴──────────┘")

	t.showInfoDialog("Token列表", content.String())
}

func handleDeleteToken(t *TUIApp) {
	tokens, _ := t.client.TokenList()
	if len(tokens) == 0 {
		t.addLog("[yellow]暂无Token可删除")
		return
	}

	var content strings.Builder
	for i, tk := range tokens {
		content.WriteString(fmt.Sprintf("%d. %s (%s)\n", i+1, tk.ID, tk.ClientName))
	}
	t.addLog("Token列表:\n%s", content.String())

	t.showInputDialog("输入要删除的序号", "1", func(idxStr string) {
		idx, err := strconv.Atoi(idxStr)
		if err != nil || idx < 1 || idx > len(tokens) {
			t.addLog("[red]无效的序号")
			return
		}
		resp, _ := t.client.TokenDelete(idx)
		if resp != nil && resp.Success {
			t.addLog("[green]Token已删除")
		} else {
			t.addLog("[red]删除失败")
		}
		t.showMenu("token")
	})
}

func handleCleanupTokens(t *TUIApp) {
	resp, err := t.client.TokenCleanup()
	if err != nil {
		t.addLog("[red]清理失败: %v", err)
	} else if resp.Success {
		t.addLog("[green]%s", resp.Message)
	}
	t.showMenu("token")
}

// ================ 客户端处理 ================

func handleClientConnect(t *TUIApp) {
	t.addLog("正在连接VPN服务器...")
	go func() {
		resp, err := t.client.ClientConnect()
		if err != nil {
			t.addLog("[red]连接失败: %v", err)
			return
		}
		if resp.Success {
			t.addLog("[green]%s", resp.Message)
		} else {
			t.addLog("[red]%s", resp.Error)
		}
	}()
}

func handleClientDisconnect(t *TUIApp) {
	t.addLog("正在断开VPN连接...")
	go func() {
		resp, err := t.client.ClientDisconnect()
		if err != nil {
			t.addLog("[red]断开失败: %v", err)
			return
		}
		if resp.Success {
			t.addLog("[green]%s", resp.Message)
		} else {
			t.addLog("[red]%s", resp.Error)
		}
	}()
}

func handleShowClientStatus(t *TUIApp) {
	status, err := t.client.ClientStatus()
	if err != nil {
		t.showInfoDialog("客户端状态", "获取失败: "+err.Error())
		return
	}

	var content strings.Builder
	if !status.Connected {
		content.WriteString("状态: [red]未连接[white]\n\n")
	} else {
		content.WriteString("状态: [green]已连接[white]\n\n")
		if status.AssignedIP != "" {
			content.WriteString(fmt.Sprintf("VPN IP: [green]%s[white]\n", status.AssignedIP))
		}
		if status.TUNDevice != "" {
			content.WriteString(fmt.Sprintf("TUN设备: %s\n", status.TUNDevice))
		}
	}
	content.WriteString(fmt.Sprintf("服务器: %s:%d\n", status.ServerAddress, status.ServerPort))

	t.showInfoDialog("客户端状态", content.String())
}

func handleSetServerAddress(t *TUIApp) {
	cfg, _ := t.client.ConfigGet()
	t.showInputDialog("服务器地址", cfg.ServerAddress, func(value string) {
		if value != "" {
			t.client.ConfigUpdate("server_address", value)
			t.addLog("[green]服务器地址已设置为: %s", value)
		}
		t.showMenu("client_settings")
	})
}

func handleSetServerPort(t *TUIApp) {
	cfg, _ := t.client.ConfigGet()
	t.showInputDialog("服务器端口", fmt.Sprintf("%d", cfg.ServerPort), func(value string) {
		if port, err := strconv.Atoi(value); err == nil && port > 0 && port < 65536 {
			t.client.ConfigUpdate("server_port", float64(port))
			t.addLog("[green]服务器端口已设置为: %d", port)
		} else {
			t.addLog("[red]无效的端口号")
		}
		t.showMenu("client_settings")
	})
}

func handleGenCSR(t *TUIApp) {
	t.showInputDialog("请输入客户端名称", "", func(clientName string) {
		if clientName == "" {
			t.addLog("[red]客户端名称不能为空")
			return
		}
		t.addLog("正在生成CSR...")
		go func() {
			resp, err := t.client.CertGenCSR(clientName)
			if err != nil {
				t.addLog("[red]生成CSR失败: %v", err)
				return
			}
			t.addLog("[green]========== CSR生成成功 ==========")
			t.addLog("私钥文件: %s", resp.KeyFile)
			t.addLog("CSR文件:  %s", resp.CSRFile)
			t.addLog("CN:       %s", resp.CN)
			t.addLog("[green]=================================")
		}()
	})
}

func handleRequestCert(t *TUIApp) {
	// 查找 CSR 文件
	files, _ := os.ReadDir(".")
	var csrFiles []string
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".csr") {
			csrFiles = append(csrFiles, f.Name())
		}
	}
	if len(csrFiles) == 0 {
		t.addLog("[yellow]未找到CSR文件，请先生成CSR")
		return
	}

	t.showInputDialog("CSR文件路径", csrFiles[0], func(csrFile string) {
		// 查找 Token 文件
		tokenFiles, _ := os.ReadDir(DefaultTokenDir)
		var validTokens []string
		for _, f := range tokenFiles {
			if strings.HasSuffix(f.Name(), ".json") {
				validTokens = append(validTokens, f.Name())
			}
		}

		defaultToken := ""
		if len(validTokens) > 0 {
			defaultToken = validTokens[0]
			t.addLog("可用Token文件:")
			for i, tf := range validTokens {
				t.addLog("  %d. %s", i+1, tf)
			}
		}

		t.showInputDialog("Token文件名", defaultToken, func(tokenFile string) {
			cfg, _ := t.client.ConfigGet()

			req := RequestCertRequest{
				CSRFile:       csrFile,
				ServerAddress: cfg.ServerAddress,
				ServerPort:    8081,
			}
			if tokenFile != "" {
				req.TokenFile = DefaultTokenDir + "/" + tokenFile
			}

			t.addLog("正在申请证书...")
			go func() {
				resp, err := t.client.CertRequest(req)
				if err != nil {
					t.addLog("[red]申请失败: %v", err)
					return
				}
				if resp.Success {
					t.addLog("[green]证书申请成功，已自动部署到 ./certs 目录")
				} else {
					t.addLog("[red]%s", resp.Error)
				}
			}()
		})
	})
}

func handleShowClientCertStatus(t *TUIApp) {
	list, err := t.client.CertList()
	if err != nil {
		t.showInfoDialog("本地证书状态", "获取失败: "+err.Error())
		return
	}

	var content strings.Builder
	content.WriteString(fmt.Sprintf("证书目录: %s\n\n", list.CertDir))

	certFiles := map[string]string{
		"ca.pem":         "CA证书 (服务端签发)",
		"client.pem":     "客户端证书",
		"client-key.pem": "客户端私钥",
	}

	for _, f := range list.Files {
		if desc, ok := certFiles[f.Name]; ok {
			if f.Exists {
				content.WriteString(fmt.Sprintf("[green]✓[white] %-20s %s\n", f.Name, desc))
				content.WriteString(fmt.Sprintf("    大小: %d bytes, 修改时间: %s\n", f.Size, f.Modified.Format("2006-01-02 15:04")))
			} else {
				content.WriteString(fmt.Sprintf("[red]✗[white] %-20s %s (不存在)\n", f.Name, desc))
			}
		}
	}

	t.showInfoDialog("本地证书状态", content.String())
}

// ================ 配置处理 ================

func handleSaveConfig(t *TUIApp) {
	resp, err := t.client.ConfigSave()
	if err != nil {
		t.addLog("[red]保存失败: %v", err)
	} else if resp.Success {
		t.addLog("[green]配置已保存")
	} else {
		t.addLog("[red]%s", resp.Error)
	}
}

func handleLoadConfig(t *TUIApp) {
	resp, err := t.client.ConfigLoad()
	if err != nil {
		t.addLog("[red]加载失败: %v", err)
	} else if resp.Success {
		t.addLog("[green]配置已加载")
		t.showMenu("config")
	} else {
		t.addLog("[red]%s", resp.Error)
	}
}

func handleShowConfig(t *TUIApp) {
	cfg, err := t.client.ConfigGet()
	if err != nil {
		t.showInfoDialog("当前配置", "获取失败: "+err.Error())
		return
	}

	var content strings.Builder
	content.WriteString("[yellow]网络配置:[white]\n")
	content.WriteString(fmt.Sprintf("  服务器地址:     %s\n", cfg.ServerAddress))
	content.WriteString(fmt.Sprintf("  服务器端口:     %d\n", cfg.ServerPort))
	content.WriteString(fmt.Sprintf("  VPN网段:        %s\n", cfg.Network))
	content.WriteString(fmt.Sprintf("  服务器IP:       %s\n", cfg.ServerIP))
	content.WriteString(fmt.Sprintf("  MTU:            %d\n", cfg.MTU))

	content.WriteString("\n[yellow]路由配置:[white]\n")
	content.WriteString(fmt.Sprintf("  路由模式:       %s\n", cfg.RouteMode))
	content.WriteString(fmt.Sprintf("  启用NAT:        %v\n", cfg.EnableNAT))
	content.WriteString(fmt.Sprintf("  NAT出口:        %s\n", cfg.NATInterface))
	content.WriteString(fmt.Sprintf("  重定向网关:     %v\n", cfg.RedirectGateway))
	content.WriteString(fmt.Sprintf("  DNS服务器:      %v\n", cfg.DNSServers))
	content.WriteString(fmt.Sprintf("  推送路由:       %v\n", cfg.PushRoutes))

	content.WriteString("\n[yellow]会话配置:[white]\n")
	content.WriteString(fmt.Sprintf("  最大连接数:     %d\n", cfg.MaxConnections))
	content.WriteString(fmt.Sprintf("  会话超时:       %v\n", cfg.SessionTimeout))
	content.WriteString(fmt.Sprintf("  心跳超时:       %v\n", cfg.KeepAliveTimeout))

	t.showInfoDialog("当前配置", content.String())
}

func handleResetConfig(t *TUIApp) {
	t.showConfirmDialog("确定恢复默认配置?", func(confirmed bool) {
		if confirmed {
			resp, _ := t.client.ConfigReset()
			if resp != nil && resp.Success {
				t.addLog("[green]已恢复默认配置")
			}
		}
	})
}

// ================ 向导处理 ================

func handleServerWizard(t *TUIApp) {
	t.addLog("开始服务端快速部署向导...")

	cfg, _ := t.client.ConfigGet()
	t.showInputDialog("VPN监听端口", fmt.Sprintf("%d", cfg.ServerPort), func(portStr string) {
		if port, err := strconv.Atoi(portStr); err == nil && port > 0 {
			t.client.ConfigUpdate("server_port", float64(port))
		}

		t.showInputDialog("VPN网段", cfg.Network, func(network string) {
			if network != "" {
				t.client.ConfigUpdate("network", network)
			}

			t.showConfirmDialog("启用全局NAT模式? (全部流量走VPN)", func(enableNAT bool) {
				if enableNAT {
					t.client.ConfigUpdate("route_mode", "full")
					t.client.ConfigUpdate("enable_nat", true)
					t.client.ConfigUpdate("redirect_gateway", true)
				} else {
					t.client.ConfigUpdate("route_mode", "split")
					t.client.ConfigUpdate("enable_nat", true)
					t.client.ConfigUpdate("redirect_gateway", false)
				}

				t.showConfirmDialog("开始初始化CA并启动服务端?", func(confirmed bool) {
					if confirmed {
						go func() {
							// 初始化 CA
							t.addLog("正在生成CA证书...")
							resp, err := t.client.CertInitCA()
							if err != nil {
								t.addLog("[red]生成证书失败: %v", err)
								return
							}
							if !resp.Success {
								t.addLog("[red]%s", resp.Error)
								return
							}
							t.addLog("[green]CA证书已生成")

							// 启动服务端
							time.Sleep(500 * time.Millisecond)
							handleServerStart(t)
							t.addLog("[green]服务端部署完成!")
						}()
					}
				})
			})
		})
	})
}

func handleClientWizard(t *TUIApp) {
	t.addLog("开始客户端快速配置向导...")

	cfg, _ := t.client.ConfigGet()
	t.showInputDialog("服务器地址", cfg.ServerAddress, func(addr string) {
		if addr != "" {
			t.client.ConfigUpdate("server_address", addr)
		}

		t.showInputDialog("服务器端口", fmt.Sprintf("%d", cfg.ServerPort), func(portStr string) {
			if port, err := strconv.Atoi(portStr); err == nil && port > 0 {
				t.client.ConfigUpdate("server_port", float64(port))
			}

			t.addLog("[green]配置已保存")

			// 检查证书
			list, _ := t.client.CertList()
			hasClientCert := false
			for _, f := range list.Files {
				if f.Name == "client.pem" && f.Exists {
					hasClientCert = true
					break
				}
			}

			if !hasClientCert {
				t.addLog("[yellow]未找到客户端证书，需要先申请证书")
				t.showConfirmDialog("是否现在生成CSR并申请证书?", func(confirmed bool) {
					if confirmed {
						t.navigateTo("client_cert")
					}
				})
			} else {
				t.showConfirmDialog("证书已就绪，是否立即连接VPN?", func(confirmed bool) {
					if confirmed {
						handleClientConnect(t)
					}
				})
			}
		})
	})
}

// ================ 系统处理 ================

func handleExit(t *TUIApp) {
	t.RequestExit()
}

// ================ 辅助函数 ================

func parseJSON(data []byte, v interface{}) {
	if data != nil {
		json.Unmarshal(data, v)
	}
}
