package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"tron-monitor/config"
	httpclient "tron-monitor/http"
	"tron-monitor/processor"
	"tron-monitor/redis"
)

// Application 应用程序结构
type Application struct {
	config         *config.Config
	redisClient    *redis.RedisClient
	httpClient     *httpclient.HTTPClient
	blockMonitor   *processor.BlockMonitor
	blockProcessor *processor.BlockProcessor
	server         *http.Server
	startTime      time.Time
}

// NewApplication 创建应用程序实例
func NewApplication(configPath string) (*Application, error) {
	// 1. 加载配置
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("加载配置失败: %w", err)
	}

	// 2. 初始化日志
	if err := initLogger(cfg); err != nil {
		return nil, fmt.Errorf("初始化日志失败: %w", err)
	}

	// 3. 初始化Redis客户端
	redisClient, err := redis.NewRedisClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("初始化Redis客户端失败: %w", err)
	}

	// 4. 初始化HTTP客户端
	httpClient := httpclient.NewHTTPClient(cfg)

	// 5. 初始化区块监控器
	blockMonitor := processor.NewBlockMonitor(cfg, redisClient, httpClient)

	// 6. 初始化区块处理器
	blockProcessor := processor.NewBlockProcessor(cfg, redisClient, httpClient)

	// 7. 初始化HTTP服务器
	server := initHTTPServer(cfg, redisClient, blockMonitor, blockProcessor)

	return &Application{
		config:         cfg,
		redisClient:    redisClient,
		httpClient:     httpClient,
		blockMonitor:   blockMonitor,
		blockProcessor: blockProcessor,
		server:         server,
		startTime:      time.Now(),
	}, nil
}

// Start 启动应用程序
func (app *Application) Start() error {
	log.Println("启动Tron区块链监控系统...")

	// 1. 健康检查
	if err := app.healthCheck(); err != nil {
		log.Printf("警告: 健康检查失败: %v，但继续启动系统", err)
		// 不返回错误，让系统继续启动
	}

	// 2. 初始化监控地址
	if err := app.initWatchAddresses(); err != nil {
		return fmt.Errorf("初始化监控地址失败: %w", err)
	}

	// 3. 启动区块处理器
	if err := app.blockProcessor.Start(); err != nil {
		return fmt.Errorf("启动区块处理器失败: %w", err)
	}

	// 4. 启动区块监控器
	if err := app.blockMonitor.Start(); err != nil {
		return fmt.Errorf("启动区块监控器失败: %w", err)
	}

	// 5. 启动HTTP服务器
	go func() {
		log.Printf("启动HTTP服务器: %s:%s", app.config.Server.Host, app.config.Server.Port)
		if err := app.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP服务器启动失败: %v", err)
		}
	}()

	log.Println("Tron区块链监控系统启动完成")
	return nil
}

// Stop 停止应用程序
func (app *Application) Stop() error {
	log.Println("正在停止Tron区块链监控系统...")

	// 1. 停止HTTP服务器
	if app.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := app.server.Shutdown(ctx); err != nil {
			log.Printf("停止HTTP服务器失败: %v", err)
		}
	}

	// 2. 停止区块监控器
	if app.blockMonitor != nil {
		if err := app.blockMonitor.Stop(); err != nil {
			log.Printf("停止区块监控器失败: %v", err)
		}
	}

	// 3. 停止区块处理器
	if app.blockProcessor != nil {
		if err := app.blockProcessor.Stop(); err != nil {
			log.Printf("停止区块处理器失败: %v", err)
		}
	}

	// 4. 关闭Redis连接
	if app.redisClient != nil {
		if err := app.redisClient.Close(); err != nil {
			log.Printf("关闭Redis连接失败: %v", err)
		}
	}

	log.Println("Tron区块链监控系统已停止")
	return nil
}

// healthCheck 健康检查
func (app *Application) healthCheck() error {
	log.Println("执行健康检查...")

	// 检查Redis连接
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 测试Redis连接
	if _, err := app.redisClient.GetQueueSize(ctx); err != nil {
		return fmt.Errorf("Redis连接检查失败: %w", err)
	}

	// 测试TronGrid API连接（使用更长的超时时间）
	apiCtx, apiCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer apiCancel()

	if _, err := app.httpClient.GetLatestBlock(apiCtx); err != nil {
		return fmt.Errorf("TronGrid API连接检查失败: %w", err)
	}

	log.Println("健康检查通过")
	return nil
}

// initWatchAddresses 初始化监控地址
func (app *Application) initWatchAddresses() error {
	log.Println("初始化监控地址...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 获取现有监控地址
	existingAddresses, err := app.redisClient.GetWatchAddresses(ctx)
	if err != nil {
		return fmt.Errorf("获取现有监控地址失败: %w", err)
	}

	// 将现有地址转换为map以便快速查找
	existingMap := make(map[string]bool)
	for _, addr := range existingAddresses {
		existingMap[addr] = true
	}

	// 添加配置中的监控地址
	for _, addr := range app.config.WatchAddresses {
		if !existingMap[addr] {
			if err := app.redisClient.AddWatchAddress(ctx, addr); err != nil {
				log.Printf("添加监控地址 %s 失败: %v", addr, err)
				continue
			}
			log.Printf("已添加监控地址: %s", addr)
		}
	}

	log.Printf("监控地址初始化完成，共 %d 个地址", len(app.config.WatchAddresses))
	return nil
}

// initLogger 初始化日志系统
func initLogger(cfg *config.Config) error {
	// 设置日志级别
	level, err := logrus.ParseLevel(cfg.Log.Level)
	if err != nil {
		return fmt.Errorf("解析日志级别失败: %w", err)
	}
	logrus.SetLevel(level)

	// 设置日志格式
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	// 如果配置了日志文件，则输出到文件
	if cfg.Log.File != "" {
		file, err := os.OpenFile(cfg.Log.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return fmt.Errorf("打开日志文件失败: %w", err)
		}
		logrus.SetOutput(file)
	}

	return nil
}

// initHTTPServer 初始化HTTP服务器
func initHTTPServer(cfg *config.Config, redisClient *redis.RedisClient, blockMonitor *processor.BlockMonitor, blockProcessor *processor.BlockProcessor) *http.Server {
	router := mux.NewRouter()

	// 健康检查端点
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "healthy",
			"time":   time.Now().Format(time.RFC3339),
		})
	}).Methods("GET")

	// 系统状态端点
	router.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// 获取各种统计信息
		monitorStats := blockMonitor.GetStats()
		processorStats := blockProcessor.GetStats()
		httpStats, _ := redisClient.GetSystemStats(r.Context())

		status := map[string]interface{}{
			"monitor":   monitorStats,
			"processor": processorStats,
			"http":      httpStats,
			"uptime":    time.Since(time.Now()).String(),
		}

		json.NewEncoder(w).Encode(status)
	}).Methods("GET")

	// 监控地址管理端点
	router.HandleFunc("/addresses", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.Method {
		case "GET":
			addresses, err := redisClient.GetWatchAddresses(r.Context())
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(addresses)

		case "POST":
			var req struct {
				Address string `json:"address"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			if err := redisClient.AddWatchAddress(r.Context(), req.Address); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			w.WriteHeader(http.StatusCreated)

		case "DELETE":
			var req struct {
				Address string `json:"address"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			if err := redisClient.RemoveWatchAddress(r.Context(), req.Address); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			w.WriteHeader(http.StatusOK)
		}
	}).Methods("GET", "POST", "DELETE")

	// 转账记录端点
	router.HandleFunc("/transfers", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		limit := int64(100) // 默认限制
		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
			if l, err := fmt.Sscanf(limitStr, "%d", &limit); err != nil || l != 1 {
				http.Error(w, "无效的limit参数", http.StatusBadRequest)
				return
			}
		}

		transfers, err := redisClient.GetRecentTransfers(r.Context(), limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(transfers)
	}).Methods("GET")

	// USDT转账记录端点
	router.HandleFunc("/usdt-transfers", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		limit := int64(100) // 默认限制
		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
			if l, err := fmt.Sscanf(limitStr, "%d", &limit); err != nil || l != 1 {
				http.Error(w, "无效的limit参数", http.StatusBadRequest)
				return
			}
		}

		transfers, err := redisClient.GetRecentUSDTTransfers(r.Context(), limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(transfers)
	}).Methods("GET")

	// USDT统计信息端点
	router.HandleFunc("/usdt-stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		transfers, err := redisClient.GetRecentUSDTTransfers(r.Context(), 1000) // 获取最近1000条记录进行统计
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// 计算USDT统计信息
		var totalAmount float64
		var totalCount int64
		var minAmount float64 = 999999999
		var maxAmount float64
		addresses := make(map[string]int64)

		for _, transfer := range transfers {
			totalAmount += transfer.Amount
			totalCount++
			
			if transfer.Amount < minAmount {
				minAmount = transfer.Amount
			}
			if transfer.Amount > maxAmount {
				maxAmount = transfer.Amount
			}

			addresses[transfer.Source]++
			addresses[transfer.Destination]++
		}

		avgAmount := float64(0)
		if totalCount > 0 {
			avgAmount = totalAmount / float64(totalCount)
		}

		// 计算最近10条记录
		recentCount := int(math.Min(10, float64(len(transfers))))
		recentTransfers := transfers[:recentCount]

		stats := map[string]interface{}{
			"total_transfers":  totalCount,
			"total_amount":     totalAmount,
			"avg_amount":       avgAmount,
			"min_amount":       minAmount,
			"max_amount":       maxAmount,
			"unique_addresses": len(addresses),
			"recent_transfers": recentTransfers,
		}

		json.NewEncoder(w).Encode(stats)
	}).Methods("GET")

	return &http.Server{
		Addr:    fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port),
		Handler: router,
	}
}

func main() {
	// 设置默认配置文件路径
	configPath := "config.yaml"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	// 创建应用程序实例
	app, err := NewApplication(configPath)
	if err != nil {
		log.Fatalf("创建应用程序失败: %v", err)
	}

	// 启动应用程序
	if err := app.Start(); err != nil {
		log.Fatalf("启动应用程序失败: %v", err)
	}

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("收到中断信号，正在关闭...")

	// 停止应用程序
	if err := app.Stop(); err != nil {
		log.Printf("停止应用程序失败: %v", err)
	}
}
