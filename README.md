# timber-stage-qualifier

`timber-stage-qualifier` 是面向水下考古保护实验室的饱水木材浸渍阶段转换资格服务。它通过版本化 JSON HTTP API 管理处理批次建档、基线冻结、过程观测、偏差暂停、纠正恢复、独立复核、证书封存与审计校验。

服务使用本地 SQLite 持久化批次聚合、不可覆盖观测、偏差、复核、幂等响应、证书和摘要链审计事件。所有写请求必须在请求体中提供 `request_id` 与 `expected_revision`，并通过 `X-Actor-ID` 请求头声明操作者。终态批次只读，批准批次会生成 SHA-256 封存的规范 JSON 证书。

## 构建

```bash
go build ./cmd/server
```

## 运行

默认监听 `127.0.0.1:19081`，数据库文件默认为 `timber-stage.db`：

```bash
go run ./cmd/server
```

可以通过 `-addr` 和 `-db` 指定回环监听地址与 SQLite 路径：

```bash
go run ./cmd/server -addr=127.0.0.1:19082 -db=./laboratory.db
```

也可以设置 `PORT`，服务会绑定 `127.0.0.1:<PORT>`。显式 `-addr` 的优先级高于 `PORT`。服务拒绝非回环监听地址和低于 `1024` 的端口。

## API

主要端点如下：

- `POST /api/v1/treatment-batches`：建立草稿批次。
- `POST /api/v1/treatment-batches/{batch_id}/baseline`：冻结浸渍基线。
- `POST /api/v1/treatment-batches/{batch_id}/observations`：提交观测并执行确定性规则评估。
- `POST /api/v1/treatment-batches/{batch_id}/deviations/{deviation_id}/correction`：登记偏差原因与纠正措施。
- `POST /api/v1/treatment-batches/{batch_id}/recovery`：批准进入连续恢复窗口。
- `POST /api/v1/treatment-batches/{batch_id}/reviews`：由独立复核员作出批准或拒绝裁定。
- `GET /api/v1/treatment-batches/{batch_id}`：查询批次详情与开放偏差。
- `GET /api/v1/treatment-batches/{batch_id}/audit`：查询并校验摘要链审计时间线。
- `GET /api/v1/treatment-batches/{batch_id}/certificate`：下载批准批次的规范证书。
- `GET /api/v1/treatment-batches/{batch_id}/certificate/verify`：复算证书和审计根完整性。
- `GET /readyz`：检查数据库迁移和连接状态。

请求体最大为 1 MiB，未知 JSON 字段会被拒绝。重复 `request_id` 仅在请求指纹完全一致时重放首次结果；不同指纹会返回冲突。修订不一致返回 `revision_conflict`，不会追加事件。

## 测试与自检

运行全部回归测试：

```bash
go test ./...
```

运行有界真实 HTTP 自检：

```bash
go run ./cmd/server -self-check -addr=127.0.0.1:19081
```

自检使用临时 SQLite 数据库，实际启动回环 HTTP 服务，完成异常暂停、纠正恢复、独立批准、证书封存和完整性校验后主动退出并清理临时数据。
