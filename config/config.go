package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config 系统配置结构体
type Config struct {
	// TronGrid API配置
	TronGrid struct {
		BaseURL    string        `mapstructure:"base_url"`
		APIKey     string        `mapstructure:"api_key"`
		Timeout    time.Duration `mapstructure:"timeout"`
		RetryMax   int           `mapstructure:"retry_max"`
		RetryDelay time.Duration `mapstructure:"retry_delay"`
	} `mapstructure:"trongrid"`

	// Redis配置
	Redis struct {
		Addr     string `mapstructure:"addr"`
		Password string `mapstructure:"password"`
		DB       int    `mapstructure:"db"`
		PoolSize int    `mapstructure:"pool_size"`
	} `mapstructure:"redis"`

	// 监控配置
	Monitor struct {
		BlockInterval    time.Duration `mapstructure:"block_interval"`     // 区块查询间隔，默认1秒
		WorkerCount      int           `mapstructure:"worker_count"`       // 工作线程数
		QueueSize        int           `mapstructure:"queue_size"`         // 队列大小
		BatchSize        int           `mapstructure:"batch_size"`         // 批处理大小
		MaxBlockHeight   int64         `mapstructure:"max_block_height"`   // 最大区块高度
		StartBlockHeight int64         `mapstructure:"start_block_height"` // 起始区块高度
	} `mapstructure:"monitor"`

	// 监控地址列表
	WatchAddresses []string `mapstructure:"watch_addresses"`

	// USDT监控配置
	USDT struct {
		ContractAddress  string  `mapstructure:"contract_address"`
		EnableMonitoring bool    `mapstructure:"enable_monitoring"`
		MinAmount        float64 `mapstructure:"min_amount"`
		MaxAmount        float64 `mapstructure:"max_amount"`
		Decimals         int     `mapstructure:"decimals"`
	} `mapstructure:"usdt"`

	// 日志配置
	Log struct {
		Level string `mapstructure:"level"`
		File  string `mapstructure:"file"`
	} `mapstructure:"log"`

	// HTTP服务配置
	Server struct {
		Port string `mapstructure:"port"`
		Host string `mapstructure:"host"`
	} `mapstructure:"server"`
}

// LoadConfig 加载配置文件
func LoadConfig(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")
	viper.AutomaticEnv()

	// 设置默认值
	setDefaults()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 验证配置
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	return &config, nil
}

// setDefaults 设置默认配置值
func setDefaults() {
	// TronGrid默认配置
	viper.SetDefault("trongrid.base_url", "https://api.trongrid.io")
	viper.SetDefault("trongrid.timeout", "30s")
	viper.SetDefault("trongrid.retry_max", 3)
	viper.SetDefault("trongrid.retry_delay", "1s")
	viper.SetDefault("trongrid.api_key", "849cc081-79af-4d12-9db1-48ec1c16417e")

	// Redis默认配置
	viper.SetDefault("redis.addr", "localhost:6379")
	viper.SetDefault("redis.db", 0)
	viper.SetDefault("redis.pool_size", 10)

	// 监控默认配置
	viper.SetDefault("monitor.block_interval", "1s") // 每秒一次查询
	viper.SetDefault("monitor.worker_count", 4)
	viper.SetDefault("monitor.queue_size", 1000)
	viper.SetDefault("monitor.batch_size", 10)
	viper.SetDefault("monitor.max_block_height", 0) // 0表示不限制

	// 日志默认配置
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.file", "")

	// USDT默认配置
	viper.SetDefault("usdt.contract_address", "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")
	viper.SetDefault("usdt.enable_monitoring", true)
	viper.SetDefault("usdt.min_amount", 100.0)
	viper.SetDefault("usdt.max_amount", 1000000.0)
	viper.SetDefault("usdt.decimals", 6)

	// HTTP服务默认配置
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("server.host", "0.0.0.0")
}

// validateConfig 验证配置
func validateConfig(config *Config) error {
	// 验证TronGrid配置
	if config.TronGrid.BaseURL == "" {
		return fmt.Errorf("TronGrid BaseURL不能为空")
	}

	// 验证Redis配置
	if config.Redis.Addr == "" {
		return fmt.Errorf("Redis地址不能为空")
	}

	// 验证监控配置
	if config.Monitor.BlockInterval < time.Second {
		return fmt.Errorf("区块查询间隔不能小于1秒")
	}

	if config.Monitor.WorkerCount <= 0 {
		return fmt.Errorf("工作线程数必须大于0")
	}

	if config.Monitor.QueueSize <= 0 {
		return fmt.Errorf("队列大小必须大于0")
	}

	// 验证监控地址格式
	for i, addr := range config.WatchAddresses {
		if !isValidTronAddress(addr) {
			return fmt.Errorf("无效的Tron地址格式: %s (索引: %d)", addr, i)
		}
	}

	return nil
}

// isValidTronAddress 验证Tron地址格式
func isValidTronAddress(address string) bool {
	// Tron地址格式验证
	// 地址长度应该是34个字符，以T开头
	if len(address) != 34 || !strings.HasPrefix(address, "T") {
		return false
	}

	// 这里可以添加更详细的地址格式验证
	// 目前只做基本的长度和前缀检查
	return true
}

// GetWatchAddressesSet 获取监控地址集合
func (c *Config) GetWatchAddressesSet() map[string]bool {
	addresses := make(map[string]bool)
	for _, addr := range c.WatchAddresses {
		addresses[addr] = true
	}
	return addresses
}

// AddWatchAddress 添加监控地址
func (c *Config) AddWatchAddress(address string) error {
	if !isValidTronAddress(address) {
		return fmt.Errorf("无效的Tron地址格式: %s", address)
	}

	// 检查是否已存在
	for _, addr := range c.WatchAddresses {
		if addr == address {
			return fmt.Errorf("地址已存在: %s", address)
		}
	}

	c.WatchAddresses = append(c.WatchAddresses, address)
	return nil
}

// RemoveWatchAddress 移除监控地址
func (c *Config) RemoveWatchAddress(address string) error {
	for i, addr := range c.WatchAddresses {
		if addr == address {
			c.WatchAddresses = append(c.WatchAddresses[:i], c.WatchAddresses[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("地址不存在: %s", address)
}
