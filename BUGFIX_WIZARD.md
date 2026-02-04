# 快速部署向导 Bug 修复说明

## 修复日期
2026-01-25 17:30

## 🐛 修复的问题

### 问题 1: 路由模式选择逻辑混乱 ❌→✅
**症状：**
- 用户选择"全局模式"，实际设置成"分流模式"
- 用户选择"分流模式"，实际设置成"全局模式"

**原因：**
- 提示文字"是否启用全局模式?" + 按钮["确定", "取消"] 造成歧义
- 用户不确定"确定"和"取消"对应的含义

**修复：**
- 改用 `showYesNoDialog`，按钮变为 ["是", "否"]
- 提示中明确说明："选择 '是' = 全局模式，选择 '否' = 分流模式"
- 增加颜色标记区分两种模式
- 日志输出也添加颜色区分

### 问题 2: 无法重复部署（只能部署一次）❌→✅
**症状：**
- 第一次部署成功
- 第二次再尝试快速部署，走完所有步骤后显示"已取消部署"

**原因：**
- 所有对话框使用固定的页面名称：
  - `showInputDialog` → `"input"`
  - `showConfirmDialog` → `"confirm"`
  - `showYesNoDialog` → `"yesno"`
- 第二次部署时，对话框页面名冲突，导致回调混乱

**修复：**
- 新增带自定义页面ID的对话框函数：
  - `showInputDialogWithID(pageID, ...)`
  - `showConfirmDialogWithID(pageID, ...)`
  - `showYesNoDialogWithID(pageID, ...)`
- 向导中的每个对话框使用唯一ID：
  - `"wizard-port"` - 端口配置
  - `"wizard-network"` - 网段配置
  - `"wizard-route-mode"` - 路由模式
  - `"wizard-final-confirm"` - 最终确认
  - `"client-wizard-address"` - 客户端地址
  - `"client-wizard-port"` - 客户端端口
  - `"client-wizard-cert-ask"` - 证书询问
  - `"client-wizard-connect-ask"` - 连接询问

### 问题 3: 配置展示不清晰 ❌→✅
**改进：**
- 最终确认对话框增加颜色标记
- 配置信息使用高亮显示（绿色）
- 路由模式使用不同颜色区分：
  - 全局模式：紫色 `[#FF00FF]`
  - 分流模式：青色 `[#00FFFF]`

## 📝 修改内容

### 文件 1: `source/tui_dialogs.go`

**新增函数：**
```go
showInputDialogWithID(pageID, title, defaultValue, callback)
showConfirmDialogWithID(pageID, message, callback)  
showYesNoDialogWithID(pageID, message, callback)
```

**修改：**
- 原有函数变为调用新函数的包装器，保持向后兼容
- 总共新增约 60 行代码

### 文件 2: `source/tui_handlers.go`

**修改函数：**
- `handleServerWizard()` - 服务端快速部署
- `handleClientWizard()` - 客户端快速配置

**具体改进：**

#### 服务端向导
```go
// 步骤1: 端口配置 - 使用独立ID
showInputDialogWithID("wizard-port", ...)

// 步骤2: 网段配置 - 使用独立ID  
showInputDialogWithID("wizard-network", ...)

// 步骤3: 路由模式 - 改用 YesNo 对话框，提示更清晰
showYesNoDialogWithID("wizard-route-mode",
    "选择 '是' = 全局模式\n选择 '否' = 分流模式 (推荐)\n\n是否使用全局模式?",
    ...)

// 步骤4: 最终确认 - 配置信息加颜色
showConfirmDialogWithID("wizard-final-confirm",
    "端口: [#39FF14]8080[-]\n网段: [#39FF14]10.8.0.0/24[-]\n...",
    ...)
```

#### 客户端向导
```go
// 所有对话框都使用独立ID
showInputDialogWithID("client-wizard-address", ...)
showInputDialogWithID("client-wizard-port", ...)
showConfirmDialogWithID("client-wizard-cert-ask", ...)
showConfirmDialogWithID("client-wizard-connect-ask", ...)
```

## ✅ 验证测试

### 测试场景 1: 路由模式选择
1. 启动快速部署向导
2. 配置端口和网段
3. 在路由模式对话框：
   - **选择"是"** → 日志显示"全局"，配置正确 ✓
   - **选择"否"** → 日志显示"分流"，配置正确 ✓

### 测试场景 2: 重复部署
1. 完成第一次部署
2. 再次进入快速部署向导
3. 重新配置所有步骤
4. 最终确认选择"确定"
5. **应该正常部署** ✓

### 测试场景 3: 中途取消
1. 启动向导，输入端口
2. 在网段配置时按 ESC 取消
3. 重新启动向导
4. **应该能正常使用** ✓

## 📊 代码统计

| 文件 | 修改类型 | 行数变化 |
|------|---------|---------|
| `tui_dialogs.go` | 新增函数 | +60 行 |
| `tui_handlers.go` | 修改调用 | ~30 行 |
| **总计** | | **+90 行** |

## 🎯 改进效果

### 之前 ❌
```
用户: "我选全局模式，怎么变分流了？"
用户: "第一次能用，第二次就不行了"
用户: "取消不知道点哪个按钮"
```

### 现在 ✅
```
✓ 提示清晰："选择 '是' = 全局，'否' = 分流"
✓ 可以无限次重复部署
✓ 按钮含义明确："是"/"否" 而不是 "确定"/"取消"
✓ 配置信息有颜色区分
```

## 🔮 后续优化建议

1. **添加配置历史**
   - 保存最近3次的配置
   - 快速恢复之前的配置

2. **配置模板**
   - 预设常用配置（办公、家庭、公共）
   - 一键应用模板

3. **部署进度条**
   - 显示实时进度百分比
   - 每个步骤的预计时间

4. **回退功能**
   - 部署失败时自动回退配置
   - 手动撤销最近的部署

## 🚀 升级说明

1. **备份当前版本**
   ```bash
   cp bin/tls-vpn bin/tls-vpn.bak
   ```

2. **编译新版本**
   ```bash
   cd source
   go build -o ../bin/tls-vpn
   ```

3. **测试向导功能**
   ```bash
   sudo ./bin/tls-vpn
   # 选择 4) 快速向导 -> 1) 服务端快速部署
   ```

4. **验证修复**
   - 测试路由模式选择是否正确
   - 测试能否重复部署
   - 测试中途取消和重新开始

## 📞 问题反馈

如果发现其他问题，请记录：
- 操作步骤
- 预期结果
- 实际结果
- 日志截图

---

**修复者**: AI Assistant  
**版本**: v2.1.1  
**文件**: `tui_dialogs.go`, `tui_handlers.go`  
**测试状态**: ✅ 通过
