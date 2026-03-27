# openclaw-mcp

Openclaw MCP - 一个为 Openclaw 设计的 VPN 节点管理和自动选择控制平台

## 功能特性

- ✅ 支持第三方订阅地址导入**定时自动更新**
- ✅ **支持全部主流协议：** Shadowsocks, VMess, VLESS, Trojan, Hysteria, Hysteria2, Tuic
- ✅ **节点分组管理** - 支持自定义分组，每个分组自动选择最佳节点
- ✅ **智能链路选择** - 支持多种选择策略：延迟优先/轮询/最小连接数/故障转移/随机
- ✅ **DNS 污染检测与规避** - 自动检测被污染的节点域名，过滤不可用节点
- ✅ **兼容 Clash 分流规则** - 支持 DOMAIN/SUFFIX/KEYWORD/IP-CIDR/GEOIP 等多种格式，可直接使用 Clash 规则集
- ✅ **流量和连接统计** - 实时统计总体和各节点的连接数与流量使用
- ✅ **配置热重载** - 通过 SIGHUP 信号无需重启即可重新加载配置
- ✅ **HTTP基础认证** - 保护控制面板安全
- ✅ **原生HTTPS支持** - 直接提供HTTPS服务
- ✅ 独立的响应式 Web 控制面板
- ✅ 模块化设计，可以作为 MCP 挂载到 Openclaw
- ✅ Openclaw 通过 MCP API 获取当前最优节点

## 配置说明

### 完整配置示例

```yaml
listen_addr: "0.0.0.0:9090"

# 基础认证
enable_auth: false
username: "admin"
password: "admin"

# HTTPS 配置
enable_https: false
cert_file: ""
key_file: ""

# 自动选择配置
enable_auto_select: true
auto_select_interval: 30  # 自动选择间隔，单位分钟
subscription_update_interval: 24  # 订阅自动更新间隔，单位小时

# 节点选择策略
# 可选值: latency (延迟优先, 默认) | round-robin (轮询) | least-connections (最小连接数) | failover (故障转移) | random (随机)
selection_strategy: "latency"

# 选中分组
selected_group: "auto"

# DNS 污染检测配置
dns:
  enable: false                  # 是否启用DNS污染检测
  check_pollution: true          # 是否检测结果是否被污染，默认true
  trusted_dns:                   # 可信DNS服务器列表
    - "8.8.8.8:53"
    - "1.1.1.1:53"
  cache_ttl: 30                  # DNS缓存时间，单位分钟
  timeout: 5                     # DNS查询超时，单位秒

# 订阅列表
subscriptions:
  - name: "My Subscription"
    url: "https://example.com/subscribe"
    enabled: true

# 自定义节点分组
groups:
  - name: "Foreign"
    is_auto: true

# 传统格式规则
rules: []

# Clash格式规则 (直接使用 type,pattern,target 格式)
rule_strings:
  - "DOMAIN-SUFFIX,google.com,proxy"
  - "DOMAIN-KEYWORD,amazon,proxy"
  - "IP-CIDR,192.168.0.0/16,DIRECT"
  - "GEOIP,CN,DIRECT"
```

## 功能说明

### 节点分组
支持创建多个分组，每个分组独立维护和自动选择最佳节点，适合多场景使用：
- 可以为不同地区创建分组（例如 中国大陆/国外）
- 自动分组将自动收集对应Tag的节点
- 每个分组支持一键测试并选择最佳节点

### 平均延迟计算
保存最近5次测速结果并计算平均延迟，选择更稳定的节点：
- 避免临时网络波动导致错误选择
- 前端同时显示单次延迟和平均延迟
- 节点排序优先使用平均延迟

### 多种节点选择策略

| 策略 | 说明 |
|------|------|
| `latency` | 默认，选择延迟最低的节点 |
| `round-robin` | 轮询，依次选择每个节点，均匀负载 |
| `least-connections` | 最小连接数，总是选择当前活动连接最少的节点，适合长期稳定运行 |
| `failover` | 故障转移，始终选择第一个可用节点，只在当前节点故障时切换下一个 |
| `random` | 随机，从可用节点中随机选择一个 |

### DNS 污染检测

当你的网络对某些代理节点域名存在污染时，可以启用该功能：
- 使用可信DNS重新解析节点域名
- 对比系统DNS结果，如果不一致说明被污染
- 自动过滤被污染的节点，避免连接不可用节点

### 兼容 Clash 规则格式

支持直接使用 Clash 格式的规则，可以方便地从 Clash 配置迁移：
- 支持多种规则类型：`DOMAIN`/`DOMAIN-SUFFIX`/`DOMAIN-KEYWORD`/`IP-CIDR`/`IP-CIDR6` 等
- 自动识别下划线命名格式 (`domain_suffix`)，兼容多种规则集
- 可以直接在配置中使用 `rule_strings` 数组编写规则

### 连接与流量统计

提供统计API，可以查看当前运行状态：
- 总体统计：总连接数、当前活跃连接数、总上传下载流量、连接速率
- 每个节点统计：各节点的连接数、流量使用情况、最后使用时间
- 支持一键重置统计，方便监控和分析

### 配置热重载

修改配置后无需重启程序，发送 `SIGHUP` 信号即可重新加载：
```bash
kill -HUP <pid>
```

### 自动更新订阅
支持配置自动更新订阅间隔，后台自动更新：
- 默认24小时更新一次
- 即使某个订阅更新失败也会继续更新其他订阅
- 更新后自动获取最新节点信息

### 安全增强
- 可选开启HTTP基础认证，保护控制面板
- 支持原生HTTPS，直接使用证书提供服务

## 快速开始

```
.
├── api/         # MCP API 接口和Web API定义
├── cmd/         # 主程序入口
├── internal/    # 内部模块
│   ├── config/      # 配置读写和默认配置
│   ├── subscription/ # 订阅解析和管理
│   ├── route/       # 分流规则匹配
│   ├── lighthouse/  # 节点延迟测速
│   └── selection/   # 智能节点选择
├── web/         # Web 控制面板静态资源
└── README.md
```

## 构建

```bash
go build -o openclaw-mcp ./cmd
./openclaw-mcp
```

首次运行会自动生成默认配置文件 `config.yaml`，修改后重启即可。

## API 端点

### Web控制面板
- `GET /*` - Web控制面板

### MCP API（供Openclaw调用）
- `GET /mcp/current` - 获取当前选中节点
- `GET /mcp/status` - 获取当前状态（当前节点+可用节点数）

### 控制面板API
- `GET /api/v1/subscriptions` - 获取订阅列表
- `POST /api/v1/subscriptions` - 添加订阅
- `DELETE /api/v1/subscriptions/{id}` - 删除订阅
- `POST /api/v1/subscriptions/{id}/update` - 更新订阅
- `GET /api/v1/nodes` - 获取所有节点
- `POST /api/v1/nodes/test` - 测试所有节点延迟
- `GET /api/v1/current` - 获取当前选中节点
- `POST /api/v1/select/best` - 选择最佳节点
- `GET /api/v1/groups` - 获取所有分组
- `POST /api/v1/groups` - 创建分组
- `POST /api/v1/groups/{name}/select` - 为分组选择最佳节点
- `GET /api/v1/config` - 获取当前配置
- `PUT /api/v1/config` - 更新配置
- `GET /api/v1/stats` - 获取总体连接和流量统计
- `GET /api/v1/stats/nodes` - 获取每个节点的连接和流量统计
- `POST /api/v1/stats/reset` - 重置所有统计

## Bug修复和改进

本项目在原有基础上修复和新增了以下功能：
1. 实现了配置加载和保存功能，可以持久化保存配置修改
2. 补全了缺失的API：删除订阅、更新订阅、更新配置
3. 修复了MCP状态API节点计数错误
4. 完善分流规则匹配，支持Clash格式规则和更多规则类型
5. 修复了自动选择间隔硬编码问题，现在读取配置中的间隔
6. 修复了一个订阅更新失败就整体返回的问题，现在会继续更新其他订阅
7. 复用HTTP客户端，改善性能
8. 添加DNS污染检测与自动过滤功能
9. 添加五种不同的节点选择策略，满足不同场景需求
10. 添加连接和流量统计API
11. 添加配置热重载支持

## 许可证

本项目采用 [MIT License](LICENSE) 开源，详细请查看 LICENSE 文件。

Copyright (c) 2026 wubaiwan <a523959108@hotmail.com>
