# goworkers

## 改进：心跳机制 PostgreSQL → Redis

### 原有方案

每次 Worker 心跳触发 2 次同步 PostgreSQL 操作：

```
POST /api/workers/:id/heartbeat
  → SELECT * FROM workers WHERE id = ?
  → UPDATE workers SET status='online', last_seen=now() WHERE id = ?
```

缺陷：
- 每次心跳都产生 DB 写压力
- 没有自动离线判断，Worker 崩溃后 status 永远停留在 `online`

### 新方案

心跳改为只操作 Redis，DB 状态由后台 goroutine 异步同步：

```
POST /api/workers/:id/heartbeat
  → EXPIRE worker:heartbeat:{id} 30s   # 纯内存操作
  → GET   worker:info:{id}             # 读缓存，不查 DB

# 后台每 15s 执行一次
SyncWorkerStatus()
  → 扫描 DB 中所有 worker
  → 对比 Redis key 是否存在
  → 仅在状态变化时才写 DB
```

Worker 30 秒内未心跳 → Redis key 自动过期 → 下次同步标记为 `offline`。

### 性能测试结果

环境：Apple M4 · go1.26.1 · arm64

**顺序调用（单 Worker）**

| 实现 | 耗时/次 | 内存/次 | 内存分配次数 |
|------|---------|---------|------------|
| PostgreSQL | ~990 µs | 11,932 B | 173 |
| Redis | ~81 µs | 1,096 B | 23 |
| 提升 | **快 12x** | 少 91% | 少 87% |

**并发调用（多 Worker 同时心跳）**

| 实现 | 耗时/次 | 内存/次 | 内存分配次数 |
|------|---------|---------|------------|
| PostgreSQL | ~818 µs | 12,150 B | 175 |
| Redis | ~20 µs | 1,108 B | 23 |
| 提升 | **快 40x** | 少 91% | 少 87% |

并发场景差距从 12x 扩大到 40x，因为 PostgreSQL 连接池在高并发下存在锁竞争，而 Redis 单线程模型无此开销。
