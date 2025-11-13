# Scanner 流程图

## 整体架构流程图

```mermaid
flowchart TD
    Start([开始]) --> NewScanner[NewScanner<br/>创建Scanner实例]
    NewScanner --> Init[初始化eventSigHashCache]
    Init --> Run[Run方法启动]
    
    Run --> InitCache[initEventSigHashCache<br/>初始化事件签名哈希缓存]
    InitCache --> LoadDB[从数据库加载BlockSync状态<br/>设置headNumberOnChain<br/>currentScannedHeight<br/>toSubmitHeight]
    LoadDB --> ConnectRPC[连接RoomServer RPC客户端]
    ConnectRPC --> StartCatchUp[启动RunCatchUp goroutine]
    
    StartCatchUp --> RunCatchUp[RunCatchUp主循环]
    
    RunCatchUp --> ConnectWS{连接WebSocket RPC}
    ConnectWS -->|失败| RetryWS[等待5秒后重试]
    RetryWS --> ConnectWS
    ConnectWS -->|成功| StartChain[启动CatchUpChain goroutine]
    
    StartChain --> Subscribe[订阅新区块头]
    Subscribe -->|失败| RetrySub[关闭连接,等待5秒后重试]
    RetrySub --> ConnectWS
    Subscribe -->|成功| ListenHeaders[监听新区块头]
    
    ListenHeaders --> CheckCtx{检查Context}
    CheckCtx -->|Done| Exit1[退出RunCatchUp]
    CheckCtx -->|继续| CheckSub{订阅错误?}
    CheckSub -->|有错误| RetrySub
    CheckSub -->|无错误| NewHeader[收到新区块头]
    NewHeader --> CheckReorg{检查链重组}
    CheckReorg -->|重组| Warn[警告日志]
    CheckReorg -->|正常| UpdateHead[更新headNumberOnChain]
    UpdateHead --> ListenHeaders
    
    StartChain --> CatchUpChain[CatchUpChain主流程]
    
    CatchUpChain --> CreateChannels[创建通道<br/>blockQueue: 200<br/>submitChan: 100]
    CreateChannels --> StartDistributor[启动任务分发协程]
    CreateChannels --> StartWorkers[启动100个Worker协程]
    CreateChannels --> StartSubmitter[启动有序提交协程]
    
    StartDistributor --> DistributorLoop[分发协程循环]
    DistributorLoop --> CheckCtx2{检查Context}
    CheckCtx2 -->|Done| Exit2[退出分发协程]
    CheckCtx2 -->|继续| CheckHeight{currentScannedHeight<br/>> headNumberOnChain?}
    CheckHeight -->|是| Wait1[等待200ms]
    Wait1 --> DistributorLoop
    CheckHeight -->|否| CheckQueue{blockQueue长度<br/><= 50?}
    CheckQueue -->|否| Wait2[等待100ms]
    Wait2 --> DistributorLoop
    CheckQueue -->|是| AddBlock[addBlockToQueue<br/>添加区块到队列<br/>currentScannedHeight++]
    AddBlock --> DistributorLoop
    
    StartWorkers --> WorkerLoop[Worker循环<br/>100个并发]
    WorkerLoop --> CheckCtx3{检查Context}
    CheckCtx3 -->|Done| Exit3[退出Worker]
    CheckCtx3 -->|继续| GetBlock[从blockQueue获取区块号]
    GetBlock --> ProcessBlock[getAndProcessBlockToBatch<br/>处理区块]
    
    ProcessBlock --> GetBlockData[通过HTTP RPC获取区块数据]
    GetBlockData --> ParseTxs[解析Optimism交易]
    ParseTxs --> ProcessEachTx[遍历每个交易<br/>processTx]
    
    ProcessEachTx --> CheckSpecial{特殊交易?<br/>to=0x4200...0015}
    CheckSpecial -->|是| Skip[跳过]
    CheckSpecial -->|否| GetReceipt[获取交易Receipt]
    GetReceipt --> CheckLogs[遍历Logs]
    CheckLogs --> MatchEvent{匹配事件类型?}
    MatchEvent -->|RoomCreated| HandleRoomCreated[处理RoomCreated事件]
    MatchEvent -->|submitCardsHash| HandleCardsHash[处理submitCardsHash事件]
    MatchEvent -->|submitCards| HandleCards[处理submitCards事件]
    MatchEvent -->|startANewRound| HandleNewRound[处理startANewRound事件]
    MatchEvent -->|不匹配| CheckLogs
    HandleRoomCreated --> BuildTx[构建proto.Transaction]
    HandleCardsHash --> BuildTx
    HandleCards --> BuildTx
    HandleNewRound --> BuildTx
    BuildTx --> CheckLogs
    Skip --> NextTx[下一个交易]
    CheckLogs -->|完成| NextTx
    NextTx -->|还有交易| ProcessEachTx
    NextTx -->|完成| BuildBatch[构建TransactionBatch]
    BuildBatch --> ProcessBlock
    
    ProcessBlock -->|失败| RetryBlock[重新放入blockQueue<br/>等待3秒]
    RetryBlock --> WorkerLoop
    ProcessBlock -->|成功| SendSubmit[发送到submitChan<br/>orderedTxBatch]
    SendSubmit --> WaitResult[等待提交结果]
    
    WaitResult --> CheckResult{提交成功?}
    CheckResult -->|失败| RetrySubmit[重新放入blockQueue<br/>等待3秒]
    RetrySubmit --> WorkerLoop
    CheckResult -->|成功| CheckSave{blockNumber % 10 == 0?}
    CheckSave -->|是| SaveDB[保存到数据库<br/>SaveBlockSync]
    CheckSave -->|否| LogSuccess[记录成功日志]
    SaveDB --> LogSuccess
    LogSuccess --> WorkerLoop
    
    StartSubmitter --> SubmitLoop[有序提交协程循环]
    SubmitLoop --> CheckCtx4{检查Context}
    CheckCtx4 -->|Done| Exit4[退出提交协程]
    CheckCtx4 -->|继续| SelectBatch{select事件}
    SelectBatch -->|收到batch| AddPending["添加到pendingBatches<br/>map(blockNumber)batch"]
    SelectBatch -->|定时器tick| TrySubmit[尝试提交]
    AddPending --> TrySubmit
    
    TrySubmit --> GetToSubmit[获取toSubmitHeight]
    GetToSubmit --> CheckPending{"存在<br/>pendingBatches(toSubmitHeight)?"}
    CheckPending -->|否| WaitNext[等待下一个区块]
    WaitNext --> SubmitLoop
    CheckPending -->|是| SubmitBatch[submitBatch<br/>提交到RoomServer]
    
    SubmitBatch --> CheckMock{RoomServer Mocked?}
    CheckMock -->|是| SkipSubmit[跳过提交]
    CheckMock -->|否| RPCSubmit[通过RPC提交<br/>超时3秒]
    RPCSubmit --> CheckSubmit{提交成功?}
    CheckSubmit -->|失败| KeepPending[保留在pendingBatches<br/>等待重试]
    KeepPending --> SubmitLoop
    CheckSubmit -->|成功| SendDone[发送结果到done channel]
    SkipSubmit --> SendDone
    SendDone --> DeletePending[从pendingBatches删除]
    DeletePending --> IncToSubmit[toSubmitHeight++]
    IncToSubmit --> TrySubmit
    
    Exit1 --> End([结束])
    Exit2 --> End
    Exit3 --> End
    Exit4 --> End
```

## 关键数据结构

```mermaid
classDiagram
    class Scanner {
        -ctx context.Context
        -cancel context.CancelFunc
        -gethWsRpc string
        -gethHttpRpc string
        -roomServerHttpRpc string
        -gethClient *ethclient.Client
        -rpcClient *eleClient.RpcClient
        -currentScannedHeight uint64
        -toSubmitHeight uint64
        -headNumberOnChain uint64
        -eventSigHashCache eventSigHashCache
        +Run()
        +RunCatchUp()
        +CatchUpChain()
        +Stop()
    }
    
    class eventSigHashCache {
        -mu sync.RWMutex
        -eventNameToHash map[string]Hash
        -eventHashToName map[Hash]string
    }
    
    class orderedTxBatch {
        +blockNumber uint64
        +batch *proto.TransactionBatch
        +done chan error
    }
    
    Scanner --> eventSigHashCache
    Scanner --> orderedTxBatch
```

## 并发模型

```mermaid
graph LR
    subgraph "主流程"
        A[Run] --> B[RunCatchUp]
        B --> C[CatchUpChain]
    end
    
    subgraph "RunCatchUp Goroutine"
        D[订阅新区块头] --> E[更新headNumberOnChain]
    end
    
    subgraph "CatchUpChain Goroutines"
        F[任务分发协程<br/>1个] --> G[blockQueue]
        G --> H[Worker协程<br/>100个并发]
        H --> I[submitChan]
        I --> J[有序提交协程<br/>1个]
    end
    
    B -.->|goroutine| D
    C -.->|goroutine| F
    C -.->|goroutine| H
    C -.->|goroutine| J
```

## 状态流转

```mermaid
stateDiagram-v2
    [*] --> 初始化: NewScanner
    初始化 --> 加载状态: initEventSigHashCache
    加载状态 --> 连接RPC: 从DB加载BlockSync
    连接RPC --> 运行中: 连接RoomServer RPC
    运行中 --> 订阅区块: 连接WebSocket RPC
    订阅区块 --> 处理区块: 收到新区块头
    处理区块 --> 分发任务: 添加到blockQueue
    分发任务 --> Worker处理: 100个Worker并发
    Worker处理 --> 构建Batch: 解析交易和事件
    构建Batch --> 等待提交: 发送到submitChan
    等待提交 --> 有序提交: 按toSubmitHeight顺序
    有序提交 --> 保存状态: 每10个区块保存DB
    保存状态 --> 处理区块: 继续下一个区块
    运行中 --> [*]: Stop()或Context取消
```

## 关键流程说明

### 1. 初始化阶段
- 初始化事件签名哈希缓存（Room和RoomManager合约事件）
- 从数据库加载上次同步的区块高度
- 连接RoomServer RPC客户端

### 2. 区块订阅阶段（RunCatchUp）
- 通过WebSocket订阅新区块头
- 实时更新链头高度（headNumberOnChain）
- 检测链重组并记录警告

### 3. 区块处理阶段（CatchUpChain）
- **任务分发协程**：控制投递速度，避免队列过载
- **Worker协程**（100个并发）：
  - 从blockQueue获取区块号
  - 通过HTTP RPC获取区块数据
  - 解析交易并提取相关事件
  - 构建TransactionBatch
- **有序提交协程**：
  - 接收Worker处理完成的batch
  - 按toSubmitHeight顺序提交到RoomServer
  - 保证提交顺序与区块顺序一致

### 4. 事件处理
支持4种事件类型：
- `RoomCreated`: 房间创建事件
- `submitCardsHash`: 提交卡牌哈希
- `submitCards`: 提交卡牌
- `startANewRound`: 开始新回合

### 5. 错误处理和重试
- 连接失败自动重试（5秒间隔）
- 区块处理失败重新放入队列
- 提交失败保留在pendingBatches等待重试
- 每10个区块保存一次状态到数据库

