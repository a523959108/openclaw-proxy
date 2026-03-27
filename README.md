# openclaw-mcp

Openclaw MCP - 一个为 Openclaw 设计的基于 Openclash 的 VPN 模块控制平台

## 功能特性

- 支持第三方订阅地址导入和自动更新
- **支持全部主流协议：** Shadowsocks, VMess, VLESS, Trojan, Hysteria, Hysteria2, Tuic
- 智能链路选择，根据延迟自动选择最佳节点
- 基于规则的分流支持（域名/IP匹配）
- 独立的 Web 控制面板，类似 Clash
- 模块化设计，可以作为 MCP 挂载到 Openclaw
- Openclaw 通过 MCP API 获取当前最优节点

## 目录结构

```
.
├── api/         # MCP API 接口定义
├── cmd/         # 主程序入口
├── internal/    # 内部模块
│   ├── config/      # 配置管理
│   ├── subscription/ # 订阅管理
│   ├── route/       # 分流规则
│   ├── lighthouse/  # 节点测速
│   └── selection/   # 智能选择
├── web/         # Web 控制面板静态资源
└── README.md
```

## 许可证

MIT
