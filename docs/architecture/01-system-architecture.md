# FlowMind 系统架构（Wails最佳实践版）

> 基于 Wails v2 + React 19 + TailwindCSS 的现代化桌面应用架构

---

## 架构原则

基于 Wails 官方建议和社区最佳实践：

1. **状态在 Go，前端只反映状态** - Wails 维护者 Lea Anthony 核心理念
2. **清晰分层** - Frontend → App → Service → Domain → Infrastructure
3. **事件驱动** - 异步通信，松耦合
4. **依赖注入** - 便于测试和扩展
5. **符合 Go 标准布局** - Standard Go Project Layout

---

## 整体架构图

```
┌─────────────────────────────────────────────────────────┐
│                  React 19 前端层                         │
│  ┌────────────────────────────────────────────────────┐ │
│  │  UI 组件 + Zustand 状态管理                         │ │
│  │  - Dashboard (仪表板)                               │ │
│  │  - GlobalPanel (全局面板)                           │ │
│  │  - AutomationEditor (自动化编辑器)                  │ │
│  │  - KnowledgeGraph (知识图谱)                        │ │
│  └────────────────────────────────────────────────────┘ │
│                          ▲                              │
│                          │ Wails Bindings               │
│                          │ (方法调用 + 事件推送)         │
│                          ▼                              │
│  ┌────────────────────────────────────────────────────┐ │
│  │  App 层 (internal/app/)                             │ │
│  │  - 前后端通信桥梁                                    │ │
│  │  - 方法导出 (Methods)                               │ │
│  │  - 事件转发 (Events)                                │ │
│  └────────────────────────────────────────────────────┘ │
│                          ▼                              │
│  ┌────────────────────────────────────────────────────┐ │
│  │  服务层 (internal/services/)                        │ │
│  │  - MonitorService (监控服务)                        │ │
│  │  - AnalyzerService (分析服务)                       │ │
│  │  - AIService (AI 服务)                              │ │
│  │  - AutomationService (自动化服务)                   │ │
│  │  - KnowledgeService (知识服务)                      │ │
│  └────────────────────────────────────────────────────┘ │
│                          ▼                              │
│  ┌────────────────────────────────────────────────────┐ │
│  │  领域层 (internal/domain/)                          │ │
│  │                                                     │ │
│  │  ┌───────────────────────────────────────────────┐ │ │
│  │  │ 监控领域 (monitor)                             │ │ │
│  │  │  - KeyboardMonitor (键盘)                      │ │ │
│  │  │  - ClipboardMonitor (剪贴板)                   │ │ │
│  │  │  - ApplicationMonitor (应用)                   │ │ │
│  │  └───────────────────────────────────────────────┘ │ │
│  │                                                     │ │
│  │  ┌───────────────────────────────────────────────┐ │ │
│  │  │ 分析领域 (analyzer)                            │ │ │
│  │  │  - PatternMiner (模式挖掘)                     │ │ │
│  │  │  - SequenceAnalyzer (序列分析)                 │ │ │
│  │  └───────────────────────────────────────────────┘ │ │
│  │                                                     │ │
│  │  ┌───────────────────────────────────────────────┐ │ │
│  │  │ AI 领域 (ai)                                   │ │ │
│  │  │  - ClaudeClient (Claude API)                  │ │ │
│  │  │  - OllamaClient (本地模型)                     │ │ │
│  │  │  - PromptEngine (提示词引擎)                   │ │ │
│  │  └───────────────────────────────────────────────┘ │ │
│  │                                                     │ │
│  │  ┌───────────────────────────────────────────────┐ │ │
│  │  │ 自动化领域 (automation)                        │ │ │
│  │  │  - ScriptGenerator (脚本生成)                  │ │ │
│  │  │  - Scheduler (任务调度)                        │ │ │
│  │  │  - Sandbox (沙箱执行)                          │ │ │
│  │  └───────────────────────────────────────────────┘ │ │
│  │                                                     │ │
│  │  ┌───────────────────────────────────────────────┐ │ │
│  │  │ 知识管理领域 (knowledge)                       │ │ │
│  │  │  - Clipper (剪藏)                              │ │ │
│  │  │  - Tagger (标签生成)                           │ │ │
│  │  │  - SemanticSearch (语义搜索)                   │ │ │
│  │  └───────────────────────────────────────────────┘ │ │
│  └────────────────────────────────────────────────────┘ │
│                          ▼                              │
│  ┌────────────────────────────────────────────────────┐ │
│  │  基础设施层 (internal/infrastructure/)              │ │
│  │                                                     │ │
│  │  ┌───────────────────────────────────────────────┐ │ │
│  │  │ 存储层 (storage)                               │ │ │
│  │  │  - SQLite (事件日志、配置)                      │ │ │
│  │  │  - BBolt (键值缓存)                            │ │ │
│  │  │  - Chromem-go (向量数据库)                     │ │ │
│  │  └───────────────────────────────────────────────┘ │ │
│  │                                                     │ │
│  │  ┌───────────────────────────────────────────────┐ │ │
│  │  │ 平台层 (platform)                              │ │ │
│  │  │  - Darwin (macOS 特定实现)                     │ │ │
│  │  │    • CGEventTap (事件捕获)                     │ │ │
│  │  │    • NSPasteboard (剪贴板)                     │ │ │
│  │  │    • NSWorkspace (应用管理)                    │ │ │
│  │  └───────────────────────────────────────────────┘ │ │
│  └────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘
```

---

## 项目结构详解

```
flowmind/
├── main.go                 # Wails 入口
│
├── internal/                       # 私有应用代码
│   ├── app/                        # App 层（前后端桥梁）
│   │   ├── app.go                  # 主 App 结构
│   │   ├── events.go               # 事件发射
│   │   ├── methods.go              # 导出方法
│   │   └── startup.go              # 初始化逻辑
│   │
│   ├── domain/                     # 领域层（核心业务）
│   │   ├── events/                 # 领域事件
│   │   │   ├── types.go
│   │   │   └── bus.go
│   │   │
│   │   ├── monitor/                # 监控领域
│   │   ├── analyzer/               # 分析领域
│   │   ├── ai/                     # AI 领域
│   │   ├── automation/             # 自动化领域
│   │   ├── knowledge/              # 知识管理领域
│   │   └── models/                 # 领域模型
│   │
│   ├── infrastructure/             # 基础设施层
│   │   ├── config/                 # 配置管理
│   │   ├── storage/                # 存储实现
│   │   ├── repositories/           # 仓储模式
│   │   ├── notify/                 # 通知系统
│   │   ├── logger/                 # 日志系统
│   │   └── platform/               # 平台相关代码
│   │       ├── darwin/             # macOS 实现
│   │       └── interface.go        # 平台接口
│   │
│   └── services/                   # 服务层（业务编排）
│       ├── monitor_service.go
│       ├── analyzer_service.go
│       ├── ai_service.go
│       ├── automation_service.go
│       └── knowledge_service.go
│
├── frontend/                       # React 19 前端
│   ├── src/
│   │   ├── main.tsx                # React 入口
│   │   ├── App.tsx                 # 主组件
│   │   ├── components/             # UI 组件
│   │   ├── hooks/                  # React Hooks
│   │   ├── stores/                 # Zustand 状态管理
│   │   ├── lib/                    # 工具库
│   │   ├── wailsjs/                # Wails 自动生成
│   │   └── styles/                 # 全局样式
│   ├── public/                     # 静态资源
│   ├── index.html
│   ├── package.json
│   ├── vite.config.js
│   ├── tailwind.config.js
│   └── postcss.config.js
│
├── pkg/                            # 公共库
│   └── events/                     # 事件系统（可复用）
│
├── build/                          # 构建资源
│   ├── appicon.png
│   ├── darwin/
│   └── windows/
│
├── configs/                        # 配置文件
│   ├── default.yaml
│   └── development.yaml
│
├── wails.json                      # Wails 配置
├── go.mod
├── go.sum
└── README.md
```

---

## 分层职责详解

### 1. App 层 (`internal/app/`)

**职责**：Wails 框架集成，前后端通信桥梁

```go
// internal/app/app.go
package app

import (
    "context"
    "github.com/chenyang-zz/internal/services"
    "github.com/chenyang-zz/internal/infrastructure/config"
    "github.com/chenyang-zz/internal/domain/events"
)

type App struct {
    ctx      context.Context
    config   *config.Config
    bus      *events.Bus

    // 服务（通过依赖注入）
    monitorSvc  *services.MonitorService
    analyzerSvc *services.AnalyzerService
    aiSvc       *services.AIService
    autoSvc     *services.AutomationService
    knowSvc     *services.KnowledgeService
}

// 导出方法（前端可调用）
func (a *App) GetDashboardData() (*DashboardData, error) {
    return a.analyzerSvc.GetDashboardData(context.Background())
}

func (a *App) CreateAutomation(req *CreateAutomationRequest) (*Automation, error) {
    return a.autoSvc.CreateAutomation(context.Background(), req)
}
```

**原则**：
- 只负责接收前端请求和返回响应
- 不包含业务逻辑，委托给 Service 层
- 通过 `runtime.EventsEmit` 推送事件到前端

---

### 2. Service 层 (`internal/services/`)

**职责**：业务流程编排，协调多个 Domain

```go
// internal/services/analyzer_service.go
package services

import (
    "context"
    "github.com/chenyang-zz/internal/domain"
)

type AnalyzerService struct {
    monitor     domain.Monitor
    patternMiner domain.PatternMiner
    ai          domain.AIService
    repo        repositories.PatternRepository
    eventBus    *events.Bus
}

func (s *AnalyzerService) AnalyzeEvents(ctx context.Context) error {
    // 1. 从监控器获取事件
    events := s.monitor.GetRecentEvents(ctx)

    // 2. 模式识别
    patterns, err := s.patternMiner.MinePatterns(ctx, events)
    if err != nil {
        return err
    }

    // 3. AI 过滤
    validPatterns, err := s.ai.FilterPatterns(ctx, patterns)
    if err != nil {
        return err
    }

    // 4. 保存并发布事件
    for _, p := range validPatterns {
        s.repo.Save(ctx, p)
        s.eventBus.Publish("pattern:discovered", p)
    }

    return nil
}
```

**原则**：
- 编排多个 Domain 协作
- 实现应用级用例
- 处理事务和错误

---

### 3. Domain 层 (`internal/domain/`)

**职责**：核心业务逻辑，领域模型

```go
// internal/domain/monitor/monitor.go
package monitor

type Monitor interface {
    Start(ctx context.Context) error
    Stop() error
    Events() <-chan Event
}

// internal/domain/monitor/monitor_impl.go
type MonitorImpl struct {
    keyboard    KeyboardMonitor
    clipboard   ClipboardMonitor
    application ApplicationMonitor
    eventBus    *events.Bus
}

func (m *MonitorImpl) Start(ctx context.Context) error {
    // 启动各个监控器
    go m.keyboard.Watch(ctx)
    go m.clipboard.Watch(ctx)
    go m.application.Watch(ctx)
    return nil
}
```

**原则**：
- 定义接口（便于测试和替换实现）
- 包含核心业务规则
- 不依赖基础设施（通过接口解耦）

---

### 4. Infrastructure 层 (`internal/infrastructure/`)

**职责**：技术实现，外部系统交互

```go
// internal/infrastructure/storage/sqlite.go
package storage

import (
    "context"
    "database/sql"
    "github.com/chenyang-zz/internal/domain/models"
)

type SQLiteRepository struct {
    db *sql.DB
}

func (r *SQLiteRepository) SaveEvent(ctx context.Context, e *models.Event) error {
    _, err := r.db.ExecContext(ctx,
        "INSERT INTO events (type, timestamp, data) VALUES (?, ?, ?)",
        e.Type, e.Timestamp, e.Data)
    return err
}

// internal/infrastructure/platform/darwin/workspace.go
package darwin

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>
*/
import "C"

func (w *DarwinWorkspace) GetActiveApp() (string, error) {
    // CGO 调用 macOS API
    appName := C.CString(Cstring(NSWorkspace.sharedWorkspace.frontmostApplication.localizedName))
    return GoString(appName), nil
}
```

**原则**：
- 实现 Domain 定义的接口
- 隔离平台特定代码
- 处理持久化和外部 API

---

## 前端架构（React 19 + TailwindCSS）

### 技术栈

- **React 19**：使用最新特性（Compiler、Actions、useOptimistic）
- **Vite 5**：快速开发和构建
- **TailwindCSS 4**：实用优先 CSS 框架
- **Zustand**：轻量级状态管理
- **组件库**：待定

### 状态管理原则

**核心思想**：前端只反映 Go 后端的状态

```jsx
// frontend/src/stores/eventStore.ts
import { create } from 'zustand';
import { EventsOn, EventsOff } from '../wailsjs/runtime';

interface EventStore {
  events: Event[];
  isSubscribed: boolean;

  subscribe: () => void;
  addEvent: (event: Event) => void;
}

export const useEventStore = create<EventStore>((set, get) => ({
  events: [],
  isSubscribed: false,

  subscribe: () => {
    if (get().isSubscribed) return;

    EventsOn('event:new', (event) => {
      set((state) => ({
        events: [...state.events, event],
      }));
    });

    set({ isSubscribed: true });
  },

  addEvent: (event) => {
    set((state) => ({
      events: [...state.events, event],
    }));
  },
}));
```

### 组件设计示例

```jsx
// frontend/src/components/Dashboard/index.jsx
import React from 'react';
import { useEventStore } from '../../stores/eventStore';
import { useWailsEvent } from '../../hooks/useWailsEvent';

export function Dashboard() {
  const { events, subscribe } = useEventStore();

  React.useEffect(() => {
    subscribe();
  }, [subscribe]);

  // 监听后端事件
  useWailsEvent('pattern:discovered', (pattern) => {
    console.log('New pattern:', pattern);
  });

  return (
    <div className="min-h-screen bg-gray-50 p-6">
      <h1 className="text-3xl font-bold">FlowMind Dashboard</h1>
      <EventList events={events} />
    </div>
  );
}
```

### React 19 最佳实践

#### 1. 使用 React Compiler（无需手动 memo）

```jsx
// React Compiler 会自动优化，无需 useMemo, useCallback
export function Dashboard() {
  const { events, subscribe, fetchEvents } = useEventStore();

  React.useEffect(() => {
    subscribe();
    fetchEvents();
  }, []);

  return <div>...</div>;
}
```

#### 2. 使用 Actions 表单处理

```jsx
import { useActionState } from 'react';

export function AutomationEditor() {
  const [state, formAction, isPending] = useActionState(
    async (prevState, formData) => {
      const result = await CreateAutomation(Object.fromEntries(formData));
      return { success: true, automation: result };
    },
    { success: false, automation: null }
  );

  return (
    <form action={formAction} className="space-y-4">
      <input name="name" type="text" />
      <button type="submit" disabled={isPending}>
        {isPending ? 'Creating...' : 'Create'}
      </button>
    </form>
  );
}
```

#### 3. 使用 useOptimistic 乐观更新

```jsx
import { useOptimistic } from 'react';

export function PatternList({ patterns, onToggleAutomation }) {
  const [optimisticPatterns, setOptimisticPatterns] = useOptimistic(
    patterns,
    (state, newPattern) => {
      return state.map(p =>
        p.id === newPattern.id ? newPattern : p
      );
    }
  );

  return (
    <ul>
      {optimisticPatterns.map(pattern => (
        <li key={pattern.id}>
          <button onClick={() => onToggleAutomation(pattern)}>
            {pattern.isAutomated ? '✓' : '○'} {pattern.name}
          </button>
        </li>
      ))}
    </ul>
  );
}
```

### TailwindCSS 配置

```javascript
// frontend/tailwind.config.js
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        primary: {
          50: '#f0f9ff',
          500: '#0ea5e9',
          600: '#0284c7',
          700: '#0369a1',
        }
      },
      animation: {
        'fade-in': 'fadeIn 0.2s ease-out',
        'slide-up': 'slideUp 0.3s ease-out',
      },
      keyframes: {
        fadeIn: {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' },
        },
        slideUp: {
          '0%': { transform: 'translateY(10px)', opacity: '0' },
          '100%': { transform: 'translateY(0)', opacity: '1' },
        },
      },
    },
  },
  plugins: [],
}
```

### Wails 事件 Hook

```typescript
// frontend/src/hooks/useWailsEvent.ts
import { useEffect } from 'react';
import { EventsOn, EventsOff } from '../wailsjs/runtime';

export function useWailsEvent(eventName: string, handler: (data: any) => void) {
  useEffect(() => {
    EventsOn(eventName, handler);

    return () => {
      EventsOff(eventName);
    };
  }, [eventName, handler]);
}
```

---

## 数据流设计

### 1. 前端 → 后端流程

```
Frontend Component
    ↓ (调用 Wails 生成的方法)
App Layer (app.go)
    ↓ (委托给)
Service Layer (services/)
    ↓ (协调)
Domain Layer (domain/)
    ↓ (使用)
Infrastructure Layer (infrastructure/)
```

**示例**：
```jsx
// 前端
const automation = await CreateAutomation({ name: "Test" });

// App 层
func (a *App) CreateAutomation(req *CreateAutomationRequest) (*Automation, error) {
    return a.autoSvc.CreateAutomation(context.Background(), req)
}

// Service 层
func (s *AutomationService) CreateAutomation(ctx context.Context, req *CreateAutomationRequest) (*Automation, error) {
    // 业务逻辑编排
    script := s.generator.Generate(ctx, req)
    return s.repo.Save(ctx, script)
}
```

### 2. 后端 → 前端事件流

```
Domain Layer (发布事件)
    ↓
Event Bus (events/bus.go)
    ↓
App Layer (订阅并转发)
    ↓ (runtime.EventsEmit)
Frontend (EventsOn 监听)
    ↓
UI Update
```

**示例**：
```go
// Domain 层
bus.Publish("pattern:discovered", pattern)

// App 层
go func() {
    for event := range eventChan {
        runtime.EventsEmit(a.ctx, "pattern:discovered", event)
    }
}()

// 前端
useWailsEvent('pattern:discovered', (pattern) => {
  toast.success(`发现新模式: ${pattern.name}`);
});
```

---

## Wails 配置

### wails.json

```json
{
  "$schema": "https://wails.io/schemas/config.v2.json",
  "name": "FlowMind",
  "outputfilename": "flowmind",
  "frontend:install": "npm install",
  "frontend:build": "npm run build",
  "frontend:dev:watcher": "npm run dev",
  "frontend:dev:serverUrl": "auto",
  "author": {
    "name": "SheepZhao",
    "email": "your@email.com"
  },
  "info": {
    "companyName": "FlowMind",
    "productName": "FlowMind",
    "productVersion": "1.0.0",
    "copyright": "Copyright 2025",
    "comments": "AI Workflow Intelligence"
  },
  "wailsjsdir": "./frontend",
  "version": "2",
  "outputType": "desktop"
}
```

### 前端配置

#### package.json
```json
{
  "name": "flowmind-frontend",
  "private": true,
  "version": "1.0.0",
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "vite build",
    "preview": "vite preview"
  },
  "dependencies": {
    "react": "^19.0.0",
    "react-dom": "^19.0.0",
    "zustand": "^5.0.0"
  },
  "devDependencies": {
    "@vitejs/plugin-react": "^4.3.0",
    "autoprefixer": "^10.4.20",
    "postcss": "^8.4.47",
    "tailwindcss": "^4.0.0",
    "vite": "^5.4.0"
  }
}
```

#### vite.config.js
```javascript
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
  },
  build: {
    outDir: 'dist',
    sourcemap: true,
  },
});
```

---

## 构建系统

### Makefile

```makefile
.PHONY: dev build clean test

dev:
	wails dev

build:
	wails build

clean:
	rm -rf frontend/dist
	rm -rf build/bin

test:
	go test ./...
	go test ./frontend/...

run: build
	open build/bin/FlowMind.app

deps:
	go mod download
	cd frontend && npm install
```

---

## 配置管理

```yaml
# configs/default.yaml
application:
  name: "FlowMind"
  version: "1.0.0"
  log_level: "info"

monitor:
  enabled_monitors:
    - keyboard
    - clipboard
    - application
  sample_rate: "100ms"

ai:
  provider: "claude"
  claude_api_key: "${CLAUDE_API_KEY}"
  ollama_url: "http://localhost:11434"
  cache_ttl: "1h"

automation:
  max_execution_time: "5m"
  allowed_paths:
    - "/tmp"
    - "${HOME}/Documents"

storage:
  sqlite_path: "${HOME}/.flowmind/flowmind.db"
  bolt_path: "${HOME}/.flowmind/cache.db"
  vector_path: "${HOME}/.flowmind/vectors"
```

---

## 依赖注入

使用 Wire 进行依赖注入：

```go
// internal/app/wire.go
//go:build wireinject
// +build wireinject

package app

import "github.com/google/wire"

func InitializeApp(cfg *config.Config) (*App, error) {
    wire.Build(
        // Infrastructure
        infrastructure.NewSQLiteDB,
        infrastructure.NewVectorDB,
        infrastructure.NewNotifier,

        // Repositories
        repositories.NewEventRepository,
        repositories.NewPatternRepository,

        // Domain
        domain.NewMonitor,
        domain.NewAnalyzer,
        domain.NewAIService,

        // Services
        services.NewMonitorService,
        services.NewAnalyzerService,

        // App
        NewApp,
    )
    return &App{}, nil
}
```

---

## 开发工作流

### 添加新功能

```bash
# 1. 在 domain 层定义接口
# internal/domain/myfeature/feature.go

# 2. 在 infrastructure 层实现
# internal/infrastructure/platform/darwin/feature_impl.go

# 3. 在 service 层编排
# internal/services/myfeature_service.go

# 4. 在 app 层暴露方法
# internal/app/methods.go

# 5. 生成 Wails 绑定
wails generate module

# 6. 前端调用
# frontend/src/components/MyFeature.jsx
```

---

## 性能优化

### 1. 批处理事件

```go
func (r *SQLiteRepository) BatchInsert(events []*Event) error {
    tx, _ := r.db.Begin()
    stmt, _ := tx.Prepare("INSERT INTO events ...")
    defer stmt.Close()

    for _, e := range events {
        stmt.Exec(e.Type, e.Timestamp, e.Data)
    }

    return tx.Commit()
}
```

### 2. 缓存 AI 响应

```go
type TTLCache struct {
    cache map[string]*cacheItem
    ttl   time.Duration
}

func (c *TTLCache) Get(key string) (interface{}, bool) {
    item, ok := c.cache[key]
    if !ok || time.Since(item.CreatedAt) > c.ttl {
        return nil, false
    }
    return item.Value, true
}
```

### 3. React 19 性能建议

- **信任 Compiler**：不需要手动 `useMemo`、`useCallback`
- **使用 Suspense**：懒加载组件
- **Transitions**：标记非紧急更新

```jsx
import { lazy, Suspense, useTransition } from 'react';

const KnowledgeGraph = lazy(() => import('./KnowledgeGraph'));

function App() {
  const [isPending, startTransition] = useTransition();

  return (
    <Suspense fallback={<Loading />}>
      <KnowledgeGraph />
    </Suspense>
  );
}
```

---

## 安全考虑

### 1. 沙箱执行

```go
type Sandbox struct {
    maxMemory    int64
    maxCPUTime   time.Duration
    allowedPaths []string
}

func (s *Sandbox) Validate(script *Script) error {
    for _, cmd := range script.Commands {
        if !s.isCommandAllowed(cmd) {
            return fmt.Errorf("command not allowed: %s", cmd)
        }
    }
    return nil
}
```

### 2. 权限管理

```go
func (pm *PermissionManager) RequestPermission(name, reason string) bool {
    runtime.EventsEmit(pm.ctx, "permission:request", map[string]interface{}{
        "name":   name,
        "reason": reason,
    })

    return <-pm.responseChan
}
```

---

## 总结

这个架构设计的核心优势：

1. **清晰分层** - 每层职责明确，易于维护
2. **符合 Wails 哲学** - 状态在 Go，前端只反映
3. **高度模块化** - 便于测试和扩展
4. **依赖注入** - 松耦合，易替换实现
5. **事件驱动** - 实时响应，异步处理
6. **平台隔离** - 便于跨平台支持
7. **React 19 优化** - 利用最新特性提升性能

**相关文档**：
- [监控引擎](./02-monitor-engine.md)
- [分析引擎](./03-analyzer-engine.md)
- [AI 服务](./04-ai-service.md)
- [自动化引擎](./05-automation-engine.md)
- [存储层](./06-storage-layer.md)
- [Wails 官方文档](https://wails.io/docs)
- [React 19 文档](https://react.dev/blog/2024/12/05/react-19)
