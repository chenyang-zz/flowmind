# 存储层 (Storage Layer)

存储层负责 FlowMind 的所有数据持久化需求，包括事件日志、用户配置、知识库、向量存储等，采用多存储引擎组合策略。

---

## 设计目标

1. **ACID 保证**：关键数据使用 SQLite
2. **高性能缓存**：BBolt 用于热数据缓存
3. **语义搜索**：Chromem-go 向量数据库
4. **本地优先**：所有数据本地存储
5. **可扩展性**：支持未来迁移到云存储

---

## 存储架构

```
┌─────────────────────────────────────────────────────────┐
│                    应用层                               │
│  - Monitor Engine                                       │
│  - Analyzer Engine                                      │
│  - Automation Engine                                    │
│  - Knowledge Manager                                    │
└─────────────────────────────────────────────────────────┘
                    ▲
                    │
┌───────────────────┼─────────────────────────────────────┐
│                   ▼                                     │
│            存储抽象层 (Repository Interface)            │
│  - EventRepository                                      │
│  - PatternRepository                                    │
│  - KnowledgeRepository                                  │
│  - AutomationRepository                                 │
└─────────────────────────────────────────────────────────┘
        │           │           │           │
        ▼           ▼           ▼           ▼
┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐
│ SQLite   │ │  BBolt   │ │ Chromem  │ │ File     │
│          │ │          │ │   -go    │ │ System   │
│ - events │ │ - cache  │ │ - vectors│ │ - clips  │
│ - config │ │ - state  │ │ - search │ │ - exports│
│ - users  │ │ - temp   │ │          │ │          │
└──────────┘ └──────────┘ └──────────┘ └──────────┘
```

---

## SQLite 数据库

### 数据库设计

```sql
-- migrations/001_init.sql

-- 事件表
CREATE TABLE IF NOT EXISTS events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid TEXT UNIQUE NOT NULL,
    type TEXT NOT NULL,
    timestamp DATETIME NOT NULL,
    data JSON,
    -- 上下文信息
    application TEXT,
    bundle_id TEXT,
    window_title TEXT,
    file_path TEXT,
    selection TEXT,
    -- 索引字段
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_events_timestamp ON events(timestamp);
CREATE INDEX idx_events_type ON events(type);
CREATE INDEX idx_events_application ON events(application);
CREATE INDEX idx_events_uuid ON events(uuid);

-- 模式表
CREATE TABLE IF NOT EXISTS patterns (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid TEXT UNIQUE NOT NULL,
    name TEXT,
    sequence_hash TEXT UNIQUE NOT NULL,
    sequence JSON NOT NULL,
    support_count INTEGER DEFAULT 1,
    confidence REAL DEFAULT 0.0,
    first_seen DATETIME NOT NULL,
    last_seen DATETIME NOT NULL,
    -- 自动化关联
    is_automated BOOLEAN DEFAULT FALSE,
    automation_id INTEGER,
    -- AI 分析
    ai_analysis TEXT,
    estimated_time_saving INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (automation_id) REFERENCES automations(id)
);

CREATE INDEX idx_patterns_automated ON patterns(is_automated);
CREATE INDEX idx_patterns_support ON patterns(support_count);

-- 知识库表
CREATE TABLE IF NOT EXISTS knowledge_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid TEXT UNIQUE NOT NULL,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    source_type TEXT, -- clip, import, api
    source_url TEXT,
    -- AI 生成
    tags JSON,
    summary TEXT,
    category TEXT,
    -- 向量搜索
    embedding_id TEXT,
    -- 统计
    view_count INTEGER DEFAULT 0,
    access_count INTEGER DEFAULT 0,
    last_accessed DATETIME,
    -- 时间戳
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_knowledge_tags ON knowledge_items(tags);
CREATE INDEX idx_knowledge_category ON knowledge_items(category);
CREATE INDEX idx_knowledge_created ON knowledge_items(created_at);

-- 知识图谱关系表
CREATE TABLE IF NOT EXISTS knowledge_relations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    from_id INTEGER NOT NULL,
    to_id INTEGER NOT NULL,
    relation_type TEXT NOT NULL, -- related, references, similar
    weight REAL DEFAULT 1.0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (from_id) REFERENCES knowledge_items(id),
    FOREIGN KEY (to_id) REFERENCES knowledge_items(id),
    UNIQUE(from_id, to_id, relation_type)
);

CREATE INDEX idx_relations_from ON knowledge_relations(from_id);
CREATE INDEX idx_relations_to ON knowledge_relations(to_id);

-- 自动化表
CREATE TABLE IF NOT EXISTS automations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    -- 触发器
    trigger_type TEXT NOT NULL, -- cron, event, manual
    trigger_config JSON NOT NULL,
    -- 步骤
    steps JSON NOT NULL,
    -- 状态
    enabled BOOLEAN DEFAULT TRUE,
    run_count INTEGER DEFAULT 0,
    -- 时间
    last_run DATETIME,
    next_run DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_automations_enabled ON automations(enabled);
CREATE INDEX idx_automations_trigger ON automations(trigger_type);

-- 执行结果表
CREATE TABLE IF NOT EXISTS execution_results (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid TEXT UNIQUE NOT NULL,
    automation_id INTEGER NOT NULL,
    run_id TEXT NOT NULL,
    status TEXT NOT NULL, -- success, failed, partial
    steps JSON,
    output TEXT,
    error TEXT,
    start_time DATETIME NOT NULL,
    end_time DATETIME NOT NULL,
    duration_ms INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (automation_id) REFERENCES automations(id)
);

CREATE INDEX idx_results_automation ON execution_results(automation_id);
CREATE INDEX idx_results_status ON execution_results(status);
CREATE INDEX idx_results_time ON execution_results(start_time);

-- 会话表
CREATE TABLE IF NOT EXISTS sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid TEXT UNIQUE NOT NULL,
    application TEXT NOT NULL,
    bundle_id TEXT,
    start_time DATETIME NOT NULL,
    end_time DATETIME,
    event_count INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_sessions_time ON sessions(start_time);

-- 配置表
CREATE TABLE IF NOT EXISTS config (
    key TEXT PRIMARY KEY,
    value JSON NOT NULL,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- AI 缓存表
CREATE TABLE IF NOT EXISTS ai_cache (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    prompt_hash TEXT UNIQUE NOT NULL,
    prompt TEXT NOT NULL,
    response TEXT NOT NULL,
    model TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME NOT NULL
);

CREATE INDEX idx_ai_cache_hash ON ai_cache(prompt_hash);
CREATE INDEX idx_ai_cache_expires ON ai_cache(expires_at);
```

### Go 实现

```go
// internal/storage/sqlite.go
package storage

import (
    "database/sql"
    "embed"
    "github.com/mattn/go-sqlite3"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

type SQLiteDB struct {
    db *sql.DB
}

func NewSQLiteDB(dbPath string) (*SQLiteDB, error) {
    db, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        return nil, err
    }

    // 启用 WAL 模式（更好的并发性能）
    if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
        return nil, err
    }

    // 设置连接池
    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(5)
    db.SetConnMaxLifetime(5 * time.Minute)

    // 运行迁移
    if err := runMigrations(db); err != nil {
        return nil, err
    }

    return &SQLiteDB{db: db}, nil
}

func runMigrations(db *sql.DB) error {
    // 创建迁移记录表
    if _, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS schema_migrations (
            version INTEGER PRIMARY KEY,
            applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
        )
    `); err != nil {
        return err
    }

    // 获取当前版本
    var currentVersion int
    db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&currentVersion)

    // 执行迁移
    files, _ := migrationFiles.ReadDir("migrations")
    for _, file := range files {
        // 提取版本号
        version := extractVersion(file.Name())

        if version > currentVersion {
            // 读取迁移文件
            content, _ := migrationFiles.ReadFile("migrations/" + file.Name())

            // 执行迁移
            if _, err := db.Exec(string(content)); err != nil {
                return fmt.Errorf("migration %s failed: %w", file.Name(), err)
            }

            // 记录迁移
            if _, err := db.Exec(
                "INSERT INTO schema_migrations (version) VALUES (?)",
                version,
            ); err != nil {
                return err
            }

            log.Info("Applied migration:", file.Name())
        }
    }

    return nil
}

func extractVersion(filename string) int {
    // 001_init.sql → 1
    var version int
    fmt.Sscanf(filename, "%d_", &version)
    return version
}

func (s *SQLiteDB) Close() error {
    return s.db.Close()
}
```

---

## 事件存储

```go
// internal/storage/event_repository.go
type EventRepository interface {
    Save(event *Event) error
    FindByUUID(uuid string) (*Event, error)
    FindByTimeRange(start, end time.Time) ([]Event, error)
    FindByType(eventType string) ([]Event, error)
    FindRecent(limit int) ([]Event, error)
    DeleteOlderThan(duration time.Duration) (int64, error)
}

type SQLiteEventRepository struct {
    db *sql.DB
}

func NewSQLiteEventRepository(db *sql.DB) *SQLiteEventRepository {
    return &SQLiteEventRepository{db: db}
}

func (r *SQLiteEventRepository) Save(event *Event) error {
    query := `
        INSERT INTO events (uuid, type, timestamp, data, application, bundle_id, window_title, file_path, selection)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
    `

    data, _ := json.Marshal(event.Data)

    _, err := r.db.Exec(query,
        event.ID,
        event.Type,
        event.Timestamp,
        data,
        event.Context.Application,
        event.Context.BundleID,
        event.Context.WindowTitle,
        event.Context.FilePath,
        event.Context.Selection,
    )

    return err
}

func (r *SQLiteEventRepository) FindByTimeRange(start, end time.Time) ([]Event, error) {
    query := `
        SELECT uuid, type, timestamp, data, application, bundle_id, window_title, file_path, selection
        FROM events
        WHERE timestamp BETWEEN ? AND ?
        ORDER BY timestamp ASC
    `

    rows, err := r.db.Query(query, start, end)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var events []Event
    for rows.Next() {
        var event Event
        var dataJSON string

        err := rows.Scan(
            &event.ID,
            &event.Type,
            &event.Timestamp,
            &dataJSON,
            &event.Context.Application,
            &event.Context.BundleID,
            &event.Context.WindowTitle,
            &event.Context.FilePath,
            &event.Context.Selection,
        )

        if err != nil {
            continue
        }

        json.Unmarshal([]byte(dataJSON), &event.Data)
        events = append(events, event)
    }

    return events, nil
}

func (r *SQLiteEventRepository) FindRecent(limit int) ([]Event, error) {
    query := `
        SELECT uuid, type, timestamp, data, application, bundle_id, window_title, file_path, selection
        FROM events
        ORDER BY timestamp DESC
        LIMIT ?
    `

    rows, err := r.db.Query(query, limit)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var events []Event
    for rows.Next() {
        var event Event
        var dataJSON string

        rows.Scan(
            &event.ID,
            &event.Type,
            &event.Timestamp,
            &dataJSON,
            &event.Context.Application,
            &event.Context.BundleID,
            &event.Context.WindowTitle,
            &event.Context.FilePath,
            &event.Context.Selection,
        )

        json.Unmarshal([]byte(dataJSON), &event.Data)
        events = append(events, event)
    }

    return events, nil
}

func (r *SQLiteEventRepository) DeleteOlderThan(duration time.Duration) (int64, error) {
    cutoff := time.Now().Add(-duration)

    result, err := r.db.Exec(
        "DELETE FROM events WHERE timestamp < ?",
        cutoff,
    )

    if err != nil {
        return 0, err
    }

    return result.RowsAffected()
}
```

---

## BBolt 键值存储

### 缓存实现

```go
// internal/storage/bbolt.go
import bbolt "go.etcd.io/bbolt"

type BBoltStore struct {
    db *bbolt.DB
}

func NewBBoltStore(dbPath string) (*BBoltStore, error) {
    db, err := bbolt.Open(dbPath, 0600, &bbolt.Options{Timeout: 1 * time.Second})
    if err != nil {
        return nil, err
    }

    // 创建 buckets
    err = db.Update(func(tx *bbolt.Tx) error {
        _, err := tx.CreateBucketIfNotExists([]byte("cache"))
        if err != nil {
            return err
        }

        _, err = tx.CreateBucketIfNotExists([]byte("session"))
        if err != nil {
            return err
        }

        _, err = tx.CreateBucketIfNotExists([]byte("temp"))
        return err
    })

    if err != nil {
        db.Close()
        return nil, err
    }

    return &BBoltStore{db: db}, nil
}

func (s *BBoltStore) Set(bucket, key string, value []byte, ttl time.Duration) error {
    return s.db.Update(func(tx *bbolt.Tx) error {
        b := tx.Bucket([]byte(bucket))
        if b == nil {
            return fmt.Errorf("bucket not found: %s", bucket)
        }

        // 创建带 TTL 的条目
        entry := struct {
            Value   []byte
            Expires time.Time
        }{
            Value:   value,
            Expires: time.Now().Add(ttl),
        }

        data, err := json.Marshal(entry)
        if err != nil {
            return err
        }

        return b.Put([]byte(key), data)
    })
}

func (s *BBoltStore) Get(bucket, key string) ([]byte, error) {
    var value []byte

    err := s.db.View(func(tx *bbolt.Tx) error {
        b := tx.Bucket([]byte(bucket))
        if b == nil {
            return fmt.Errorf("bucket not found: %s", bucket)
        }

        data := b.Get([]byte(key))
        if data == nil {
            return nil
        }

        // 解析条目
        var entry struct {
            Value   []byte
            Expires time.Time
        }

        if err := json.Unmarshal(data, &entry); err != nil {
            return err
        }

        // 检查过期
        if time.Now().After(entry.Expires) {
            return fmt.Errorf("expired")
        }

        value = entry.Value
        return nil
    })

    return value, err
}

func (s *BBoltStore) Delete(bucket, key string) error {
    return s.db.Update(func(tx *bbolt.Tx) error {
        b := tx.Bucket([]byte(bucket))
        if b == nil {
            return nil
        }

        return b.Delete([]byte(key))
    })
}

func (s *BBoltStore) Close() error {
    return s.db.Close()
}
```

---

## 向量数据库

### Chromem-go 集成

```go
// internal/storage/vector.go
import "github.com/philippgille/chromem-go"

type VectorStore struct {
    collection *chromem.Collection
    aiService  *ai.AIService
}

func NewVectorStore(aiService *ai.AIService) (*VectorStore, error) {
    // 创建 embedding 函数
    embedFunc := func(ctx context.Context, text string) ([]float32, error) {
        return aiService.Embed(text)
    }

    // 创建集合
    collection, err := chromem.NewCollection(
        chromem.WithEmbeddingFunc(embedFunc),
        chromem.WithName("knowledge"),
    )

    if err != nil {
        return nil, err
    }

    return &VectorStore{
        collection: collection,
        aiService:  aiService,
    }, nil
}

func (v *VectorStore) AddDocument(id, content string, metadata map[string]string) error {
    return v.collection.AddDocument(
        context.Background(),
        chromem.Document{
            ID:       id,
            Content:  content,
            Metadata: metadata,
        },
    )
}

func (v *VectorStore) Search(query string, topK int) ([]chromem.Result, error) {
    results, err := v.collection.Query(
        context.Background(),
        query,
        topK,
        nil, // 不需要额外的过滤条件
    )

    if err != nil {
        return nil, err
    }

    return results, nil
}

func (v *VectorStore) Delete(id string) error {
    return v.collection.Delete(context.Background(), id)
}
```

---

## 知识库存储

```go
// internal/storage/knowledge_repository.go
type KnowledgeRepository interface {
    Save(item *KnowledgeItem) error
    FindByUUID(uuid string) (*KnowledgeItem, error)
    FindByCategory(category string) ([]KnowledgeItem, error)
    FindByTag(tag string) ([]KnowledgeItem, error)
    Search(query string, limit int) ([]KnowledgeItem, error)
    Update(id int, item *KnowledgeItem) error
    Delete(id int) error
    FindRelated(id int, limit int) ([]KnowledgeItem, error)
}

type KnowledgeItem struct {
    ID          int
    UUID        string
    Title       string
    Content     string
    SourceType  string
    SourceURL   string
    Tags        []string
    Summary     string
    Category    string
    ViewCount   int
    AccessCount int
    LastAccessed *time.Time
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

type SQLiteKnowledgeRepository struct {
    db          *sql.DB
    vectorStore *VectorStore
}

func NewSQLiteKnowledgeRepository(db *sql.DB, vs *VectorStore) *SQLiteKnowledgeRepository {
    return &SQLiteKnowledgeRepository{
        db:          db,
        vectorStore: vs,
    }
}

func (r *SQLiteKnowledgeRepository) Save(item *KnowledgeItem) error {
    query := `
        INSERT INTO knowledge_items
        (uuid, title, content, source_type, source_url, tags, summary, category)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?)
    `

    tags, _ := json.Marshal(item.Tags)

    result, err := r.db.Exec(query,
        item.UUID,
        item.Title,
        item.Content,
        item.SourceType,
        item.SourceURL,
        tags,
        item.Summary,
        item.Category,
    )

    if err != nil {
        return err
    }

    // 获取 ID
    id, _ := result.LastInsertId()
    item.ID = int(id)

    // 添加到向量数据库
    if r.vectorStore != nil {
        r.vectorStore.AddDocument(
            item.UUID,
            item.Content,
            map[string]string{
                "title":    item.Title,
                "category": item.Category,
            },
        )
    }

    return nil
}

func (r *SQLiteKnowledgeRepository) Search(query string, limit int) ([]KnowledgeItem, error) {
    // 先用向量搜索
    if r.vectorStore != nil {
        results, err := r.vectorStore.Search(query, limit)
        if err == nil {
            // 根据 UUID 查询完整信息
            var items []KnowledgeItem
            for _, result := range results {
                if item, err := r.FindByUUID(result.ID); err == nil {
                    items = append(items, *item)
                }
            }
            return items, nil
        }
    }

    // 降级到 LIKE 搜索
    query = "%" + query + "%"
    sqlQuery := `
        SELECT id, uuid, title, content, source_type, source_url, tags, summary, category,
               view_count, access_count, last_accessed, created_at, updated_at
        FROM knowledge_items
        WHERE title LIKE ? OR content LIKE ? OR summary LIKE ?
        ORDER BY created_at DESC
        LIMIT ?
    `

    return r.executeQuery(sqlQuery, query, query, query, limit)
}

func (r *SQLiteKnowledgeRepository) FindRelated(id int, limit int) ([]KnowledgeItem, error) {
    // 查找相关的知识项（通过关系表）
    query := `
        SELECT ki.id, ki.uuid, ki.title, ki.content, ki.source_type, ki.source_url,
               ki.tags, ki.summary, ki.category, ki.view_count, ki.access_count,
               ki.last_accessed, ki.created_at, ki.updated_at
        FROM knowledge_items ki
        JOIN knowledge_relations kr ON ki.id = kr.to_id
        WHERE kr.from_id = ?
        ORDER BY kr.weight DESC
        LIMIT ?
    `

    return r.executeQuery(query, id, limit)
}
```

---

## 自动化存储

```go
// internal/storage/automation_repository.go
type AutomationRepository interface {
    Save(script *AutomationScript) error
    FindByUUID(uuid string) (*AutomationScript, error)
    FindAll() ([]AutomationScript, error)
    FindAllEnabled() ([]AutomationScript, error)
    FindByTriggerType(triggerType string) ([]AutomationScript, error)
    Update(script *AutomationScript) error
    Delete(id int) error
}

type SQLiteAutomationRepository struct {
    db *sql.DB
}

func NewSQLiteAutomationRepository(db *sql.DB) *SQLiteAutomationRepository {
    return &SQLiteAutomationRepository{db: db}
}

func (r *SQLiteAutomationRepository) Save(script *AutomationScript) error {
    query := `
        INSERT INTO automations
        (uuid, name, description, trigger_type, trigger_config, steps, enabled)
        VALUES (?, ?, ?, ?, ?, ?, ?)
    `

    triggerConfig, _ := json.Marshal(script.Trigger.Config)
    steps, _ := json.Marshal(script.Steps)

    result, err := r.db.Exec(query,
        script.ID,
        script.Name,
        script.Description,
        script.Trigger.Type,
        triggerConfig,
        steps,
        script.Enabled,
    )

    if err != nil {
        return err
    }

    id, _ := result.LastInsertId()
    // 保存到结构
    return nil
}

func (r *SQLiteAutomationRepository) FindAllEnabled() ([]AutomationScript, error) {
    query := `
        SELECT id, uuid, name, description, trigger_type, trigger_config, steps,
               enabled, run_count, last_run, next_run, created_at, updated_at
        FROM automations
        WHERE enabled = TRUE
        ORDER BY created_at DESC
    `

    rows, err := r.db.Query(query)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var scripts []AutomationScript
    for rows.Next() {
        var script AutomationScript
        var triggerConfigJSON, stepsJSON string

        err := rows.Scan(
            &script.ID,
            &script.UUID,
            &script.Name,
            &script.Description,
            &script.Trigger.Type,
            &triggerConfigJSON,
            &stepsJSON,
            &script.Enabled,
            &script.RunCount,
            &script.LastRun,
            &script.NextRun,
            &script.CreatedAt,
            &script.UpdatedAt,
        )

        if err != nil {
            continue
        }

        json.Unmarshal([]byte(triggerConfigJSON), &script.Trigger.Config)
        json.Unmarshal([]byte(stepsJSON), &script.Steps)

        scripts = append(scripts, script)
    }

    return scripts, nil
}
```

---

## 数据库维护

### 清理任务

```go
// internal/storage/maintenance.go
type Maintenance struct {
    db *sql.DB
}

func NewMaintenance(db *sql.DB) *Maintenance {
    return &Maintenance{db: db}
}

// 清理旧事件
func (m *Maintenance) CleanupOldEvents(retention time.Duration) (int64, error) {
    cutoff := time.Now().Add(-retention)

    result, err := m.db.Exec(
        "DELETE FROM events WHERE timestamp < ?",
        cutoff,
    )

    if err != nil {
        return 0, err
    }

    return result.RowsAffected()
}

// 清理过期缓存
func (m *Maintenance) CleanupExpiredCache() (int64, error) {
    result, err := m.db.Exec(
        "DELETE FROM ai_cache WHERE expires_at < ?",
        time.Now(),
    )

    if err != nil {
        return 0, err
    }

    return result.RowsAffected()
}

// 优化数据库
func (m *Maintenance) Optimize() error {
    if _, err := m.db.Exec("VACUUM"); err != nil {
        return err
    }

    if _, err := m.db.Exec("ANALYZE"); err != nil {
        return err
    }

    return nil
}

// 备份数据库
func (m *Maintenance) Backup(backupPath string) error {
    // 使用 SQLite 的备份 API
    // 或简单地复制文件

    return fmt.Errorf("not implemented")
}

// 统计信息
func (m *Maintenance) Stats() (*DatabaseStats, error) {
    stats := &DatabaseStats{}

    // 事件数量
    m.db.QueryRow("SELECT COUNT(*) FROM events").Scan(&stats.EventCount)

    // 知识库数量
    m.db.QueryRow("SELECT COUNT(*) FROM knowledge_items").Scan(&stats.KnowledgeCount)

    // 自动化数量
    m.db.QueryRow("SELECT COUNT(*) FROM automations WHERE enabled = TRUE").Scan(&stats.AutomationCount)

    // 数据库大小
    var size int64
    m.db.QueryRow("SELECT page_count * page_size as size FROM pragma_page_count(), pragma_page_size()").Scan(&size)
    stats.DatabaseSize = size

    return stats, nil
}

type DatabaseStats struct {
    EventCount       int64
    KnowledgeCount   int64
    AutomationCount  int64
    DatabaseSize     int64
}
```

---

## 性能优化

### 批量插入

```go
func (r *SQLiteEventRepository) SaveBatch(events []Event) error {
    tx, err := r.db.Begin()
    if err != nil {
        return err
    }

    defer tx.Rollback()

    stmt, err := tx.Prepare(`
        INSERT INTO events (uuid, type, timestamp, data, application, bundle_id, window_title, file_path, selection)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
    `)

    if err != nil {
        return err
    }

    defer stmt.Close()

    for _, event := range events {
        data, _ := json.Marshal(event.Data)

        _, err := stmt.Exec(
            event.ID,
            event.Type,
            event.Timestamp,
            data,
            event.Context.Application,
            event.Context.BundleID,
            event.Context.WindowTitle,
            event.Context.FilePath,
            event.Context.Selection,
        )

        if err != nil {
            return err
        }
    }

    return tx.Commit()
}
```

### 连接池配置

```go
func configureConnectionPool(db *sql.DB) {
    // 最大打开连接数
    db.SetMaxOpenConns(25)

    // 最大空闲连接数
    db.SetMaxIdleConns(5)

    // 连接最大生命周期
    db.SetConnMaxLifetime(5 * time.Minute)

    // 连接最大空闲时间
    db.SetConnMaxIdleTime(1 * time.Minute)
}
```

---

## 使用示例

```go
// 初始化存储
db, _ := NewSQLiteDB("/path/to/flowmind.db")
bbolt, _ := NewBBoltStore("/path/to/cache.db")
vectorStore, _ := NewVectorStore(aiService)

// 创建仓库
eventRepo := NewSQLiteEventRepository(db)
knowledgeRepo := NewSQLiteKnowledgeRepository(db, vectorStore)
automationRepo := NewSQLiteAutomationRepository(db)

// 保存事件
event := &Event{
    ID:        generateUUID(),
    Type:      "keyboard",
    Timestamp: time.Now(),
    Data:      map[string]interface{}{"keycode": 46},
    Context:   &EventContext{Application: "VS Code"},
}
eventRepo.Save(event)

// 查询事件
events, _ := eventRepo.FindByTimeRange(
    time.Now().Add(-24*time.Hour),
    time.Now(),
)

// 添加知识
item := &KnowledgeItem{
    UUID:    generateUUID(),
    Title:   "Rust 异步编程",
    Content: "...",
    Tags:    []string{"rust", "async"},
}
knowledgeRepo.Save(item)

// 语义搜索
results, _ := knowledgeRepo.Search("异步编程最佳实践", 10)
```

---

**相关文档**：
- [系统架构](./01-system-architecture.md)
- [数据库设计](../design/01-database-design.md)
- [API 设计](../design/02-api-design.md)
