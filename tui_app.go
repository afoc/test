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
	app          *tview.Application
	pages        *tview.Pages
	menuList     *tview.List
	logView      *tview.TextView
	statusBar    *tview.TextView
	logBuffer    *LogBuffer
	menuStack    []string
	stopChan     chan struct{}
	client       *ControlClient // API 客户端
	lastLogSeq   uint64         // 最后获取的日志序号
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
	formatted := fmt.Sprintf("[gray]%s[white] %s", timestamp, line)
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
func NewTUIApp() *TUIApp {
	t := &TUIApp{
		app:       tview.NewApplication(),
		pages:     tview.NewPages(),
		logBuffer: NewLogBuffer(500),
		menuStack: []string{"main"},
		stopChan:  make(chan struct{}),
		client:    NewControlClient(),
	}
	t.setupUI()
	return t
}

// setupUI 设置 UI 布局
func (t *TUIApp) setupUI() {
	// 日志视图
	t.logView = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)
	t.logView.SetBorder(true).SetTitle(" 系统日志 (F2清空) ")

	// 菜单列表
	t.menuList = tview.NewList().
		SetHighlightFullLine(true).
		SetSelectedBackgroundColor(tcell.ColorDarkCyan).
		SetSelectedTextColor(tcell.ColorWhite)
	t.menuList.SetBorder(true)

	// 状态栏
	t.statusBar = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetWrap(false)
	t.statusBar.SetBackgroundColor(tcell.ColorDarkBlue)

	// 布局
	leftPanel := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(t.menuList, 0, 1, true)

	mainFlex := tview.NewFlex().
		AddItem(leftPanel, 40, 0, true).
		AddItem(t.logView, 0, 1, false)

	layout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(mainFlex, 0, 1, true).
		AddItem(t.statusBar, 1, 0, false)

	t.pages.AddPage("main", layout, true, true)

	// 全局按键
	t.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		frontPage, _ := t.pages.GetFrontPage()
		if frontPage == "input" || frontPage == "confirm" || frontPage == "info" {
			return event
		}

		switch event.Key() {
		case tcell.KeyF2:
			t.logBuffer.Clear()
			t.logView.Clear()
			t.logView.SetText("")
			t.addLog("日志已清空")
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
	serverStatus := "[red]未运行"
	clientStatus := "[red]未连接"

	// 通过 API 获取状态
	if t.client.IsServiceRunning() {
		if status, err := t.client.ServerStatus(); err == nil && status.Running {
			serverStatus = fmt.Sprintf("[green]运行中[white] (%d客户端)", status.ClientCount)
		}
		if status, err := t.client.ClientStatus(); err == nil && status.Connected {
			if status.AssignedIP != "" {
				clientStatus = fmt.Sprintf("[green]已连接[white] (%s)", status.AssignedIP)
			} else {
				clientStatus = "[yellow]连接中..."
			}
		}
	}

	t.statusBar.SetText(fmt.Sprintf(" [white]服务端: %s  │  客户端: %s  │  [gray]Tab切换 F2清日志 Esc返回 q退出", serverStatus, clientStatus))
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
		// 根据日志级别添加颜色
		var colorTag string
		switch entry.Level {
		case "error":
			colorTag = "[red]"
		case "warn":
			colorTag = "[yellow]"
		default:
			colorTag = "[white]"
		}

		// 格式化时间
		logTime := time.UnixMilli(entry.Time).Format("15:04:05")

		// 添加到日志缓冲区（带颜色和时间戳）
		formatted := fmt.Sprintf("[gray]%s %s%s[white]", logTime, colorTag, entry.Message)
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
		t.addLog("[red]未知菜单: %s", menuName)
		return
	}

	t.menuList.SetTitle(" " + menuDef.Title + " ")

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
		t.menuList.AddItem("[yellow]返回上级", "", 'b', func() {
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

	// 检查服务是否运行
	if !t.client.IsServiceRunning() {
		t.logBuffer.AddLine("[yellow]后台服务未运行，部分功能不可用")
		t.logBuffer.AddLine("[yellow]请先运行: ./tls-vpn --service")
		t.logView.SetText(t.logBuffer.GetContent())
	}

	go func() {
		time.Sleep(100 * time.Millisecond)
		t.startUpdater()
	}()

	return t.app.Run()
}

// Stop 停止应用
func (t *TUIApp) Stop() {
	// 使用 sync.Once 或检查来避免重复关闭
	select {
	case <-t.stopChan:
		// 已经关闭
	default:
		close(t.stopChan)
	}
	t.app.Stop()
}
