# Aittco / new-api 上游升级记录

- 核对日期：2026-10-18。
- 官方发布：`v1.0.0-rc.45`，2026-10-17 发布。
- 官方提交：`6cf0d90b5ff27f901697dddd61e0c28aba8efefc`。
- 升级前本地提交：`ec2cc00815b6f7a63ce2b0bae10b062e9f437658`。
- 本地版本标记：`v1.0.0-rc.45+aittco`。
- 本次只做本地代码升级，不推送远端、不部署、不迁移运行中的数据库。

## 保留的本地功能

- Aittco 首页、登录/注册外观、Logo 与本地 `/docs` 页面。
- Root 管理员自定义 API key、key 片段搜索与模型映射编辑修复。
- 原生 Gemini 图片请求、Visionary 渠道及 Base64/URL-safe 图片兼容。
- 图片异步生成/编辑、并发限制、任务明细、COS/S3 结果存储和 Webhook。
- 原工作区六种语言中 300 个未提交译值，迁移至 `web/src/i18n/locales/`；报告在新目录重新生成。
- new-api、QuantumNous 的上游身份、许可证与署名。

## 上游结构变化

前端入口从 `web/default/` 迁移为 `web/`，语言文件从
`web/default/src/i18n/locales/` 迁移为 `web/src/i18n/locales/`。
`relaykit/` 与 `tokenkit/` 是独立 Go 模块，需分别验证。
构建要求以各模块的 `go.mod` 和 `web/package.json` 为准。

旧定制 Visionary 渠道编号 `58` 与官方 AdvancedCustom 冲突，当前编号为 `64`。
仅在检测到旧异步图片表且尚无官方任务插件表时迁移旧编号，并保存迁移标记；
新建的官方 AdvancedCustom 渠道不会再次转换。

## 图片任务兼容修复

普通图片任务保存冻结计费信息，数据库预扣与任务额度快照在同一事务内完成；
退款不修改尚未计入的用户/渠道 usage。Redis 开启批量写入时保留待落库的额度差量，
避免检查点主动清缓存后重新读到旧余额。

官方图片插件子任务独占计费与退款；本地父任务通过持久化提交屏障、
`Attempt` 和 `WorkerNode` 校验避免旧 worker 重复提交。
已提交但没有保存子任务的父任务到期进入失败终态，不自动重复提交；
有子任务的父任务继续读取子任务结果。通用超时扫描不处理本地图片父任务。

超时失败事务锁定并读取最新任务，保留并发增加的预扣额后再退款；
旧 `Attempt` 或不同 `WorkerNode` 不能将新任务改为失败。相应回归覆盖完整退款和旧 worker 隔离。

Windows 上渠道亲和性统计测试中的时间戳标识可能碰撞。测试夹具改用 UUID 隔离，
三个用例各重复 30 次通过；未改变生产统计逻辑。

## 原工作区保护

集成前对原 10 个未提交文件逐一核对 SHA256；原始补丁、文件副本和清单
保留在升级 worktree 父目录。六种语言的 300 个译值已逐值迁移进新路径。
集成时额外保留 Git stash，不向已删除的 `web/default/` 重复套用旧补丁。

## 实际部署前

1. 停止写入并完整备份主数据库、独立日志库、配置和图片存储。代码回退不等于数据库回退。
2. 在数据库副本上先验证启动迁移，并再次启动确认幂等性。
3. MySQL/PostgreSQL 旧库的 `users.quota`、`used_quota`、`aff_quota`、`aff_history` 必须先检查是否为 64 位整数。官方启动会拒绝旧 32 位列；本次没有更改生产 schema。扩列操作需保留实际列的默认值、NULL 和其他约束。不要用 `SKIP_64BIT_QUOTA_SCHEMA_CHECK` 跳过真实类型问题。
4. 保留 `SESSION_SECRET`、数据库/Redis 连接、COS/S3 配置及已有密钥。不要把生产 `.env` 提交 Git。
5. 本地构建命令为 `docker compose --env-file .env -f docker-compose.prod.yml.example build`。启动属于后续部署操作，本次未执行。
6. 上游新增登录/权限与计费机制，部署副本应检查管理员登录、普通用户权限、钱包/订阅扣费、同步/异步图片及历史任务查询。

Redis 通用额度缓存在批量待写入期间自然过期或被其他路径删除的情况，仍受上游通用缓存恢复语义约束；本次修复避免图片检查点主动制造该窗口。

## 验证结果

- 根 Go 模块：`GOWORK=off go test -p 2 ./... -count=1 -timeout=300s -json`、`go vet -p 2 ./...` 通过。
- Windows/amd64 和 Linux/amd64 的 `CGO_ENABLED=0 go build` 通过；Linux 首次依赖下载超时，换源重试成功。
- `relaykit/`、`tokenkit/` 独立模块的 test、vet、build 通过。
- 真实 SQLite、MySQL 8.0、PostgreSQL 15 的重复启动迁移、Visionary 编号迁移、图片预扣/退款、worker 所有权和任务结算回归通过。
- 前端分批跑完 202/202 个唯一测试文件，2,380 项测试通过，无失败、跳过或未完成文件。Windows 上最终批次使用 `--pool=threads --maxWorkers=1 --testTimeout=60000`，仓库默认超时未改。
- 前端全量 lint 仍有上游既有问题（157 errors、66 warnings）；本地定制差异无 lint error 或 warning。版权头检查有 25 处既有发现，未删除或改写受保护的署名。
- 六种语言原有 300 个未提交译值在新路径中全部保留。

## 后续同步

当前 `origin` 指向 QuantumNous/new-api，`aigclee` 指向自有仓库。
后续先备份本地未提交改动，创建隔离分支，fetch 官方 tag 后 merge；不要用
`reset --hard` 覆盖定制代码。冲突解决后，分别运行根模块、relaykit、tokenkit 的
test/vet/build 和前端 typecheck/build，再验证三种数据库。

本次详细命令、测试结果与原始翻译补丁保存在升级 worktree 的父目录；
最终验证结果以该目录的 `upgrade-logs/` 为准。
