# Homeward

A self-hosted mesh VPN, built to replace Tailscale with your own infrastructure.

## Why

Chinese home broadband typically sits behind carrier-grade NAT (CG-NAT) — there is no public IPv4 address, so you cannot reach your home machine from outside. Tailscale solves this, but routes your traffic through their servers.

Homeward does the same thing on infrastructure you control.

```
MacBook (outside)
    │
    │  WireGuard tunnel
    │
   VPS (your relay, public IP)
    │
    │  punches through CG-NAT
    │
Home machine (no public IP)
```

## Goals

- Connect to a home machine behind CG-NAT from anywhere
- All traffic relayed through your own VPS, not a third-party service
- Learn how mesh VPNs, NAT traversal, and WireGuard actually work

## How It Works

### The problem: CG-NAT

Home broadband in China typically looks like this:

```
traceroute 8.8.8.8

1  192.168.0.1    ← home router
2  192.168.1.1    ← ISP modem
3  172.17.0.1     ← ISP CG-NAT device   ← the blocker
4  183.x.x.x      ← public internet
```

Three layers of private addresses before reaching the public internet. Nothing outside can initiate a connection inward.

### The solution: relay + hole punching

```
1. Both devices connect outbound to the control plane (outbound traffic is never blocked)
2. Control plane exchanges their public keys and addresses
3. Attempt direct connection via UDP hole punching
4. If that fails (common under CG-NAT), relay traffic through the DERP server
5. All traffic is WireGuard-encrypted end-to-end regardless of path
```

## Architecture

Two binaries, two roles:

```
homeward-server   (runs on VPS)
├── Control plane — node registration, key distribution, topology sync
└── DERP relay    — encrypted packet forwarding when direct connection fails

homeward-client   (runs on MacBook and home machine — same binary)
├── Register with control plane
├── Sync peer list
├── Manage local WireGuard config
├── TUN virtual network interface
└── NAT traversal (STUN + UDP hole punching → fallback to DERP)
```

MacBook and the home machine run identical client code. There is no fixed "client" or "server" among peers — any node can initiate or accept connections.

## Components

| Component | Purpose |
|-----------|---------|
| Control plane | Key exchange and topology distribution |
| DERP relay | Fallback relay when direct connection is not possible |
| WireGuard | Encrypted tunnel between peers |
| TUN interface | Virtual network device, assigns each node a stable IP |
| NAT traversal | STUN discovery + UDP hole punching for direct connection |

Connection priority:

```
1. IPv6 direct     (best latency, works if home ISP provides IPv6)
2. UDP hole punch  (direct IPv4, works through single-layer NAT)
3. DERP relay      (always works, including CG-NAT)
```

## Project Structure

```
homeward/
├── cmd/
│   ├── server/          # VPS binary: control plane + DERP
│   └── client/          # Node binary: MacBook and home machine
└── internal/
    ├── control/         # Control plane logic
    ├── derp/            # Relay server
    ├── wg/              # WireGuard management
    ├── tun/             # Virtual network interface
    └── nat/             # NAT traversal
```

## Test Environment

To develop and test the full stack you need three machines:

| Machine | Role | Network |
|---------|------|---------|
| VPS | Run `homeward-server` | Public IP, reachable from anywhere |
| Home machine | Peer behind CG-NAT | No public IPv4 (your real home machine works) |
| MacBook | Peer outside | Any network (4G, office, cafe) |

## Tech Stack

- **Go** — same language Tailscale uses
- **wireguard-go** — WireGuard implementation in userspace
- **pion/ice** — ICE/STUN/TURN for NAT traversal

## References

- [WireGuard whitepaper](https://www.wireguard.com/papers/wireguard.pdf)
- [Tailscale's NAT traversal writeup](https://tailscale.com/blog/how-nat-traversal-works)
- [Headscale](https://github.com/juanfont/headscale) — open source Tailscale control plane, worth reading
- [NetBird](https://github.com/netbirdio/netbird) — similar architecture, clean codebase
