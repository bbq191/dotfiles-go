package installer

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

// ParallelInstaller 并行安装器
type ParallelInstaller struct {
	installer    *Installer
	logger       *logrus.Logger
	maxWorkers   int
	semaphore    chan struct{} // 信号量控制并发数
	progressMgr  *ProgressManager
	results      []*InstallResult
	resultsMutex sync.Mutex
}

// NewParallelInstaller 创建并行安装器
func NewParallelInstaller(installer *Installer, maxWorkers int) *ParallelInstaller {
	if maxWorkers <= 0 {
		maxWorkers = runtime.NumCPU()
	}
	
	return &ParallelInstaller{
		installer:  installer,
		logger:     installer.logger,
		maxWorkers: maxWorkers,
		semaphore:  make(chan struct{}, maxWorkers),
		results:    make([]*InstallResult, 0),
	}
}

// InstallPackagesParallel 并行安装多个包
func (pi *ParallelInstaller) InstallPackagesParallel(ctx context.Context, packages []string, opts InstallOptions) ([]*InstallResult, error) {
	// 检查包管理器是否支持并行安装
	if !pi.supportsParallel() {
		pi.logger.Warn("当前包管理器不支持并行安装，回退到串行模式")
		return pi.installer.InstallPackages(ctx, packages, opts)
	}

	pi.logger.Infof("启动并行安装模式：%d 个工作协程，安装 %d 个包", pi.maxWorkers, len(packages))
	
	// 创建进度管理器
	pi.progressMgr = NewProgressManager(packages, pi.logger, opts.Quiet)
	
	// 启动进度显示（除非是quiet模式）
	if !opts.Quiet {
		pi.progressMgr.Start()
		defer pi.progressMgr.Close()
	}
	
	// 创建错误组进行并发控制
	g, ctx := errgroup.WithContext(ctx)
	
	// 创建任务通道
	packageChan := make(chan string, len(packages))
	
	// 发送所有包到通道
	for _, pkg := range packages {
		packageChan <- pkg
	}
	close(packageChan)
	
	// 启动worker协程
	for i := 0; i < pi.maxWorkers; i++ {
		workerID := i
		g.Go(func() error {
			return pi.worker(ctx, workerID, packageChan, opts)
		})
	}
	
	// 等待所有worker完成
	if err := g.Wait(); err != nil {
		pi.logger.Errorf("并行安装过程中出现错误: %v", err)
		// 继续处理，不要因为部分失败而终止
	}
	
	// 显示总结（除非是quiet模式）
	if !opts.Quiet {
		time.Sleep(100 * time.Millisecond)
		pi.progressMgr.PrintSummaryTable()
	}
	
	// 统计结果
	pi.resultsMutex.Lock()
	results := make([]*InstallResult, len(pi.results))
	copy(results, pi.results)
	pi.resultsMutex.Unlock()
	
	successful := 0
	failed := 0
	for _, result := range results {
		if result.Success {
			successful++
		} else {
			failed++
		}
	}
	
	pi.logger.Infof("并行安装完成 - 成功: %d, 失败: %d", successful, failed)
	
	return results, nil
}

// worker 工作协程
func (pi *ParallelInstaller) worker(ctx context.Context, workerID int, packageChan <-chan string, opts InstallOptions) error {
	pi.logger.Debugf("Worker %d 启动", workerID)
	defer pi.logger.Debugf("Worker %d 退出", workerID)
	
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case pkg, ok := <-packageChan:
			if !ok {
				// 通道已关闭，无更多任务
				return nil
			}
			
			// 获取信号量（控制并发数）
			select {
			case pi.semaphore <- struct{}{}:
				// 成功获取信号量，执行安装
				err := pi.installPackageWithProgress(ctx, pkg, opts, workerID)
				<-pi.semaphore // 释放信号量
				
				if err != nil {
					pi.logger.Errorf("Worker %d 安装包 %s 失败: %v", workerID, pkg, err)
					// 不返回错误，继续处理其他包
				}
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}

// installPackageWithProgress 带进度更新的包安装
func (pi *ParallelInstaller) installPackageWithProgress(ctx context.Context, pkg string, opts InstallOptions, workerID int) error {
	pi.logger.Debugf("Worker %d 开始安装包: %s", workerID, pkg)
	
	// 发送开始安装事件
	if pi.progressMgr != nil {
		pi.progressMgr.SendEvent(ProgressEvent{
			Type:        ProgressStart,
			PackageName: pkg,
			Message:     "开始安装",
		})
	}
	
	// 执行安装
	result, err := pi.installer.InstallPackage(ctx, pkg, opts)
	
	// 添加结果到列表
	pi.resultsMutex.Lock()
	pi.results = append(pi.results, result)
	pi.resultsMutex.Unlock()
	
	// 添加结果到进度管理器
	if pi.progressMgr != nil {
		pi.progressMgr.AddResult(result)
	}
	
	// 发送相应的进度事件
	if pi.progressMgr != nil {
		if err != nil {
			pi.progressMgr.SendEvent(ProgressEvent{
				Type:        ProgressFail,
				PackageName: pkg,
				Manager:     result.Manager,
				Error:       err,
			})
		} else if result.Success {
			if result.Skipped {
				pi.progressMgr.SendEvent(ProgressEvent{
					Type:        ProgressSkip,
					PackageName: pkg,
					Manager:     result.Manager,
					Message:     "包已存在",
				})
			} else {
				pi.progressMgr.SendEvent(ProgressEvent{
					Type:        ProgressSuccess,
					PackageName: pkg,
					Manager:     result.Manager,
					Message:     "安装成功",
				})
			}
		}
	}
	
	pi.logger.Debugf("Worker %d 完成安装包: %s", workerID, pkg)
	return err
}

// supportsParallel 检查当前包管理器是否支持并行安装
func (pi *ParallelInstaller) supportsParallel() bool {
	availableManagers := pi.installer.GetAvailableManagers()
	if len(availableManagers) == 0 {
		return false
	}
	
	// 获取最高优先级的管理器
	manager := pi.installer.SelectManager()
	if manager == nil {
		return false
	}
	
	// 检查包管理器是否支持并行
	switch manager.Name() {
	case "pacman":
		// Pacman 不支持真正的并行安装（会有锁冲突）
		return false
	case "winget":
		// Winget 支持并行安装
		return true
	case "yay":
		// Yay 不支持并行安装（基于pacman）
		return false
	default:
		// 默认假设不支持并行
		return false
	}
}

// GetOptimalWorkerCount 获取最佳工作协程数
func GetOptimalWorkerCount(packageCount int) int {
	cpuCount := runtime.NumCPU()
	
	// 基于包数量和CPU核心数计算最佳工作协程数
	if packageCount <= 2 {
		return 1 // 包数量很少时，串行更快
	}
	
	if packageCount <= cpuCount {
		return packageCount // 包数量少于CPU核心数时，每个包一个协程
	}
	
	// 包数量多时，使用CPU核心数的1.5倍（考虑I/O等待）
	return int(float64(cpuCount) * 1.5)
}

// ParallelCapability 并行能力检查结果
type ParallelCapability struct {
	Supported         bool
	RecommendedWorkers int
	Reason            string
}

// CheckParallelCapability 检查并行安装能力
func (pi *ParallelInstaller) CheckParallelCapability(packages []string) *ParallelCapability {
	capability := &ParallelCapability{
		Supported: false,
		Reason:    "未知原因",
	}
	
	// 检查包管理器支持
	if !pi.supportsParallel() {
		manager := pi.installer.SelectManager()
		managerName := "未知"
		if manager != nil {
			managerName = manager.Name()
		}
		capability.Reason = fmt.Sprintf("包管理器 %s 不支持并行安装", managerName)
		return capability
	}
	
	// 检查包数量
	if len(packages) <= 1 {
		capability.Reason = "包数量太少，并行安装无优势"
		return capability
	}
	
	// 支持并行安装
	capability.Supported = true
	capability.RecommendedWorkers = GetOptimalWorkerCount(len(packages))
	capability.Reason = fmt.Sprintf("支持并行安装，推荐 %d 个工作协程", capability.RecommendedWorkers)
	
	return capability
}