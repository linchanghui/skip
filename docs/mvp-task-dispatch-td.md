# SKIPP MVP 技术设计（TD）

## 1. 文档目标

本文件是 SKIPP MVP 的实施级设计文档，目标是让后续开发可直接按文档推进：

1. 明确业务边界与模块边界
2. 明确数据库表结构、索引、约束
3. 明确项目代码层级与职责
4. 明确 API 契约优先的开发顺序
5. 明确“自上而下”分阶段落地路线（先 schema + API，再 service）

---

## 2. MVP 范围与核心假设

## 2.1 范围

- 单城市
- 单类任务：`queue_for_me`
- 单一主场景：咖啡/餐饮排队
- 半自动履约：系统分发 + 人工兜底

## 2.2 核心假设

- 早期目标不是做“实时排队数据平台”，而是验证“代排队履约闭环”
- 排队数据是“决策辅助信号”，不是“平台承诺实时真值”
- 信任体系通过“小规模可控 runner 池 + 履约证据链”达成最小可用

---

## 3. 领域模型与模块边界

## 3.1 领域对象

- `Store`：门店
- `QueueReport`：门店排队上报（带有效期）
- `Task`：用户任务
- `TaskAttempt`：任务分发尝试（每轮匹配）
- `Runner`：履约人员
- `RunnerAvailability`：runner 可接单状态
- `TaskProof`：履约证据（到店、排队中、完成）
- `TaskEvent`：任务状态流转事件审计

## 3.2 模块职责

- `Queue Signal`：接收上报、判断过期、输出可展示信号
- `Dispatch`：执行匹配、广播、改派、超时处理
- `Runner Management`：入池、状态、可接单开关
- `Trust & Ops`：证据审计、异常单处理、手动改派

---

## 4. 项目层级设计（建议落地到当前仓库）

当前仓库已有 `internal/domain`、`internal/repository`、`internal/httpserver`，建议在此基础上扩展。

```text
cmd/server/main.go

internal/
  db/
    migrate.go
    migrations/
      001_init.sql
      002_task_dispatch_mvp.sql
  domain/
    types.go                    # 现有门店/状态模型
    task_types.go               # Task/Runner/QueueReport 枚举与结构
    validate.go                 # 参数和状态机校验
  repository/
    store.go                    # 现有门店读写
    task_repo.go                # tasks/task_events/task_attempts
    runner_repo.go              # runners/runner_availability
    queue_repo.go               # queue_reports + latest signal 查询
    proof_repo.go               # task_proofs
  service/
    task_service.go             # 创建任务、取消任务、状态流转
    dispatch_service.go         # 自动匹配、超时改派
    runner_service.go           # runner 入池、上下线
    queue_service.go            # 上报/过期逻辑
    ops_service.go              # 手工指派、异常处理
  httpserver/
    server.go
    handler_task.go
    handler_runner.go
    handler_queue.go
    handler_ops.go
    middleware_auth.go
```

---

## 5. 数据库设计（SQLite，MVP）

说明：

- 时间字段统一使用 UTC 的 RFC3339 字符串或 SQLite datetime
- 关键状态字段使用 `TEXT + CHECK`，保证状态机可控
- 所有任务核心表都保留 `created_at/updated_at`

## 5.1 复用现有表

- `stores`
- `status_reports`
- `store_status_latest`

现有表继续服务门店展示；新增排队信号统一落 `queue_reports`，后续可逐步替换 `status_reports` 或双写过渡。

## 5.2 新增表（DDL 草案）

```sql
-- 002_task_dispatch_mvp.sql

CREATE TABLE IF NOT EXISTS runners (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  phone TEXT NOT NULL UNIQUE,
  status TEXT NOT NULL CHECK (status IN (
    'candidate','approved','probation','active','suspended','offboarded'
  )),
  service_area TEXT NOT NULL DEFAULT 'changi',
  reliability_score REAL NOT NULL DEFAULT 0.5,
  agreement_version TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_runners_status_area
  ON runners(status, service_area);

CREATE TABLE IF NOT EXISTS runner_availability (
  runner_id TEXT PRIMARY KEY REFERENCES runners(id) ON DELETE CASCADE,
  is_online INTEGER NOT NULL CHECK (is_online IN (0,1)),
  last_ping_at TEXT NOT NULL DEFAULT (datetime('now')),
  current_lng REAL,
  current_lat REAL,
  active_task_id TEXT,
  updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_runner_availability_online
  ON runner_availability(is_online, updated_at DESC);

CREATE TABLE IF NOT EXISTS tasks (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  store_id TEXT NOT NULL REFERENCES stores(id) ON DELETE RESTRICT,
  task_type TEXT NOT NULL CHECK (task_type IN ('queue_for_me')),
  status TEXT NOT NULL CHECK (status IN (
    'created','matching','accepted','arrived','queuing','completed','failed','cancelled'
  )),
  requested_at TEXT NOT NULL,
  accepted_runner_id TEXT REFERENCES runners(id) ON DELETE SET NULL,
  quoted_price_cents INTEGER,
  sla_accept_by TEXT NOT NULL,
  sla_arrive_by TEXT,
  fail_reason TEXT,
  cancelled_by TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_tasks_status_created
  ON tasks(status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_tasks_runner_status
  ON tasks(accepted_runner_id, status);

CREATE TABLE IF NOT EXISTS task_attempts (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  attempt_no INTEGER NOT NULL,
  strategy TEXT NOT NULL CHECK (strategy IN ('auto_batch','manual_assign')),
  candidate_runner_ids TEXT, -- JSON array string
  selected_runner_id TEXT REFERENCES runners(id) ON DELETE SET NULL,
  result TEXT NOT NULL CHECK (result IN ('pending','accepted','timeout','rejected','cancelled')),
  started_at TEXT NOT NULL DEFAULT (datetime('now')),
  ended_at TEXT,
  UNIQUE(task_id, attempt_no)
);

CREATE INDEX IF NOT EXISTS idx_task_attempts_task
  ON task_attempts(task_id, attempt_no DESC);

CREATE TABLE IF NOT EXISTS task_events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  from_status TEXT,
  to_status TEXT NOT NULL,
  actor_type TEXT NOT NULL CHECK (actor_type IN ('user','runner','system','ops')),
  actor_id TEXT,
  payload TEXT, -- JSON
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_task_events_task_time
  ON task_events(task_id, created_at DESC);

CREATE TABLE IF NOT EXISTS task_proofs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  runner_id TEXT REFERENCES runners(id) ON DELETE SET NULL,
  proof_type TEXT NOT NULL CHECK (proof_type IN (
    'arrived_photo','queue_progress_photo','completion_photo','text_note'
  )),
  media_url TEXT,
  note TEXT,
  captured_at TEXT NOT NULL DEFAULT (datetime('now')),
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_task_proofs_task_time
  ON task_proofs(task_id, captured_at DESC);

CREATE TABLE IF NOT EXISTS queue_reports (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  store_id TEXT NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
  reporter_type TEXT NOT NULL CHECK (reporter_type IN ('runner','user','operator')),
  reporter_id TEXT,
  queue_length INTEGER,
  wait_minutes_est INTEGER,
  busy_level TEXT NOT NULL CHECK (busy_level IN ('quiet','moderate','busy','closed')),
  evidence_url TEXT,
  confidence_flag TEXT NOT NULL DEFAULT 'normal' CHECK (confidence_flag IN ('normal','low')),
  reported_at TEXT NOT NULL DEFAULT (datetime('now')),
  expires_at TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_queue_reports_store_time
  ON queue_reports(store_id, reported_at DESC);

CREATE INDEX IF NOT EXISTS idx_queue_reports_store_expiry
  ON queue_reports(store_id, expires_at DESC);
```

## 5.3 关键约束规则

- `tasks.status` 只能按状态机流转（应用层强校验）
- `runner_availability.active_task_id` 非空时，不应再接新单
- `queue_reports.expires_at` 必填，推荐 `reported_at + 20~30 min`
- `task_attempts` 记录每轮匹配，便于失败复盘和 SLA 分析

---

## 6. 核心流程设计

## 6.1 任务创建与分发

1. 用户创建任务：`created`
2. 系统写入 `task_events(created)`
3. 系统进入 `matching`，创建 `task_attempts(attempt_no=1, pending)`
4. 按规则筛 runner，批量通知
5. 任一 runner 接受后：
   - `task_attempts.result=accepted`
   - `tasks.status=accepted`
   - `tasks.accepted_runner_id=...`
6. 若超时无人接单：
   - 当前 `attempts.result=timeout`
   - 创建下一轮 `attempt_no + 1`
   - 超过阈值后 `tasks.status=failed`

## 6.2 履约与证据

runner 操作：

- 到店：`accepted -> arrived`，上传 `arrived_photo`
- 排队中：`arrived -> queuing`，可上传进度
- 完成：`queuing -> completed`，上传 `completion_photo` 或文本凭证

所有动作必须写入：

- `task_events`
- `task_proofs`（若有证据）

## 6.3 门店排队上报与过期

1. runner/user/operator 提交 `queue_report`
2. 后端计算 `expires_at = reported_at + ttl_minutes`
3. 查询门店信号时：
   - 若最新 report 未过期，返回数值
   - 若过期，返回 `status_expired=true` 且隐藏旧数值

---

## 7. API 设计（MVP v1）

注意：现有 `api/openapi.yaml` 先保留门店相关接口。以下为下一步增量接口，建议先写入 OpenAPI，再开发实现。

## 7.1 用户任务接口

- `POST /v1/tasks`
- `GET /v1/tasks/{id}`
- `POST /v1/tasks/{id}/cancel`

`POST /v1/tasks` request:

```json
{
  "user_id": "u_001",
  "store_id": "sb-jewel",
  "task_type": "queue_for_me",
  "note": "need before 6pm"
}
```

`GET /v1/tasks/{id}` response (简化):

```json
{
  "id": "t_001",
  "status": "matching",
  "store_id": "sb-jewel",
  "accepted_runner_id": null,
  "sla_accept_by": "2026-04-23T12:05:00Z",
  "latest_event_at": "2026-04-23T12:01:20Z"
}
```

## 7.2 runner 接单接口

- `POST /v1/runners/apply`
- `POST /v1/runners/{id}/availability`
- `POST /v1/tasks/{id}/accept`
- `POST /v1/tasks/{id}/arrive`
- `POST /v1/tasks/{id}/complete`
- `POST /v1/tasks/{id}/proofs`

## 7.3 排队信号接口

- `POST /v1/stores/{id}/queue-reports`
- `GET /v1/stores/{id}/queue-signal`

`GET /v1/stores/{id}/queue-signal` response:

```json
{
  "store_id": "sb-jewel",
  "status_expired": true,
  "last_updated_at": "2026-04-23T14:00:00Z",
  "last_updated_x_mins_ago": 62,
  "signal": null
}
```

## 7.4 运营接口

- `POST /v1/ops/tasks/{id}/assign`
- `POST /v1/ops/tasks/{id}/fail`
- `POST /v1/ops/queue-reports/{id}/hide`

---

## 8. Service 层设计（接口优先）

## 8.1 TaskService

建议方法：

- `CreateTask(ctx, input) (Task, error)`
- `CancelTask(ctx, taskID, actor) error`
- `GetTask(ctx, taskID) (TaskDetail, error)`
- `Transition(ctx, taskID, command) error`

职责：

- 状态机合法性校验
- SLA 字段计算
- 事件写入

## 8.2 DispatchService

建议方法：

- `StartMatching(ctx, taskID) error`
- `RunnerAccept(ctx, taskID, runnerID) error`
- `HandleAttemptTimeout(ctx, taskID, attemptNo) error`
- `ManualAssign(ctx, taskID, runnerID, opsID) error`

职责：

- runner 候选筛选
- attempt 生命周期
- 自动补位与失败收敛

## 8.3 RunnerService

建议方法：

- `Apply(ctx, input) (Runner, error)`
- `SetAvailability(ctx, runnerID, online, location) error`
- `Suspend(ctx, runnerID, reason) error`

## 8.4 QueueService

建议方法：

- `Report(ctx, storeID, input) (QueueReport, error)`
- `GetSignal(ctx, storeID) (QueueSignal, error)`

职责：

- TTL 计算
- 过期判定
- 简单置信度标记

---

## 9. 自上而下开发阶段（强约束）

## Phase 0：冻结契约与状态机（0.5 天）

- 产出：
  - 本 TD 定稿
  - 状态流转表定稿
  - 字段命名与枚举定稿
- DoD：
  - 团队评审通过

## Phase 1：先定义 DB Schema（1 天）

- 产出：
  - `002_task_dispatch_mvp.sql`
  - migration 回滚策略（至少文档说明）
- DoD：
  - `go test ./internal/db/...` 通过（若有）
  - 本地迁移可重复执行

## Phase 2：先定义 OpenAPI（1 天）

- 产出：
  - 更新 `api/openapi.yaml`，新增 task/runner/queue/ops 接口
  - 错误码与响应模型统一
- DoD：
  - `go test ./api/...` 通过
  - lint/spectral 通过（如果接入）

## Phase 3：Repository 层（2-3 天）

- 产出：
  - `task_repo.go` `runner_repo.go` `queue_repo.go` `proof_repo.go`
  - 事务边界：接单竞争、状态流转原子性
- DoD：
  - `go test ./internal/repository/...` 通过
  - 覆盖典型并发场景（两个 runner 同时抢单）

## Phase 4：Service 层（2-3 天）

- 产出：
  - `task_service.go` `dispatch_service.go` `queue_service.go`
  - 状态机、SLA、补位逻辑
- DoD：
  - service 单测覆盖关键路径和失败路径

## Phase 5：HTTP Handler 与鉴权（1-2 天）

- 产出：
  - 新增 handler 文件并挂载路由
  - runner 与 ops 的最小鉴权策略（MVP 可 API key/固定 token）
- DoD：
  - `go test ./internal/httpserver/...` 通过
  - e2e smoke：创建任务 -> 接单 -> 到店 -> 完成

## Phase 6：前端最小接入（2 天）

- 产出：
  - 任务提交页
  - runner 接单页（内部）
  - store detail 中 queue-signal 的“过期提示”
- DoD：
  - `cd web && npm test` 通过

## Phase 7：灰度运行与指标（持续）

- 产出：
  - 关键指标看板：接单时长、完成率、改派率、过期信号占比
- DoD：
  - 可以支持 1 城市小规模真实测试

---

## 10. 测试策略（按层）

- migration：重复执行与空库初始化
- repository：事务一致性、唯一约束、外键约束
- service：状态机合法流转、非法流转拦截、SLA 超时逻辑
- http：契约一致性、错误码一致性
- web：过期信号 UI 与任务状态展示

建议最小回归命令：

```bash
go test ./api/...
go test ./internal/repository/...
go test ./internal/httpserver/...
go test ./...
cd web && npm test
```

---

## 11. MVP 非目标与后续扩展

MVP 非目标：

- 多城市自动化调度
- 复杂定价
- 完整 escrow 支付与赔付引擎
- 高级反作弊模型

后续扩展方向：

- 从 `queue_for_me` 扩展到 `collect_for_me`、`buy_for_me`
- 将 `queue_reports` 与现有 `status_reports` 做统一聚合视图
- 引入更细粒度 runner 风险分层和自动质检
