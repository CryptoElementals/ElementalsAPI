# 锦标赛排队相关表结构（建议）

> 目标：支撑「报名后冻结 token、首轮定时开赛、后续轮按胜者推进」。

### 1) `tournament`

用于一届锦标赛主记录。

| 字段 | 类型示例 | 说明 |
|------|----------|------|
| `id` | bigint pk | 锦标赛ID |
| `name` | varchar | 名称 |
| `status` | varchar | `registration_open` / `in_progress` / `finished` |
| `scheduled_start_at` | datetime | 首轮计划开赛时间（整点） |
| `registration_deadline` | datetime | 报名截止时间 |
| `entry_fee` | int | 报名费 |
| `created_at/updated_at` | datetime | 审计字段 |

---

### 2) `tournament_participant`

用于记录玩家报名与参赛状态（每人每届一条）。

| 字段 | 类型示例 | 说明 |
|------|----------|------|
| `id` | bigint pk | 参赛记录ID |
| `tournament_id` | bigint fk | 关联 `tournament.id` |
| `player_id` | bigint | 玩家ID |
| `temp_address` | varchar | 临时地址（与现有系统对齐） |
| `status` | varchar | `queued` / `eliminated` / `champion` |
| `seed` | int nullable | 种子位（可选） |
| `created_at/updated_at` | datetime | 审计字段 |

建议约束：
- `UNIQUE(tournament_id, player_id)`（防止重复报名）

---

### 3) `tournament_round`

用于记录每一轮状态（可选但推荐，有助于幂等和观测）。

| 字段 | 类型示例 | 说明 |
|------|----------|------|
| `id` | bigint pk | 轮次记录ID |
| `tournament_id` | bigint fk | 关联 `tournament.id` |
| `round_no` | int | 第几轮（1,2,3...） |
| `status` | varchar | `pending` / `matched` / `playing` / `completed` |
| `created_at/updated_at` | datetime | 审计字段 |

建议约束：
- `UNIQUE(tournament_id, round_no)`

---

### 4) `tournament_match`

用于记录每轮配对、胜者与关联的 PVP `game_id`。

| 字段 | 类型示例 | 说明 |
|------|----------|------|
| `id` | bigint pk | 对阵ID |
| `tournament_id` | bigint fk | 关联 `tournament.id` |
| `round_no` | int | 所属轮次 |
| `match_no` | int | 轮内序号 |
| `player_a_participant_id` | bigint fk | 选手A（关联 `tournament_participant.id`） |
| `player_b_participant_id` | bigint fk nullable | 选手B（轮空可空） |
| `winner_participant_id` | bigint fk nullable | 胜者 |
| `game_id` | bigint nullable | 关联 RoomServer 的 PVP 对局ID |
| `status` | varchar | `matched` / `playing` / `completed` / `bye` |
| `created_at/updated_at` | datetime | 审计字段 |

建议约束：
- `UNIQUE(tournament_id, round_no, match_no)`（防重复建对阵）

---

### 5) 与 `locked_user_tokens` 的关系（整届锁）

锦标赛 token 锁建议沿用现有 `locked_user_tokens`，并扩展：

| 字段 | 说明 |
|------|------|
| `tournament_id` (nullable) | 非空表示该锁属于锦标赛整届报名锁 |
| `tournament_participant_id` (nullable) | 对应到 `tournament_participant.id`（便于对账） |
| `game_id` | 锦标赛整届锁可保持 `0`，不随每轮变化 |

语义：
- 报名成功时创建一条整届锁（`tournament_id` 非空）。
- 单场 `GameCompleted` 不删除整届锁。
- 玩家出局或夺冠（本届对该玩家结束）时再解锁。
