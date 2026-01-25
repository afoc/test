package main

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// TUIApp TUI 应用程序
type TUIApp struct {
	app        *tview.Application
	pages      *tview.Pages
	menuList   *tview.List
	logView    *tview.TextView
	statusBar  *tview.TextView
	logBuffer  *LogBuffer
	menuStack  []string
	stopChan   chan struct{}
	lastLogSeq uint64         // 最后获取的日志序号
	client     *ControlClient // IPC 客户端
}

// LogBuffer 环形日志缓冲区
type LogBuffer struct {
	lines    []string
	maxLines int
	mu       sync.Mutex
}

// NewLogBuffer 创建日志缓冲区
func NewLogBuffer(maxLines int) *LogBuffer {
	return &LogBuffer{
		lines:    make([]string, 0, maxLines),
		maxLines: maxLines,
	}
}

// Write 实现 io.Writer 接口
func (lb *LogBuffer) Write(p []byte) (n int, err error) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	for _, line := range strings.Split(strings.TrimRight(string(p), "\n"), "\n") {
		if line != "" {
			lb.addLineNoLock(line)
		}
	}
	return len(p), nil
}

// AddLine 添加日志行（自动添加时间戳）
func (lb *LogBuffer) AddLine(line string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.addLineNoLock(line)
}

// AddLineRaw 添加原始日志行（不添加时间戳，用于已格式化的日志）
func (lb *LogBuffer) AddLineRaw(line string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.lines = append(lb.lines, line)
	if len(lb.lines) > lb.maxLines {
		lb.lines = lb.lines[1:]
	}
}

func (lb *LogBuffer) addLineNoLock(line string) {
	timestamp := time.Now().Format("15:04:05")
	// 使用 Cyberpunk 配色
	formatted := fmt.Sprintf("[#786496]%s[-] [#C8C8DC]%s[-]", timestamp, line)
	lb.lines = append(lb.lines, formatted)
	if len(lb.lines) > lb.maxLines {
		lb.lines = lb.lines[1:]
	}
}

// GetContent 获取所有日志
func (lb *LogBuffer) GetContent() string {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	return strings.Join(lb.lines, "\n")
}

// Clear 清空日志
func (lb *LogBuffer) Clear() {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.lines = lb.lines[:0]
}

// NewTUIApp 创建 TUI 应用
func NewTUIApp(client *ControlClient) *TUIApp {
	t := &TUIApp{
		app:       tview.NewApplication(),
		pages:     tview.NewPages(),
		logBuffer: NewLogBuffer(500),
		menuStack: []string{"main"},
		stopChan:  make(chan struct{}),
		client:    client,
	}
	t.setupUI()
	return t
}

// setupUI 设置 UI 布局
func (t *TUIApp) setupUI() {
	// ASCII Art 标题
	logoView := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetText(ASCIILogoCompact)
	logoView.SetBackgroundColor(ColorBgDeep)

	// 日志视图 - 双线边框
	t.logView = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)
	t.logView.SetBorder(true).
		SetTitle(" [#00FFFF]◈[-] [#FF00FF]系统日志[-] ").
		SetTitleAlign(tview.AlignLeft).
		SetBorderColor(ColorNeonPurple).
		SetBackgroundColor(ColorBgMain).
		SetTitleColor(ColorNeonCyan)
	t.logView.SetBorderPadding(0, 0, 1, 1)

	// 菜单列表 - 双线边框
	t.menuList = tview.NewList().
		SetHighlightFullLine(true).
		SetSelectedBackgroundColor(ColorBgSelect).
		SetSelectedTextColor(ColorNeonCyan).
		SetMainTextColor(ColorTextNormal).
		SetSecondaryTextColor(ColorTextDim).
		SetShortcutColor(ColorNeonPink)
	t.menuList.SetBorder(true).
		SetBorderColor(ColorNeonPurple).
		SetBackgroundColor(ColorBgMain).
		SetTitleColor(ColorNeonCyan)
	t.menuList.SetBorderPadding(0, 0, 1, 1)

	// 状态栏 - 霓虹风格
	t.statusBar = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetWrap(false)
	t.statusBar.SetBackgroundColor(ColorBgPanel)
	t.statusBar.SetTextColor(ColorTextNormal)

	// 帮助栏 - 霓虹快捷键
	helpBar := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetText("[#FF00FF]Tab[-] 切换  [#00FFFF]F2[-] 清屏  [#39FF14]Esc[-] 返回  [#FFE900]q[-] 退出")
	helpBar.SetBackgroundColor(ColorBgDeep)
	helpBar.SetTextColor(ColorTextDim)

	// 左侧面板（菜单）
	leftPanel := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(t.menuList, 0, 1, true)

	// 主内容区
	mainFlex := tview.NewFlex().
		AddItem(leftPanel, 40, 0, true).
		AddItem(t.logView, 0, 1, false)

	// 整体布局
	layout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(logoView, 4, 0, false).
		AddItem(mainFlex, 0, 1, true).
		AddItem(t.statusBar, 1, 0, false).
		AddItem(helpBar, 1, 0, false)

	t.pages.AddPage("main", layout, true, true)

	// 设置全局背景色
	t.app.SetBeforeDrawFunc(func(screen tcell.Screen) bool {
		screen.Clear()
		return false
	})

	// 全局按键
	t.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		frontPage, _ := t.pages.GetFrontPage()
		if frontPage == "input" || frontPage == "confirm" || frontPage == "info" || frontPage == "exitDialog" || frontPage == "loading" {
			return event
		}

		switch event.Key() {
		case tcell.KeyF2:
			t.logBuffer.Clear()
			t.logView.Clear()
			t.logView.SetText("")
			t.addLog("[#39FF14]⚡ 日志已清空[-]")
			t.app.ForceDraw()
			return nil
		case tcell.KeyEsc:
			if len(t.menuStack) > 1 {
				t.menuStack = t.menuStack[:len(t.menuStack)-1]
				t.showMenu(t.menuStack[len(t.menuStack)-1])
				return nil
			}
		case tcell.KeyTab:
			if t.menuList.HasFocus() {
				t.app.SetFocus(t.logView)
			} else {
				t.app.SetFocus(t.menuList)
			}
			return nil
		}
		return event
	})

	t.app.SetRoot(t.pages, true)
	t.updateStatusBar()
}

// addLog 添加日志
func (t *TUIApp) addLog(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	t.logBuffer.AddLine(msg)
	go func() {
		t.app.QueueUpdateDraw(func() {
			t.logView.SetText(t.logBuffer.GetContent())
			t.logView.ScrollToEnd()
		})
	}()
}

// updateStatusBar 更新状态栏
func (t *TUIApp) updateStatusBar() {
	serverIcon := "[#FF5F1F]" + StatusStopped + "[-]" // 霓虹橙圆点
	serverStatus := "[#FF5F1F]未运行[-]"
	clientIcon := "[#FF5F1F]" + StatusDisconnected + "[-]"
	clientStatus := "[#FF5F1F]未连接[-]"

	if t.client.IsServiceRunning() {
		if status, err := t.client.ServerStatus(); err == nil && status.Running {
			serverIcon = "[#39FF14]" + StatusRunning + "[-]" // 荧光绿圆点
			serverStatus = fmt.Sprintf("[#39FF14]运行中[-] [#00FFFF](%d)[-]", status.ClientCount)
		}
		if status, err := t.client.ClientStatus(); err == nil && status.Connected {
			clientIcon = "[#39FF14]" + StatusConnected + "[-]"
			if status.AssignedIP != "" {
				clientStatus = fmt.Sprintf("[#39FF14]已连接[-] [#00FFFF](%s)[-]", status.AssignedIP)
			} else {
				clientStatus = "[#FFE900]连接中...[-]"
			}
		}
	}

	t.statusBar.SetText(fmt.Sprintf(" [#FF00FF]◈[-] 服务端: %s %s   [#BF00FF]│[-]   [#00FFFF]◇[-] 客户端: %s %s ", serverIcon, serverStatus, clientIcon, clientStatus))
}

// startUpdater 启动状态更新器
func (t *TUIApp) startUpdater() {
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		lastContent := ""
		for {
			select {
			case <-t.stopChan:
				return
			case <-ticker.C:
				// 从后台服务获取新日志
				t.fetchServiceLogs()

				content := t.logBuffer.GetContent()
				if content != lastContent {
					lastContent = content
					t.app.QueueUpdateDraw(func() {
						t.logView.SetText(content)
						t.logView.ScrollToEnd()
					})
				}
				t.app.QueueUpdateDraw(func() {
					t.updateStatusBar()
				})
			}
		}
	}()
}

// fetchServiceLogs 从后台服务获取日志
func (t *TUIApp) fetchServiceLogs() {
	if !t.client.IsServiceRunning() {
		return
	}

	resp, err := t.client.LogFetch(t.lastLogSeq, 50)
	if err != nil {
		return
	}

	// 添加新日志到本地缓冲区
	for _, entry := range resp.Logs {
		// 根据日志级别使用 Cyberpunk 配色
		var colorTag string
		switch entry.Level {
		case "error":
			colorTag = "[#FF5F1F]" // 霓虹橙
		case "warn":
			colorTag = "[#FFE900]" // 激光黄
		default:
			colorTag = "[#C8C8DC]" // 普通文字
		}

		// 格式化时间
		logTime := time.UnixMilli(entry.Time).Format("15:04:05")

		// 添加到日志缓冲区（带颜色和时间戳）
		formatted := fmt.Sprintf("[#786496]%s[-] %s%s[-]", logTime, colorTag, entry.Message)
		t.logBuffer.AddLineRaw(formatted)
	}

	// 更新最后序号
	if resp.LastSeq > t.lastLogSeq {
		t.lastLogSeq = resp.LastSeq
	}
}

// showMenu 显示菜单
func (t *TUIApp) showMenu(menuName string) {
	t.menuList.Clear()

	menus := GetMenus()
	menuDef, ok := menus[menuName]
	if !ok {
		t.addLog("[#FF5F1F]✖ 未知菜单: %s[-]", menuName)
		return
	}

	t.menuList.SetTitle(" [#00FFFF]" + menuDef.Title + "[-] ")

	for _, item := range menuDef.Items {
		itemCopy := item // 避免闭包问题
		t.menuList.AddItem(itemCopy.Label, itemCopy.Desc, itemCopy.Shortcut, func() {
			if itemCopy.NavTo != "" {
				// 导航到子菜单
				t.navigateTo(itemCopy.NavTo)
			} else if itemCopy.Handler != nil {
				// 执行处理函数
				itemCopy.Handler(t)
			}
		})
	}

	// 自动添加返回上级
	if menuDef.Parent != "" {
		t.menuList.AddItem("[#FFE900]◀ 返回上级[-]", "", 'b', func() {
			if len(t.menuStack) > 1 {
				t.menuStack = t.menuStack[:len(t.menuStack)-1]
				t.showMenu(t.menuStack[len(t.menuStack)-1])
			}
		})
	}

	t.menuList.SetCurrentItem(0)
	t.app.SetFocus(t.menuList)
	t.app.ForceDraw()
}

// navigateTo 导航到子菜单
func (t *TUIApp) navigateTo(menuName string) {
	t.menuStack = append(t.menuStack, menuName)
	t.showMenu(menuName)
}

// Run 运行应用
func (t *TUIApp) Run() error {
	t.showMenu("main")
	t.logBuffer.AddLine("TLS VPN 管理系统已启动")
	t.logView.SetText(t.logBuffer.GetContent())

	go func() {
		time.Sleep(100 * time.Millisecond)
		t.startUpdater()
	}()

	return t.app.Run()
}

// Stop 停止应用
func (t *TUIApp) Stop() {
	select {
	case <-t.stopChan:
		return // 已经关闭
	default:
		close(t.stopChan)
	}
	t.app.Stop()
}

// RequestExit 请求退出
func (t *TUIApp) RequestExit() {
	// 检查是否有活跃的 VPN 服务
	hasActiveVPN := false
	if t.client.IsServiceRunning() {
		if status, err := t.client.ServerStatus(); err == nil && status.Running {
			hasActiveVPN = true
		}
		if status, err := t.client.ClientStatus(); err == nil && status.Connected {
			hasActiveVPN = true
		}
	}

	if hasActiveVPN {
		// 有 VPN 运行，显示对话框让用户选择
		t.showExitDialog()
	} else {
		// 没有 VPN 运行，直接停止后台服务并退出
		t.addLog("正在停止后台服务...")
		t.client.Shutdown()
		t.Stop()
	}
}

// showExitDialog 显示退出确认对话框（仅当有 VPN 运行时）
func (t *TUIApp) showExitDialog() {
	modal := tview.NewModal().
		SetText("[#00FFFF]⚡[-] VPN 服务正在运行\n\n退出后服务将继续在后台运行").
		AddButtons([]string{"退出界面", "停止服务", "取消"}).
		SetBackgroundColor(ColorBgPanel).
		SetTextColor(ColorTextBright).
		SetButtonBackgroundColor(ColorNeonCyan).
		SetButtonTextColor(ColorBgDeep).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			t.pages.RemovePage("exitDialog")
			t.app.SetFocus(t.menuList)

			switch buttonIndex {
			case 0: // 退出管理界面（服务继续后台运行）
				t.addLog("[#39FF14]⚡ 退出管理界面，服务继续后台运行[-]")
				t.Stop()
			case 1: // 停止后台服务
				t.addLog("[#FFE900]⏳ 正在停止后台服务...[-]")
				if _, err := t.client.Shutdown(); err != nil {
					t.addLog("[#FF5F1F]✖ 停止失败: %v[-]", err)
				} else {
					t.addLog("[#39FF14]✔ 后台服务已停止[-]")
				}
				t.Stop()
			case 2: // 取消
				t.addLog("[#786496]取消退出[-]")
			}
		})

	t.pages.AddPage("exitDialog", modal, true, true)
	t.app.SetFocus(modal)
}
