# TokenHub

智源（BAAI）内部统一模型接入平台，基于 [New API](https://github.com/QuantumNous/new-api) 二次开发。

本仓库是 New API 的**修改版**（Modified Version），按 AGPLv3 第 13 条公开对应源代码。

## 上游项目

| 项 | 值 |
|---|---|
| 原项目 | [QuantumNous/new-api](https://github.com/QuantumNous/new-api) |
| 同步基线 | `34003e9c2eed2c20e617f583fdfe54bf1bf42237`（2026-10-25） |
| 许可证 | GNU Affero General Public License v3.0（见 [LICENSE](./LICENSE)） |
| 上游说明 | 见 [NOTICE](./NOTICE) |

## 改动说明

<!-- 本段按 AGPLv3 §7(c) 要求标注修改内容；每次向上游同步后更新此处。 -->

### 1. 组织（部门）与部门额度池

新增部门层级，用于按部门管控额度分配。**额度语义为「分配池」**：部门维护一个充值池
（`organizations.quota`）与已分配总额（`organizations.allocated`），成员额度只能从池里分配，
不变式为 `allocated <= quota`。成员调用模型时仍然只扣个人余额，扣费主流程未改变。

涉及文件：

- `model/organization.go`（新增）：`Organization` 表、`AllocateQuota` / `ReleaseQuota` 原子原语、对账 `RecalcAllocated`
- `model/user.go`：`User` 新增 `organization_id` 字段；`EditWithTx` 在换部门时迁移额度归属
- `model/user_quota_adjustment.go`：管理员改额度时校验部门池
- `model/topup.go`：兑换码与在线支付到账时校验部门池（`creditTopUpQuota`）
- `model/main.go`：`Organization` 加入 AutoMigrate
- `controller/organization.go`（新增）：管理端 CRUD
- `router/api-router.go`：`/api/organizations` 路由（`handlePermissionRoute`，声明 Access Token scope）
- `service/authz/resources_organization.go`（新增）：`organization:read` / `organization:write` / `organization:member` 权限点
- `controller/audit.go`：新增部门操作审计模板
- 前端 `web/src/features/organizations/`、`web/src/routes/_authenticated/organizations/`

行为变化：

- 未归属部门的用户（`organization_id = 0`）行为与上游完全一致
- 部门池不足时，新增额度/兑换码/切换部门会被拒绝；已分配的余额不受影响
- 删除部门会解除成员归属，但不影响成员个人余额

### 2. 渠道公开名对用户可见

新增 `channels.public_name`，用于在用户侧用量日志中显示供应商名（如「中云」「FlagOS」）。
**留空表示该渠道对用户不可见**，可用于隐藏灰度或测试渠道。

涉及文件：

- `model/channel.go`：`Channel` 新增 `PublicName` 字段
- `model/log.go`：`formatUserLogs` 改为填充 `public_name`，并置零内部 `ChannelId`
- `controller/channel_authz.go`：`public_name` 归入非敏感可写字段
- 前端 `web/src/features/usage-logs/components/columns/common-logs-columns.tsx`：普通用户新增只读「Provider」列

行为变化（安全相关）：

- 用户侧日志不再下发内部渠道 ID 和内部渠道名，只显示 `public_name`
- `model/log_format_test.go`、`controller/relay_error_log_test.go` 的断言相应调整


## 保留声明

按 [NOTICE](./NOTICE) 中 AGPLv3 §7 附加条款的要求：

- 用户界面保留署名：`Frontend design and development by New API contributors.`
- 用户界面保留原项目链接：https://github.com/QuantumNous/new-api
- 不以原项目名义对外宣传本修改版

## 许可证

本项目整体遵循 AGPLv3，详见 [LICENSE](./LICENSE)。第三方依赖的许可证见 [THIRD-PARTY-LICENSES.md](./THIRD-PARTY-LICENSES.md)。

若你通过本平台使用服务，可以按 AGPLv3 第 13 条获取本服务的对应源代码，即本仓库。
