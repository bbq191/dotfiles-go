package commands

import (
	"fmt"

	"github.com/bbq191/dotfiles-go/internal/config"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	strictMode bool
)

// validateCmd 验证配置命令
var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "验证配置文件",
	Long: `验证配置文件的格式和内容是否正确。

验证项目:
  • JSON 语法检查
  • 必填字段验证
  • 路径有效性检查
  • 包名正确性验证
  • 平台兼容性检查

示例:
  dotfiles validate                 # 验证默认配置
  dotfiles validate --strict       # 严格模式验证
  dotfiles validate --config=my.json  # 验证指定配置`,
	RunE: runValidate,
}

func init() {
	rootCmd.AddCommand(validateCmd)

	validateCmd.Flags().BoolVarP(&strictMode, "strict", "s", false, "严格模式验证")
}

func runValidate(cmd *cobra.Command, args []string) error {
	logger := GetLogger()
	
	logger.Info("开始配置验证流程")
	
	if strictMode {
		logger.Info("使用严格模式验证")
	}
	
	// 加载配置
	configDir := getConfigDir()
	loader := loadConfig(configDir, logger)
	
	config, err := loader.LoadConfig()
	if err != nil {
		return fmt.Errorf("配置加载失败: %w", err)
	}
	
	// 验证配置
	validator := createValidator(logger)
	if err := validator.ValidateConfig(config); err != nil {
		fmt.Printf("❌ 配置验证失败:\n%v\n", err)
		return err
	}
	
	// 显示验证结果
	fmt.Println("✅ 配置验证通过")
	fmt.Printf("用户: %s (%s)\n", config.User.Name, config.User.Email)
	fmt.Printf("版本: %s\n", config.Version)
	
	if config.ZshConfig != nil {
		fmt.Printf("Zsh 集成: 已启用\n")
		if config.ZshConfig.XDGDirectories.Enabled {
			fmt.Printf("XDG 目录: 已启用\n")
		}
	}
	
	if config.Packages != nil {
		categoryCount := len(config.Packages.Categories)
		managerCount := len(config.Packages.Managers)
		fmt.Printf("包配置: %d 个分类, %d 个包管理器\n", categoryCount, managerCount)
	}
	
	logger.Info("配置验证完成")
	return nil
}

// getConfigDir 获取配置目录
func getConfigDir() string {
	return config.GetConfigDir()
}

// loadConfig 创建配置加载器
func loadConfig(configDir string, logger *logrus.Logger) *config.ConfigLoader {
	return config.NewConfigLoader(configDir, logger)
}

// createValidator 创建配置验证器
func createValidator(logger *logrus.Logger) *config.ConfigValidator {
	return config.NewConfigValidator(logger)
}