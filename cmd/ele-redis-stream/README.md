# ele-redis-stream

Redis Stream 测试工具，用于 room_events 流的生产、消费和裁剪。基于 `stream` 包抽象，底层使用 Redis 实现。

## 构建

```bash
make redisstream
# 输出: bin/ele-redis-stream
```

## 配置

通过 `-c` 指定配置文件，需包含 `redis` 段：

```yaml
redis:
  address: localhost:6379
  password: ""
  size: 10

log:
  level: info
```

## 子命令

### consume - 消费

从 `room_events` 流中读取事件并输出到控制台。

```bash
./bin/ele-redis-stream consume -c config.yaml
./bin/ele-redis-stream consume -c config.yaml -b 2000   # block 超时 2000ms
```

| 参数 | 说明 | 默认 |
|------|------|------|
| `-c, --config` | 配置文件路径 | config.yaml |
| `-b, --block` | XREAD 阻塞超时 (ms) | 1000 |

### produce - 生产

向 `room_events` 流写入测试事件。

```bash
./bin/ele-redis-stream produce -c config.yaml
./bin/ele-redis-stream produce -c config.yaml -n 10 -i 1s   # 发送 10 条，间隔 1 秒
./bin/ele-redis-stream produce -c config.yaml -t my_topic   # 指定 topic
```

| 参数 | 说明 | 默认 |
|------|------|------|
| `-c, --config` | 配置文件路径 | config.yaml |
| `-t, --topic` | 主题 | test_topic_0x123_0x456 |
| `-i, --interval` | 发送间隔 | 1s |
| `-n, --count` | 发送条数 (0=持续直到 Ctrl+C) | 0 |

### trim - 裁剪

删除超过指定时间的旧条目，执行前会打印将被删除的内容。

```bash
./bin/ele-redis-stream trim -c config.yaml -m 1h
./bin/ele-redis-stream trim -c config.yaml -m 10s -n   # dry-run，不实际删除
./bin/ele-redis-stream trim -c config.yaml -s room_events -m 30m
```

| 参数 | 说明 | 默认 |
|------|------|------|
| `-c, --config` | 配置文件路径 | config.yaml |
| `-m, --max-age` | 删除超过此时间的条目 (如 1h, 30m, 10s) | 1h |
| `-s, --stream` | 流名称 | room_events |
| `-n, --dry-run` | 仅预览，不执行删除 | false |

## 使用示例

```bash
# 终端 1: 启动消费者
./bin/ele-redis-stream consume -c stream_consumer_config.yaml

# 终端 2: 生产 5 条测试消息
./bin/ele-redis-stream produce -c stream_config.yaml -n 5 -i 500ms

# 查看redis里的消息
redis-cli XRANGE room_events - +

# 裁剪 1 小时前的数据（先 dry-run 预览）
./bin/ele-redis-stream trim -c stream_config.yaml -m 30s -n
./bin/ele-redis-stream trim -c stream_config.yaml -m 30s
```

## Redis 版本

- **XTRIM MINID** 需要 Redis 6.2+
- Redis 6.0 及以下会自动回退到 XDEL 逐条删除

## 消息格式

流中每条消息包含字段：`topic`、`payload`（base64 编码的 protobuf）、`ts`。与 Room Server PubSub.Publish 格式一致。
