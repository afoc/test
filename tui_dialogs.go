package main

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// showInputDialog 显示输入对话框
func (t *TUIApp) showInputDialog(title, defaultValue string, callback func(string)) {
	previousFocus := t.menuList

	inputField := tview.NewInputField().
		SetText(defaultValue).
		SetFieldWidth(35).
		SetFieldBackgroundColor(ColorBgSelect).
		SetFieldTextColor(ColorTextBright).
		SetPlaceholder("请输入...").
		SetPlaceholderTextColor(ColorTextDim).
		SetLabelColor(ColorNeonCyan)

	container := tview.NewFlex().SetDirection(tview.FlexRow)

	titleText := tview.NewTextView().
		SetDynamicColors(true).
		SetText("[#FF00FF]✦[-] " + title).
		SetTextAlign(tview.AlignCenter)
	titleText.SetBackgroundColor(ColorBgPanel)
	titleText.SetTextColor(ColorNeonCyan)

	inputRow := tview.NewFlex().
		AddItem(nil, 2, 0, false).
		AddItem(inputField, 0, 1, true).
		AddItem(nil, 2, 0, false)

	hintText := tview.NewTextView().
		SetDynamicColors(true).
		SetText("[#00FFFF]Enter[-] 确认    [#FF00FF]Esc[-] 取消").
		SetTextAlign(tview.AlignCenter)
	hintText.SetBackgroundColor(ColorBgPanel)
	hintText.SetTextColor(ColorTextDim)

	container.
		AddItem(titleText, 1, 0, false).
		AddItem(nil, 1, 0, false).
		AddItem(inputRow, 1, 0, true).
		AddItem(nil, 1, 0, false).
		AddItem(hintText, 1, 0, false)

	container.SetBorder(true).
		SetBorderColor(ColorNeonCyan).
		SetBackgroundColor(ColorBgPanel).
		SetBorderPadding(1, 1, 2, 2)

	inputField.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			value := inputField.GetText()
			t.pages.RemovePage("input")
			t.app.SetFocus(previousFocus)
			t.app.ForceDraw()
			callback(value)
		}
	})

	inputField.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			t.pages.RemovePage("input")
			t.app.SetFocus(previousFocus)
			t.app.ForceDraw()
			return nil
		}
		return event
	})

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(container, 9, 0, true).
			AddItem(nil, 0, 1, false), 50, 0, true).
		AddItem(nil, 0, 1, false)

	t.pages.AddPage("input", modal, true, true)
	t.app.SetFocus(inputField)
}

// showConfirmDialog 显示确认对话框
func (t *TUIApp) showConfirmDialog(message string, callback func(bool)) {
	previousFocus := t.menuList

	modal := tview.NewModal().
		SetText("[#FFE900]⚠[-] " + message).
		AddButtons([]string{"  确定  ", "  取消  "}).
		SetBackgroundColor(ColorBgPanel).
		SetTextColor(ColorTextBright).
		SetButtonBackgroundColor(ColorNeonCyan).
		SetButtonTextColor(ColorBgDeep).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			t.pages.RemovePage("confirm")
			t.app.SetFocus(previousFocus)
			t.app.ForceDraw()
			callback(buttonIndex == 0)
		})

	t.pages.AddPage("confirm", modal, true, true)
}

// showInfoDialog 显示信息对话框
func (t *TUIApp) showInfoDialog(title, content string) {
	previousFocus := t.menuList

	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetText(content).
		SetScrollable(true).
		SetWrap(true)
	textView.SetBackgroundColor(ColorBgPanel)
	textView.SetTextColor(ColorTextNormal)

	hintText := tview.NewTextView().
		SetDynamicColors(true).
		SetText("[#00FFFF]↑↓[-] 滚动    [#FF00FF]Enter/Esc[-] 关闭").
		SetTextAlign(tview.AlignCenter)
	hintText.SetBackgroundColor(ColorBgPanel)
	hintText.SetTextColor(ColorTextDim)

	container := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(textView, 0, 1, true).
		AddItem(hintText, 1, 0, false)

	container.SetBorder(true).
		SetTitle(" [#FF00FF]◈[-] " + title + " ").
		SetTitleAlign(tview.AlignCenter).
		SetTitleColor(ColorNeonCyan).
		SetBorderColor(ColorNeonCyan).
		SetBackgroundColor(ColorBgPanel).
		SetBorderPadding(0, 0, 1, 1)

	textView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape || event.Key() == tcell.KeyEnter {
			t.pages.RemovePage("info")
			t.app.SetFocus(previousFocus)
			t.app.ForceDraw()
			return nil
		}
		return event
	})

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(container, 0, 3, true).
			AddItem(nil, 0, 1, false), 0, 2, true).
		AddItem(nil, 0, 1, false)

	t.pages.AddPage("info", modal, true, true)
	t.app.SetFocus(textView)
}

// showLoadingDialog 显示加载动画对话框
func (t *TUIApp) showLoadingDialog(message string) *Spinner {
	spinner := NewSpinner(t.app)
	spinner.SetText("[#00FFFF]"+message+"[-]", "")

	container := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(spinner.GetView(), 1, 0, false).
		AddItem(nil, 0, 1, false)

	container.SetBorder(true).
		SetBorderColor(ColorNeonPink).
		SetBackgroundColor(ColorBgPanel).
		SetBorderPadding(1, 1, 2, 2)

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(container, 5, 0, false).
			AddItem(nil, 0, 1, false), 40, 0, false).
		AddItem(nil, 0, 1, false)

	t.pages.AddPage("loading", modal, true, true)
	spinner.Start()

	return spinner
}

// hideLoadingDialog 隐藏加载动画对话框
func (t *TUIApp) hideLoadingDialog(spinner *Spinner) {
	if spinner != nil {
		spinner.Stop()
	}
	t.pages.RemovePage("loading")
	t.app.SetFocus(t.menuList)
}
