package processor

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"tron-monitor/config"
	"tron-monitor/http"
	"tron-monitor/redis"
)

// BlockMonitor 区块监控器
type BlockMonitor struct {
	config      *config.Config
	redisClient *redis.RedisClient
	httpClient  *http.HTTPClient
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	running     bool
	mu          sync.RWMutex

	// 统计信息
	lastProcessedBlock int64
	processedBlocks    int64
	errors             int64
}

// NewBlockMonitor 创建区块监控器
func NewBlockMonitor(cfg *config.Config, redisClient *redis.RedisClient, httpClient *http.HTTPClient) *BlockMonitor {
	ctx, cancel := context.WithCancel(context.Background())

	return &BlockMonitor{
		config:      cfg,
		redisClient: redisClient,
		httpClient:  httpClient,
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start 启动区块监控
func (bm *BlockMonitor) Start() error {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	if bm.running {
		return fmt.Errorf("区块监控器已在运行")
	}

	bm.running = true
	bm.wg.Add(1)

	go func() {
		defer bm.wg.Done()
		bm.monitorBlocks()
	}()

	log.Println("区块监控器已启动")
	return nil
}

// Stop 停止区块监控
func (bm *BlockMonitor) Stop() error {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	if !bm.running {
		return fmt.Errorf("区块监控器未运行")
	}

	bm.running = false
	bm.cancel()
	bm.wg.Wait()

	log.Println("区块监控器已停止")
	return nil
}

// IsRunning 检查是否正在运行
func (bm *BlockMonitor) IsRunning() bool {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	return bm.running
}

// monitorBlocks 监控区块循环
func (bm *BlockMonitor) monitorBlocks() {
	ticker := time.NewTicker(bm.config.Monitor.BlockInterval)
	defer ticker.Stop()

	log.Printf("开始监控区块，查询间隔: %v", bm.config.Monitor.BlockInterval)

	for {
		select {
		case <-bm.ctx.Done():
			log.Println("区块监控器收到停止信号")
			return
		case <-ticker.C:
			log.Printf("开始处理最新区块...")
			if err := bm.processLatestBlock(); err != nil {
				log.Printf("处理最新区块失败: %v", err)
				bm.errors++
			}
		}
	}
}

// processLatestBlock 处理最新区块
func (bm *BlockMonitor) processLatestBlock() error {
	// 获取最新区块
	blockData, err := bm.httpClient.GetLatestBlock(bm.ctx)
	if err != nil {
		return fmt.Errorf("获取最新区块失败: %w", err)
	}

	log.Printf("获取到区块高度: %d, 上次处理区块: %d", blockData.Height, bm.lastProcessedBlock)

	// 检查是否为新区块
	if blockData.Height <= bm.lastProcessedBlock {
		log.Printf("区块 %d 不是新区块，跳过", blockData.Height)
		return nil // 不是新区块，跳过
	}

	// 检查区块高度限制
	if bm.config.Monitor.MaxBlockHeight > 0 && blockData.Height > bm.config.Monitor.MaxBlockHeight {
		log.Printf("区块高度 %d 超过限制 %d，跳过", blockData.Height, bm.config.Monitor.MaxBlockHeight)
		return nil
	}

	// 检查起始区块高度
	if bm.config.Monitor.StartBlockHeight > 0 && blockData.Height < bm.config.Monitor.StartBlockHeight {
		log.Printf("区块高度 %d 低于起始高度 %d，跳过", blockData.Height, bm.config.Monitor.StartBlockHeight)
		return nil
	}

	// 处理缺失的区块（限制最多处理10个区块，避免性能问题）
	startBlock := bm.lastProcessedBlock + 1
	endBlock := blockData.Height
	maxGap := int64(10) // 最多处理10个缺失区块

	if startBlock < endBlock {
		gap := endBlock - startBlock + 1
		if gap > maxGap {
			log.Printf("缺失区块过多 (%d 个)，只处理最近的 %d 个区块", gap, maxGap)
			startBlock = endBlock - maxGap + 1
		}
		
		log.Printf("发现缺失区块，处理区块范围: %d - %d", startBlock, endBlock)
		
		for blockNum := startBlock; blockNum <= endBlock; blockNum++ {
			// 获取特定区块
			specificBlockData, err := bm.httpClient.GetBlockByNumber(bm.ctx, blockNum)
			if err != nil {
				log.Printf("获取区块 %d 失败: %v", blockNum, err)
				continue
			}

			// 推送区块数据到Redis队列
			if err := bm.redisClient.PushBlockData(bm.ctx, specificBlockData); err != nil {
				log.Printf("推送区块 %d 数据到队列失败: %v", blockNum, err)
				continue
			}

			log.Printf("已处理缺失区块 %d", blockNum)
			bm.processedBlocks++
		}
	} else {
		// 推送最新区块数据到Redis队列
		if err := bm.redisClient.PushBlockData(bm.ctx, blockData); err != nil {
			return fmt.Errorf("推送区块数据到队列失败: %w", err)
		}
	}

	// 更新统计信息
	bm.lastProcessedBlock = blockData.Height
	bm.processedBlocks++

	log.Printf("已处理区块 %d，队列大小: %d", blockData.Height, bm.getQueueSize())

	return nil
}

// getQueueSize 获取队列大小
func (bm *BlockMonitor) getQueueSize() int64 {
	size, err := bm.redisClient.GetQueueSize(bm.ctx)
	if err != nil {
		log.Printf("获取队列大小失败: %v", err)
		return 0
	}
	return size
}

// GetStats 获取监控器统计信息
func (bm *BlockMonitor) GetStats() map[string]interface{} {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	queueSize, _ := bm.redisClient.GetQueueSize(bm.ctx)

	return map[string]interface{}{
		"running":              bm.running,
		"last_processed_block": bm.lastProcessedBlock,
		"processed_blocks":     bm.processedBlocks,
		"errors":               bm.errors,
		"queue_size":           queueSize,
		"block_interval":       bm.config.Monitor.BlockInterval,
	}
}

// ProcessHistoricalBlocks 处理历史区块
func (bm *BlockMonitor) ProcessHistoricalBlocks(startBlock, endBlock int64) error {
	log.Printf("开始处理历史区块: %d - %d", startBlock, endBlock)

	for blockNum := startBlock; blockNum <= endBlock; blockNum++ {
		select {
		case <-bm.ctx.Done():
			return fmt.Errorf("处理被中断")
		default:
		}

		// 获取区块数据
		blockData, err := bm.httpClient.GetBlockByNumber(bm.ctx, blockNum)
		if err != nil {
			log.Printf("获取区块 %d 失败: %v", blockNum, err)
			continue
		}

		// 推送区块数据到Redis队列
		if err := bm.redisClient.PushBlockData(bm.ctx, blockData); err != nil {
			log.Printf("推送区块 %d 到队列失败: %v", blockNum, err)
			continue
		}

		log.Printf("已处理历史区块 %d", blockNum)
	}

	log.Printf("历史区块处理完成")
	return nil
}

// SyncToLatestBlock 同步到最新区块
func (bm *BlockMonitor) SyncToLatestBlock() error {
	// 获取最新区块
	latestBlock, err := bm.httpClient.GetLatestBlock(bm.ctx)
	if err != nil {
		return fmt.Errorf("获取最新区块失败: %w", err)
	}

	latestHeight := latestBlock.Height
	currentHeight := bm.lastProcessedBlock

	if currentHeight >= latestHeight {
		log.Printf("已是最新区块，当前: %d, 最新: %d", currentHeight, latestHeight)
		return nil
	}

	log.Printf("开始同步区块: %d -> %d", currentHeight+1, latestHeight)

	return bm.ProcessHistoricalBlocks(currentHeight+1, latestHeight)
}

// ResetStats 重置统计信息
func (bm *BlockMonitor) ResetStats() {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	bm.processedBlocks = 0
	bm.errors = 0
	bm.lastProcessedBlock = 0
}

// GetLastProcessedBlock 获取最后处理的区块高度
func (bm *BlockMonitor) GetLastProcessedBlock() int64 {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	return bm.lastProcessedBlock
}

// SetLastProcessedBlock 设置最后处理的区块高度
func (bm *BlockMonitor) SetLastProcessedBlock(height int64) {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	bm.lastProcessedBlock = height
}
