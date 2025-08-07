package commands

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile    string
	verbose    bool
	rootLogger *logrus.Logger
)

// rootCmd 是应用的根命令
var rootCmd = &cobra.Command{
	Use:   "dotfiles",
	Short: "跨平台 dotfiles 配置管理工具",
	Long: `专为 WSL2 (Arch Linux + Zsh) 和 PowerShell 设计的
现代化配置管理工具，提供统一的开发环境配置体验。

支持功能：
  • 软件包并行安装
  • 配置文件模板生成  
  • 跨平台路径处理
  • XDG 规范支持`,
	Version: "0.1.0",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		initLogger()
	},
}

// Execute 执行根命令
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// 全局参数
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "配置文件路径")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "详细输出")

	// 绑定到 viper
	viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
}

// initConfig 初始化配置
func initConfig() {
	if cfgFile != "" {
		// 使用指定的配置文件
		viper.SetConfigFile(cfgFile)
	} else {
		// 搜索默认配置文件位置
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}

		// 添加配置文件搜索路径
		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.AddConfigPath("./configs")
		viper.SetConfigType("json")
		viper.SetConfigName("shared")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		if verbose {
			fmt.Println("Using config file:", viper.ConfigFileUsed())
		}
	}
}

// initLogger 初始化日志系统
func initLogger() {
	rootLogger = logrus.New()
	
	// 设置日志级别
	if verbose || viper.GetBool("verbose") {
		rootLogger.SetLevel(logrus.DebugLevel)
	} else {
		rootLogger.SetLevel(logrus.InfoLevel)
	}

	// 设置日志格式
	rootLogger.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: false,
		FullTimestamp:    true,
		TimestampFormat:  "15:04:05",
	})

	rootLogger.Debug("日志系统初始化完成")
}

// GetLogger 获取日志实例
func GetLogger() *logrus.Logger {
	return rootLogger
}