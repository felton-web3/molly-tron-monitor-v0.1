package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"time"

	"tron-monitor/config"
	"tron-monitor/models"
)

// HTTPClient HTTP客户端
type HTTPClient struct {
	config     *config.Config
	client     *http.Client
	baseURL    string
	timeout    time.Duration
	retryMax   int
	retryDelay time.Duration

	// 请求统计相关字段
	requestCount    int64
	lastRequestTime time.Time
	errorCount      int64
	successCount    int64
}

// NewHTTPClient 创建HTTP客户端
func NewHTTPClient(cfg *config.Config) *HTTPClient {
	return &HTTPClient{
		config:     cfg,
		baseURL:    cfg.TronGrid.BaseURL,
		timeout:    cfg.TronGrid.Timeout,
		retryMax:   cfg.TronGrid.RetryMax,
		retryDelay: cfg.TronGrid.RetryDelay,
		client: &http.Client{
			Timeout: cfg.TronGrid.Timeout,
		},
	}
}

// GetLatestBlock 获取最新区块
func (c *HTTPClient) GetLatestBlock(ctx context.Context) (*models.BlockData, error) {
	url := fmt.Sprintf("%s/wallet/getnowblock", c.baseURL)

	// 先解析为原始响应结构
	var rawResponse struct {
		BlockID      string                `json:"blockID"`
		BlockHeader  *models.BlockHeader   `json:"block_header"`
		Transactions []*models.Transaction `json:"transactions"`
	}

	err := c.makeRequest(ctx, "GET", url, nil, &rawResponse)
	if err != nil {
		return nil, fmt.Errorf("获取最新区块失败: %w", err)
	}

	// 构建 BlockData
	blockData := &models.BlockData{
		BlockHash: rawResponse.BlockID,
		CreatedAt: time.Now(),
	}

	// 从区块头中获取区块高度和时间戳
	if rawResponse.BlockHeader != nil && rawResponse.BlockHeader.RawData != nil {
		blockData.Height = rawResponse.BlockHeader.RawData.Number
		blockData.Timestamp = rawResponse.BlockHeader.RawData.Timestamp
	}

	// 构建 Block 结构
	blockData.Block = &models.Block{
		BlockHeader: rawResponse.BlockHeader,
		Trans:       rawResponse.Transactions,
	}

	// 更新统计信息
	atomic.AddInt64(&c.successCount, 1)
	atomic.StoreInt64(&c.requestCount, atomic.AddInt64(&c.requestCount, 1))
	c.lastRequestTime = time.Now()

	return blockData, nil
}

// GetBlockByNumber 根据区块号获取区块
func (c *HTTPClient) GetBlockByNumber(ctx context.Context, blockNumber int64) (*models.BlockData, error) {
	url := fmt.Sprintf("%s/wallet/getblockbynum", c.baseURL)

	requestBody := map[string]interface{}{
		"num": blockNumber,
	}

	// 先解析为原始响应结构
	var rawResponse struct {
		BlockID      string                `json:"blockID"`
		BlockHeader  *models.BlockHeader   `json:"block_header"`
		Transactions []*models.Transaction `json:"transactions"`
	}

	err := c.makeRequest(ctx, "POST", url, requestBody, &rawResponse)
	if err != nil {
		return nil, fmt.Errorf("获取区块 %d 失败: %w", blockNumber, err)
	}

	// 构建 BlockData
	blockData := &models.BlockData{
		BlockHash: rawResponse.BlockID,
		CreatedAt: time.Now(),
	}

	// 从区块头中获取区块高度和时间戳
	if rawResponse.BlockHeader != nil && rawResponse.BlockHeader.RawData != nil {
		blockData.Height = rawResponse.BlockHeader.RawData.Number
		blockData.Timestamp = rawResponse.BlockHeader.RawData.Timestamp
	}

	// 构建 Block 结构
	blockData.Block = &models.Block{
		BlockHeader: rawResponse.BlockHeader,
		Trans:       rawResponse.Transactions,
	}

	// 更新统计信息
	atomic.AddInt64(&c.successCount, 1)
	atomic.StoreInt64(&c.requestCount, atomic.AddInt64(&c.requestCount, 1))
	c.lastRequestTime = time.Now()

	return blockData, nil
}

// GetTransactionInfo 获取交易信息
func (c *HTTPClient) GetTransactionInfo(ctx context.Context, txID string) (*models.TransactionInfo, error) {
	url := fmt.Sprintf("%s/wallet/gettransactioninfobyid", c.baseURL)

	requestBody := map[string]string{
		"value": txID,
	}

	var txInfo models.TransactionInfo
	err := c.makeRequest(ctx, "POST", url, requestBody, &txInfo)
	if err != nil {
		return nil, fmt.Errorf("获取交易信息失败: %w", err)
	}

	return &txInfo, nil
}

// GetAccountInfo 获取账户信息
func (c *HTTPClient) GetAccountInfo(ctx context.Context, address string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/v1/accounts/%s", c.baseURL, address)

	var accountInfo map[string]interface{}
	err := c.makeRequest(ctx, "GET", url, nil, &accountInfo)
	if err != nil {
		return nil, fmt.Errorf("获取账户信息失败: %w", err)
	}

	return accountInfo, nil
}

// GetTokenTransfers 获取代币转账记录
func (c *HTTPClient) GetTokenTransfers(ctx context.Context, address string, limit int) ([]map[string]interface{}, error) {
	url := fmt.Sprintf("%s/v1/accounts/%s/transactions/trc20", c.baseURL, address)
	if limit > 0 {
		url = fmt.Sprintf("%s?limit=%d", url, limit)
	}

	var transfers []map[string]interface{}
	err := c.makeRequest(ctx, "GET", url, nil, &transfers)
	if err != nil {
		return nil, fmt.Errorf("获取代币转账记录失败: %w", err)
	}

	return transfers, nil
}

// makeRequest 执行HTTP请求
func (c *HTTPClient) makeRequest(ctx context.Context, method, url string, body interface{}, result interface{}) error {
	var requestBody []byte
	var err error

	if body != nil {
		requestBody, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("序列化请求体失败: %w", err)
		}
	}

	// 重试机制
	var lastErr error
	for i := 0; i <= c.retryMax; i++ {
		if i > 0 {
			// 等待重试延迟
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(c.retryDelay):
			}
		}

		err := c.doRequest(ctx, method, url, requestBody, result)
		if err == nil {
			return nil
		}

		lastErr = err
		atomic.AddInt64(&c.errorCount, 1)

		// 如果不是最后一次重试，继续重试
		if i < c.retryMax {
			continue
		}
	}

	return fmt.Errorf("请求失败，已重试 %d 次: %w", c.retryMax+1, lastErr)
}

// doRequest 执行单次HTTP请求
func (c *HTTPClient) doRequest(ctx context.Context, method, url string, body []byte, result interface{}) error {
	var req *http.Request
	var err error

	if body != nil {
		req, err = http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(body))
		if err != nil {
			return fmt.Errorf("创建请求失败: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequestWithContext(ctx, method, url, nil)
		if err != nil {
			return fmt.Errorf("创建请求失败: %w", err)
		}
	}

	// 添加API密钥（如果配置了）
	if c.config.TronGrid.APIKey != "" {
		req.Header.Set("TRON-PRO-API-KEY", c.config.TronGrid.APIKey)
	}

	// 添加用户代理
	req.Header.Set("User-Agent", "TronMonitor/1.0")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应体失败: %w", err)
	}

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
	}

	// 解析响应
	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("解析响应失败: %w", err)
		}
	}

	return nil
}

// GetStats 获取请求统计信息
func (c *HTTPClient) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"request_count":     atomic.LoadInt64(&c.requestCount),
		"success_count":     atomic.LoadInt64(&c.successCount),
		"error_count":       atomic.LoadInt64(&c.errorCount),
		"last_request_time": c.lastRequestTime,
		"success_rate": func() float64 {
			total := atomic.LoadInt64(&c.requestCount)
			if total == 0 {
				return 0
			}
			return float64(atomic.LoadInt64(&c.successCount)) / float64(total) * 100
		}(),
	}
}

// ResetStats 重置统计信息
func (c *HTTPClient) ResetStats() {
	atomic.StoreInt64(&c.requestCount, 0)
	atomic.StoreInt64(&c.successCount, 0)
	atomic.StoreInt64(&c.errorCount, 0)
	c.lastRequestTime = time.Time{}
}
