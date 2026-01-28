# 数据库设计

FlowMind 使用 SQLite 作为主要数据库，采用关系型设计，结合向量数据库实现语义搜索。

---

## 数据库选型

### SQLite

**选择理由**：
- ✅ 轻量级，无需独立服务
- ✅ ACID 保证
- ✅ 优秀的并发性能（WAL 模式）
- ✅ 跨平台支持
- ✅ 丰富的 Go 生态

**适用场景**：
- 事件日志存储
- 配置管理
- 用户数据
- 自动化脚本
- 执行历史

### Chromem-go（向量数据库）

**选择理由**：
- ✅ 纯 Go 实现
- ✅ 本地优先
- ✅ 支持自定义 embedding 函数
- ✅ 内存性能优化

**适用场景**：
- 知识库语义搜索
- 内容推荐
- 相似度匹配

---

## 核心表设计

详细表结构请参考：[存储层 - SQLite 数据库](../architecture/06-storage-layer.md#sqlite-数据库)

### 1. events（事件表）

```sql
CREATE TABLE events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid TEXT UNIQUE NOT NULL,
    type TEXT NOT NULL,
    timestamp DATETIME NOT NULL,
    data JSON,
    application TEXT,
    bundle_id TEXT,
    window_title TEXT,
    file_path TEXT,
    selection TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

**索引**：
- `idx_events_timestamp` - 时间范围查询
- `idx_events_type` - 按类型过滤
- `idx_events_application` - 按应用过滤
- `idx_events_uuid` - UUID 查找

**分区策略**（未来）：
- 按月分区，便于清理旧数据

### 2. patterns（模式表）

```sql
CREATE TABLE patterns (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid TEXT UNIQUE NOT NULL,
    sequence_hash TEXT UNIQUE NOT NULL,
    sequence JSON NOT NULL,
    support_count INTEGER DEFAULT 1,
    first_seen DATETIME NOT NULL,
    last_seen DATETIME NOT NULL,
    is_automated BOOLEAN DEFAULT FALSE,
    automation_id INTEGER,
    ai_analysis TEXT
);
```

**关系**：
- `automation_id` → `automations(id)`

### 3. knowledge_items（知识库表）

```sql
CREATE TABLE knowledge_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid TEXT UNIQUE NOT NULL,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    tags JSON,
    summary TEXT,
    category TEXT,
    embedding_id TEXT,
    view_count INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

**关系**：
- 通过 `knowledge_relations` 表建立知识图谱

### 4. automations（自动化表）

```sql
CREATE TABLE automations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    trigger_type TEXT NOT NULL,
    trigger_config JSON NOT NULL,
    steps JSON NOT NULL,
    enabled BOOLEAN DEFAULT TRUE,
    run_count INTEGER DEFAULT 0
);
```

**关系**：
- `execution_results.automation_id` → `automations(id)`

---

## ER 图

```
┌─────────────┐
│   events    │
└─────────────┘
       │
       │ (时间聚合)
       ↓
┌─────────────┐
│  sessions   │
└─────────────┘
       │
       │ (模式挖掘)
       ↓
┌─────────────┐       ┌──────────────┐
│  patterns   │ ──→  │ automations  │
└─────────────┘       └──────────────┘
                            │
                            │ (执行)
                            ↓
                     ┌──────────────┐
                     │execution_res │
                     └──────────────┘

┌─────────────┐       ┌──────────────┐
│knowledge_item│ ←─→ │knowledge_rel │
└─────────────┘       └──────────────┘
       │
       │ (向量搜索)
       ↓
┌─────────────┐
│ vector_store│
└─────────────┘
```

---

## 数据迁移策略

### 版本控制

```go
// 使用 migrations 目录
migrations/
├── 001_init.sql
├── 002_add_patterns_table.sql
├── 003_add_knowledge_relations.sql
└── ...
```

### 迁移记录表

```sql
CREATE TABLE schema_migrations (
    version INTEGER PRIMARY KEY,
    applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

---

## 性能优化

### WAL 模式

```sql
PRAGMA journal_mode=WAL;
```

**优点**：
- 更好的并发读写
- 更快的提交速度
- 减少磁盘 I/O

### 连接池配置

```go
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(5 * time.Minute)
```

### 批量操作

```go
// 使用事务批量插入
tx, _ := db.Begin()
stmt, _ := tx.Prepare("INSERT INTO events ...")
for _, event := range events {
    stmt.Exec(...)
}
tx.Commit()
```

---

## 备份与恢复

### 在线备份

```go
// 使用 SQLite backup API
err := db.Backup(targetDB, 0)
```

### 导出 SQL

```bash
sqlite3 flowmind.db .dump > backup.sql
```

### 恢复

```bash
sqlite3 flowmind.db < backup.sql
```

---

## 数据保留策略

### 事件数据

- **热数据**：最近 7 天，常驻内存
- **温数据**：7-30 天，存储在磁盘
- **冷数据**：30 天以上，归档或删除

### 自动清理

```go
// 定期清理任务
func (m *Maintenance) CleanupOldEvents(retention time.Duration) {
    cutoff := time.Now().Add(-retention)
    db.Exec("DELETE FROM events WHERE timestamp < ?", cutoff)
}
```

---

## 监控指标

### 关键指标

- 事件插入速率（events/sec）
- 查询延迟（ms）
- 数据库大小（MB）
- 缓存命中率（%）
- 连接池使用率（%）

### 监控查询

```sql
-- 数据库大小
SELECT page_count * page_size / 1024 / 1024 AS size_mb
FROM pragma_page_count(), pragma_page_size();

-- 表大小
SELECT name, (pgsize * 1024) / 1024 AS size_mb
FROM pragma_page_count()
JOIN sqlite_master USING(name);

-- 慢查询
-- 需要启用 SQLITE_DEBUG
```

---

**相关文档**：
- [存储层](../architecture/06-storage-layer.md)
- [API 设计](./02-api-design.md)
- [安全设计](./05-security-design.md)
