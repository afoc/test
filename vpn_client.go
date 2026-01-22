package main

import (
	"crypto/tls"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"github.com/songgao/water"
)

// VPNClient VPN客户端结构
type VPNClient struct {
	tlsConfig      *tls.Config
	conn           *tls.Conn
	connMutex      sync.Mutex
	assignedIP     net.IP
	running        bool
	reconnect      bool
	config         VPNConfig
	packetHandler  func([]byte) error
	heartbeatStop  chan struct{}
	heartbeatMutex sync.Mutex
	tunDevice      *water.Interface
	sendSeq        uint32        // 新增：发送序列号
	recvSeq        uint32        // 新增：接收序列号
	seqMutex       sync.Mutex    // 新增：序列号锁
	routeManager   *RouteManager // 新增：路由管理器
}

// NewVPNClient 创建新的VPN客户端
func NewVPNClient(certManager *CertificateManager, config VPNConfig) *VPNClient {
	return &VPNClient{
		tlsConfig:     certManager.ClientTLSConfig(),
		running:       true,
		reconnect:     true,
		config:        config,
		packetHandler: nil,
	}
}

// InitializeTUN 初始化TUN设备
func (c *VPNClient) InitializeTUN() error {
	// 检查root权限
	if err := checkRootPrivileges(); err != nil {
		return err
	}

	// 创建TUN设备（自动选择可用名称）
	tun, err := createTUNDevice("tun")
	if err != nil {
		return err
	}
	c.tunDevice = tun

	log.Printf("客户端TUN设备已创建: %s，等待IP分配...", tun.Name())
	return nil
}

// ConfigureTUN 配置TUN设备（在获得IP后调用）
func (c *VPNClient) ConfigureTUN() error {
	if c.tunDevice == nil {
		return fmt.Errorf("TUN设备未创建")
	}
	if c.assignedIP == nil {
		return fmt.Errorf("未分配IP地址")
	}

	// 配置TUN设备IP地址
	ipAddr := fmt.Sprintf("%s/24", c.assignedIP.String())
	if err := configureTUNDevice(c.tunDevice.Name(), ipAddr, c.config.MTU); err != nil {
		return err
	}

	log.Printf("客户端TUN设备已配置: %s", ipAddr)
	return nil
}

// Connect 连接到VPN服务器
func (c *VPNClient) Connect() error {
	address := fmt.Sprintf("%s:%d", c.config.ServerAddress, c.config.ServerPort)

	conn, err := tls.Dial("tcp", address, c.tlsConfig)
	if err != nil {
		return fmt.Errorf("连接失败: %v", err)
	}

	err = conn.Handshake()
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("TLS握手失败: %v", err)
	}

	// 验证TLS版本
	if conn.ConnectionState().Version != tls.VersionTLS13 {
		_ = conn.Close()
		return fmt.Errorf("未使用TLS 1.3协议")
	}

	c.connMutex.Lock()
	c.conn = conn
	c.connMutex.Unlock()
	log.Println("成功连接到VPN服务器，使用TLS 1.3协议")

	// 读取分配的IP - 读取新的消息头格式（13字节）
	header := make([]byte, 13)
	_, err = io.ReadFull(c.conn, header)
	if err != nil {
		return fmt.Errorf("读取消息头失败: %v", err)
	}

	// 手动解析消息头
	msgType := MessageType(header[0])
	length := binary.BigEndian.Uint32(header[1:5])

	// 读取消息体
	payload := make([]byte, length)
	_, err = io.ReadFull(c.conn, payload)
	if err != nil {
		return fmt.Errorf("读取消息体失败: %v", err)
	}

	if msgType == MessageTypeIPAssignment && len(payload) >= 4 {
		c.assignedIP = net.IP(payload)
		log.Printf("分配的VPN IP: %s", c.assignedIP)
	} else {
		return fmt.Errorf("未收到有效的IP分配信息: type=%d, length=%d", msgType, length)
	}

	// 接收服务器推送的配置
	header = make([]byte, 13)
	_, err = io.ReadFull(c.conn, header)
	if err != nil {
		log.Printf("警告：未接收到服务器配置: %v", err)
		return nil // 配置是可选的，不影响连接
	}

	msgType = MessageType(header[0])
	length = binary.BigEndian.Uint32(header[1:5])

	if msgType == MessageTypeControl && length > 0 {
		payload = make([]byte, length)
		_, err = io.ReadFull(c.conn, payload)
		if err != nil {
			log.Printf("警告：读取服务器配置失败: %v", err)
			return nil
		}

		var serverConfig ClientConfig
		if err := json.Unmarshal(payload, &serverConfig); err != nil {
			log.Printf("警告：解析服务器配置失败: %v", err)
		} else {
			// 应用服务器推送的路由配置
			if serverConfig.RouteMode != "" {
				c.config.RouteMode = serverConfig.RouteMode
				log.Printf("服务器配置路由模式: %s", serverConfig.RouteMode)
			}
			if len(serverConfig.ExcludeRoutes) > 0 {
				c.config.ExcludeRoutes = serverConfig.ExcludeRoutes
			}
			c.config.RedirectGateway = serverConfig.RedirectGateway
			c.config.RedirectDNS = serverConfig.RedirectDNS
			if len(serverConfig.DNS) > 0 {
				c.config.DNSServers = serverConfig.DNS
			}
			if len(serverConfig.Routes) > 0 {
				c.config.PushRoutes = serverConfig.Routes
			}
			// 保存ServerIP供路由设置使用
			if serverConfig.ServerIP != "" {
				c.config.ServerIP = serverConfig.ServerIP
			}
			log.Printf("已应用服务器配置: RouteMode=%s, RedirectGateway=%v, RedirectDNS=%v",
				c.config.RouteMode, c.config.RedirectGateway, c.config.RedirectDNS)
		}
	}

	return nil
}

// SendData 发送数据
func (c *VPNClient) SendData(data []byte) error {
	c.connMutex.Lock()
	conn := c.conn
	c.connMutex.Unlock()

	if conn == nil {
		return fmt.Errorf("连接未建立")
	}

	// 获取并递增发送序列号
	c.seqMutex.Lock()
	seq := c.sendSeq
	c.sendSeq++
	c.seqMutex.Unlock()

	// 计算校验和（可选）
	checksum := uint32(0)
	if len(data) > 0 {
		checksum = crc32.ChecksumIEEE(data)
	}

	msg := &Message{
		Type:     MessageTypeData,
		Length:   uint32(len(data)),
		Sequence: seq,
		Checksum: checksum,
		Payload:  data,
	}

	serialized, err := msg.Serialize()
	if err != nil {
		return fmt.Errorf("序列化消息失败: %v", err)
	}

	_, err = conn.Write(serialized)
	return err
}

// SendHeartbeat 发送心跳
func (c *VPNClient) SendHeartbeat() error {
	c.connMutex.Lock()
	conn := c.conn
	c.connMutex.Unlock()

	if conn == nil {
		return fmt.Errorf("连接未建立")
	}

	msg := &Message{
		Type:     MessageTypeHeartbeat,
		Length:   0,
		Sequence: 0, // 心跳不使用序列号
		Checksum: 0,
		Payload:  []byte{},
	}

	serialized, err := msg.Serialize()
	if err != nil {
		return fmt.Errorf("序列化心跳消息失败: %v", err)
	}

	_, err = conn.Write(serialized)
	return err
}

// ReceiveData 接收数据，返回消息类型和数据
func (c *VPNClient) ReceiveData() (MessageType, []byte, error) {
	c.connMutex.Lock()
	conn := c.conn
	c.connMutex.Unlock()

	if conn == nil {
		return 0, nil, fmt.Errorf("连接未建立")
	}

	// 读取消息头（13字节）
	header := make([]byte, 13)
	_, err := io.ReadFull(conn, header)
	if err != nil {
		return 0, nil, err
	}

	// 手动解析消息头
	msgType := MessageType(header[0])
	length := binary.BigEndian.Uint32(header[1:5])
	sequence := binary.BigEndian.Uint32(header[5:9])
	checksum := binary.BigEndian.Uint32(header[9:13])

	// 读取消息体
	payload := make([]byte, length)
	if length > 0 {
		_, err = io.ReadFull(conn, payload)
		if err != nil {
			return 0, nil, err
		}
	}

	// 验证序列号（心跳和IP分配消息除外）
	if msgType != MessageTypeHeartbeat && msgType != MessageTypeIPAssignment {
		c.seqMutex.Lock()
		// 检测重放攻击（序列号回退）
		if sequence < c.recvSeq {
			c.seqMutex.Unlock()
			return 0, nil, fmt.Errorf("检测到重放攻击：期望序列号 >= %d，收到 %d", c.recvSeq, sequence)
		}
		// 检测消息丢失（序列号跳跃）
		if sequence > c.recvSeq+1 && c.recvSeq > 0 {
			log.Printf("警告：检测到消息丢失，期望序列号 %d，收到 %d", c.recvSeq+1, sequence)
		}
		c.recvSeq = sequence
		c.seqMutex.Unlock()
	}

	// 验证校验和（如果提供）
	if checksum != 0 && len(payload) > 0 {
		actualChecksum := crc32.ChecksumIEEE(payload)
		if actualChecksum != checksum {
			return 0, nil, fmt.Errorf("消息校验和不匹配: 期望 %d, 收到 %d", actualChecksum, checksum)
		}
	}

	return msgType, payload, nil
}

// Run 运行客户端
func (c *VPNClient) Run() {
	for c.running && c.reconnect {
		err := c.Connect()
		if err != nil {
			log.Printf("连接失败: %v，%v秒后重试", err, c.config.ReconnectDelay/time.Second)
			time.Sleep(c.config.ReconnectDelay)
			continue
		}

		log.Println("VPN客户端已连接，开始数据传输...")

		// 如果有TUN设备，配置它
		if c.tunDevice != nil && c.assignedIP != nil {
			if err := c.ConfigureTUN(); err != nil {
				log.Printf("配置TUN设备失败: %v", err)
				c.Close()
				continue
			}

			// 配置路由
			if err := c.setupRoutes(); err != nil {
				log.Printf("配置路由失败: %v", err)
				c.Close()
				continue
			}
		}

		// 初始化心跳停止通道
		c.heartbeatMutex.Lock()
		c.heartbeatStop = make(chan struct{})
		c.heartbeatMutex.Unlock()

		// 启动心跳协程
		go c.startHeartbeat()

		// 如果有TUN设备，启动TUN读取协程
		if c.tunDevice != nil {
			go c.handleTUNRead()
		}

		// 数据传输循环
		c.dataLoop()

		// 停止心跳协程
		c.stopHeartbeat()

		if c.reconnect {
			log.Println("连接断开，尝试重连...")
			time.Sleep(c.config.ReconnectDelay)
		}
	}

	log.Println("VPN客户端已退出")
}

// startHeartbeat 开始心跳
func (c *VPNClient) startHeartbeat() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	c.heartbeatMutex.Lock()
	stopChan := c.heartbeatStop
	c.heartbeatMutex.Unlock()

	for {
		select {
		case <-ticker.C:
			c.connMutex.Lock()
			conn := c.conn
			c.connMutex.Unlock()

			if conn == nil {
				return
			}

			// 发送心跳包
			err := c.SendHeartbeat()
			if err != nil {
				log.Printf("发送心跳失败: %v", err)
				return
			}
		case <-stopChan:
			return
		}
	}
}

// stopHeartbeat 停止心跳协程
func (c *VPNClient) stopHeartbeat() {
	c.heartbeatMutex.Lock()
	defer c.heartbeatMutex.Unlock()
	if c.heartbeatStop != nil {
		close(c.heartbeatStop)
		c.heartbeatStop = nil
	}
}

// handleTUNRead 处理从TUN设备读取的数据
func (c *VPNClient) handleTUNRead() {
	packet := make([]byte, c.config.MTU)

	for c.running {
		n, err := c.tunDevice.Read(packet)
		if err != nil {
			if c.running {
				log.Printf("从TUN设备读取失败: %v", err)
			}
			break
		}

		if n < 20 { // IP header minimum size
			continue
		}

		// 发送数据包到服务器
		err = c.SendData(packet[:n])
		if err != nil {
			log.Printf("发送数据包失败: %v", err)
			break
		}
	}
}

// dataLoop 数据传输循环
func (c *VPNClient) dataLoop() {
	for c.running {
		c.connMutex.Lock()
		conn := c.conn
		c.connMutex.Unlock()

		if conn == nil {
			break
		}

		_ = conn.SetReadDeadline(time.Now().Add(c.config.KeepAliveTimeout))

		msgType, data, err := c.ReceiveData()
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				log.Printf("连接超时")
				break
			}
			log.Printf("读取数据失败: %v", err)
			break
		}

		// 处理心跳响应 - 不打印日志
		if msgType == MessageTypeHeartbeat {
			continue
		}

		// 处理控制消息
		if msgType == MessageTypeControl {
			if len(data) > 0 {
				var config ClientConfig
				if err := json.Unmarshal(data, &config); err != nil {
					log.Printf("解析服务器配置失败: %v", err)
				} else {
					if err := c.applyServerConfig(&config); err != nil {
						log.Printf("应用服务器配置失败: %v", err)
					}
				}
			}
			continue
		}

		// 处理数据包
		if msgType == MessageTypeData && data != nil && len(data) > 0 {
			if c.tunDevice != nil {
				// 写入TUN设备
				_, err := c.tunDevice.Write(data)
				if err != nil {
					log.Printf("写入TUN设备失败: %v", err)
				}
			} else if c.packetHandler != nil {
				// 使用自定义处理器
				err := c.packetHandler(data)
				if err != nil {
					log.Printf("处理数据包失败: %v", err)
				}
			} else {
				// 默认处理：打印数据包信息
				log.Printf("接收到数据包，长度: %d, 内容: %s", len(data), hex.EncodeToString(data[:min(len(data), 16)]))
			}
		}
	}
}

// setupRoutes 设置路由（根据配置模式）
func (c *VPNClient) setupRoutes() error {
	// 创建路由管理器
	rm, err := NewRouteManager()
	if err != nil {
		return fmt.Errorf("创建路由管理器失败: %v", err)
	}
	c.routeManager = rm

	// 获取服务器IP（去掉CIDR后缀）
	serverIP := c.config.ServerAddress

	// 添加到VPN服务器的路由，确保不走VPN
	serverRoute := serverIP + "/32"
	if err := rm.AddRoute(serverRoute, rm.defaultGateway, rm.defaultIface); err != nil {
		log.Printf("警告：添加到VPN服务器的路由失败: %v", err)
	}

	// 根据路由模式设置路由
	switch c.config.RouteMode {
	case "full":
		return c.setupFullTunnelRoutes(rm)
	case "split":
		return c.setupSplitTunnelRoutes(rm)
	default:
		log.Printf("警告：未知的路由模式 %s，使用分流模式", c.config.RouteMode)
		return c.setupSplitTunnelRoutes(rm)
	}
}

// setupFullTunnelRoutes 设置全流量模式路由
func (c *VPNClient) setupFullTunnelRoutes(rm *RouteManager) error {
	log.Println("配置全流量代理模式...")

	// 获取VPN网关（服务器在VPN网络中的IP）
	vpnGateway := ""
	if c.config.ServerIP != "" {
		// 解析服务器IP，去掉CIDR
		for i := 0; i < len(c.config.ServerIP); i++ {
			if c.config.ServerIP[i] == '/' {
				vpnGateway = c.config.ServerIP[:i]
				break
			}
		}
	}
	if vpnGateway == "" {
		vpnGateway = "10.8.0.1" // 默认值
	}

	// 添加 0.0.0.0/1 和 128.0.0.0/1 路由（覆盖所有IP）
	tunDeviceName := c.tunDevice.Name()
	routes := []string{"0.0.0.0/1", "128.0.0.0/1"}
	for _, route := range routes {
		// 检查是否被排除
		if c.isExcluded(route) {
			log.Printf("跳过被排除的路由: %s", route)
			continue
		}

		if err := rm.AddRoute(route, vpnGateway, tunDeviceName); err != nil {
			log.Printf("警告：添加路由 %s 失败: %v", route, err)
		}
	}

	// 处理排除路由 - 添加到原始网关
	for _, excludeRoute := range c.config.ExcludeRoutes {
		if err := rm.AddRoute(excludeRoute, rm.defaultGateway, rm.defaultIface); err != nil {
			log.Printf("警告：添加排除路由 %s 失败: %v", excludeRoute, err)
		}
	}

	// 配置DNS（如果启用）
	if c.config.RedirectDNS && len(c.config.DNSServers) > 0 {
		if err := rm.SaveDNS(); err != nil {
			log.Printf("警告：保存DNS配置失败: %v", err)
		} else {
			if err := rm.SetDNS(c.config.DNSServers); err != nil {
				log.Printf("警告：设置DNS失败: %v", err)
			}
		}
	}

	log.Println("全流量代理模式配置完成")
	return nil
}

// setupSplitTunnelRoutes 设置分流模式路由
func (c *VPNClient) setupSplitTunnelRoutes(rm *RouteManager) error {
	log.Println("配置分流模式...")

	// 获取VPN网关
	vpnGateway := ""
	if c.config.ServerIP != "" {
		for i := 0; i < len(c.config.ServerIP); i++ {
			if c.config.ServerIP[i] == '/' {
				vpnGateway = c.config.ServerIP[:i]
				break
			}
		}
	}
	if vpnGateway == "" {
		vpnGateway = "10.8.0.1"
	}

	// 只添加 push_routes 中的路由
	tunDeviceName := c.tunDevice.Name()
	for _, route := range c.config.PushRoutes {
		if err := rm.AddRoute(route, vpnGateway, tunDeviceName); err != nil {
			log.Printf("警告：添加路由 %s 失败: %v", route, err)
		}
	}

	log.Printf("分流模式配置完成，已添加 %d 条路由", len(c.config.PushRoutes))
	return nil
}

// isExcluded 检查路由是否被排除
func (c *VPNClient) isExcluded(route string) bool {
	for _, excludeRoute := range c.config.ExcludeRoutes {
		if route == excludeRoute {
			return true
		}
	}
	return false
}

// Close 关闭客户端
func (c *VPNClient) Close() {
	c.running = false
	c.reconnect = false

	// 停止心跳协程
	c.stopHeartbeat()

	// 清理路由和DNS
	if c.routeManager != nil {
		c.routeManager.CleanupRoutes()
		_ = c.routeManager.RestoreDNS()
	}

	c.connMutex.Lock()
	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}
	c.connMutex.Unlock()

	// 清理TUN设备
	if c.tunDevice != nil {
		deviceName := c.tunDevice.Name()
		_ = c.tunDevice.Close()
		cleanupTUNDevice(deviceName)
	}
}

// applyServerConfig 应用服务器推送的配置
func (c *VPNClient) applyServerConfig(config *ClientConfig) error {
	log.Printf("收到服务器配置: DNS=%v, Routes=%v, MTU=%d", config.DNS, config.Routes, config.MTU)

	// 记录DNS服务器（可以修改/etc/resolv.conf，但需谨慎）
	for _, dns := range config.DNS {
		log.Printf("推荐DNS: %s", dns)
	}

	// 添加路由
	tunDeviceName := c.tunDevice.Name()
	for _, route := range config.Routes {
		// 解析服务器IP以获取网关（去掉CIDR后缀）
		serverIPStr := config.ServerIP
		for i := 0; i < len(serverIPStr); i++ {
			if serverIPStr[i] == '/' {
				serverIPStr = serverIPStr[:i]
				break
			}
		}

		output, err := runCmdCombined("ip", "route", "add", route, "via", serverIPStr, "dev", tunDeviceName)
		if err != nil {
			log.Printf("警告：添加路由 %s 失败: %v, 输出: %s", route, err, string(output))
		} else {
			log.Printf("已添加路由: %s via %s dev %s", route, serverIPStr, tunDeviceName)
		}
	}

	return nil
}
