# Openclaw MCP 集成说明

本文档说明 Openclaw 如何与 Openclaw-proxy MCP 服务集成，获取自动选择的最佳节点。

## 什么是 MCP

MCP (Master Control Program) 是 Openclaw-proxy 提供的节点管理和自动选择服务，它独立运行，通过 HTTP API 为 Openclaw 提供当前最佳节点信息。

## 为什么使用 MCP

- **独立运行**：MCP 独立管理订阅更新、延迟测速、自动选择，不影响 Openclaw 主进程
- **自动选择**：MCP 会定期自动测试所有节点并选择最佳节点
- **灵活策略**：支持多种选择策略（延迟优先/轮询/最小连接数/故障转移/随机）
- **DNS 污染检测**：自动检测并过滤 DNS 被污染的节点
- **Web 管理界面**：提供完整的 Web 控制面板管理节点和订阅

## 集成方式

Openclaw 需要配置为从 MCP 获取当前节点。MCP 提供两个 API 端点供 Openclaw 调用。

### 1. 获取当前选中节点

**API 端点：**
```
GET http://<mcp-host>:<port>/mcp/current
```

**响应示例：**
```json
{
  "name": "Node-Name",
  "server": "example.com",
  "port": 8388,
  "password": "password",
  "method": "chacha20-ietf-poly1305",
  "protocol": "shadowsocks",
  "latency": 120,
  "average_latency": 115.5,
  "available": true,
  "group": "HK",
  ...
}
```

Openclaw 调用此 API 获取当前 MCP 自动选择的最佳节点，然后使用该节点建立连接。

### 2. 获取状态信息

**API 端点：**
```
GET http://<mcp-host>:<port>/mcp/status
```

**响应示例：**
```json
{
  "current_node": { /* 当前节点信息 */ },
  "available_nodes": 25,
  "total_nodes": 30,
  "strategy": "latency"
}
```

## 何时调用 MCP API

### 每次建立新连接时调用

这是推荐的方式：
1. Openclaw 需要建立出站连接时
2. 调用 `GET /mcp/current` 获取当前最佳节点
3. 使用返回的节点信息连接

优点：
- 总是使用最新选择的节点
- 如果 MCP 已经重新选择了更好的节点，立刻就能生效

### 定期轮询（可选）

如果不想每次连接都请求 API，可以：
1. 启动时调用一次获取节点
2. 定期（比如每 5 分钟）调用一次刷新
3. 一直使用缓存的节点直到下次刷新

缺点：
- 不能立刻享受到节点重新选择的好处

## Openclaw 配置示例

在 Openclaw 配置文件中添加 MCP 配置：

```yaml
# Openclaw config example
mcp:
  enable: true
  endpoint: "http://127.0.0.1:9090/mcp/current"
  timeout: 5  # HTTP 请求超时，单位秒
```

## 工作流程图

```
┌─────────────┐
│ Openclaw    │
│  需要出站   │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ 调用 /mcp/  │
│ current     │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ MCP 返回     │
│ 当前最佳节点 │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ Openclaw    │
│ 使用该节点  │
│ 建立连接     │
└─────────────┘
```

## MCP 启动

### 直接运行

```bash
# 下载 Linux 版本
wget https://github.com/a523959108/openclaw-proxy/releases/download/v1.0.0/openclaw-proxy-linux-amd64
chmod +x openclaw-proxy-linux-amd64

# 首次运行生成默认配置
./openclaw-proxy-linux-amd64

# 编辑配置文件 config.yaml，添加你的订阅
vim config.yaml

# 后台运行
nohup ./openclaw-proxy-linux-amd64 &
```

### 使用 Docker

```bash
docker build -t openclaw-proxy .
docker run -d -p 9090:9090 -v $(pwd)/config.yaml:/config.yaml openclaw-proxy
```

## MCP 配置示例

关键配置：

```yaml
# config.yaml
listen_addr: "0.0.0.0:9090"

# 自动选择配置
enable_auto_select: true
auto_select_interval: 30  # 每 30 分钟自动重新选择一次

# 选择策略
# latency - 延迟最低 (默认)
# round-robin - 轮询
# least-connections - 最小连接数
# failover - 故障转移
# random - 随机
selection_strategy: "latency"

# 订阅列表
subscriptions:
  - name: "My Subscription"
    url: "https://your-subscription-url.com"
    enabled: true

# DNS 污染检测 (如果节点域名容易被污染请开启)
dns:
  enable: true
  check_pollution: true
  trusted_dns:
    - "8.8.8.8:53"
    - "1.1.1.1:53"
```

## 故障排查

### MCP 返回空节点
- 检查订阅是否可访问
- 检查是否有可用节点
- 检查 DNS 污染检测是否过滤了太多节点，可尝试关闭 `check_pollution`

### 连接 MCP 超时
- 检查 MCP 是否正常运行
- 检查防火墙是否允许 Openclaw 访问 MCP 端口
- 检查 `listen_addr` 是否监听了正确的接口（不要用 `127.0.0.1` 如果 Openclaw 在其他容器/主机）

## 常用命令

```bash
# 查看日志
ps aux | grep openclaw-proxy

# 热重载配置（修改配置后无需重启）
kill -HUP <pid>

# 停止服务
kill <pid>
```

## 更多文档

- 完整配置和功能说明请查看 [README.md](./README.md)
- API 完整列表请查看 [README.md](./README.md#api-端点)