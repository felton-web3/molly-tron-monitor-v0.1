# Tron区块链监控系统

一个高性能的Tron区块链监控系统，专门用于监控Tron转账活动。系统采用区块监控架构，通过HTTPS请求定期获取区块数据，确保数据完整性和可靠性。系统支持每秒一次的查询频率，充分利用TronGrid API的查询权限。

## 核心特性

- **高性能监控**: 每秒一次查询频率，实时监控Tron网络活动
- **多地址支持**: 支持监控多个Tron地址的转账活动
- **多代币支持**: 支持TRX、TRC10、TRC20代币转账监控
- **USDT监控**: 专门监控USDT转账交易，支持金额范围过滤
- **数据缓存**: 使用Redis进行数据缓存和队列管理
- **多线程处理**: 采用多线程架构，提高处理效率
- **RESTful API**: 提供完整的HTTP API接口
- **Docker支持**: 完整的容器化部署方案
- **健康检查**: 完善的健康检查和监控机制

## 系统架构

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   TronGrid API  │    │   Block Monitor │    │ Block Processor │
│                 │◄──►│                 │◄──►│                 │
│  HTTPS请求      │    │   每秒查询      │    │   多线程处理    │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │                        │
                                ▼                        ▼
                       ┌─────────────────┐    ┌─────────────────┐
                       │   Redis Queue   │    │  Transfer Events│
                       │                 │    │                 │
                       │   区块数据队列   │    │   转账事件存储   │
                       └─────────────────┘    └─────────────────┘
                                │                        │
                                ▼                        ▼
                       ┌─────────────────┐    ┌─────────────────┐
                       │  HTTP Server    │    │  Watch Addresses│
                       │                 │    │                 │
                       │   RESTful API   │    │   监控地址管理   │
                       └─────────────────┘    └─────────────────┘
```

## 技术栈

- **语言**: Go 1.21+
- **数据存储**: Redis
- **区块链连接**: TronGrid API
- **Web框架**: Gorilla Mux
- **日志**: Logrus
- **配置管理**: Viper
- **容器化**: Docker & Docker Compose

## 快速开始

### 前置要求

- Go 1.21+
- Redis 6.0+
- Docker & Docker Compose (可选)

### 安装

#### 方法1: 本地安装

1. 克隆仓库
```bash
git clone <repository-url>
cd tron-monitor
```

2. 安装依赖
```bash
make deps
```

3. 构建应用程序
```bash
make build
```

4. 启动Redis
```bash
redis-server
```

5. 运行应用程序
```bash
make run
```

#### 方法2: Docker部署

1. 克隆仓库
```bash
git clone <repository-url>
cd tron-monitor
```

2. 启动服务
```bash
make compose-up
```

### 配置

编辑 `config.yaml` 文件来配置系统：

```yaml
# TronGrid API配置
trongrid:
  base_url: "https://api.trongrid.io"
  api_key: ""  # 可选，如果需要更高的API限制
  timeout: "30s"
  retry_max: 3
  retry_delay: "1s"

# Redis配置
redis:
  addr: "localhost:6379"
  password: ""
  db: 0
  pool_size: 10

# 监控配置
monitor:
  block_interval: "1s"  # 区块查询间隔，每秒一次
  worker_count: 4       # 工作线程数
  queue_size: 1000      # 队列大小
  batch_size: 10        # 批处理大小
  max_block_height: 0   # 最大区块高度，0表示不限制
  start_block_height: 0 # 起始区块高度，0表示从最新区块开始

# 监控地址列表
watch_addresses:
  - "TJRabPrwbZy45sbavfcjinPJC18kjpRTv8"  # 示例地址1
  - "TUpMhErZL2fhh4sVNULAbNKLokS4GjC1F4"  # 示例地址2

# USDT监控配置
usdt:
  contract_address: "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"  # USDT合约地址
  enable_monitoring: true  # 启用USDT监控
  min_amount: 10  # 最小监控金额（USDT）
  max_amount: 1000000  # 最大监控金额（USDT）
  decimals: 6  # USDT精度

# 日志配置
log:
  level: "info"  # debug, info, warn, error
  file: ""       # 日志文件路径，空表示输出到控制台

# HTTP服务配置
server:
  host: "0.0.0.0"
  port: "8080"
```

## API接口

### 健康检查

```bash
GET /health
```

响应:
```json
{
  "status": "healthy",
  "time": "2024-01-01T12:00:00Z"
}
```

### 系统状态

```bash
GET /status
```

响应:
```json
{
  "monitor": {
    "running": true,
    "last_processed_block": 12345678,
    "processed_blocks": 100,
    "errors": 0,
    "queue_size": 5,
    "block_interval": "1s"
  },
  "processor": {
    "running": true,
    "processed_blocks": 100,
    "transfers_found": 50,
    "errors": 0,
    "worker_count": 4
  },
  "http": {
    "total_blocks_processed": 100,
    "total_transfers_found": 50,
    "last_processed_block": 12345678,
    "last_processed_time": "2024-01-01T12:00:00Z",
    "uptime": "1h30m",
    "error_count": 0,
    "success_count": 100
  },
  "uptime": "1h30m"
}
```

### 监控地址管理

#### 获取监控地址列表

```bash
GET /addresses
```

响应:
```json
[
  "TJRabPrwbZy45sbavfcjinPJC18kjpRTv8",
  "TUpMhErZL2fhh4sVNULAbNKLokS4GjC1F4"
]
```

#### 添加监控地址

```bash
POST /addresses
Content-Type: application/json

{
  "address": "TJRabPrwbZy45sbavfcjinPJC18kjpRTv8"
}
```

#### 移除监控地址

```bash
DELETE /addresses
Content-Type: application/json

{
  "address": "TJRabPrwbZy45sbavfcjinPJC18kjpRTv8"
}
```

### 转账记录

```bash
GET /transfers?limit=100
```

响应:
```json
[
  {
    "source": "TJRabPrwbZy45sbavfcjinPJC18kjpRTv8",
    "destination": "TUpMhErZL2fhh4sVNULAbNKLokS4GjC1F4",
    "amount": 100.5,
    "fee": 0.1,
    "tx_hash": "abc123...",
    "block_height": 12345678,
    "timestamp": 1704067200000,
    "confirmations": 1,
    "token_type": "TRX",
    "contract_address": "",
    "asset_name": ""
  }
]
```

### USDT转账记录

```bash
GET /usdt-transfers?limit=100
```

响应:
```json
[
  {
    "source": "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",
    "destination": "TYPjL2iwqvcev7jDBe4M85Jq2FYpvkMvAH",
    "amount": 1000.0,
    "fee": 0,
    "tx_hash": "abc123...",
    "block_height": 12345678,
    "timestamp": 1704067200000,
    "confirmations": 0,
    "token_type": "USDT",
    "contract_address": "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",
    "is_usdt": true,
    "usd_value": 1000.0
  }
]
```

### USDT统计信息

```bash
GET /usdt-stats
```

响应:
```json
{
  "total_transfers": 150,
  "total_amount": 50000.0,
  "avg_amount": 333.33,
  "min_amount": 10.0,
  "max_amount": 5000.0,
  "unique_addresses": 45,
  "recent_transfers": [...]
}
```

## 开发

### 项目结构

```
tron-monitor/
├── config/          # 配置管理
├── http/           # HTTP客户端
├── models/         # 数据模型
├── processor/      # 区块处理
├── redis/          # Redis客户端
├── config.yaml     # 配置文件
├── Dockerfile      # Docker配置
├── docker-compose.yml # Docker Compose配置
├── Makefile        # 构建脚本
├── go.mod          # Go模块文件
├── go.sum          # Go依赖校验
├── main.go         # 主程序
└── README.md       # 项目文档
```

### 开发命令

```bash
# 下载依赖
make deps

# 开发模式运行
make dev

# 运行测试
make test

# 代码格式化
make fmt

# 代码检查
make lint

# 构建
make build

# 清理
make clean
```

### Docker开发

```bash
# 构建Docker镜像
make docker-build

# 运行Docker容器
make docker-run

# 停止Docker容器
make docker-stop

# 清理Docker镜像
make docker-clean
```

### Docker Compose开发

```bash
# 启动服务
make compose-up

# 查看日志
make compose-logs

# 停止服务
make compose-down

# 重启服务
make compose-restart
```

## 部署

### 生产环境部署

1. 使用Docker Compose部署（推荐）

```bash
# 启动生产环境
docker-compose -f docker-compose.yml up -d

# 查看日志
docker-compose logs -f

# 停止服务
docker-compose down
```

2. 本地部署

```bash
# 构建应用程序
make build

# 安装到系统
make install

# 运行服务
tron-monitor
```

### 监控和日志

系统提供以下监控端点：

- `/health` - 健康检查
- `/status` - 系统状态
- `/addresses` - 监控地址管理
- `/transfers` - 转账记录查询
- `/usdt-transfers` - USDT转账记录查询
- `/usdt-stats` - USDT统计信息

日志级别可通过配置文件调整：
- `debug` - 详细调试信息
- `info` - 一般信息
- `warn` - 警告信息
- `error` - 错误信息

## 性能优化

### 系统调优

1. **Redis优化**
   - 增加连接池大小
   - 启用持久化
   - 配置内存限制

2. **网络优化**
   - 使用TronGrid API密钥
   - 调整请求超时时间
   - 配置重试策略

3. **处理优化**
   - 调整工作线程数
   - 优化队列大小
   - 配置批处理大小

### 监控指标

系统提供以下关键指标：

- 区块处理速度
- 转账事件发现率
- API请求成功率
- 队列积压情况
- 错误率统计

## 故障排除

### 常见问题

1. **Redis连接失败**
   - 检查Redis服务状态
   - 验证连接配置
   - 检查网络连通性

2. **TronGrid API请求失败**
   - 检查网络连接
   - 验证API密钥
   - 检查请求频率限制

3. **区块处理延迟**
   - 检查队列积压情况
   - 调整工作线程数
   - 优化处理逻辑

### 日志分析

系统日志包含以下关键信息：

- 区块监控状态
- 转账事件处理
- API请求统计
- 错误详情

## 贡献

欢迎提交Issue和Pull Request来改进项目。

### 开发指南

1. Fork项目
2. 创建功能分支
3. 提交更改
4. 创建Pull Request

### 代码规范

- 遵循Go语言规范
- 添加适当的注释
- 编写单元测试
- 保持代码简洁

## 许可证

本项目采用MIT许可证。详见LICENSE文件。

## 联系方式

如有问题或建议，请通过以下方式联系：

- 提交Issue
- 发送邮件
- 参与讨论

---

**注意**: 这是一个监控系统，请确保遵守相关法律法规和API使用条款。
