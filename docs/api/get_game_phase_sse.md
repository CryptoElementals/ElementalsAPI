# GetGamePhase SSE API 文档

## 概述

GetGamePhase SSE API 是一个基于Server-Sent Events (SSE)的实时游戏阶段监听接口，用于订阅和接收特定玩家的游戏状态变化。

## 基本信息

- **接口名称**: GetGamePhaseSSE
- **请求方式**: SSE (Server-Sent Events)
- **认证方式**: 需要玩家地址认证
- **超时时间**: 可配置，默认600秒（10分钟）

## 请求参数

### 请求结构

```json
{
  "Action": "GetGamePhaseSSE",
  "RequestUUID": "string",
  "TempAddress": "string",
  "Duration": "integer"
}
```

### 参数说明

| 参数名 | 类型 | 必填 | 默认值 | 说明 |
|--------|------|------|--------|------|
| Action | string | 是 | - | 固定值："GetGamePhaseSSE" |
| RequestUUID | string | 是 | - | 请求唯一标识符 |
| TempAddress | string | 是 | - | 临时地址，用于标识特定的游戏会话 |
| Duration | integer | 是 | 600 | 监听持续时间（秒），范围：1-3600 |

### 请求示例

```json
{
  "Action": "GetGamePhaseSSE",
  "RequestUUID": "req-12345-67890",
  "TempAddress": "0x1234567890abcdef",
  "Duration": 1800
}
```

## 响应格式

### SSE 事件格式

API 使用标准的 SSE 格式返回数据：

```
data: {"type":"status_update","data":{"status":"started","address":"0x...","tempAddress":"0x...","duration":1800},"requestUUID":"req-12345-67890"}

data: {"type":"game_phase_update","data":{"phase":"waiting","players":3,"maxPlayers":10},"requestUUID":"req-12345-67890"}

data: {"type":"status_update","data":{"status":"completed"},"requestUUID":"req-12345-67890"}
```

### 事件类型

#### 1. 开始事件 (status_update)
```json
{
  "type": "status_update",
  "data": {
    "status": "started",
    "address": "0x1234567890abcdef",
    "tempAddress": "0x1234567890abcdef",
    "duration": 1800
  },
  "requestUUID": "req-12345-67890"
}
```

#### 2. 游戏阶段更新事件 (game_phase_update)
```json
{
  "type": "game_phase_update",
  "data": {
    "phase": "waiting|playing|finished",
    "players": 3,
    "maxPlayers": 10,
    "gameId": "game-12345"
  },
  "requestUUID": "req-12345-67890"
}
```

#### 3. 结束事件 (status_update)
```json
{
  "type": "status_update",
  "data": {
    "status": "completed"
  },
  "requestUUID": "req-12345-67890"
}
```

## 使用示例

### JavaScript 客户端示例

```javascript
// 建立 SSE 连接
const eventSource = new EventSource('/api/match/get_game_phase_sse', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': 'Bearer your-token'
  },
  body: JSON.stringify({
    Action: "GetGamePhaseSSE",
    RequestUUID: "req-" + Date.now(),
    TempAddress: "0x1234567890abcdef",
    Duration: 1800
  })
});

// 监听消息
eventSource.onmessage = function(event) {
  const data = JSON.parse(event.data);
  
  switch(data.type) {
    case 'status_update':
      if (data.data.status === 'started') {
        console.log('开始监听游戏阶段变化');
      } else if (data.data.status === 'completed') {
        console.log('监听结束');
        eventSource.close();
      }
      break;
      
    case 'game_phase_update':
      console.log('游戏阶段更新:', data.data.phase);
      console.log('当前玩家数:', data.data.players);
      break;
  }
};

// 错误处理
eventSource.onerror = function(error) {
  console.error('SSE 连接错误:', error);
  eventSource.close();
};
```

### cURL 示例

```bash
curl -X POST http://localhost:8080/api/match/get_game_phase_sse \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-token" \
  -d '{
    "Action": "GetGamePhaseSSE",
    "RequestUUID": "req-12345-67890",
    "TempAddress": "0x1234567890abcdef",
    "Duration": 1800
  }'
```

## 错误处理

### 常见错误码

| 错误码 | 说明 | 解决方案 |
|--------|------|----------|
| 400 | 参数验证失败 | 检查请求参数格式和必填项 |
| 401 | 认证失败 | 确保提供有效的玩家地址 |
| 500 | 内部服务器错误 | 检查服务器日志 |

### 错误响应示例

```json
{
  "Action": "GetGamePhaseSSEResponse",
  "RequestUUID": "req-12345-67890",
  "Success": false,
  "Message": "参数验证失败: TempAddress is required"
}
```

## 技术细节

### 连接管理

- 连接建立后会自动发送开始事件
- 通过 gRPC 订阅 RoomServer 的实时事件
- 支持优雅断开和超时处理
- 最大监听时间为 3600 秒（1小时）

### 性能考虑

- 使用 SSE 实现实时推送，减少轮询开销
- 支持多客户端并发连接
- 自动处理连接断开和重连

### 安全说明

- 需要有效的玩家地址认证
- 地址会自动转换为小写格式
- 支持请求超时和资源清理

## 注意事项

1. **连接超时**: 默认监听时间为 600 秒，可根据需要调整
2. **地址格式**: 所有地址都会自动转换为小写格式
3. **事件顺序**: 事件按时间顺序发送，建议客户端保持事件状态
4. **重连机制**: 客户端应实现自动重连逻辑
5. **资源清理**: 连接结束后会自动清理相关资源

## 更新日志

- v1.0.0: 初始版本，支持基本的游戏阶段监听功能 