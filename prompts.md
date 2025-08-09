# Tron区块链监控系统 - 提示信息

## 项目概述

这是一个用于监控Tron区块链转账的系统，支持TRX、TRC10和TRC20代币转账的实时监控，特别针对USDT转账进行了优化。

## 核心功能

### 1. 转账监控
- **TRX转账**：监控原生TRX转账
- **TRC10转账**：监控TRC10代币转账
- **TRC20转账**：监控TRC20代币转账，包括USDT
- **USDT专项监控**：专门的USDT转账监控功能

### 2. 地址管理
- 支持添加/删除监控地址
- 实时统计监控地址的转账活动
- 地址余额和交易历史追踪

### 3. API接口
- RESTful API设计
- 支持转账记录查询
- 提供统计信息接口
- USDT专项API

## 技术架构

### 1. 核心模块
```
├── main.go              # 主程序入口
├── processor/           # 区块处理器
│   └── block_processor.go
├── config/             # 配置管理
│   └── config.go
├── models/             # 数据模型
│   └── types.go
├── http/               # HTTP服务
│   └── client.go
└── redis/              # Redis存储
    └── redis.go
```

### 2. 关键功能实现

#### 区块处理
- 多工作线程并行处理区块
- 智能合约调用解析
- 转账事件提取和验证

#### 地址转换
- 十六进制地址转Base58格式
- TRX地址格式标准化
- 地址校验和验证

#### USDT专项处理
- USDT合约地址识别
- 6位小数精度处理
- 金额范围过滤
- 专门的存储和API

## 配置说明

### 1. 监控配置
```yaml
monitor:
  worker_count: 4        # 工作线程数
  block_interval: 3000   # 区块间隔(毫秒)
  addresses:             # 监控地址列表
    - "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"
```

### 2. USDT配置
```yaml
usdt:
  contract_address: "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"
  enable_monitoring: true
  min_amount: 10
  max_amount: 1000000
  decimals: 6
```

### 3. Redis配置
```yaml
redis:
  host: "localhost"
  port: 6379
  db: 0
```

## API接口

### 1. 系统状态
```
GET /health          # 健康检查
GET /status          # 系统状态
```

### 2. 地址管理
```
GET    /addresses    # 获取监控地址
POST   /addresses    # 添加监控地址
DELETE /addresses    # 删除监控地址
```

### 3. 转账记录
```
GET /transfers?limit=100          # 获取转账记录
GET /usdt-transfers?limit=100     # 获取USDT转账记录
GET /usdt-stats                   # 获取USDT统计信息
```

## 部署方式

### 1. 本地运行
```bash
# 启动Redis
redis-server

# 启动应用
go run main.go

# 或编译后运行
go build -o tron-monitor .
./tron-monitor
```

### 2. Docker部署
```bash
# 使用docker-compose
docker-compose up -d

# 或单独构建
docker build -t tron-monitor .
docker run -d tron-monitor
```

### 3. 脚本管理
```bash
# 启动系统
./scripts/start.sh

# 停止系统
./scripts/stop.sh

# 演示功能
./scripts/demo.sh

# 测试USDT监控
./scripts/test_usdt_monitor.sh
```

## 开发提示

### 1. 代码结构
- 遵循Go语言最佳实践
- 模块化设计，职责分离
- 错误处理和日志记录完善

### 2. 性能优化
- 多线程并行处理
- Redis缓存优化
- 内存使用优化

### 3. 扩展性
- 支持添加新的代币类型
- 可配置的监控参数
- 插件化的处理器设计

## 故障排除

### 1. 常见问题
- **连接失败**：检查Redis服务是否启动
- **地址转换错误**：验证地址格式是否正确
- **数据解析失败**：检查合约调用数据格式

### 2. 调试方法
- 查看日志文件：`logs/tron-monitor.log`
- 使用调试脚本：`./scripts/debug_usdt.sh`
- 检查API状态：`curl http://localhost:8080/health`

### 3. 性能监控
- 监控内存使用情况
- 检查Redis连接状态
- 观察区块处理速度

## 安全考虑

### 1. 网络安全
- API接口访问控制
- 敏感信息保护
- 请求频率限制

### 2. 数据安全
- Redis访问权限控制
- 数据备份策略
- 日志安全存储

## 维护建议

### 1. 定期维护
- 清理过期日志
- 监控系统资源使用
- 更新依赖包

### 2. 监控指标
- 区块处理速度
- 转账事件数量
- 系统响应时间
- 错误率统计

### 3. 备份策略
- 配置文件备份
- 数据库备份
- 日志文件归档

---

*此文档提供了项目的完整概述和操作指南，帮助开发者快速理解和使用系统。*
