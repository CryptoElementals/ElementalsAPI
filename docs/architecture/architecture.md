# 架构概览

```mermaid
%%{init: {'theme': 'base', 'themeVariables': {'lineColor':'#333','primaryBorderColor':'#333','primaryTextColor':'#333','secondaryBorderColor':'#333','tertiaryBorderColor':'#333'}}}%%
flowchart LR
    subgraph LEFT["架构图"]
        direction TB
        C[多终端 / Web]
        ALB[ALB · HTTPS]
        AP[ele-apiserver · 多实例]
        RD[(Redis · 鉴权)]
        NLBR[NLB · TCP]
        NLBL[NLB · TCP]
        RS[ele-roomserver · 多实例]
        LS[Lobby Server · 多实例]
        MQ[("消息队列")]
        SC[ele-scanner · 多实例]
        L2[Layer 2 · Node ]
        DB[mysql]
        C --> ALB --> AP --> RD
        AP --> DB
        AP --> NLBR --> RS --> DB
        AP --> NLBL --> LS --> DB
        RS -->|发布| MQ
        LS -->|发布| MQ
        MQ -->|订阅| LS
        MQ -->|仅订阅| AP
        SC --> | submit txs | NLBR
        SC -->|扫块| L2
        SC --> DB
    end
```

---

## 核心服务

| 服务 | 职责 |
|------|------|
| **ele-apiserver** | **多实例**；HTTP、Session；**鉴权等用 Redis**；经 NLB 调 Room / Lobby；**只从 MQ 消费** |
| **Lobby Server** | **多实例**；大厅与匹配；**向 MQ 发布并订阅** |
| **ele-roomserver** | **多实例**；对战、链上交互；**向 MQ 发布** |
| **消息队列** | **Room / Lobby 生产**；**Lobby、API 消费** |
| **ele-scanner** | **多实例**；**扫 Layer2**；**经 NLB（Room）** 调 Room；落库 |

---