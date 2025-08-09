# Windows Terminal Rose Pine Dawn 快速配置脚本
# 运行方法: PowerShell -ExecutionPolicy Bypass -File quick-setup.ps1

Write-Host "🌸 Windows Terminal Rose Pine Dawn 配置工具" -ForegroundColor Magenta
Write-Host "=============================================" -ForegroundColor Cyan

# 检查 Windows Terminal 是否安装
$wtInstalled = Get-Command "wt" -ErrorAction SilentlyContinue
if (-not $wtInstalled) {
    Write-Host "❌ 未检测到 Windows Terminal，请先安装" -ForegroundColor Red
    Write-Host "安装命令: winget install Microsoft.WindowsTerminal" -ForegroundColor Yellow
    exit 1
}

Write-Host "✅ 检测到 Windows Terminal" -ForegroundColor Green

# 检查 Scoop 是否安装
$scoopInstalled = Get-Command "scoop" -ErrorAction SilentlyContinue
if ($scoopInstalled) {
    Write-Host "✅ 检测到 Scoop 包管理器" -ForegroundColor Green
    $choice = Read-Host "是否使用 Scoop 自动安装 JetBrains Mono Nerd Font? (y/n)"
    if ($choice -eq "y" -or $choice -eq "Y") {
        Write-Host "🔽 正在安装 JetBrains Mono Nerd Font..." -ForegroundColor Yellow
        scoop bucket add nerd-fonts
        scoop install JetBrains-Mono-NF
        Write-Host "✅ 字体安装完成" -ForegroundColor Green
    }
} else {
    Write-Host "⚠️  未检测到 Scoop，需要手动安装字体" -ForegroundColor Yellow
    Write-Host "字体下载地址: https://github.com/ryanoasis/nerd-fonts/releases" -ForegroundColor Cyan
    Write-Host "请下载 JetBrainsMono.zip 并安装字体文件" -ForegroundColor Cyan
    $continue = Read-Host "字体安装完成后按 Enter 继续..."
}

# 获取 WSL 配置文件列表
Write-Host "🔍 正在获取 WSL 配置文件..." -ForegroundColor Yellow
$profiles = wt --list-profiles | ConvertFrom-Json

Write-Host "检测到的配置文件:" -ForegroundColor Cyan
for ($i = 0; $i -lt $profiles.Count; $i++) {
    $profile = $profiles[$i]
    Write-Host "[$i] $($profile.name) - $($profile.guid)" -ForegroundColor White
}

$selection = Read-Host "请输入要配置的 WSL 配置文件编号"
if ($selection -lt 0 -or $selection -ge $profiles.Count) {
    Write-Host "❌ 无效的选择" -ForegroundColor Red
    exit 1
}

$selectedProfile = $profiles[$selection]
$wslGuid = $selectedProfile.guid
Write-Host "✅ 已选择: $($selectedProfile.name) ($wslGuid)" -ForegroundColor Green

# 查找 Windows Terminal 设置文件路径
$settingsPath = "$env:LOCALAPPDATA\Packages\Microsoft.WindowsTerminal_8wekyb3d8bbwe\LocalState\settings.json"
if (-not (Test-Path $settingsPath)) {
    # 尝试预览版路径
    $settingsPath = "$env:LOCALAPPDATA\Packages\Microsoft.WindowsTerminalPreview_8wekyb3d8bbwe\LocalState\settings.json"
}

if (-not (Test-Path $settingsPath)) {
    Write-Host "❌ 找不到 Windows Terminal 设置文件" -ForegroundColor Red
    Write-Host "请手动配置，设置文件通常位于:" -ForegroundColor Yellow
    Write-Host "$env:LOCALAPPDATA\Packages\Microsoft.WindowsTerminal_8wekyb3d8bbwe\LocalState\settings.json" -ForegroundColor Cyan
    exit 1
}

Write-Host "✅ 找到设置文件: $settingsPath" -ForegroundColor Green

# 备份现有配置
$backupPath = "$settingsPath.backup.$(Get-Date -Format 'yyyyMMdd-HHmmss')"
Copy-Item $settingsPath $backupPath
Write-Host "✅ 已备份原配置到: $backupPath" -ForegroundColor Green

# 读取配置模板
$configTemplate = Get-Content "windows-terminal-rose-pine-dawn.json" -Raw -Encoding UTF8

if (-not $configTemplate) {
    Write-Host "❌ 找不到配置文件模板: windows-terminal-rose-pine-dawn.json" -ForegroundColor Red
    Write-Host "请确保配置文件在当前目录" -ForegroundColor Yellow
    exit 1
}

# 替换 GUID
$configContent = $configTemplate -replace '\{请替换为你的WSL2-GUID\}', $wslGuid

# 写入配置文件
$configContent | Out-File -FilePath $settingsPath -Encoding UTF8 -Force

Write-Host "✅ 配置文件已更新" -ForegroundColor Green

# 重启 Windows Terminal
Write-Host "🔄 正在重启 Windows Terminal..." -ForegroundColor Yellow
$wtProcesses = Get-Process "WindowsTerminal" -ErrorAction SilentlyContinue
if ($wtProcesses) {
    $wtProcesses | Stop-Process -Force
    Start-Sleep -Seconds 2
}

# 启动 Windows Terminal
Start-Process "wt"

Write-Host "" 
Write-Host "🎉 配置完成！" -ForegroundColor Green
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Cyan
Write-Host "✨ Rose Pine Dawn 透明主题已应用" -ForegroundColor Magenta
Write-Host "📝 原配置已备份到: $backupPath" -ForegroundColor Cyan
Write-Host "🎮 查看完整快捷键和自定义选项，请参考配置指南" -ForegroundColor Yellow
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Cyan

$openGuide = Read-Host "是否打开配置指南? (y/n)"
if ($openGuide -eq "y" -or $openGuide -eq "Y") {
    Start-Process "Windows-Terminal-配置指南.md"
}

Write-Host "🌸 享受你的新终端体验！" -ForegroundColor Magenta