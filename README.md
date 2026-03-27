# openclaw-mcp

Openclaw MCP - 一个为 Openclaw 设计的 VPN 节点管理和自动选择控制平台

## 功能特性

- ✅ 支持第三方订阅地址导入**定时自动更新**
- ✅ **支持全部主流协议：** Shadowsocks, VMess, VLESS, Trojan, Hysteria, Hysteria2, Tuic
- ✅ **节点分组管理** - 支持自定义分组，每个分组自动选择最佳节点
- ✅ **智能链路选择** - 基于平均延迟选择最稳定节点，避免单次网络波动
- ✅ **基于规则的分流支持** - 域名后缀/关键词/IP-CIDR匹配
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

# 选中分组
selected_group: "auto"

# 订阅列表
subscriptions:
  - name: "My Subscription"
    url: "https://example.com/subscribe"
    enabled: true

# 自定义节点分组
groups:
  - name: "Foreign"
    is_auto: true

# 分流规则
rules: []
```

## 新增功能说明

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

## Bug修复和改进

本项目在原有基础上修复了以下问题：
1. 实现了配置加载和保存功能，可以持久化保存配置修改
2. 补全了缺失的API：删除订阅、更新订阅、更新配置
3. 修复了MCP状态API节点计数错误
4. 实现了IP-CIDR规则匹配，分流规则完整可用
5. 修复了自动选择间隔硬编码问题，现在读取配置中的间隔
6. 修复了一个订阅更新失败就整体返回的问题，现在会继续更新其他订阅
7. 复用HTTP客户端，改善性能

## 许可证

本项目采用 [MIT License](LICENSE) 开源，详细请查看 LICENSE 文件。

Copyright (c) 2026 wubaiwan <a523959108@hotmail.com>
