# Homeward

自托管的 Mesh VPN，用你自己的基础设施替代 Tailscale。

## 为什么做这个

中国家庭宽带通常处于运营商级 NAT（CG-NAT）之后——没有公网 IPv4 地址，因此无法从外部访问家里的机器。Tailscale 解决了这个问题，但流量会经过他们的服务器中转。

Homeward 做的是同样的事，但跑在你自己控制的基础设施上。

```
MacBook（外网）
    │
    │  WireGuard 隧道
    │
   VPS（你的中继服务器，有公网 IP）
    │
    │  穿透 CG-NAT
    │
家庭机器（无公网 IP）
```

## 目标

- 从任何地方连接到 CG-NAT 后的家庭机器
- 所有流量通过你自己的 VPS 中继，而非第三方服务
- 学习 Mesh VPN、NAT 穿透和 WireGuard 的实际工作原理

## 工作原理

### 问题：CG-NAT

中国家庭宽带的网络路径通常是这样的：

```
traceroute 8.8.8.8

1  192.168.0.1    ← 家用路由器
2  192.168.1.1    ← 运营商光猫
3  172.17.0.1     ← 运营商 CG-NAT 设备   ← 阻断点
4  183.x.x.x      ← 公网
```

到达公网前经过三层私有地址。外部任何设备都无法主动向内发起连接。

### 解决方案：中继 + 打洞

```
1. 两端设备均主动向控制平面发起出站连接（出站流量从不被拦截）
2. 控制平面交换双方的公钥和地址
3. 尝试通过 UDP 打洞直连
4. 若失败（CG-NAT 下很常见），通过 DERP 服务器中继流量
5. 无论走哪条路径，所有流量均由 WireGuard 端到端加密
```

## 架构

两个二进制文件，两种角色：

```
homeward-server   （运行在 VPS 上）
├── 控制平面 — 节点注册、密钥分发、拓扑同步
└── DERP 中继 — 直连失败时的加密包转发

homeward-client   （运行在 MacBook 和家庭机器上，同一个二进制文件）
├── 向控制平面注册
├── 同步对等节点列表
├── 管理本地 WireGuard 配置
├── TUN 虚拟网络接口
└── NAT 穿透（STUN + UDP 打洞 → 回退到 DERP）
```

MacBook 和家庭机器运行相同的客户端代码。节点之间没有固定的"客户端"或"服务端"之分——任何节点都可以主动发起或接受连接。

## 组件

| 组件 | 用途 |
|------|------|
| 控制平面 | 密钥交换与拓扑分发 |
| DERP 中继 | 直连不可用时的回退中继 |
| WireGuard | 节点间的加密隧道 |
| TUN 接口 | 虚拟网络设备，为每个节点分配固定 IP |
| NAT 穿透 | STUN 探测 + UDP 打洞实现直连 |

连接优先级：

```
1. IPv6 直连     （延迟最低，家庭 ISP 提供 IPv6 时可用）
2. UDP 打洞      （IPv4 直连，单层 NAT 下可用）
3. DERP 中继     （始终可用，包括 CG-NAT 场景）
```

## 项目结构

```
homeward/
├── cmd/
│   ├── server/          # VPS 二进制：控制平面 + DERP
│   └── client/          # 节点二进制：MacBook 和家庭机器
└── internal/
    ├── control/         # 控制平面逻辑
    ├── derp/            # 中继服务器
    ├── wg/              # WireGuard 管理
    ├── tun/             # 虚拟网络接口
    └── nat/             # NAT 穿透
```

## 测试环境

完整测试需要三台机器：

| 机器 | 角色 | 网络 |
|------|------|------|
| VPS | 运行 `homeward-server` | 公网 IP，任何地方可达 |
| 家庭机器 | CG-NAT 后的节点 | 无公网 IPv4（用你真实的家庭机器即可） |
| MacBook | 外网节点 | 任意网络（4G、公司、咖啡馆） |

## 技术栈

- **Go** — Tailscale 同款语言
- **wireguard-go** — 用户态 WireGuard 实现
- **pion/ice** — 用于 NAT 穿透的 ICE/STUN/TURN

## 参考资料

- [WireGuard 白皮书](https://www.wireguard.com/papers/wireguard.pdf)
- [Tailscale NAT 穿透详解](https://tailscale.com/blog/how-nat-traversal-works)
- [Headscale](https://github.com/juanfont/headscale) — 开源 Tailscale 控制平面，值得一读
- [NetBird](https://github.com/netbirdio/netbird) — 相似架构，代码整洁
