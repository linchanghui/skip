# 东南亚门店排队可视化（阶段 1）— 设计文档

**MVP：Web + 樟宜机场 Demo**

## 1. 文档目的与范围

**目的**：在不做完整 LBS 平台的前提下，用地图呈现「门店/柜台相对排队繁忙程度」，先验证东南亚场景下的产品价值与数据闭环。

**阶段 1 范围**：

- 地图展示（Google Maps 或本地图商 SDK 二选一，见第 5 节）。
- 以**樟宜机场**为 Demo 区域：有限数量的 POI（店铺/柜台）+ 排队/繁忙状态。
- **MVP 形态**：Web；后续预留 App、WhatsApp 嵌入（同一套 API）。

**本阶段明确不做**：完整 LBS 引擎、全量东南亚 POI、与 Google「Popular times」官方数据直连（原因见第 3 节）。

---

## 2. 产品形态与演进

| 阶段 | 形态 | 说明 |
|------|------|------|
| MVP | Web（响应式） | 地图 + POI 列表 + 状态详情；运营/商户可录入或对接简单数据源 |
| 后续 | 原生 App | 复用同一后端 API；可加强推送、离线、深度链接 |
| 后续 | WhatsApp 嵌入 | 典型为 **WhatsApp Flows / 业务消息中的链接卡片** 跳转到 Web，或轻量 H5；核心仍是 HTTP API |

**设计原则**：前后端分离；**API 优先**（便于 Web / App / WA 共用）；地图仅作展示层，可替换图商。

---

## 3. 门店排队数怎么获取？

### 3.1 Google 官方能力边界（结论）

根据 Google 帮助中心说明：

- **Popular times / wait times / visit duration** 来自已开启 **Location History / Timeline** 的用户数据的**聚合与匿名化**结果；**不能由商家手动添加**；且仅在 Google 认为**数据量足够**时才会展示。  
  参考：[About popular times, wait times & visit duration data](https://support.google.com/business/answer/6263531?hl=en)

- **区域繁忙（Busy area）** 同样是基于区域内场所趋势的聚合，并强调隐私与差分隐私等保护措施；**不面向第三方开放「可拉取的实时排队人数 API」**这一形态。  
  参考：[Get information about busy areas from Google Maps](https://support.google.com/maps/answer/11323117?hl=en)

**因此**：若产品目标是「每个门店的实时排队号/人数」，**不能依赖** Google 面向商家的公开接口作为稳定数据源；Google 展示的是「相对繁忙/典型等待」类信息，且获取方式、覆盖度、时效性对第三方产品**不可控**。

### 3.2 可落地的数据源方案（按推荐顺序）

| 方案 | 内容 | 优点 | 缺点 / 风险 |
|------|------|------|-------------|
| **A. 商户/机场运营自报（MVP 主推）** | 后台或受控表单上报：当前排队长度、预计等待分钟、是否暂停取号 | 实现快、语义清晰、合规可控 | 需运营纪律；有主观性 |
| **B. 合作数据接口** | 与樟宜商户或机场系统对接（若有队列系统 API） | 准、可规模化 | 商务与集成成本高 |
| **C. 用户众包（谨慎）** | 用户一键「现在很挤/一般/很空」+ 反作弊 | 可补充盲区 | 易被刷；需算法与治理 |
| **D. 非授权抓取 Maps 前端** | 技术上可能解析页面 | — | **强烈不推荐**：违反服务条款与法律风险 |

**MVP 建议**：**A 为主**（樟宜 Demo 用种子数据 + 运营后台手动更新），接口上预留 **B** 的适配层（同一套领域模型，不同 `source`）。

**与 Google 文档的关系**：把 Google 的说明当作**「为什么不能指望 Google 给我们排队 API」**的依据；产品上若需要「类似 popular times 的相对热度」，未来可考虑**自研聚合**（来自 A/B/C），而不是复刻 Google 黑盒数据。

---

## 4. Demo 范围：樟宜机场

- **地理边界**：机场多航站楼 polygon（GeoJSON）或 bbox，用于列表过滤与地图默认视野。
- **POI 集（MVP 收敛）**：先**写死**「樟宜机场附近两家星巴克」两条记录即可验证地图 + 排队状态闭环；后续再扩到 20～50 个点。
- **状态**：至少支持  
  - `queue_length`（可选整数：当前叫号差、或人数档位）  
  - `wait_minutes_est`（可选）  
  - `busy_level` 枚举：`quiet` / `moderate` / `busy` / `closed`  
  - `as_of`（观测时间）、`source`（`merchant` / `operator` / `integration` / `crowd`）

### 4.1 两家星巴克的坐标：要你提供吗？能用 Google 查吗？

| 方式 | 说明 |
|------|------|
| **你提供（可选）** | 若你有更准确的「航站楼 + 店铺名 + 坐标」或 Google Maps 分享链接，可直接写进种子 JSON/SQL，**不强制**。 |
| **不写死、用接口查一次** | 可用 **Places API**（例如 Text Search：`Starbucks near Changi Airport`）在开发阶段解析出 `place_id`、名称、坐标，再**固化进仓库种子数据**。 |
| **运行时每次调 Google 拉列表** | 不推荐作为 MVP：配额与费用、条款对缓存/展示有要求，且 Demo 只需要两个稳定锚点。 |

**结论**：**不必专门由你提供数据**；实现上默认在代码里写死两条门店记录（名称、area、lat、lng 可从公开地图或一次 Places 查询得到）。若你愿意核对「到底是哪两家店（T1/T3 等）」，只影响展示文案准确性，不影响技术路径。

**注意**：Google **没有**「给任意关键词返回实时排队人数」的开放接口；Places 解决的是**地点检索与元数据**，与排队业务数据仍分离（排队仍走自报/API 第 3 节方案）。

---

## 5. 技术方案总览（建议栈）

| 层 | 选型 | 说明 |
|----|------|------|
| 前端 | TypeScript + React（或团队惯用栈）+ **Google Maps JavaScript API**（Demo 在新加坡，合规与体验较顺） | 后续可抽象 `MapProvider`，换 MapLibre + 自有瓦片 |
| 后端 | **Go** + `chi`/`echo` 等轻量路由（与二进制部署模式一致） | 单二进制、易交叉编译 |
| 存储 | **SQLite**（单文件，如 `data/app.db`；MVP 用 lat/lng 浮点即可，不上 PostGIS） | 迁移用 goose/atlas/`migrate` 或手写 SQL；部署时随二进制目录或持久卷挂载 |
| 鉴权 MVP | 公开读；写接口用 **API Key** 或 **JWT（运营账号）** | 后续再接 OAuth |
| 配置 | 环境变量 + `.env.example` | |
| 测试 | **Go：`go test ./...`**；契约测试可用 **schemathesis** 或手写 HTTP 集成测试 | **每个阶段结束必须绿** |

### 5.1 Google Cloud 项目里要提前开什么、Key 怎么用

在 [Google Cloud Console](https://console.cloud.google.com/) 同一项目中启用 **Maps JavaScript API**（Web 地图必开），并**绑定结算账号**（Maps Platform 按用量计费，有免费额度政策以官网为准）。

| API（控制台里的名称） | MVP 是否必需 | 用途 |
|----------------------|--------------|------|
| **Maps JavaScript API** | **必需** | 浏览器里加载地图、Marker、InfoWindow 等 |
| **Places API** 或 **Places API (New)** | **可选** | 仅在「用文本搜索解析两家星巴克坐标/place_id」时用；若坐标已写死在种子数据里，可不开 |
| **Geocoding API** | **可选** | 仅当你用「地址字符串 → 坐标」且不走 Places 时需要 |

**Key 建议（安全实践）**：

1. **浏览器 Key（给前端）**：只开 **Maps JavaScript API**；在凭据里加 **HTTP 引荐来源网址限制**（如 `http://localhost:5173/*`、生产域名 `https://yourdomain/*`）。不要把带 Places/无限制的 Key 暴露在前端。  
2. **服务端 Key（给 Go 后端，若后端调 Places）**：开 **Places API（或 New）**；使用 **IP 限制**（服务器公网 IP 或 Cloud NAT 出口）。前端不持有此 Key。

环境变量命名示例：`VITE_GOOGLE_MAPS_API_KEY`（仅 Maps JS）、`GOOGLE_MAPS_API_KEY`（服务端 Places，可选）。

---

## 6. 自上而下分阶段落地（每阶段完成即跑测试）

### 阶段 0：冻结范围与契约（0.5～1 天）

**交付物**：本设计文档定稿版、樟宜种子 POI 列表（CSV/JSON）、状态枚举定义。

**测试**：文档评审 checklist（非自动化）。

---

### 阶段 1：API 设计优先（1～2 天）

**目标**：发布 **OpenAPI 3.1**（或 3.0）规范，前后端可并行。

**核心资源**（建议 REST 风格）：

1. `GET /healthz` — 存活探针。  
2. `GET /v1/areas/changi` — Demo 区域元数据（边界、默认中心、缩放）。  
3. `GET /v1/stores?area_id=changi` — 门店列表（含最新快照或 embed `latest_status`）。  
4. `GET /v1/stores/{id}` — 单店详情 + 最近 N 条状态历史（可选 query）。  
5. `POST /v1/stores/{id}/status-reports` — 上报新状态（运营/商户；需鉴权）。  
6. （可选）`POST /v1/admin/seed` — 仅非生产环境或受保护的管理入口，导入种子数据。

**测试（本阶段必须绿）**：

- OpenAPI **lint**（spectral）。  
- **契约测试**：用 openapi 生成 minimal mock 或 dredd/schemathesis 对 stub server 跑一轮（若时间紧：手写「请求/响应 JSON 与 schema 一致」的 golden test）。

---

### 阶段 2：Service + 持久化（3～5 天）

**目标**：实现真实读写；读路径低延迟；写路径审计。

**模块划分**：

- `internal/domain`：Store、StatusReport、BusyLevel。  
- `internal/repository`：**SQLite** 实现（`database/sql` + 驱动，或 `modernc.org/sqlite` 等纯 Go 方案便于 `CGO_ENABLED=0` 交叉编译）。  
- `internal/http`：handler、中间件（鉴权、request id、日志）。  
- `cmd/server`：入口、`migrate` 子命令或启动时 migrate。

**表（最小）**：

- `stores`：id, name, category, terminal, floor, lat, lng, area_id, external_ref, created_at…  
- `status_reports`：id, store_id, busy_level, queue_length, wait_minutes_est, source, reported_at, reporter_id…  
- `store_status_latest`：可用**应用层在写入 status 时同步更新**同一行，或 SQLite 触发器；MVP 避免物化视图复杂度。

**测试（本阶段必须绿）**：

- **Repository 集成测试**：使用临时 SQLite 文件或 `:memory:`，`go test -tags=integration ./...`（无需 Docker Postgres）。  
- **HTTP 集成测试**：`httptest.Server` 打全链路 `GET/POST`。

---

### 阶段 3：Web 前端（3～5 天）

**目标**：地图 + 标记 + 侧栏列表；点击 marker 展示最新状态与时间。

**测试**：

- **Vitest/RTL** 组件与 hooks 测试。  
- **Playwright** 1～2 条 E2E：加载地图壳、mock API 返回固定 JSON、断言列表与详情文案（地图 canvas 难断言则断言 DOM 与 API 调用）。

---

### 阶段 4：樟宜 Demo 数据与运营流（1～2 天）

**交付物**：种子 SQL/迁移、简单「更新状态」页面（仅登录运营可用）或 CLI。

**测试**：E2E 或脚本校验「种子店铺数量、每个店至少一条 latest」。

---

### 阶段 5：部署与观测（1～2 天）

对齐参考仓库中的部署脚本模式（见第 8 节）：

- 本地 **`CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build`** 产出 Linux amd64 单二进制。  
- **`rsync -avz`** 同步到 EC2 目录（排除 `.git`、`node_modules` 等）。  
- 远端 **`fuser -k $PORT/tcp`** 释放端口后 **`nohup ./server -addr :PORT`** 后台运行，日志落文件。  
- 若有静态前端：构建产物一并 rsync；**Nginx** `proxy_pass` 到本机端口，**子路径挂载**（`MOUNT_BASE`、`VITE_BASE_PATH` 一类变量与主站共存）。

**测试**：部署后 smoke：`curl` healthz、`GET /v1/stores`；可选 Uptime 探针。

---

## 7. API 草案（字段级摘要）

**`Store`（响应）**

```json
{
  "id": "store_01",
  "name": "Burger Counter T3",
  "area_id": "changi",
  "terminal": "T3",
  "floor": "B2",
  "location": { "lat": 1.234, "lng": 103.987 },
  "category": "food",
  "latest_status": {
    "busy_level": "busy",
    "queue_length": 12,
    "wait_minutes_est": 25,
    "as_of": "2026-04-20T10:15:00Z",
    "source": "operator"
  }
}
```

**`POST /v1/stores/{id}/status-reports`**

```json
{
  "busy_level": "moderate",
  "queue_length": 5,
  "wait_minutes_est": 10,
  "source": "merchant",
  "note": "optional"
}
```

错误模型：统一 `application/problem+json` 或 `{ "error": { "code", "message" } }`（全项目一致即可）。

---

## 8. 部署方案（参考 `se-take-home-assignment/deploy.sh`）

参考路径：`GolandProjects/se-take-home-assignment/deploy.sh`（与本 skip 仓库可并存，部署时复制/adapt 到本项目的 `deploy/` 或根目录脚本）。

该脚本体现的模式要点：

- 本地交叉编译 Linux amd64 单二进制（`CGO_ENABLED=0`）。  
- `rsync` 同步到 EC2，SSH 选项避免 GSSAPI 长时间卡住（`BatchMode`、`ConnectTimeout`、`GSSAPIAuthentication=no`）。  
- 远端 `timeout fuser -k PORT/tcp` 释放端口；`nohup ./binary -addr :PORT -base MOUNT_BASE` 后台运行。  
- 前端存在时：`npm ci` + `VITE_BASE_PATH` 构建后再同步；Nginx 示例配置反代子路径。

**新项目建议**：复用同一套环境变量命名（`EC2_HOST`、`REMOTE_DIR`、`SERVICE_PORT`、`MOUNT_BASE`、`VITE_BASE_PATH`），仅替换二进制名与项目路径。

---

## 9. 风险与合规（摘要）

- **地图 ToS**：Google Maps Platform 有单独条款；keys 需服务端/前端分离与配额限制。  
- **数据真实性**：商户自报需免责声明（「估算/以现场为准」）。  
- **个人信息**：若众包带账号，需最小化存储与保留策略（新加坡 **PDPA** 等本地法规划线）。

---

## 10. 阶段完成定义（DoD）汇总

| 阶段 | DoD |
|------|-----|
| 1 | OpenAPI 定稿 + lint 绿 + 契约/ golden 测试绿 |
| 2 | `go test`（含 integration）绿；核心 CRUD 与 latest 聚合正确 |
| 3 | 单元/E2E（mock API）绿；Demo 可手动验收 |
| 4 | 樟宜种子数据可重复初始化 |
| 5 | 一键部署脚本 + smoke 通过 |

---

## 附录：本仓库实现进度（与阶段对齐）

| 设计阶段 | 仓库状态 |
|----------|----------|
| 0 契约 | 设计文档 + 两家星巴克种子数据（`internal/repository/store.go` `SeedDemo`） |
| 1 API | `api/openapi.yaml` + `api/openapi_test.go` |
| 2 Service + SQLite | `cmd/server`、`internal/httpserver`、`internal/repository`、`internal/db/migrations`；`go test ./...`（需本地 **Go 1.22+**，因使用 `log/slog`） |
| 3 Web | `web/`（Vite + React）；**无** `VITE_GOOGLE_MAPS_API_KEY` 时地图区为占位说明；配置后加载 Maps JS 与 Marker |
| 4 Demo 数据 | 启动服务时自动 `SeedDemo`（空库写入两家店 + 初始状态） |
| 5 部署 | 仓库根目录 `deploy.sh`；Nginx 示例 `deploy/nginx-dota2master-skip.conf.example`（与 dota2master 同域、`/skip/` 子路径、端口 `18181`） |

**本地联调**：终端 A `go run ./cmd/server`（默认 `:8080`）；终端 B `cd web && npm run dev`（Vite 将 `/v1`、`/healthz` 代理到 `127.0.0.1:8080`）。**最后再**在 `web/.env` 中设置 `VITE_GOOGLE_MAPS_API_KEY`（参见 `web/.env.example`）。

**生产子路径**：后端 `-base /skip -static ./web/dist`；前端构建 `VITE_BASE_PATH=/skip/`；与 FeedMe 相同，Nginx `proxy_pass` 到本机端口且**不要**在 `proxy_pass` 末尾加 `/`，由 Go `StripPrefix` 处理。

**线上入口**：`dota2master.com` 当前由 **Caddy** 反代（`dota2-replayer/deploy/Caddyfile.ec2`），非 Nginx；更新 Caddyfile 后若容器内仍为旧内容，可 `sudo docker restart caddy`。
