# 排队匹配流程

## 时序图：PVP JoinQueue

```mermaid
%%{init: {'theme': 'base', 'themeVariables': {'lineColor':'#333','primaryBorderColor':'#333','primaryTextColor':'#333','secondaryBorderColor':'#333','tertiaryBorderColor':'#333'}}}%%
sequenceDiagram
    participant Client
    participant API
    participant Lobby
    participant RoomServer
    participant DB
    participant Redis

    Client->>API: JoinQueue (Mode, TempAddress, PlayerID)
    API->>API: 从 Session 获取 PlayerID
    API->>Lobby: JoinQueue(PlayerAddress)
    Lobby->>Lobby: 获取分布式锁

    alt 玩家已在队列
        Lobby-->>API: error: player already in queue
    end

    Lobby->>DB: 是否在Match表中
    alt 玩家已经Match
        Lobby-->>API: error: player already matched
    end

    Lobby->>DB: LockUserToken(id, tempAddr, minToken)
    alt Token 不足或已锁定, 检查locked_user_tokens表/tournament_participant表
        DB-->>Lobby: error
        Lobby-->>API: error
    end

    Lobby->>DB: 插入tournament_participant记录

    Lobby-->>API: ok
    API-->>Client: JoinQueueResponse
```

## 时序图：PVP StartBattle
```mermaid
%%{init: {'theme': 'base', 'themeVariables': {'lineColor':'#333','primaryBorderColor':'#333','primaryTextColor':'#333','secondaryBorderColor':'#333','tertiaryBorderColor':'#333'}}}%%
sequenceDiagram
    participant Client
    participant API
    participant Lobby
    participant RoomServer
    participant Contract
    participant Scanner
    participant Redis
    

    API->>Lobby: StartBattle
    Lobby->>API: Response
    API->>Lobby: StartBattle
    Lobby->>API: Response
    Lobby->>RoomServer: CreateGame
    RoomServer->>Contract: CreateRoom(有吗？)
    Contract->>RoomServer: CreateRoom TX OK
    RoomServer->>Lobby: GameCreating
    Lobby->>Lobby: 更新Match状态
    Contract->>Scanner: GameCreatedEvent
    Scanner->>RoomServer: SubmitTransactions
    RoomServer->>Redis: GameCreatedEvent
    Redis->>API: GameCreatedEvent
    API->>Client: GameCreatedEvent
```

## 时序图：PVP 超时后未双方Start
```mermaid
%%{init: {'theme': 'base', 'themeVariables': {'lineColor':'#333','primaryBorderColor':'#333','primaryTextColor':'#333','secondaryBorderColor':'#333','tertiaryBorderColor':'#333'}}}%%
sequenceDiagram
    participant Client
    participant API
    participant Lobby
    participant DB
    participant Redis

    Lobby->>Redis: PartStartedEvent
    Lobby->>DB: 更新Match状态/UnlockToken
    Redis->>API: 读取/消费 PartStartedEvent
    API->>Client: PartStartedEvent
```

## 时序图：PVP Game决出结果后结束
游戏双方决出胜负后结束。
```mermaid
%%{init: {'theme': 'base', 'themeVariables': {'lineColor':'#333','primaryBorderColor':'#333','primaryTextColor':'#333','secondaryBorderColor':'#333','tertiaryBorderColor':'#333'}}}%%
sequenceDiagram
    participant Client
    participant API
    participant Lobby
    participant RoomServer
    participant Contract
    participant DB
    participant Redis

    RoomServer->>DB: Settlement/UnlockToken
    RoomServer->>Redis: GameCompletedEvent
    Redis->>API: 读取/消费 GameCompletedEvent
    API->>Client: GameCompletedEvent

    Redis->>Lobby: GameCompletedEvent
    Lobby->>DB: 更新Match状态/LockToken/新增Match记录
    Lobby->>Redis: MatchedEvent(含MatchID)
    Redis->>API: MatchedEvent
    API->>Client: MatchedEvent
```

## 时序图：PVP Game至少一方弃赛后结束
至少一方，弃赛，超时后结束。
```mermaid
%%{init: {'theme': 'base', 'themeVariables': {'lineColor':'#333','primaryBorderColor':'#333','primaryTextColor':'#333','secondaryBorderColor':'#333','tertiaryBorderColor':'#333'}}}%%
sequenceDiagram
    participant Client
    participant API
    participant Lobby
    participant RoomServer
    participant Contract
    participant DB
    participant Redis

    RoomServer->>DB: Settlement
    RoomServer->>Redis: PlayerOfflineEvent
    Redis->>API: 读取/消费 PlayerOfflineEvent
    API->>Client: PlayerOfflineEvent

    Redis->>Lobby: PlayerOfflineEvent
    Lobby->>DB: 更新Match状态/UnlockToken
```

## 时序图：PVP Game决出结果结束后，至少一方不再继续
```mermaid
%%{init: {'theme': 'base', 'themeVariables': {'lineColor':'#333','primaryBorderColor':'#333','primaryTextColor':'#333','secondaryBorderColor':'#333','tertiaryBorderColor':'#333'}}}%%
sequenceDiagram
    participant Client
    participant API
    participant Lobby
    participant RoomServer
    participant Contract
    participant DB
    participant Redis

    API->>Lobby: CancelBattle
    Lobby->>DB: 更新Match状态/UnlockToken
    Lobby->>Redis: BattleCancelledEvent
    Redis->>API: 读取/消费 BattleCancelledEvent
    API->>Client: BattleCancelledEvent 
```

## 时序图：PVP Game决出结果结束后，至少一方没有操作后超时
统一到PVP 超时后未双方Start
