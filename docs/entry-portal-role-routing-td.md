# SKIPP 用户入口页（角色分流）TD

## 1. 目标与背景

基于《SKIPP – Time Arbitrage Marketplace》企划和当前 MVP 实现，本设计定义一个统一入口页：

1. 先让用户选择角色：`Post Tasks` / `Accept Tasks`
2. 如果选择下任务：按 SKIPP 功能分类创建任务
3. 如果选择接任务：进入地图工作台，看到门店、当前任务、并支持排队上报

目标是把“用户进入后不知道该做什么”的问题，改为单页明确分流与可执行动作。

---

## 2. 术语与范围

## 2.1 角色

- `Requester`：发布任务（买时间）
- `Runner`：接受任务并执行（卖时间）
- `Ops`：当前不作为入口角色，仅在后台/管理接口处理异常

## 2.2 版本范围（本 TD）

- Web 端入口页与两条主流程
- 不新增支付流程
- 不新增复杂身份认证（沿用 MVP 轻量身份输入）

---

## 3. 信息架构（IA）

## 3.1 一级入口页

页面路径建议：`/` 或 `/skip/`

主要区块：

1. Hero 区：一句话价值主张（Send someone, save your time）
2. 角色选择卡片：
   - `Post Tasks`
   - `Accept Tasks`
3. 最近任务快照（可选，后续）

## 3.2 分流后页面

1. `Post Tasks` -> `Task Creation Hub`
2. `Accept Tasks` -> `Runner Map Console`

---

## 4. 下任务（Requester）逻辑设计

## 4.1 分类模型（来自企划总结）

一级分类（入口可选）：

1. `Daily Micro-Tasks`
2. `High-Stakes Time`
3. `Physical Presence`
4. `B2B Time Outsourcing`（MVP 可先隐藏为 invited / not available）

二级模板（示例）：

1. `Daily Micro-Tasks`
   - Queue for food/coffee
   - Parcel pickup
   - Small errand
2. `High-Stakes Time`
   - Visa/government queue
   - Limited drop purchase
   - Slot holding
3. `Physical Presence`
   - Attend for me
   - Hold place for me
   - Verify for me
4. `B2B Time Outsourcing`
   - Document handling
   - On-ground presence
   - Last-mile execution

## 4.2 MVP 字段映射策略

现阶段后端 `task_type` 仅支持 `queue_for_me`。因此：

1. 前端保留“业务分类 + 模板”用于用户输入与分析
2. 发送到后端时：
   - `task_type` 固定传 `queue_for_me`
   - 把分类信息写入 `note`（结构化文本/JSON）

建议 `note` 格式：

```json
{
  "category": "high_stakes_time",
  "template": "limited_drop_purchase",
  "user_note": "Need before 6 PM"
}
```

## 4.3 创建任务流程

1. 选择一级分类
2. 选择二级模板
3. 选择门店/地点（先用已有 stores）
4. 输入说明
5. 提交任务
6. 返回任务状态页（初始 `matching`）

---

## 5. 接任务（Runner）逻辑设计

## 5.1 Runner Map Console 目标

Runner 进入地图页后，至少可完成三件事：

1. 看门店与队列态势（已有地图能力）
2. 看当前可接任务/我已接任务
3. 上报门店排队信息（queue report）

## 5.2 页面布局建议

左侧栏：

1. Runner 身份区（Runner ID + online/offline）
2. `Available Tasks`（可接任务）
3. `My Active Tasks`（已接且未完成）
4. `Queue Report Form`（选店 + queue_length + wait + busy + ttl）

右侧：

1. 地图（Marker 显示 queue number）
2. 点击门店显示 store name + queue/wait + stale 状态

## 5.3 Runner 主流程

1. 设置在线状态
2. 从 `Available Tasks` 选择任务并 `Accept`
3. 任务执行时上报门店排队信息
4. 任务状态推进（arrive / complete）

---

## 6. 前端状态机（入口页）

## 6.1 状态定义

1. `role_unselected`
2. `role_requester`
3. `role_runner`
4. `requester_submitting`
5. `runner_accepting`
6. `error`

## 6.2 状态转移

1. `role_unselected -> role_requester`：点击 `Post Tasks`
2. `role_unselected -> role_runner`：点击 `Accept Tasks`
3. `role_requester -> requester_submitting`：提交任务
4. `role_runner -> runner_accepting`：接单动作
5. 任意状态出错 -> `error`（展示英文错误文案）

---

## 7. API 对齐与缺口

## 7.1 当前已可复用

- `POST /v1/tasks`
- `POST /v1/tasks/{id}/accept`
- `POST /v1/stores/{id}/queue-reports`
- `GET /v1/stores`
- `GET /v1/stores/{id}`
- `GET /v1/stores/{id}/queue-signal`

## 7.2 建议补充（便于 Runner Console 完整）

1. `GET /v1/tasks?status=matching` -> Available tasks
2. `GET /v1/tasks?runner_id={id}&status=accepted,arrived,queuing` -> My active tasks
3. `POST /v1/runners/{id}/availability`（已存在）前端接入

若短期不补 API，可用临时方案：

1. 从 seed/mock 任务静态读取
2. runner 先手输 Task ID 接单（当前已支持）

---

## 8. 页面交互与文案规范（英文站点）

## 8.1 入口页文案

- Title: `Choose Your Role`
- Card A: `Post Tasks`
- Card B: `Accept Tasks`
- Helper: `You can switch roles anytime`

## 8.2 错误文案（统一英文）

- `Failed to load data.`
- `Please select a store first.`
- `Failed to create task.`
- `Failed to accept task.`
- `Failed to submit queue report.`

---

## 9. 分阶段开发计划（入口页专项）

## Phase A：入口分流骨架

交付：

1. 新增角色选择 UI
2. 两个分流容器（Requester / Runner）

测试：

1. `App.test.tsx`：点击角色卡片切换视图

## Phase B：Requester 分类建单

交付：

1. 一级/二级分类选择器
2. 创建任务请求映射（`task_type=queue_for_me` + 分类写入 note）

测试：

1. 前端单测：分类选择后可提交
2. 后端单测：`POST /v1/tasks` 正常创建

## Phase C：Runner Console

交付：

1. 任务列表区（available/my tasks）
2. 门店排队上报表单
3. 地图联动任务与门店状态

测试：

1. 前端单测：接单成功后状态更新
2. 前端单测：上报成功后出现确认提示
3. 后端单测：`queue-reports` 写入与 `queue-signal` 更新

## Phase D：回归与可观测

交付：

1. 增加关键事件埋点（role_selected / task_created / task_accepted / queue_reported）
2. 入口页错误边界与空状态完善

测试：

1. `go test ./...`
2. `npm test -- --run`

---

## 10. 验收标准（DoD）

1. 用户首次进入能在 1 次点击内明确进入目标角色流
2. Requester 能按分类创建任务（即使后端仍用单 task_type）
3. Runner 能看到门店地图、当前任务并完成排队上报
4. 所有用户可见文案与错误文案为英文
5. 前后端测试通过
