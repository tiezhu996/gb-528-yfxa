# 舞台吊挂 Cue 联锁离线推演

`stage-rigging-cue-interlock` 是供剧场吊挂程序员与安全复核人员使用的离线排练工作台。系统维护设备运动边界、Cue 动作与依赖、版本化联锁规则，将锁定 Cue 展开成确定性时间线并保存不可覆盖的碰撞与规则证据。

> **安全边界：本项目不连接吊机、控制台、PLC 或任何舞台机械，不发送动作命令，也不提供现场操作许可。所有结果只用于离线排练规划和人工安全复核，不能替代主管、设备检查或既有安全程序。**

## Docker Compose 快速启动

```bash
cp .env.example .env
docker compose up -d --build
docker compose ps
```

服务地址：

- Web 工作台：<http://127.0.0.1:18528>
- 后端健康：<http://127.0.0.1:19528/healthz>
- 后端就绪：<http://127.0.0.1:19528/readyz>
- 前端代理健康：<http://127.0.0.1:18528/api/healthz>
- API 前缀：<http://127.0.0.1:19528/api/v1>

种子账号：

| 角色 | 用户名 | 密码 | 主要权限 |
| --- | --- | --- | --- |
| 吊挂程序员 | `programmer` | `programmer123` | 设备、Cue、规则维护，运行推演，提交复核 |
| 安全复核员 | `reviewer` | `reviewer123` | Cue 批准/退回/锁定，规则启停，推演批准/拒绝，审计 |
| 管理员 | `admin` | `admin123` | 程序员和复核员权限 |

停止并只清理本项目容器、网络和命名卷：

```bash
docker compose down -v --remove-orphans
```

## 主要功能

- `RiggingDevice`：设备代码、类型、载荷/速度/行程、安全区、状态和乐观锁版本；设备页同时展示适用规则。
- `CueDefinition`：序号、绝对起始时间、时长、动作 JSON、依赖 JSON、创建人、批准人和完整状态流。
- `InterlockRule`：负载、速度、行程、安全区互斥和依赖间隔五类规则，保存设备范围、结构化阈值、严重度、启停与规则版本。
- `RehearsalRun`：不可覆盖的 Cue/规则版本快照、动作时间线、规则结果、碰撞窗口、最高严重度与人工复核记录。
- 五个业务页：设备模型、Cue 编排、联锁规则、离线推演、审计复核；共享时间线和证据表只消费真实 API。
- JWT/RBAC、request ID、结构化访问日志、panic recovery、本地限流、统一错误响应、事务状态迁移和追加式审计。

项目不包含票务、场地预约、工单、库存、订单或财务功能。

## 算法与假设

1. 对选中的锁定 Cue 构建依赖图；先检查缺失前置、重复依赖和重复序号，再用 DFS 返回闭合循环证据路径，最后以序号和 Cue 代码稳定排序完成拓扑序。
2. 每个动作展开为半开区间 `[start, end)`；动作绝对时间等于 Cue 起点加动作相对偏移。首尾相接不算重叠。
3. 位置按端点线性插值：`position(t) = from + (to - from) × elapsed / duration`；速度为端点距离除以动作秒数。
4. 联锁评估检查设备载荷、线速度、行程端点、同安全区重叠和依赖完成间隔。每条证据包含规则编号、Cue、设备、时间窗口、实际值、阈值、单位、严重度和说明。
5. `cue_set_version` 由 Cue ID/版本和启用规则 ID/版本稳定散列得到；相同锁定版本与规则版本生成相同时间线和规则证据。

算法只做模型比较。它不处理真实加速度、结构挠度、控制器响应、制动距离、人员位置或设备通信，因此不能形成“可执行”结论。

## 状态与枚举位置

`CueStatus = draft | pending_review | approved | locked | archived`

- 数据库：`cue_definitions.cue_status` 的显式 `CHECK` 约束。
- 后端：`backend/internal/constants/cue.go`，贯穿 `model/cue_definition.go`、repository/service/handler/router。
- 前端：`frontend/src/types/cue.ts`、`stores/cues.ts`、`components/common/CueStatusBadge.vue`、`pages/CuesPage.vue`、`pages/AuditPage.vue`。

状态固定为：

```text
draft -> pending_review -> approved -> locked -> archived
             |               |
             +----> draft <---+
```

批准、退回和锁定只能由 `safety_reviewer` 或 `admin` 执行。只有 `locked` Cue 版本可进入推演。

`InterlockResult = pass | warning | blocker | invalid`

- 数据库：`rehearsal_runs.highest_severity` 显式 `CHECK` 约束，完整结果保存在 `rule_results_json`。
- 后端：`backend/internal/constants/interlock.go`、`internal/interlock/evaluator.go`、`dto/rehearsal_run.go`。
- 前端：`frontend/src/types/interlock.ts`、`types/rehearsal.ts`、`components/common/RuleEvidenceTable.vue`、`pages/RulesPage.vue`、`pages/RehearsalsPage.vue`。

包含 blocker 或 invalid 的运行保存为 `blocked`，不能提交或批准。正常运行按 `evaluated -> pending_review -> approved_for_rehearsal | rejected` 流转；批准只代表离线证据已由人员复核。

## API

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `POST` | `/api/v1/auth/login` | 种子账号登录并签发 JWT |
| `GET/POST` | `/api/v1/devices` | 设备列表、创建设备模型 |
| `GET/PUT` | `/api/v1/devices/:id` | 设备详情、版本化限制更新 |
| `GET/POST` | `/api/v1/cues` | Cue 列表、创建草稿 |
| `GET/PUT` | `/api/v1/cues/:id` | Cue 详情、草稿动作更新 |
| `POST` | `/api/v1/cues/:id/{submit,approve,reject,lock,archive}` | 事务化 Cue 状态迁移 |
| `GET/POST` | `/api/v1/rules` | 规则列表、创建规则 |
| `GET/PUT` | `/api/v1/rules/:id` | 规则详情、版本化阈值更新 |
| `POST` | `/api/v1/rules/:id/toggle` | 复核员启停规则 |
| `POST` | `/api/v1/rules/:id/test` | 单阈值离线探针 |
| `GET` | `/api/v1/rehearsals`、`/rehearsals/:id` | 运行列表和不可覆盖快照 |
| `POST` | `/api/v1/rehearsals/run` | 对锁定 Cue 集执行确定性推演 |
| `POST` | `/api/v1/rehearsals/:id/submit` | 提交安全复核 |
| `POST` | `/api/v1/rehearsals/:id/review` | 复核员批准或拒绝 |
| `GET` | `/api/v1/rehearsals/:id/compare?other_id=` | 比较两个运行版本 |
| `GET` | `/api/v1/audit-events` | 复核员读取追加式审计事件 |

统一响应包含 `data`（列表另含 `meta`）和 `request_id`；错误包含 `error.code`、`error.message`、可选 `error.details` 与 `request_id`。主要错误码包括 `CUE_DEPENDENCY_CYCLE`、`MISSING_CUE_DEPENDENCY`、`DUPLICATE_CUE_SEQUENCE`、`ACTION_OUT_OF_CUE_BOUNDS`、`CUE_NOT_LOCKED`、`BLOCKER_RUN_NOT_APPROVABLE`、各实体版本冲突、`AUTH_REQUIRED` 与 `FORBIDDEN`。

## 技术栈与结构

| 层 | 技术 |
| --- | --- |
| 前端 | Vue 3、TypeScript、Vite、Element Plus、Pinia、Vue Router、Lucide |
| 后端 | Go 1.22、Gin、GORM、validator/v10、JWT、`slog` |
| 数据库 | Compose 使用 PostgreSQL 16；runtime smoke 使用独立 SQLite 内存库 |
| 部署 | Nginx SPA 与 `/api` 反向代理、三服务健康依赖 |

```text
backend/internal/{config,model,dto,repository,service,handler,router,middleware,constants,interlock,util}
frontend/src/{api,stores,types,components/common,hooks,pages,router,utils}
```

后端保持 `handler -> service -> repository -> model` 单向依赖和构造器注入。依赖图、时间线展开、位置插值、规则求值分别位于四个 `interlock/` 文件；四个核心实体逐层分文件。

## 环境变量

| 变量 | 默认用途 |
| --- | --- |
| `COMPOSE_PROJECT_NAME` | `stage-rigging-cue-interlock` |
| `FRONTEND_PORT/BACKEND_PORT/DB_PORT` | `18528/19528/57528` |
| `POSTGRES_DB/POSTGRES_USER/POSTGRES_PASSWORD` | PostgreSQL 业务库配置 |
| `DB_DRIVER/DB_DSN/DB_AUTO_MIGRATE` | 双驱动、连接串与迁移开关 |
| `GIN_MODE` | Compose 默认使用 `release` 模式 |
| `JWT_SECRET/JWT_TTL_MINUTES` | JWT 签名与有效期，部署时必须替换密钥 |
| `CORS_ORIGINS` | 允许的浏览器源，逗号分隔 |
| `TIMELINE_STEP_MS` | 保存到推演快照的离线时间步长 |
| `MAX_CUES_PER_RUN` | 单次推演允许的最大 Cue 数 |
| `RATE_LIMIT_PER_MINUTE` | 单客户端本地限流 |
| `LOG_LEVEL/SHUTDOWN_TIMEOUT_SECONDS` | 结构化日志与优雅停机 |

## 本地开发与验证

```bash
go work sync
go build ./backend/...
go vet ./backend/...
go test ./backend/...
go test -race ./backend/...
npm --prefix frontend ci
npm --prefix frontend run test
npm --prefix frontend run typecheck
npm --prefix frontend run build


docker compose config --quiet
```

`runtime_smoke.json` 只描述 `20528` 端口的 SQLite 内存启动方式，不保存验收结果。真实构建、PostgreSQL、API、Browser、截图和停服证据写入 `output/execution.md`。

## 排错

- Compose 服务未变为 healthy：执行 `docker compose logs db backend frontend`，优先检查 PostgreSQL DSN、JWT 密钥长度和 Nginx 代理。
- 返回 `CUE_DEPENDENCY_CYCLE`：查看 `error.details.evidence_path`，它包含闭合循环路径；修改草稿依赖并重新走复核/锁定。
- 返回 `CUE_NOT_LOCKED`：所选 Cue 仍是草稿、待审或仅批准状态，需安全复核员锁定该明确版本。
- 返回 `BLOCKER_RUN_NOT_SUBMITTABLE`：打开推演证据表，按规则编号、设备和时间窗口修正新 Cue 版本；历史运行不会被覆盖。
- 返回 409 版本冲突：刷新实体后基于最新 `version` 或 `rule_version` 重试，不要复用旧表单版本。

## License

MIT
