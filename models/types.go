package models

import (
	"time"
)

// BlockData 区块数据结构
type BlockData struct {
	Height    int64     `json:"height"`
	BlockHash string    `json:"blockID"`
	Timestamp int64     `json:"timestamp"`
	Block     *Block    `json:"block"`
	CreatedAt time.Time `json:"created_at"`
}

// Block Tron区块结构
type Block struct {
	BlockHeader *BlockHeader   `json:"block_header"`
	Trans       []*Transaction `json:"transactions"`
}

// BlockHeader 区块头结构
type BlockHeader struct {
	RawData          *BlockHeaderRaw `json:"raw_data"`
	WitnessSignature string          `json:"witness_signature"`
}

// BlockHeaderRaw 区块头原始数据
type BlockHeaderRaw struct {
	Timestamp        int64  `json:"timestamp"`
	TxTrieRoot       string `json:"txTrieRoot"`
	ParentHash       string `json:"parentHash"`
	Number           int64  `json:"number"`
	WitnessId        int64  `json:"witness_id"`
	WitnessAddress   string `json:"witness_address"`
	Version          int32  `json:"version"`
	AccountStateRoot string `json:"accountStateRoot"`
}

// Transaction 交易结构
type Transaction struct {
	RawData   *TransactionRaw      `json:"raw_data"`
	Signature []string             `json:"signature"`
	TxID      string               `json:"txID"`
	Ret       []*TransactionResult `json:"ret"`
}

// TransactionRaw 交易原始数据
type TransactionRaw struct {
	Contract      []*Contract `json:"contract"`
	RefBlockBytes string      `json:"ref_block_bytes"`
	RefBlockHash  string      `json:"ref_block_hash"`
	Expiration    int64       `json:"expiration"`
	Timestamp     int64       `json:"timestamp"`
	FeeLimit      int64       `json:"fee_limit"`
}

// Contract 合约结构
type Contract struct {
	Type      string      `json:"type"`
	Parameter interface{} `json:"parameter"`
}

// TransferContract 转账合约
type TransferContract struct {
	OwnerAddress string `json:"owner_address"`
	ToAddress    string `json:"to_address"`
	Amount       int64  `json:"amount"`
}

// TransferAssetContract 资产转账合约
type TransferAssetContract struct {
	AssetName    string `json:"asset_name"`
	OwnerAddress string `json:"owner_address"`
	ToAddress    string `json:"to_address"`
	Amount       int64  `json:"amount"`
}

// TriggerSmartContract 智能合约触发
type TriggerSmartContract struct {
	OwnerAddress    string `json:"owner_address"`
	ContractAddress string `json:"contract_address"`
	Data            string `json:"data"`
	CallValue       int64  `json:"call_value"`
}

// TransactionResult 交易结果
type TransactionResult struct {
	ContractRet string `json:"contractRet"`
}

// TransferEvent 转账事件
type TransferEvent struct {
	Source          string  `json:"source"`
	Destination     string  `json:"destination"`
	Amount          float64 `json:"amount"`
	Fee             float64 `json:"fee"`
	TxHash          string  `json:"tx_hash"`
	BlockHeight     int64   `json:"block_height"`
	Timestamp       int64   `json:"timestamp"`
	Confirmations   int     `json:"confirmations"`
	TokenType       string  `json:"token_type"` // TRX, TRC10, TRC20, USDT
	ContractAddress string  `json:"contract_address,omitempty"`
	AssetName       string  `json:"asset_name,omitempty"`
	IsUSDT          bool    `json:"is_usdt,omitempty"` // 是否为USDT转账
	USDValue        float64 `json:"usd_value,omitempty"` // USD价值（如果是USDT）
}

// SystemStats 系统统计信息
type SystemStats struct {
	TotalBlocksProcessed int64         `json:"total_blocks_processed"`
	TotalTransfersFound  int64         `json:"total_transfers_found"`
	LastProcessedBlock   int64         `json:"last_processed_block"`
	LastProcessedTime    time.Time     `json:"last_processed_time"`
	Uptime               time.Duration `json:"uptime"`
	ErrorCount           int64         `json:"error_count"`
	SuccessCount         int64         `json:"success_count"`
}

// WatchAddress 监控地址信息
type WatchAddress struct {
	Address       string    `json:"address"`
	AddedAt       time.Time `json:"added_at"`
	LastSeen      time.Time `json:"last_seen,omitempty"`
	TransferCount int64     `json:"transfer_count"`
}

// APIResponse TronGrid API响应结构
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Error   string      `json:"error,omitempty"`
}

// BlockResponse 区块响应
type BlockResponse struct {
	BlockHeader *BlockHeader   `json:"block_header"`
	Trans       []*Transaction `json:"transactions"`
}

// TransactionInfo 交易信息
type TransactionInfo struct {
	ID              string              `json:"id"`
	Fee             int64               `json:"fee"`
	BlockNumber     int64               `json:"blockNumber"`
	BlockTimeStamp  int64               `json:"blockTimeStamp"`
	ContractResult  []string            `json:"contractResult"`
	ContractAddress string              `json:"contract_address,omitempty"`
	Receipt         *TransactionReceipt `json:"receipt"`
	Log             []*TransactionLog   `json:"log"`
}

// TransactionReceipt 交易收据
type TransactionReceipt struct {
	EnergyUsage       int64  `json:"energy_usage"`
	EnergyFee         int64  `json:"energy_fee"`
	OriginEnergyUsage int64  `json:"origin_energy_usage"`
	EnergyUsageTotal  int64  `json:"energy_usage_total"`
	NetUsage          int64  `json:"net_usage"`
	NetFee            int64  `json:"net_fee"`
	Result            string `json:"result"`
}

// TransactionLog 交易日志
type TransactionLog struct {
	Address string   `json:"address"`
	Topics  []string `json:"topics"`
	Data    string   `json:"data"`
}
