package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"tron-monitor/config"
	"tron-monitor/models"
)

// RedisClient Redis客户端
type RedisClient struct {
	client *redis.Client
	config *config.Config
}

// NewRedisClient 创建Redis客户端
func NewRedisClient(cfg *config.Config) (*RedisClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
		PoolSize: cfg.Redis.PoolSize,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("Redis连接失败: %w", err)
	}

	return &RedisClient{
		client: client,
		config: cfg,
	}, nil
}

// Close 关闭Redis连接
func (r *RedisClient) Close() error {
	return r.client.Close()
}

// PushBlockData 推送区块数据到队列
func (r *RedisClient) PushBlockData(ctx context.Context, blockData *models.BlockData) error {
	data, err := json.Marshal(blockData)
	if err != nil {
		return fmt.Errorf("序列化区块数据失败: %w", err)
	}

	key := "block_queue"
	err = r.client.LPush(ctx, key, data).Err()
	if err != nil {
		return fmt.Errorf("推送区块数据到队列失败: %w", err)
	}

	// 限制队列大小
	r.client.LTrim(ctx, key, 0, int64(r.config.Monitor.QueueSize-1))

	return nil
}

// PopBlockData 从队列弹出区块数据
func (r *RedisClient) PopBlockData(ctx context.Context) (*models.BlockData, error) {
	key := "block_queue"
	result, err := r.client.BRPop(ctx, 5*time.Second, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // 队列为空
		}
		return nil, fmt.Errorf("从队列弹出区块数据失败: %w", err)
	}

	if len(result) < 2 {
		return nil, fmt.Errorf("队列数据格式错误")
	}

	var blockData models.BlockData
	if err := json.Unmarshal([]byte(result[1]), &blockData); err != nil {
		return nil, fmt.Errorf("反序列化区块数据失败: %w", err)
	}

	return &blockData, nil
}

// SaveTransferEvent 保存转账事件
func (r *RedisClient) SaveTransferEvent(ctx context.Context, event *models.TransferEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("序列化转账事件失败: %w", err)
	}

	// 使用交易哈希作为键
	key := fmt.Sprintf("transfer:%s", event.TxHash)
	err = r.client.Set(ctx, key, data, 24*time.Hour).Err()
	if err != nil {
		return fmt.Errorf("保存转账事件失败: %w", err)
	}

	// 添加到转账列表
	listKey := "transfers"
	r.client.LPush(ctx, listKey, data)
	r.client.LTrim(ctx, listKey, 0, 9999) // 保留最近10000条记录

	// 如果是USDT转账，单独保存到USDT转账列表
	if event.IsUSDT {
		usdtListKey := "usdt_transfers"
		r.client.LPush(ctx, usdtListKey, data)
		r.client.LTrim(ctx, usdtListKey, 0, 9999) // 保留最近10000条USDT转账记录
	}

	return nil
}

// GetTransferEvent 获取转账事件
func (r *RedisClient) GetTransferEvent(ctx context.Context, txHash string) (*models.TransferEvent, error) {
	key := fmt.Sprintf("transfer:%s", txHash)
	data, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("获取转账事件失败: %w", err)
	}

	var event models.TransferEvent
	if err := json.Unmarshal([]byte(data), &event); err != nil {
		return nil, fmt.Errorf("反序列化转账事件失败: %w", err)
	}

	return &event, nil
}

// AddWatchAddress 添加监控地址
func (r *RedisClient) AddWatchAddress(ctx context.Context, address string) error {
	key := "watch_addresses"
	err := r.client.SAdd(ctx, key, address).Err()
	if err != nil {
		return fmt.Errorf("添加监控地址失败: %w", err)
	}

	// 保存地址信息
	addrInfo := models.WatchAddress{
		Address: address,
		AddedAt: time.Now(),
	}
	
	addrData, err := json.Marshal(addrInfo)
	if err != nil {
		return fmt.Errorf("序列化地址信息失败: %w", err)
	}

	addrKey := fmt.Sprintf("address_info:%s", address)
	err = r.client.Set(ctx, addrKey, addrData, 0).Err()
	if err != nil {
		return fmt.Errorf("保存地址信息失败: %w", err)
	}

	return nil
}

// RemoveWatchAddress 移除监控地址
func (r *RedisClient) RemoveWatchAddress(ctx context.Context, address string) error {
	key := "watch_addresses"
	err := r.client.SRem(ctx, key, address).Err()
	if err != nil {
		return fmt.Errorf("移除监控地址失败: %w", err)
	}

	// 删除地址信息
	addrKey := fmt.Sprintf("address_info:%s", address)
	r.client.Del(ctx, addrKey)

	return nil
}

// GetWatchAddresses 获取所有监控地址
func (r *RedisClient) GetWatchAddresses(ctx context.Context) ([]string, error) {
	key := "watch_addresses"
	addresses, err := r.client.SMembers(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("获取监控地址失败: %w", err)
	}

	return addresses, nil
}

// IsWatchAddress 检查是否为监控地址
func (r *RedisClient) IsWatchAddress(ctx context.Context, address string) (bool, error) {
	key := "watch_addresses"
	exists, err := r.client.SIsMember(ctx, key, address).Result()
	if err != nil {
		return false, fmt.Errorf("检查监控地址失败: %w", err)
	}

	return exists, nil
}

// UpdateAddressStats 更新地址统计信息
func (r *RedisClient) UpdateAddressStats(ctx context.Context, address string, event *models.TransferEvent) error {
	addrKey := fmt.Sprintf("address_info:%s", address)
	
	// 获取现有地址信息
	addrData, err := r.client.Get(ctx, addrKey).Result()
	var addrInfo models.WatchAddress
	
	if err == redis.Nil {
		// 地址信息不存在，创建新的
		addrInfo = models.WatchAddress{
			Address: address,
			AddedAt: time.Now(),
		}
	} else if err != nil {
		return fmt.Errorf("获取地址信息失败: %w", err)
	} else {
		// 解析现有地址信息
		if err := json.Unmarshal([]byte(addrData), &addrInfo); err != nil {
			return fmt.Errorf("反序列化地址信息失败: %w", err)
		}
	}

	// 更新统计信息
	addrInfo.LastSeen = time.Unix(event.Timestamp/1000, 0)
	addrInfo.TransferCount++

	// 保存更新后的地址信息
	newAddrData, err := json.Marshal(addrInfo)
	if err != nil {
		return fmt.Errorf("序列化地址信息失败: %w", err)
	}

	err = r.client.Set(ctx, addrKey, newAddrData, 0).Err()
	if err != nil {
		return fmt.Errorf("保存地址信息失败: %w", err)
	}

	return nil
}

// SaveSystemStats 保存系统统计信息
func (r *RedisClient) SaveSystemStats(ctx context.Context, stats *models.SystemStats) error {
	data, err := json.Marshal(stats)
	if err != nil {
		return fmt.Errorf("序列化系统统计信息失败: %w", err)
	}

	key := "system_stats"
	err = r.client.Set(ctx, key, data, 0).Err()
	if err != nil {
		return fmt.Errorf("保存系统统计信息失败: %w", err)
	}

	return nil
}

// GetSystemStats 获取系统统计信息
func (r *RedisClient) GetSystemStats(ctx context.Context) (*models.SystemStats, error) {
	key := "system_stats"
	data, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return &models.SystemStats{}, nil
		}
		return nil, fmt.Errorf("获取系统统计信息失败: %w", err)
	}

	var stats models.SystemStats
	if err := json.Unmarshal([]byte(data), &stats); err != nil {
		return nil, fmt.Errorf("反序列化系统统计信息失败: %w", err)
	}

	return &stats, nil
}

// GetQueueSize 获取队列大小
func (r *RedisClient) GetQueueSize(ctx context.Context) (int64, error) {
	key := "block_queue"
	size, err := r.client.LLen(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("获取队列大小失败: %w", err)
	}

	return size, nil
}

// ClearQueue 清空队列
func (r *RedisClient) ClearQueue(ctx context.Context) error {
	key := "block_queue"
	err := r.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("清空队列失败: %w", err)
	}

	return nil
}

// GetRecentTransfers 获取最近的转账记录
func (r *RedisClient) GetRecentTransfers(ctx context.Context, limit int64) ([]*models.TransferEvent, error) {
	key := "transfers"
	data, err := r.client.LRange(ctx, key, 0, limit-1).Result()
	if err != nil {
		return nil, fmt.Errorf("获取最近转账记录失败: %w", err)
	}

	var events []*models.TransferEvent
	for _, item := range data {
		var event models.TransferEvent
		if err := json.Unmarshal([]byte(item), &event); err != nil {
			continue // 跳过无效数据
		}
		events = append(events, &event)
	}

	return events, nil
}

// GetRecentUSDTTransfers 获取最近的USDT转账记录
func (r *RedisClient) GetRecentUSDTTransfers(ctx context.Context, limit int64) ([]*models.TransferEvent, error) {
	key := "usdt_transfers"
	data, err := r.client.LRange(ctx, key, 0, limit-1).Result()
	if err != nil {
		return nil, fmt.Errorf("获取最近USDT转账记录失败: %w", err)
	}

	var events []*models.TransferEvent
	for _, item := range data {
		var event models.TransferEvent
		if err := json.Unmarshal([]byte(item), &event); err != nil {
			continue // 跳过无效数据
		}
		events = append(events, &event)
	}

	return events, nil
}
