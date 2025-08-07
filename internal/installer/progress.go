package installer

import (
	"fmt"
	"sync"
	"time"

	"github.com/schollz/progressbar/v3"
	"github.com/sirupsen/logrus"
)

// ProgressEvent 进度事件类型
type ProgressEvent struct {
	Type        ProgressEventType
	PackageName string
	Manager     string
	Message     string
	Error       error
	Timestamp   time.Time
}

// ProgressEventType 进度事件类型枚举
type ProgressEventType int

const (
	ProgressStart ProgressEventType = iota // 开始安装
	ProgressUpdate                         // 安装进度更新
	ProgressSuccess                        // 安装成功
	ProgressFail                           // 安装失败
	ProgressSkip                           // 跳过安装
)

// ProgressManager 进度管理器
type ProgressManager struct {
	packages     []string
	events       chan ProgressEvent
	results      map[string]*InstallResult
	progressBar  *progressbar.ProgressBar
	logger       *logrus.Logger
	mu           sync.RWMutex
	started      bool
	totalPkgs    int
	completedPkgs int
}

// NewProgressManager 创建进度管理器
func NewProgressManager(packages []string, logger *logrus.Logger, quiet bool) *ProgressManager {
	pm := &ProgressManager{
		packages:  packages,
		events:    make(chan ProgressEvent, 100), // 缓冲通道避免阻塞
		results:   make(map[string]*InstallResult),
		logger:    logger,
		totalPkgs: len(packages),
	}
	
	// 只在非静默模式时创建进度条
	if !quiet {
		pm.progressBar = progressbar.NewOptions(len(packages),
			progressbar.OptionSetDescription("📦 安装进度"),
			progressbar.OptionSetWidth(50),
			progressbar.OptionSetTheme(progressbar.Theme{
				Saucer:        "█",
				SaucerPadding: "░",
				BarStart:      "▐",
				BarEnd:        "▌",
			}),
			progressbar.OptionShowCount(),
			progressbar.OptionShowIts(),
			progressbar.OptionSetItsString("pkg"),
			progressbar.OptionOnCompletion(func() {
				fmt.Printf("\n✨ 安装完成！\n\n")
			}),
			progressbar.OptionSpinnerType(14),
			progressbar.OptionFullWidth(),
			progressbar.OptionSetRenderBlankState(true),
		)
	}
	
	return pm
}

// Start 启动进度显示
func (pm *ProgressManager) Start() {
	pm.mu.Lock()
	pm.started = true
	pm.mu.Unlock()
	
	// 只在非静默模式时显示启动消息
	if pm.progressBar != nil {
		fmt.Printf("🚀 准备安装 %d 个包...\n\n", pm.totalPkgs)
	}
	
	// 启动事件处理协程
	go pm.processEvents()
}

// SendEvent 发送进度事件
func (pm *ProgressManager) SendEvent(event ProgressEvent) {
	if pm.started {
		event.Timestamp = time.Now()
		select {
		case pm.events <- event:
		default:
			// 通道满时丢弃事件，避免阻塞安装过程
			pm.logger.Warn("进度事件通道已满，丢弃事件")
		}
	}
}

// processEvents 处理进度事件
func (pm *ProgressManager) processEvents() {
	for event := range pm.events {
		pm.handleEvent(event)
	}
}

// handleEvent 处理单个进度事件
func (pm *ProgressManager) handleEvent(event ProgressEvent) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	switch event.Type {
	case ProgressStart:
		pm.updatePackageStatus(event.PackageName, "🔄", "安装中", "yellow")
		
	case ProgressSuccess:
		pm.updatePackageStatus(event.PackageName, "✅", "已完成", "green")
		pm.completedPkgs++
		if pm.progressBar != nil {
			pm.progressBar.Add(1)
		}
		
	case ProgressFail:
		pm.updatePackageStatus(event.PackageName, "❌", "失败", "red")
		pm.completedPkgs++
		if pm.progressBar != nil {
			pm.progressBar.Add(1)
		}
		
	case ProgressSkip:
		pm.updatePackageStatus(event.PackageName, "⏭️", "已跳过", "blue")
		pm.completedPkgs++
		if pm.progressBar != nil {
			pm.progressBar.Add(1)
		}
	}
	
	// 更新进度条描述
	pm.updateProgressDescription()
}

// updatePackageStatus 更新包状态显示
func (pm *ProgressManager) updatePackageStatus(packageName, icon, status, color string) {
	// 只在非静默模式时显示状态
	if pm.progressBar != nil {
		fmt.Printf("\r%s %s (%s)    \n", icon, packageName, status)
	}
}

// updateProgressDescription 更新进度条描述
func (pm *ProgressManager) updateProgressDescription() {
	if pm.progressBar != nil {
		desc := fmt.Sprintf("📦 安装进度 (%d/%d)", pm.completedPkgs, pm.totalPkgs)
		pm.progressBar.Describe(desc)
	}
}

// Close 关闭进度管理器
func (pm *ProgressManager) Close() {
	pm.mu.Lock()
	pm.started = false
	pm.mu.Unlock()
	
	close(pm.events)
	if pm.progressBar != nil {
		pm.progressBar.Finish()
	}
}

// GetSummary 获取安装总结
func (pm *ProgressManager) GetSummary() *InstallSummary {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	summary := &InstallSummary{
		TotalPackages: pm.totalPkgs,
		Successful:    0,
		Failed:        0,
		Skipped:       0,
		Results:       make([]*InstallResult, 0, len(pm.results)),
	}
	
	for _, result := range pm.results {
		summary.Results = append(summary.Results, result)
		if result.Success {
			summary.Successful++
		} else {
			summary.Failed++
		}
	}
	
	return summary
}

// InstallSummary 安装总结
type InstallSummary struct {
	TotalPackages int
	Successful    int
	Failed        int
	Skipped       int
	Results       []*InstallResult
	TotalDuration float64
}

// AddResult 添加安装结果
func (pm *ProgressManager) AddResult(result *InstallResult) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	pm.results[result.PackageName] = result
}

// IsCompleted 检查是否完成
func (pm *ProgressManager) IsCompleted() bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	return pm.completedPkgs >= pm.totalPkgs
}

// PrintSummaryTable 打印总结表格
func (pm *ProgressManager) PrintSummaryTable() {
	summary := pm.GetSummary()
	
	fmt.Printf("\n📊 安装结果统计:\n")
	fmt.Printf("┌─────────────────────┬──────────────┬────────────┬──────────┐\n")
	fmt.Printf("│ 包名                │ 包管理器     │ 状态       │ 耗时(秒) │\n")
	fmt.Printf("├─────────────────────┼──────────────┼────────────┼──────────┤\n")
	
	totalTime := 0.0
	for _, result := range summary.Results {
		status := "❌ 失败"
		if result.Success {
			status = "✅ 成功"
		}
		
		totalTime += result.Duration
		
		fmt.Printf("│ %-19s │ %-12s │ %-10s │ %8.2f │\n",
			truncateString(result.PackageName, 19),
			result.Manager,
			status,
			result.Duration,
		)
	}
	
	fmt.Printf("└─────────────────────┴──────────────┴────────────┴──────────┘\n")
	fmt.Printf("总计: 成功 %d, 失败 %d, 总耗时: %.2f秒\n", 
		summary.Successful, summary.Failed, totalTime)
}

// truncateString 截断字符串到指定长度
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}