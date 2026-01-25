package main

import (
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ============================================================================
// Cyberpunk 配色方案
// ============================================================================

// 背景色系
var (
	ColorBgDeep   = tcell.NewRGBColor(13, 2, 33)   // #0D0221 深紫黑
	ColorBgMain   = tcell.NewRGBColor(20, 10, 45)  // #140A2D 主背景
	ColorBgPanel  = tcell.NewRGBColor(30, 15, 60)  // #1E0F3C 面板背景
	ColorBgSelect = tcell.NewRGBColor(60, 20, 100) // #3C1464 选中背景
)

// 霓虹色系
var (
	ColorNeonPink   = tcell.NewRGBColor(255, 0, 255) // #FF00FF 霓虹粉
	ColorNeonCyan   = tcell.NewRGBColor(0, 255, 255) // #00FFFF 电光蓝
	ColorNeonGreen  = tcell.NewRGBColor(57, 255, 20) // #39FF14 荧光绿
	ColorNeonYellow = tcell.NewRGBColor(255, 233, 0) // #FFE900 激光黄
	ColorNeonOrange = tcell.NewRGBColor(255, 95, 31) // #FF5F1F 霓虹橙
	ColorNeonPurple = tcell.NewRGBColor(191, 0, 255) // #BF00FF 霓虹紫
)

// 文字色系
var (
	ColorTextBright = tcell.NewRGBColor(255, 255, 255) // #FFFFFF 高亮白
	ColorTextNormal = tcell.NewRGBColor(200, 200, 220) // #C8C8DC 普通文字
	ColorTextDim    = tcell.NewRGBColor(120, 100, 150) // #786496 暗淡文字
)

// 语义化颜色别名
var (
	ColorAccent    = ColorNeonCyan   // 主强调色
	ColorAccentAlt = ColorNeonPink   // 次强调色
	ColorSuccess   = ColorNeonGreen  // 成功
	ColorWarning   = ColorNeonYellow // 警告
	ColorError     = ColorNeonOrange // 错误
	ColorSpecial   = ColorNeonPurple // 特殊

	ColorBorder       = ColorNeonPurple // 普通边框
	ColorBorderActive = ColorNeonCyan   // 激活边框

	ColorBgSecondary   = ColorBgPanel  // 次要背景
	ColorBgHighlight   = ColorBgSelect // 高亮背景
	ColorTextPrimary   = ColorTextBright
	ColorTextSecondary = ColorTextNormal
	ColorTextMuted     = ColorTextDim
)

// ============================================================================
// ASCII Art 标题
// ============================================================================

const ASCIILogo = `[#00FFFF]╔══════════════════════════════════════════════════════════════╗
║[#FF00FF]  ████████╗██╗     ███████╗   ██╗   ██╗██████╗ ███╗   ██╗  [#00FFFF]║
║[#FF00FF]  ╚══██╔══╝██║     ██╔════╝   ██║   ██║██╔══██╗████╗  ██║  [#00FFFF]║
║[#FF00FF]     ██║   ██║     ███████╗   ██║   ██║██████╔╝██╔██╗ ██║  [#00FFFF]║
║[#FF00FF]     ██║   ██║     ╚════██║   ╚██╗ ██╔╝██╔═══╝ ██║╚██╗██║  [#00FFFF]║
║[#FF00FF]     ██║   ███████╗███████║    ╚████╔╝ ██║     ██║ ╚████║  [#00FFFF]║
║[#FF00FF]     ╚═╝   ╚══════╝╚══════╝     ╚═══╝  ╚═╝     ╚═╝  ╚═══╝  [#00FFFF]║
╚══════════════════════════════════════════════════════════════╝[-]`

const ASCIILogoCompact = `[#00FFFF]╔════════════════════════════════════════════╗
║[#FF00FF]  ▀█▀ █   █▀   █ █ █▀█ █▄ █  [#00FFFF]║
║[#FF00FF]   █  █▄▄ ▄█   ▀▄▀ █▀▀ █ ▀█  [#00FFFF]║
╚════════════════════════════════════════════╝[-]`

// ============================================================================
// 菜单图标 (霓虹风格)
// ============================================================================

const (
	IconServer     = "◈ "
	IconClient     = "◇ "
	IconConfig     = "⚙ "
	IconWizard     = "★ "
	IconExit       = "✖ "
	IconStart      = "▶ "
	IconStop       = "■ "
	IconSettings   = "⚙ "
	IconCert       = "⬡ "
	IconToken      = "◆ "
	IconUsers      = "⬢ "
	IconKick       = "⊗ "
	IconStats      = "▤ "
	IconNetwork    = "◎ "
	IconRoute      = "➤ "
	IconDNS        = "◉ "
	IconSave       = "▼ "
	IconLoad       = "▲ "
	IconView       = "◐ "
	IconReset      = "↻ "
	IconAdd        = "+ "
	IconRemove     = "- "
	IconBack       = "◀ "
	IconConnect    = "⚡"
	IconDisconnect = "⦸ "
	IconGenerate   = "✦ "
	IconRequest    = "➚ "
	IconStatus     = "▣ "

	// 状态指示符
	StatusRunning      = "●"
	StatusStopped      = "○"
	StatusConnected    = "●"
	StatusDisconnected = "○"
)

// 应用信息
const (
	AppName    = "TLS VPN"
	AppVersion = "v1.0"
	AppTitle   = "⚡ " + AppName + " " + AppVersion
)

// ============================================================================
// Spinner 加载动画
// ============================================================================

// Spinner 加载动画组件
type Spinner struct {
	frames   []string
	current  int
	running  bool
	mu       sync.Mutex
	textView *tview.TextView
	prefix   string
	suffix   string
	stopCh   chan struct{}
	app      *tview.Application
}

// 霓虹风格 Spinner 帧
var SpinnerFrames = []string{
	"[#FF00FF]⠋[-]", "[#FF00FF]⠙[-]", "[#FF00FF]⠹[-]", "[#00FFFF]⠸[-]",
	"[#00FFFF]⠼[-]", "[#00FFFF]⠴[-]", "[#39FF14]⠦[-]", "[#39FF14]⠧[-]",
	"[#39FF14]⠇[-]", "[#FFE900]⠏[-]", "[#FFE900]⠋[-]", "[#FF00FF]⠙[-]",
}

var SpinnerFramesBlock = []string{
	"[#FF00FF]▰▱▱▱▱[-]", "[#FF00FF]▰▰▱▱▱[-]", "[#00FFFF]▰▰▰▱▱[-]",
	"[#00FFFF]▰▰▰▰▱[-]", "[#39FF14]▰▰▰▰▰[-]", "[#39FF14]▱▰▰▰▰[-]",
	"[#FFE900]▱▱▰▰▰[-]", "[#FFE900]▱▱▱▰▰[-]", "[#FF00FF]▱▱▱▱▰[-]",
}

// NewSpinner 创建新的 Spinner
func NewSpinner(app *tview.Application) *Spinner {
	tv := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	tv.SetBackgroundColor(ColorBgPanel)

	return &Spinner{
		frames:   SpinnerFrames,
		textView: tv,
		app:      app,
		stopCh:   make(chan struct{}),
	}
}

// SetText 设置 Spinner 文本
func (s *Spinner) SetText(prefix, suffix string) *Spinner {
	s.mu.Lock()
	s.prefix = prefix
	s.suffix = suffix
	s.mu.Unlock()
	return s
}

// GetView 获取 TextView
func (s *Spinner) GetView() *tview.TextView {
	return s.textView
}

// Start 启动动画
func (s *Spinner) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.stopCh = make(chan struct{})
	s.mu.Unlock()

	go func() {
		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-s.stopCh:
				return
			case <-ticker.C:
				s.mu.Lock()
				s.current = (s.current + 1) % len(s.frames)
				frame := s.frames[s.current]
				text := s.prefix + " " + frame + " " + s.suffix
				s.mu.Unlock()

				s.app.QueueUpdateDraw(func() {
					s.textView.SetText(text)
				})
			}
		}
	}()
}

// Stop 停止动画
func (s *Spinner) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		close(s.stopCh)
		s.running = false
	}
}

// SetFrames 设置动画帧
func (s *Spinner) SetFrames(frames []string) *Spinner {
	s.mu.Lock()
	s.frames = frames
	s.mu.Unlock()
	return s
}
