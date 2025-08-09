# Windows Terminal Rose Pine Dawn 透明配置指南

## 🌸 配置文件说明

已生成的配置文件：`windows-terminal-rose-pine-dawn.json`

这是专为 WSL2 + ArchLinux + ZSH 环境优化的 Rose Pine Dawn 透明主题配置。

## 📋 使用步骤

### 步骤 1: 安装字体

1. **下载 JetBrains Mono Nerd Font**
   ```bash
   # 方法1: 通过网站下载
   # 访问: https://github.com/ryanoasis/nerd-fonts/releases
   # 下载: JetBrainsMono.zip
   
   # 方法2: 通过 Scoop 安装 (推荐)
   scoop bucket add nerd-fonts
   scoop install JetBrains-Mono-NF
   ```

2. **安装字体**
   - 解压下载的字体文件
   - 右键点击 `.ttf` 文件选择"安装"
   - 或复制到 `C:\Windows\Fonts\` 目录

### 步骤 2: 获取 WSL2 配置文件 GUID

1. **打开 PowerShell**，运行：
   ```powershell
   wt --list-profiles
   ```

2. **找到你的 ArchLinux 配置文件**，复制其 GUID
   例如：`{12345678-1234-5678-9abc-123456789012}`

### 步骤 3: 配置 Windows Terminal

#### 方法 1: 完整替换配置（推荐新用户）

1. **打开 Windows Terminal**
2. **按 `Ctrl + ,` 打开设置**
3. **点击左下角 "打开 JSON 文件"**
4. **备份现有配置**：
   ```bash
   # 复制原配置文件到备份文件
   cp settings.json settings.json.backup
   ```
5. **替换配置内容**：
   - 将 `windows-terminal-rose-pine-dawn.json` 的内容完整复制
   - 替换 `{请替换为你的WSL2-GUID}` 为实际的 GUID
   - 保存文件

#### 方法 2: 合并配置（推荐有现有配置的用户）

1. **只添加配色方案**：
   ```json
   "schemes": [
       // 你的现有配色方案...
       {
           "name": "Rose Pine Dawn Enhanced",
           // ... 复制 schemes 部分的内容
       }
   ]
   ```

2. **修改默认配置文件**：
   ```json
   "profiles": {
       "defaults": {
           // 添加以下配置
           "useAcrylic": true,
           "acrylicOpacity": 0.4,
           "font": {
               "face": "JetBrains Mono Nerd Font",
               "size": 11,
               "weight": "medium"
           },
           "colorScheme": "Rose Pine Dawn Enhanced"
       }
   }
   ```

### 步骤 4: 测试配置

1. **重启 Windows Terminal**
2. **创建新标签页**，应该看到：
   - 透明背景效果
   - Rose Pine Dawn 配色
   - JetBrains Mono Nerd Font 字体
3. **测试中文显示**：
   ```bash
   echo "测试中文显示效果 Hello World 🌸"
   ```

## ⚙️ 个性化调整

### 调整透明度

```json
"acrylicOpacity": 0.4,  // 60% 透明度
// 0.3 = 70% 透明度 (更透明)
// 0.5 = 50% 透明度 (更不透明)
```

### 调整字体大小

```json
"font": {
    "size": 11,  // 默认大小
    // 10 = 小字体
    // 12 = 大字体
    // 13 = 超大字体
}
```

### 调整字体粗细

```json
"font": {
    "weight": "medium",  // 默认
    // "normal" = 正常粗细
    // "semiBold" = 半粗体
    // "bold" = 粗体
}
```

## 🎮 快捷键说明

| 功能 | 快捷键 |
|------|--------|
| 复制 | `Ctrl + C` |
| 粘贴 | `Ctrl + V` |
| 水平分割 | `Alt + Shift + -` |
| 垂直分割 | `Alt + Shift + +` |
| 调整透明度 (减少) | `Ctrl + Shift + -` |
| 调整透明度 (增加) | `Ctrl + Shift + +` |
| 新建标签页 | `Ctrl + Shift + T` |
| 关闭标签页 | `Ctrl + Shift + W` |
| 切换标签页 | `Ctrl + Tab` |
| 窗格导航 | `Alt + 方向键` |

## 🔧 故障排除

### 问题 1: 字体不显示或显示异常

**解决方案：**
1. 确认已安装 JetBrains Mono Nerd Font
2. 检查字体名称是否正确：`"JetBrains Mono Nerd Font"`
3. 尝试重启 Windows Terminal

### 问题 2: 透明效果不生效

**解决方案：**
1. 确认 Windows 系统支持 Acrylic 效果
2. 检查系统设置 > 个性化 > 颜色 > 透明效果已开启
3. 尝试将 `"useAcrylic"` 设为 `false`，使用纯透明

### 问题 3: 配色显示不正确

**解决方案：**
1. 确认配色方案名称正确：`"Rose Pine Dawn Enhanced"`
2. 检查 JSON 语法是否正确（使用在线 JSON 验证器）
3. 尝试重新加载配置文件

### 问题 4: WSL2 无法正常启动

**解决方案：**
1. 确认 WSL2 已正确安装和配置
2. 检查 ArchLinux 发行版是否正常运行：
   ```powershell
   wsl -l -v
   ```
3. 更新 GUID 为正确的配置文件 ID

### 问题 5: 中文显示模糊

**解决方案：**
1. 增大字体大小到 12 或 13
2. 调整字体粗细到 "semiBold"
3. 确认系统 DPI 设置正确

## 🎨 进阶自定义

### 创建多个配色变体

```json
"schemes": [
    {
        "name": "Rose Pine Dawn Light",
        "acrylicOpacity": 0.3,  // 更透明
        // ... 其他配置
    },
    {
        "name": "Rose Pine Dawn Dark",
        "acrylicOpacity": 0.5,  // 更不透明
        // ... 其他配置
    }
]
```

### 为不同用途创建专用配置文件

```json
"profiles": {
    "list": [
        {
            "name": "编程专用",
            "font": {"size": 10},
            "colorScheme": "Rose Pine Dawn Enhanced"
        },
        {
            "name": "演示专用", 
            "font": {"size": 14},
            "acrylicOpacity": 0.2
        }
    ]
}
```

## 📚 相关资源

- [Windows Terminal 官方文档](https://docs.microsoft.com/en-us/windows/terminal/)
- [Rose Pine 主题官网](https://rosepinetheme.com/)
- [JetBrains Mono 字体](https://www.jetbrains.com/lp/mono/)
- [Nerd Fonts 图标字体](https://www.nerdfonts.com/)

## 🆘 获取帮助

如果遇到问题：
1. 检查 Windows Terminal 版本是否为最新
2. 查看 Windows Terminal 官方文档
3. 确认 WSL2 环境配置正确
4. 备份并重置配置文件测试

---

*配置已针对你的 WSL2 + ArchLinux + ZSH 环境优化，享受优雅的终端体验！* 🌸