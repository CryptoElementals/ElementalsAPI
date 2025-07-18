# GetGamePhase API 文档

## 概述

GetGamePhase API 用于获取当前玩家的游戏阶段状态，包括PvP对战信息和对战玩家列表。

## 基本信息

- **接口名称**: GetGamePhase
- **请求方式**: POST
- **认证方式**: 需要玩家地址认证
- **响应格式**: JSON

## 请求参数

### 请求结构

```json
{
  "Action": "GetGamePhase",
  "RequestUUID": "string",
  "TempAddress": "string"
}
```

### 参数说明

| 参数名 | 类型 | 描述信息 | 必填 |
|--------|------|----------|------|
| **Action** | string | GetGamePhase | **Yes** |
| **RequestUUID** | string | 请求唯一标识符 | **Yes** |
| **TempAddress** | string | 临时地址，用于标识特定的游戏会话 | **Yes** |

### 请求示例

```json
{
  "Action": "GetGamePhase",
  "RequestUUID": "req-12345-67890",
  "TempAddress": "0x1234567890abcdef"
}
```

## 响应字段

### 响应结构

```json
{
  "RetCode": 0,
  "Action": "GetGamePhaseResponse",
  "RequestUUID": "req-12345-67890",
  "Message": "Player is in match queue",
  "Mode": "PvP",
  "PvPInfo": {
    "Phase": "Queueing",
    "MatchId": "match-12345",
    "RoomId": "room-67890"
  },
  "Players": [
    {
      "Address": "0x1234567890abcdef",
      "Name": "Player1",
      "AvatarURL": "https://example.com/avatar1.jpg",
      "IsMyself": true,
      "Confirmed": false
    }
  ]
}
```

### 响应字段说明

| 字段名 | 类型 | 描述信息 | 必填 |
|--------|------|----------|------|
| **RetCode** | uint | 返回状态码，为 0 则为成功返回，非 0 为失败 | **Yes** |
| **Action** | string | GetGamePhaseResponse | **Yes** |
| **RequestUUID** | string | 请求唯一标识符 | **Yes** |
| **Message** | string | 返回消息，描述当前状态 | No |
| **Mode** | string | 游戏模式，None 或 PvP | **Yes** |
| **PvPInfo** | object | PvP对战信息 | No |
| **Players** | array | 对战玩家列表 | No |

### PvPInfo 字段说明

| 字段名 | 类型 | 描述信息 | 必填 |
|--------|------|----------|------|
| **Phase** | string | 游戏阶段：None, Queueing, Matching, InBattle | **Yes** |
| **MatchId** | string | 匹配ID | No |
| **RoomId** | string | 房间ID | No |

### Players 字段说明

| 字段名 | 类型 | 描述信息 | 必填 |
|--------|------|----------|------|
| **Address** | string | 玩家钱包地址 | **Yes** |
| **Name** | string | 玩家昵称 | **Yes** |
| **AvatarURL** | string | 玩家头像URL | **Yes** |
| **IsMyself** | boolean | 是否为当前玩家 | **Yes** |
| **Confirmed** | boolean | 是否已确认对战 | **Yes** |

## 游戏阶段说明

### Phase 状态说明

| 状态值 | 描述 | 说明 |
|--------|------|------|
| None | 未参与游戏 | 玩家当前未参与任何游戏活动 |
| Queueing | 排队中 | 玩家已加入匹配队列，等待匹配 |
| Matching | 匹配中 | 玩家已匹配成功，等待确认 |
| InBattle | 战斗中 | 玩家已进入对战房间 |

### Mode 说明

| 模式值 | 描述 |
|--------|------|
| None | 未参与任何游戏模式 |
| PvP | 参与PvP对战模式 |

## 响应示例

### 成功响应 - 排队中

```json
{
  "RetCode": 0,
  "Action": "GetGamePhaseResponse",
  "RequestUUID": "req-12345-67890",
  "Message": "Player is in match queue",
  "Mode": "PvP",
  "PvPInfo": {
    "Phase": "Queueing",
    "MatchId": "",
    "RoomId": ""
  },
  "Players": []
}
```

### 成功响应 - 匹配中

```json
{
  "RetCode": 0,
  "Action": "GetGamePhaseResponse",
  "RequestUUID": "req-12345-67890",
  "Message": "Player matched, waiting for confirmation",
  "Mode": "PvP",
  "PvPInfo": {
    "Phase": "Matching",
    "MatchId": "match-12345",
    "RoomId": "room-67890"
  },
  "Players": [
    {
      "Address": "0x1234567890abcdef",
      "Name": "Player1",
      "AvatarURL": "https://example.com/avatar1.jpg",
      "IsMyself": true,
      "Confirmed": false
    },
    {
      "Address": "0xfedcba0987654321",
      "Name": "Player2",
      "AvatarURL": "https://example.com/avatar2.jpg",
      "IsMyself": false,
      "Confirmed": false
    }
  ]
}
```

### 成功响应 - 战斗中

```json
{
  "RetCode": 0,
  "Action": "GetGamePhaseResponse",
  "RequestUUID": "req-12345-67890",
  "Message": "Player has entered battle",
  "Mode": "PvP",
  "PvPInfo": {
    "Phase": "InBattle",
    "MatchId": "match-12345",
    "RoomId": "room-67890"
  },
  "Players": [
    {
      "Address": "0x1234567890abcdef",
      "Name": "Player1",
      "AvatarURL": "https://example.com/avatar1.jpg",
      "IsMyself": true,
      "Confirmed": true
    },
    {
      "Address": "0xfedcba0987654321",
      "Name": "Player2",
      "AvatarURL": "https://example.com/avatar2.jpg",
      "IsMyself": false,
      "Confirmed": true
    }
  ]
}
```

### 成功响应 - 未参与游戏

```json
{
  "RetCode": 0,
  "Action": "GetGamePhaseResponse",
  "RequestUUID": "req-12345-67890",
  "Message": "Player is not participating in any game",
  "Mode": "None",
  "PvPInfo": {
    "Phase": "None",
    "MatchId": "",
    "RoomId": ""
  },
  "Players": []
}
```

## 错误处理

### 错误码说明

| 错误码 | 说明 | 解决方案 |
|--------|------|----------|
| 0 | 成功 | - |
| 1001 | 参数解析失败 | 检查请求参数格式 |
| 1001 | 获取玩家地址失败 | 确保提供有效的玩家地址 |
| 1002 | 连接RoomServer失败 | 检查RoomServer服务状态 |
| 1003 | RoomServer GetPlayerInfo失败 | 检查RoomServer日志 |

### 错误响应示例

#### 参数验证失败

```json
{
  "RetCode": 1001,
  "Action": "GetGamePhaseResponse",
  "RequestUUID": "req-12345-67890",
  "Message": "Parameter parsing failed",
  "Mode": "None",
  "PvPInfo": {
    "Phase": "None"
  }
}
```

#### 认证失败

```json
{
  "RetCode": 1001,
  "Action": "GetGamePhaseResponse",
  "RequestUUID": "req-12345-67890",
  "Message": "Failed to get player address",
  "Mode": "None",
  "PvPInfo": {
    "Phase": "None"
  }
}
```

#### RoomServer连接失败

```json
{
  "RetCode": 1002,
  "Action": "GetGamePhaseResponse",
  "RequestUUID": "req-12345-67890",
  "Message": "Failed to connect to RoomServer: connection refused",
  "Mode": "None",
  "PvPInfo": {
    "Phase": "None"
  }
}
```

## 使用示例

### cURL 示例

```bash
curl -X POST http://localhost:8080/api/match/get_game_phase \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-token" \
  -d '{
    "Action": "GetGamePhase",
    "RequestUUID": "req-12345-67890",
    "TempAddress": "0x1234567890abcdef"
  }'
```

### JavaScript 示例

```javascript
const response = await fetch('/api/match/get_game_phase', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': 'Bearer your-token'
  },
  body: JSON.stringify({
    Action: "GetGamePhase",
    RequestUUID: "req-" + Date.now(),
    TempAddress: "0x1234567890abcdef"
  })
});

const data = await response.json();

if (data.RetCode === 0) {
  console.log('游戏模式:', data.Mode);
  console.log('游戏阶段:', data.PvPInfo.Phase);
  console.log('玩家数量:', data.Players.length);
} else {
  console.error('请求失败:', data.Message);
}
```

## 技术细节

### 地址处理

- 所有钱包地址都会自动转换为小写格式
- 临时地址也会转换为小写格式以确保一致性

### 数据来源

- 通过gRPC调用RoomServer的GetPlayerInfo接口获取玩家状态
- 如果MatchId存在，会从数据库查询对战玩家信息
- 玩家昵称和头像从用户档案中获取

### 性能考虑

- 支持并发请求
- 自动处理gRPC连接和断开
- 数据库查询失败时使用默认值

## 注意事项

1. **地址格式**: 所有地址都会自动转换为小写格式
2. **玩家信息**: 如果无法获取用户档案，会使用地址作为默认昵称
3. **状态同步**: 建议定期调用此接口以获取最新状态
4. **错误处理**: 客户端应妥善处理各种错误码
5. **认证要求**: 需要有效的玩家地址认证

## 更新日志

- v1.0.0: 初始版本，支持基本的游戏阶段查询功能
- v1.1.0: 新增对战玩家列表功能 