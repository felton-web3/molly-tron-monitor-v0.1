package processor

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"tron-monitor/config"
	"tron-monitor/http"
	"tron-monitor/models"
	"tron-monitor/redis"

	"github.com/btcsuite/btcutil/base58"
)

// BlockProcessor 区块处理器
type BlockProcessor struct {
	config      *config.Config
	redisClient *redis.RedisClient
	httpClient  *http.HTTPClient
	workers     []*BlockWorker
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	running     bool
	mu          sync.RWMutex

	// 统计信息
	processedBlocks int64
	transfersFound  int64
	errors          int64
}

// BlockWorker 区块工作线程
type BlockWorker struct {
	id        int
	processor *BlockProcessor
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	running   bool
	mu        sync.RWMutex
}

// NewBlockProcessor 创建区块处理器
func NewBlockProcessor(cfg *config.Config, redisClient *redis.RedisClient, httpClient *http.HTTPClient) *BlockProcessor {
	ctx, cancel := context.WithCancel(context.Background())

	processor := &BlockProcessor{
		config:      cfg,
		redisClient: redisClient,
		httpClient:  httpClient,
		ctx:         ctx,
		cancel:      cancel,
	}

	// 创建工作线程
	processor.workers = make([]*BlockWorker, cfg.Monitor.WorkerCount)
	for i := 0; i < cfg.Monitor.WorkerCount; i++ {
		workerCtx, workerCancel := context.WithCancel(ctx)
		processor.workers[i] = &BlockWorker{
			id:        i,
			processor: processor,
			ctx:       workerCtx,
			cancel:    workerCancel,
		}
	}

	return processor
}

// Start 启动区块处理器
func (bp *BlockProcessor) Start() error {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	if bp.running {
		return fmt.Errorf("区块处理器已在运行")
	}

	bp.running = true

	// 启动所有工作线程
	for _, worker := range bp.workers {
		bp.wg.Add(1)
		go func(w *BlockWorker) {
			defer bp.wg.Done()
			w.start()
		}(worker)
	}

	log.Printf("区块处理器已启动，工作线程数: %d", len(bp.workers))
	return nil
}

// Stop 停止区块处理器
func (bp *BlockProcessor) Stop() error {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	if !bp.running {
		return fmt.Errorf("区块处理器未运行")
	}

	bp.running = false
	bp.cancel()

	// 停止所有工作线程
	for _, worker := range bp.workers {
		worker.stop()
	}

	bp.wg.Wait()
	log.Println("区块处理器已停止")
	return nil
}

// IsRunning 检查是否正在运行
func (bp *BlockProcessor) IsRunning() bool {
	bp.mu.RLock()
	defer bp.mu.RUnlock()
	return bp.running
}

// GetStats 获取处理器统计信息
func (bp *BlockProcessor) GetStats() map[string]interface{} {
	bp.mu.RLock()
	defer bp.mu.RUnlock()

	return map[string]interface{}{
		"running":          bp.running,
		"processed_blocks": bp.processedBlocks,
		"transfers_found":  bp.transfersFound,
		"errors":           bp.errors,
		"worker_count":     len(bp.workers),
	}
}

// ResetStats 重置统计信息
func (bp *BlockProcessor) ResetStats() {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	bp.processedBlocks = 0
	bp.transfersFound = 0
	bp.errors = 0
}

// start 启动工作线程
func (w *BlockWorker) start() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.running {
		return
	}

	w.running = true
	w.wg.Add(1)

	go func() {
		defer w.wg.Done()
		w.processBlocks()
	}()

	log.Printf("工作线程 %d 已启动", w.id)
}

// stop 停止工作线程
func (w *BlockWorker) stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.running {
		return
	}

	w.running = false
	w.cancel()
	w.wg.Wait()

	log.Printf("工作线程 %d 已停止", w.id)
}

// processBlocks 处理区块循环
func (w *BlockWorker) processBlocks() {
	for {
		select {
		case <-w.ctx.Done():
			return
		default:
		}

		// 从Redis队列获取区块数据
		blockData, err := w.processor.redisClient.PopBlockData(w.ctx)
		if err != nil {
			log.Printf("工作线程 %d: 获取区块数据失败: %v", w.id, err)
			time.Sleep(time.Second)
			continue
		}

		if blockData == nil {
			// 队列为空，等待一段时间
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// 处理区块
		if err := w.processBlock(blockData); err != nil {
			log.Printf("工作线程 %d: 处理区块 %d 失败: %v", w.id, blockData.Height, err)
			w.processor.errors++
		} else {
			w.processor.processedBlocks++
		}
	}
}

// processBlock 处理单个区块
func (w *BlockWorker) processBlock(blockData *models.BlockData) error {
	log.Printf("工作线程 %d: 处理区块 %d，Block: %v, Trans: %v",
		w.id, blockData.Height, blockData.Block != nil,
		func() interface{} {
			if blockData.Block != nil {
				return len(blockData.Block.Trans)
			}
			return "nil"
		}())

	if blockData.Block == nil || blockData.Block.Trans == nil {
		return fmt.Errorf("区块数据无效")
	}

	var transfers []*models.TransferEvent

	// 处理区块中的每个交易
	for _, tx := range blockData.Block.Trans {
		txTransfers, err := w.extractTransfers(tx, blockData)
		if err != nil {
			log.Printf("工作线程 %d: 提取交易 %s 的转账信息失败: %v", w.id, tx.TxID, err)
			continue
		}

		transfers = append(transfers, txTransfers...)
	}

	// 保存转账事件
	for _, transfer := range transfers {
		if err := w.processor.redisClient.SaveTransferEvent(w.ctx, transfer); err != nil {
			log.Printf("工作线程 %d: 保存转账事件失败: %v", w.id, err)
			continue
		}

		w.processor.transfersFound++
	}

	return nil
}

// extractTransfers 提取转账事件
func (w *BlockWorker) extractTransfers(tx *models.Transaction, blockData *models.BlockData) ([]*models.TransferEvent, error) {
	var transfers []*models.TransferEvent

	if tx.RawData == nil || len(tx.RawData.Contract) == 0 {
		return transfers, nil
	}

	// 获取监控地址集合
	watchAddresses, err := w.processor.redisClient.GetWatchAddresses(w.ctx)
	if err != nil {
		return nil, fmt.Errorf("获取监控地址失败: %w", err)
	}

	watchAddressSet := make(map[string]bool)
	for _, addr := range watchAddresses {
		watchAddressSet[addr] = true
	}

	// 处理每个合约
	for _, contract := range tx.RawData.Contract {
		transfer, err := w.extractTransferFromContract(contract, tx, blockData, watchAddressSet)
		if err != nil {
			log.Printf("提取合约转账信息失败: %v", err)
			continue
		}

		if transfer != nil {
			transfers = append(transfers, transfer)
		}
	}

	return transfers, nil
}

// extractTransferFromContract 从合约中提取转账信息
func (w *BlockWorker) extractTransferFromContract(contract *models.Contract, tx *models.Transaction, blockData *models.BlockData, watchAddressSet map[string]bool) (*models.TransferEvent, error) {
	switch contract.Type {
	case "TransferContract":
		return w.extractTRXTransfer(contract, tx, blockData, watchAddressSet)
	case "TransferAssetContract":
		return w.extractTRC10Transfer(contract, tx, blockData, watchAddressSet)
	case "TriggerSmartContract":
		return w.extractTRC20Transfer(contract, tx, blockData, watchAddressSet)
	default:
		return nil, nil // 不支持的合约类型
	}
}

// extractTRXTransfer 提取TRX转账
func (w *BlockWorker) extractTRXTransfer(contract *models.Contract, tx *models.Transaction, blockData *models.BlockData, watchAddressSet map[string]bool) (*models.TransferEvent, error) {
	// 解析转账合约参数
	paramData, ok := contract.Parameter.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("无效的转账合约参数")
	}

	// 获取value字段中的实际转账数据
	valueData, ok := paramData["value"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("无效的转账value数据")
	}

	ownerAddress, _ := valueData["owner_address"].(string)
	toAddress, _ := valueData["to_address"].(string)
	amount, _ := valueData["amount"].(float64)

	// 转换地址格式
	fromAddr := w.convertHexToBase58(ownerAddress)
	toAddr := w.convertHexToBase58(toAddress)

	// 检查是否涉及监控地址（暂时注释掉，显示所有转账事件）
	// if !watchAddressSet[fromAddr] && !watchAddressSet[toAddr] {
	// 	return nil, nil
	// }

	// 显示转账详情
	transferTime := time.Unix(blockData.Timestamp/1000, 0).Format("2006-01-02 15:04:05")
	log.Printf("TRX转账事件 - From: %s, To: %s, Amount: %.6f TRX, Time: %s, TxHash: %s",
		fromAddr, toAddr, amount/1e6, transferTime, tx.TxID)

	// 更新地址统计信息
	if watchAddressSet[fromAddr] {
		w.updateAddressStats(fromAddr, blockData)
	}
	if watchAddressSet[toAddr] {
		w.updateAddressStats(toAddr, blockData)
	}

	return &models.TransferEvent{
		Source:      fromAddr,
		Destination: toAddr,
		Amount:      amount / 1e6, // TRX精度为6位小数
		Fee:         0,            // 需要从交易收据获取
		TxHash:      tx.TxID,
		BlockHeight: blockData.Height,
		Timestamp:   blockData.Timestamp,
		TokenType:   "TRX",
	}, nil
}

// extractTRC10Transfer 提取TRC10代币转账
func (w *BlockWorker) extractTRC10Transfer(contract *models.Contract, tx *models.Transaction, blockData *models.BlockData, watchAddressSet map[string]bool) (*models.TransferEvent, error) {
	// 解析资产转账合约参数
	paramData, ok := contract.Parameter.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("无效的资产转账合约参数")
	}

	// 获取value字段中的实际转账数据
	valueData, ok := paramData["value"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("无效的资产转账value数据")
	}

	ownerAddressHex, _ := valueData["owner_address"].(string)
	toAddressHex, _ := valueData["to_address"].(string)
	amount, _ := valueData["amount"].(float64)
	assetName, _ := valueData["asset_name"].(string)

	// 将十六进制地址转换为base58格式的TRX地址
	ownerAddress := w.convertHexToBase58(ownerAddressHex)
	toAddress := w.convertHexToBase58(toAddressHex)

	// 检查是否涉及监控地址
	if !watchAddressSet[ownerAddress] && !watchAddressSet[toAddress] {
		return nil, nil
	}

	// 显示转账详情
	transferTime := time.Unix(blockData.Timestamp/1000, 0).Format("2006-01-02 15:04:05")
	log.Printf("TRC10转账事件 - From: %s, To: %s, Amount: %.0f %s, Time: %s, TxHash: %s",
		ownerAddress, toAddress, amount, assetName, transferTime, tx.TxID)

	// 更新地址统计信息
	if watchAddressSet[ownerAddress] {
		w.updateAddressStats(ownerAddress, blockData)
	}
	if watchAddressSet[toAddress] {
		w.updateAddressStats(toAddress, blockData)
	}

	return &models.TransferEvent{
		Source:      ownerAddress,
		Destination: toAddress,
		Amount:      amount,
		Fee:         0,
		TxHash:      tx.TxID,
		BlockHeight: blockData.Height,
		Timestamp:   blockData.Timestamp,
		TokenType:   "TRC10",
		AssetName:   assetName,
	}, nil
}

// extractTRC20Transfer 提取TRC20代币转账
func (w *BlockWorker) extractTRC20Transfer(contract *models.Contract, tx *models.Transaction, blockData *models.BlockData, watchAddressSet map[string]bool) (*models.TransferEvent, error) {
	// 解析智能合约触发参数

	paramData, ok := contract.Parameter.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("无效的智能合约参数")
	}

	// 获取value字段中的实际转账数据
	valueData, ok := paramData["value"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("无效的智能合约value数据")
	}
	ownerAddressHex, _ := valueData["owner_address"].(string)
	contractAddressHex, _ := valueData["contract_address"].(string)
	data, _ := valueData["data"].(string)

	// 将十六进制地址转换为base58格式的TRX地址
	ownerAddress := w.convertHexToBase58(ownerAddressHex)
	contractAddress := w.convertHexToBase58(contractAddressHex)
	// 检查是否为USDT转账
	isUSDT := w.isUSDTContract(contractAddress)

	// 如果是USDT转账，检查是否启用USDT监控
	if isUSDT && !w.processor.config.USDT.EnableMonitoring {
		log.Printf("USDT监控已禁用，跳过处理")
		return nil, nil
	}

	// 解析TRC20转账数据
	transfer, err := w.parseTRC20TransferData(data, ownerAddress, contractAddress, tx, blockData, isUSDT)
	if err != nil {
		log.Printf("解析TRC20转账数据失败: %v", err)
		return nil, err
	}
	if transfer != nil {
		// 显示转账详情（非USDT的TRC20转账）
		if !transfer.IsUSDT {
			transferTime := time.Unix(blockData.Timestamp/1000, 0).Format("2006-01-02 15:04:05")
			log.Printf("TRC20转账事件 - From: %s, To: %s, Amount: %.0f, Contract: %s, Time: %s, TxHash: %s",
				transfer.Source, transfer.Destination, transfer.Amount, contractAddress, transferTime, tx.TxID)
		}

		// 检查是否涉及监控地址（发送方或接收方）
		if !watchAddressSet[transfer.Source] && !watchAddressSet[transfer.Destination] {
			return nil, nil
		}

		// 更新地址统计信息
		if watchAddressSet[transfer.Source] {
			w.updateAddressStats(transfer.Source, blockData)
		}
		if watchAddressSet[transfer.Destination] {
			w.updateAddressStats(transfer.Destination, blockData)
		}
	}

	return transfer, nil
}

// isUSDTContract 检查是否为USDT合约
func (w *BlockWorker) isUSDTContract(contractAddress string) bool {
	return contractAddress == w.processor.config.USDT.ContractAddress
}

// parseTRC20TransferData 解析TRC20转账数据
func (w *BlockWorker) parseTRC20TransferData(data, ownerAddress, contractAddress string, tx *models.Transaction, blockData *models.BlockData, isUSDT bool) (*models.TransferEvent, error) {
	// TRC20 transfer函数的数据格式为: a9059cbb + 32字节的to地址 + 32字节的amount
	// 安全获取数据前缀
	dataPrefix := data
	if len(data) > 10 {
		dataPrefix = data[:10]
	}
	if len(data) < 74 || !strings.HasPrefix(data, "a9059cbb") {
		log.Printf("数据不符合TRC20 transfer格式 - 长度: %d, 前缀: %s", len(data), dataPrefix)
		return nil, nil // 不是transfer调用
	}

	// 数据没有0x前缀，直接移除函数选择器 (a9059cbb)
	if len(data) < 8 {
		log.Printf("数据长度不足，无法移除函数选择器: %d", len(data))
		return nil, fmt.Errorf("数据长度不足")
	}
	data = data[8:]

	// 解析接收地址 (32字节，64个十六进制字符)
	if len(data) < 64 {
		log.Printf("地址数据长度不足: %d", len(data))
		return nil, fmt.Errorf("地址数据长度不足")
	}
	toAddressHex := data[:64]

	// 移除地址部分，获取金额数据
	data = data[64:]

	// 解析金额 (32字节，64个十六进制字符)
	if len(data) < 64 {
		log.Printf("金额数据长度不足: %d", len(data))
		return nil, fmt.Errorf("金额数据长度不足")
	}
	amountHex := data[:64]

	// 转换地址格式 (从hex转换为base58)
	// 移除前导零，确保地址格式正确
	toAddressHex = strings.TrimLeft(toAddressHex, "0")
	if len(toAddressHex) < 40 {
		// 如果地址长度不足40个字符，在前面补0
		toAddressHex = strings.Repeat("0", 40-len(toAddressHex)) + toAddressHex
	}

	// 添加41前缀（Tron地址前缀）
	fullAddressHex := "41" + toAddressHex

	// 转换为Base58格式
	toAddress := w.convertHexToBase58(fullAddressHex)

	// 解析金额
	amount, err := w.parseHexAmount(amountHex)
	if err != nil {
		log.Printf("解析金额失败: %v", err)
		return nil, fmt.Errorf("解析金额失败: %w", err)
	}

	// 如果是USDT，需要根据精度调整金额
	if isUSDT {
		amount = amount / math.Pow(10, float64(w.processor.config.USDT.Decimals))
	}

	tokenType := "TRC20"
	if isUSDT {
		tokenType = "USDT"
	}

	// 转换发送方地址格式
	fromAddress := w.convertHexToBase58(ownerAddress)

	// 如果是USDT转账，立即打印出来
	if isUSDT {
		transferTime := time.Unix(blockData.Timestamp/1000, 0).Format("2006-01-02 15:04:05")
		log.Printf("USDT转账事件 - From: %s, To: %s, Amount: %.6f USDT, Time: %s, TxHash: %s",
			fromAddress, toAddress, amount, transferTime, tx.TxID)
	}

	return &models.TransferEvent{
		Source:          fromAddress,
		Destination:     toAddress,
		Amount:          amount,
		Fee:             0,
		TxHash:          tx.TxID,
		BlockHeight:     blockData.Height,
		Timestamp:       blockData.Timestamp,
		TokenType:       tokenType,
		ContractAddress: contractAddress,
		IsUSDT:          isUSDT,
		USDValue:        amount, // USDT的USD价值等于其数量
	}, nil
}

// parseHexAmount 解析十六进制金额
func (w *BlockWorker) parseHexAmount(hexStr string) (float64, error) {
	// 移除前导零
	hexStr = strings.TrimLeft(hexStr, "0")
	if hexStr == "" {
		return 0, nil
	}

	// 转换为十进制
	amount, err := strconv.ParseUint(hexStr, 16, 64)
	if err != nil {
		return 0, fmt.Errorf("解析十六进制金额失败: %w", err)
	}

	return float64(amount), nil
}

// convertHexToBase58 将hex地址转换为base58格式
func (w *BlockWorker) convertHexToBase58(hexAddr string) string {
	// 移除0x前缀
	hexAddr = strings.TrimPrefix(hexAddr, "0x")

	// 如果地址为空或格式不正确，返回原地址
	if len(hexAddr) == 0 {
		return hexAddr
	}

	// 如果地址已经是base58格式（以T开头），直接返回
	if strings.HasPrefix(hexAddr, "T") {
		return hexAddr
	}

	// 使用正确的Tron地址转换方法
	tronAddress, err := w.hexToTronAddress(hexAddr)
	if err != nil {
		log.Printf("地址转换失败: %v", err)
		return hexAddr
	}

	return tronAddress
}

// hexToTronAddress 将TRON的十六进制地址转换为Base58Check格式的地址
func (w *BlockWorker) hexToTronAddress(hexAddress string) (string, error) {
	// 1. 解码十六进制字符串为字节数组
	// TRON地址的十六进制表示通常以 "41" 开头
	addressBytes, err := hex.DecodeString(hexAddress)
	if err != nil {
		return "", fmt.Errorf("解码十六进制地址失败: %v", err)
	}

	// 2. 计算校验和：对地址字节进行两次SHA256哈希，取结果的前4个字节
	hash1 := sha256.Sum256(addressBytes)
	hash2 := sha256.Sum256(hash1[:])
	checksum := hash2[:4]

	// 3. 将校验和追加到地址字节数组的末尾
	payload := append(addressBytes, checksum...)

	// 4. 使用Base58对拼接后的数据进行编码
	tronAddress := base58.Encode(payload)

	return tronAddress, nil
}

// updateAddressStats 更新地址统计信息
func (w *BlockWorker) updateAddressStats(address string, blockData *models.BlockData) {
	// 创建临时的转账事件用于更新统计
	tempEvent := &models.TransferEvent{
		Source:      address,
		Destination: address,
		Amount:      0,
		Fee:         0,
		TxHash:      "",
		BlockHeight: blockData.Height,
		Timestamp:   blockData.Timestamp,
		TokenType:   "TRX",
	}

	if err := w.processor.redisClient.UpdateAddressStats(w.ctx, address, tempEvent); err != nil {
		log.Printf("更新地址 %s 统计信息失败: %v", address, err)
	}
}
