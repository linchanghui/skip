# skip

樟宜机场 Demo：门店排队状态 + Web 地图（Maps JavaScript API 可在最后配置）。

## 文档

- [阶段 1：门店排队地图（Web MVP / 樟宜 Demo）设计文档](docs/queue-map-phase1-design.md)

## 实现阶段（与测试）

| 阶段 | 内容 | 完成后运行 |
|------|------|------------|
| 1 | OpenAPI：`api/openapi.yaml` | `go test ./api/...` |
| 2 | Go 服务 + SQLite | `go test ./...`（需 **Go 1.22+**） |
| 3 | Web：`web/`（无 Maps Key 时地图为占位） | `cd web && npm ci && npm test` |
| 4 | 种子数据 | 启动服务自动写入空库 |
| 5 | 部署 | 根目录 `deploy.sh` + `deploy/nginx-dota2master-skip.conf.example`（与 dota2master 同域） |

## 本地运行

**终端 A — 后端**（默认 `:8080`，SQLite `data/app.db`）：

```bash
go run ./cmd/server
```

上报排队需设置环境变量 `SKIP_ADMIN_API_KEY`，请求头携带 `X-Admin-Key`（与设计一致）。

**终端 B — 前端**（Vite 将 `/v1`、`/healthz` 代理到 `http://127.0.0.1:8080`）：

```bash
cd web
npm ci
npm run dev
```

浏览器打开终端里提示的地址（如 `http://localhost:5173`）。

**模拟生产子路径（可选）**：与线上 `/skip/` 一致时，先 `cd web && VITE_BASE_PATH=/skip/ npm run build`，再：

```bash
go run ./cmd/server -addr :8080 -base /skip -static ./web/dist
```

访问 `http://127.0.0.1:8080/skip/`（`/skip` 会重定向到 `/skip/`）。

**最后再接入 Google 地图**：在 `web/` 下 `cp .env.example .env`，填写 `VITE_GOOGLE_MAPS_API_KEY`，重启 `npm run dev`。未配置时仍可查看门店列表与详情。

生产构建若 API 与页面不同源，可设置 `VITE_API_BASE` 指向 API 根 URL；与 Vite `base` 同域时一般不用设置，由 `import.meta.env.BASE_URL` 拼出 `/skip/v1/...`。

## 部署到 EC2（与 dota2master 同域名）

默认与 `se-take-home-assignment/deploy.sh` 同一台主机、同一域名，仅子路径与端口不同：

| 变量 | 默认 |
|------|------|
| `EC2_HOST` | `ec2-user@15.164.92.89` |
| `REMOTE_DIR` | `~/skip` |
| `SERVICE_PORT` | `18181`（FeedMe 为 `18080`） |
| `MOUNT_BASE` / `VITE_BASE_PATH` | `/skip` / `/skip/` |

```bash
./deploy.sh
```

**入口**：线上使用 **Caddy**（非 Nginx），配置在 `dota2-replayer/deploy/Caddyfile.ec2` 的 `handle /skip*` → `127.0.0.1:18181`。更新该文件并 `rsync` 到 EC2 后，建议执行 `sudo docker restart caddy`（`rsync` 原地替换文件 inode 时，bind 挂载可能仍指向旧内容；重启可确保生效），或 `sudo docker exec caddy caddy reload --config /etc/caddy/Caddyfile`。Nginx 示例见 `deploy/nginx-dota2master-skip.conf.example`（仅供参考）。

**上报密钥**：远程 `~/skip/env` 中一行 `SKIP_ADMIN_API_KEY=...`（`chmod 600`）。`deploy.sh` 启动时会自动 `source` 该文件。查看：`ssh ec2-user@15.164.92.89 'cat ~/skip/env'`（勿提交到仓库）。

## 仓库说明

- `go.sum` 已提交；若升级依赖请在 **Go 1.22+** 下执行 `go mod tidy`。
- 当前沙盒/CI 若 Go 版本低于 1.22，将无法编译（`log/slog`）。
