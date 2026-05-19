# 接口说明

## 文档范围

本文档基于以下内容整理：

1. `route/route.go` 中的实际路由注册
2. `api/web.go`、`api/scan.go` 中的控制器实现
3. 相关模型、服务层校验逻辑与测试代码

当前服务地址默认是：

- 本地访问：[http://127.0.0.1:8081](http://127.0.0.1:8081)

## 返回格式

绝大多数 JSON 接口统一返回：

```json
{
  "code": 200,
  "data": {},
  "msg": "ok"
}
```

规则：

- 成功时 `code` 一般等于 HTTP 状态码
- 失败时 `code` 也直接等于 HTTP 状态码
- 主要错误信息在 `msg`
- 更详细错误通常在 `data.error`

示例：

```json
{
  "code": 400,
  "data": {
    "error": "startPath is empty"
  },
  "msg": "创建任务失败"
}
```

## 鉴权说明

系统存在两套鉴权方式。

### 1. 旧接口 `/img/*`

使用 HTTP Basic Auth。

默认配置来自根目录 `config.yaml`：

- 用户名：`admin`
- 密码：`admin@123`

### 2. Web 管理台接口 `/api/*`

登录方式：

- 先调用 `POST /api/auth/login`
- 登录成功后服务端下发 Cookie：`img_process_session`
- 后续 `/api/*` 接口依赖该 Cookie

注意：

- Session 存在于当前进程内存中，服务重启后会失效
- `/api/auth/login` 不需要登录
- 其余 `/api/*` 路由默认都需要登录

未登录返回示例：

```json
{
  "code": 401,
  "data": {
    "authenticated": false
  },
  "msg": "未登录"
}
```

## 通用查询参数

很多列表接口支持分页，分页规则由服务端统一处理：

- `page`：页码，默认 `1`
- `pageSize`：每页大小，默认 `20`

说明：

- 非法或非正数会回退默认值
- `GET /api/jobs/:id/logs` 如果未显式传 `pageSize`，代码会把默认值从 20 提高到 200

## 扫描参数模型 `DoScanImgArg`

以下接口会直接或间接使用扫描参数：

- `POST /img/scan`
- `GET /img/scan`
- `POST /api/jobs`
- `POST /api/schedules`
- `PUT /api/schedules/:id`

字段如下：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `deleteShow` | `boolean` | 是否展示删除候选 |
| `moveFileShow` | `boolean` | 是否展示移动候选 |
| `modifyDateShow` | `boolean` | 是否展示修改拍摄时间候选 |
| `renameFileShow` | `boolean` | 是否展示重命名候选 |
| `md5Show` | `boolean` | 是否展示重复文件候选 |
| `deleteAction` | `boolean` | 是否直接执行删除动作 |
| `moveFileAction` | `boolean` | 是否直接执行移动动作 |
| `modifyDateAction` | `boolean` | 是否直接执行修改拍摄时间动作 |
| `renameFileAction` | `boolean` | 是否直接执行重命名动作 |
| `startPath` | `string` | 扫描根目录，必填且必须为已存在目录 |
| `startPathBak` | `string` | 备份目录，可选；如果传了也必须为已存在目录 |

补充规则：

- 若调用方未传部分字段，服务端会用配置默认值补齐
- `startPath` 为空会被补成本地配置中的默认扫描目录
- `startPathBak` 为空会被补成配置中的默认备份目录
- 如果最终 `startPath` 不存在或不是目录，接口返回 400

## 接口清单

### 1. 登录接口

#### `POST /api/auth/login`

用途：Web 管理台登录。

鉴权：无需登录。

请求体：

```json
{
  "username": "admin",
  "password": "admin"
}
```

字段说明：

| 字段 | 必填 | 说明 |
| --- | --- | --- |
| `username` | 是 | 登录用户名 |
| `password` | 是 | 登录密码 |

成功响应：

```json
{
  "code": 200,
  "data": {
    "username": "admin",
    "authenticated": true
  },
  "msg": "登录成功"
}
```

失败场景：

- JSON 不合法或缺字段 -> `400`
- 用户名或密码错误 -> `401`

#### `POST /api/auth/logout`

用途：退出登录并清理 Session Cookie。

鉴权：需要登录。

成功响应：

```json
{
  "code": 200,
  "data": {
    "authenticated": false
  },
  "msg": "退出成功"
}
```

#### `GET /api/auth/me`

用途：获取当前登录用户信息。

鉴权：需要登录。

成功响应：

```json
{
  "code": 200,
  "data": {
    "username": "admin",
    "authenticated": true
  },
  "msg": "ok"
}
```

### 2. 任务接口

#### `GET /api/jobs`

用途：查询任务列表。

鉴权：需要登录。

查询参数：

| 参数 | 类型 | 说明 |
| --- | --- | --- |
| `page` | `int` | 页码 |
| `pageSize` | `int` | 每页条数 |
| `status` | `string` | 任务状态过滤 |
| `source` | `string` | 任务来源过滤 |
| `keyword` | `string` | 关键字过滤 |
| `hasAction` | `boolean` 字符串 | 是否包含真实执行动作，仅识别 `"true"` 为真 |
| `startCreated` | `RFC3339 datetime` | 创建时间起点，需要和 `endCreated` 成对出现 |
| `endCreated` | `RFC3339 datetime` | 创建时间终点，需要和 `startCreated` 成对出现 |

成功响应字段：

| 字段 | 说明 |
| --- | --- |
| `list` | 任务列表 |
| `total` | 总记录数 |

单个任务对象主要字段：

| 字段 | 说明 |
| --- | --- |
| `id` | 任务 ID |
| `jobUuid` | 任务 UUID |
| `scanUuid` | 扫描 UUID |
| `source` | 来源 |
| `status` | 状态 |
| `scheduleId` | 来源计划 ID |
| `queueAt` | 入队时间 |
| `startAt` | 开始时间 |
| `endAt` | 结束时间 |
| `currentPhase` | 当前阶段 |
| `lastHeartbeatAt` | 最近心跳 |
| `processedCount` | 已处理数量 |
| `totalCount` | 总数量 |
| `totalFolderCount` | 总目录数，从 `summary_json` 中提取 |
| `hasAction` | 是否包含真实动作 |
| `scanArgs` | 解析后的扫描参数对象 |
| `summary` | 解析后的汇总对象 |
| `pendingActionCount` | 待执行动作数 |
| `executedActionCount` | 已执行动作数 |
| `artifactPath` | 产物路径 |
| `errorMessage` | 错误信息 |
| `createdAt` | 创建时间 |
| `updatedAt` | 更新时间 |

#### `POST /api/jobs`

用途：创建新任务。

鉴权：需要登录。

请求体：

```json
{
  "displayName": "手工扫描",
  "source": "manual",
  "scheduleId": null,
  "scanArgs": {
    "startPath": "/data/pic-lab",
    "md5Show": true
  }
}
```

字段说明：

| 字段 | 必填 | 说明 |
| --- | --- | --- |
| `displayName` | 否 | 当前控制器未实际使用 |
| `source` | 否 | 任务来源，空时默认 `manual` |
| `scheduleId` | 否 | 计划 ID，计划触发场景可传 |
| `scanArgs` | 是 | 扫描参数对象 |

成功响应：

- HTTP 状态码：`201`
- `data.job` 为任务对象

失败场景：

- 请求 JSON 不合法 -> `400`
- `startPath` 无效、Cron 等校验类错误 -> `400`
- 其他内部错误 -> `500`

#### `GET /api/jobs/:id`

用途：查询单个任务详情。

鉴权：需要登录。

路径参数：

| 参数 | 说明 |
| --- | --- |
| `id` | 任务 ID，必须为正整数 |

成功响应：

- `data.job` 为任务对象

失败场景：

- `id` 非法 -> `400`
- 任务不存在 -> `404`

#### `GET /api/jobs/:id/events`

用途：分页查询任务事件流。

鉴权：需要登录。

参数：

- 路径参数 `id`
- 分页参数 `page`、`pageSize`

成功响应：

```json
{
  "code": 200,
  "data": {
    "list": [],
    "total": 0
  },
  "msg": "ok"
}
```

列表项直接对应 `scan_event` 表结构。

#### `GET /api/jobs/:id/logs`

用途：分页查询任务日志。

鉴权：需要登录。

参数：

- 路径参数 `id`
- 分页参数 `page`、`pageSize`

说明：

- 默认 `pageSize` 会被服务端提升到 200

成功响应：

- `data.list`：日志列表
- `data.total`：总数

列表项直接对应 `scan_job_log` 表结构。

#### `GET /api/jobs/:id/action-items`

用途：查询任务动作明细。

鉴权：需要登录。

参数：

| 参数 | 类型 | 说明 |
| --- | --- | --- |
| `id` | 路径参数 | 任务 ID |
| `tab` | `string` | 标签页，默认 `pending` |
| `type` | `string` | 动作类型过滤 |
| `status` | `string` | 动作状态过滤 |
| `keyword` | `string` | 关键字过滤 |
| `page` | `int` | 页码 |
| `pageSize` | `int` | 每页大小 |

成功响应：

```json
{
  "code": 200,
  "data": {
    "list": [],
    "total": 0,
    "counts": {},
    "groupedCounts": {}
  },
  "msg": "ok"
}
```

字段说明：

| 字段 | 说明 |
| --- | --- |
| `list` | 动作明细列表 |
| `total` | 当前查询条件下总数 |
| `counts` | 当前 tab 下的动作统计 |
| `groupedCounts` | 按待执行/已执行/错误分组的动作统计 |

动作明细对象主要结构：

| 字段 | 说明 |
| --- | --- |
| `id` | 动作 ID |
| `actionType` | 动作类型 |
| `objectType` | 对象类型 |
| `sourcePath` | 源路径 |
| `targetPath` | 目标路径 |
| `reasonCode` | 原因编码 |
| `reasonText` | 原因说明 |
| `stage` | 阶段 |
| `status` | 状态 |
| `discoveredAt` | 发现时间 |
| `executedAt` | 执行时间 |
| `errorMessage` | 错误信息 |
| `metadataJson` | 原始元数据 JSON |
| `duplicateGroup` | 重复分组 |
| `detail` | 普通动作详情 |
| `pair` | 重复项对比预览 |

说明：

- 普通动作会返回 `detail`
- 重复删除动作会返回 `pair`

#### `GET /api/jobs/:id/action-preview`

用途：预览动作相关图片文件，直接返回文件流，不是 JSON。

鉴权：需要登录。

查询参数：

| 参数 | 必填 | 说明 |
| --- | --- | --- |
| `itemId` | 是 | 动作 ID |
| `slot` | 否 | 预览槽位，默认 `source` |

`slot` 可选值：

- `source`：普通动作的源文件
- `pair_a`：重复项 A
- `pair_b`：重复项 B

响应特点：

- 成功时返回文件内容
- 会设置 `Cache-Control: private, max-age=60`
- 会设置 `Content-Disposition: inline`

失败场景：

- `id` 或 `itemId` 非法 -> `400`
- 文件不存在 -> `404`
- 路径越权、动作不属于任务、没有可预览路径 -> `400`

#### `GET /api/jobs/:id/stream`

用途：通过 SSE 实时订阅任务状态和新增事件。

鉴权：需要登录。

响应格式：

- `Content-Type: text/event-stream`
- 事件名可能为 `job`、`event`、`error`

行为说明：

- 服务端每秒轮询一次任务状态和新事件
- 任务状态变化时推送 `job`
- 新事件到达时逐条推送 `event`
- 当任务进入完成态时，流会自动结束

完成态包括：

- `succeeded`
- `failed`
- `interrupted`
- `skipped`

### 3. 动作执行接口

#### `POST /api/jobs/:id/actions/delete-duplicates`

用途：根据历史删除清单，对任务中的待处理重复文件批量执行删除。

鉴权：需要登录。

成功响应：

```json
{
  "code": 200,
  "data": {
    "jobId": 1
  },
  "msg": "重复文件删除已执行"
}
```

约束：

- 任务必须已成功完成，否则会失败
- 该接口会结合旧产物目录中的 `dump_delete_list` 执行

#### `POST /api/jobs/:id/action-items/:itemId/delete`

用途：执行单条删除类动作。

鉴权：需要登录。

支持动作类型：

- `delete`
- `delete_empty_dir`

成功响应：

- `data.jobId`
- `data.itemId`

失败场景：

- `id` 或 `itemId` 非法 -> `400`
- 动作不属于该任务 -> `400`
- 动作不是待执行删除动作 -> `400`
- 文件系统执行失败 -> `500`

#### `POST /api/jobs/:id/action-items/:itemId/delete-duplicate`

用途：执行单条重复项删除动作，并允许指定删除 A 侧还是 B 侧。

鉴权：需要登录。

请求体：

```json
{
  "side": "A"
}
```

字段说明：

| 字段 | 必填 | 说明 |
| --- | --- | --- |
| `side` | 是 | `A` 或 `B`，不区分大小写 |

规则：

- `A`：删除系统推荐删除的一侧，通常是 `sourcePath`
- `B`：删除保留侧，属于人工覆盖系统建议

成功响应：

```json
{
  "code": 200,
  "data": {
    "jobId": 1,
    "itemId": 2,
    "side": "A"
  },
  "msg": "重复项删除已执行"
}
```

失败场景：

- JSON 不合法 -> `400`
- `side` 非 `A/B` -> `400`
- 动作不属于任务 -> `400`
- 动作不是待执行重复删除动作 -> `400`
- 删除路径为空 -> `400`

#### `POST /api/jobs/:id/action-items/:itemId/modify-shoot-time`

用途：执行单条拍摄时间修正动作。

鉴权：需要登录。

请求体：无

规则：

- 目标日期优先取动作元数据中的 `targetDate`
- 若没有则退回 `minDate`
- 服务端最终会把日期格式转成 EXIF 写入格式 `YYYY:MM:DD 00:00:00`

成功响应：

```json
{
  "code": 200,
  "data": {
    "jobId": 1,
    "itemId": 2
  },
  "msg": "拍摄时间已变更"
}
```

失败场景：

- 动作不是待执行 `modify_time` -> `400`
- 目标日期为空 -> `400`
- 目标日期格式非法 -> `400`

#### `POST /api/jobs/:id/action-items/:itemId/move`

用途：执行单条移动动作。

鉴权：需要登录。

请求体：无

规则：

- 目标路径优先取元数据中的 `targetPath`
- 若没有则退回动作表 `target_path`

成功响应：

```json
{
  "code": 200,
  "data": {
    "jobId": 1,
    "itemId": 2
  },
  "msg": "位置已变更"
}
```

失败场景：

- 动作不是待执行 `move` -> `400`
- 目标路径为空 -> `400`

#### `POST /api/jobs/:id/action-items/:itemId/rename`

用途：执行单条重命名动作。

鉴权：需要登录。

请求体：无

规则与移动接口类似，只是业务语义不同。

成功响应：

```json
{
  "code": 200,
  "data": {
    "jobId": 1,
    "itemId": 2
  },
  "msg": "文件名已变更"
}
```

失败场景：

- 动作不是待执行 `rename` -> `400`
- 目标路径为空 -> `400`

#### `POST /api/jobs/:id/actions/delete-all`

用途：批量执行任务下所有待删除类动作。

鉴权：需要登录。

成功响应：

```json
{
  "code": 200,
  "data": {
    "jobId": 1,
    "count": 12
  },
  "msg": "删除类动作已全部执行"
}
```

说明：

- 仅批量执行删除类动作
- 成功后会写入一条生命周期事件

### 4. 计划任务接口

#### `GET /api/schedules`

用途：查询计划列表。

鉴权：需要登录。

查询参数：

| 参数 | 类型 | 说明 |
| --- | --- | --- |
| `page` | `int` | 页码 |
| `pageSize` | `int` | 每页大小 |
| `enabled` | `boolean` 字符串 | 启用状态过滤，仅 `"true"` 识别为真 |

成功响应：

- `data.list`：计划列表
- `data.total`：总数

#### `POST /api/schedules`

用途：创建计划。

鉴权：需要登录。

请求体：

```json
{
  "name": "每日扫描",
  "enabled": true,
  "mode": "daily",
  "cronExpr": "",
  "scheduleConfig": "{\"hour\":2,\"minute\":30}",
  "scanArgs": {
    "startPath": "/data/pic-lab"
  }
}
```

字段说明：

| 字段 | 必填 | 说明 |
| --- | --- | --- |
| `name` | 是 | 计划名称 |
| `enabled` | 否 | 是否启用 |
| `timezone` | 否 | 兼容字段；为空时服务端从运行环境读取时区，前端默认不再传入 |
| `mode` | 否 | `hourly`、`daily`、`weekly`、`monthly`、`custom` |
| `cronExpr` | 否 | 高级 Cron 模式下直接使用 |
| `scheduleConfig` | 否 | 非高级 Cron 模式用来生成 Cron 的 JSON 字符串 |
| `scanArgs` | 是 | 扫描参数 |

校验规则：

- 非 `custom` 模式时，`cronExpr` 可为空，服务端会根据 `scheduleConfig` 生成
- `hourly` 模式使用 `scheduleConfig.minute` 生成 `minute * * * *`
- `weekly` 模式必须在 `scheduleConfig` 中提供 `weekdays`
- `custom` 模式下最终 `cronExpr` 不能为空
- `scanArgs.startPath` 必须有效
- 管理台会按当前表单值实时展示对应的 5 段 Cron 预览，不额外包含 `CRON_TZ=`

成功响应：

- HTTP 状态码：`201`
- `data.schedule`：计划对象

#### `PUT /api/schedules/:id`

用途：更新计划。

鉴权：需要登录。

请求体与创建计划一致。

成功响应：

- `data.schedule`：更新后的计划对象

失败场景：

- `id` 非法 -> `400`
- 参数错误 -> `400`
- 计划不存在或内部错误 -> `500`

#### `DELETE /api/schedules/:id`

用途：删除计划。

鉴权：需要登录。

成功响应：

```json
{
  "code": 200,
  "data": {
    "id": 1
  },
  "msg": "计划已删除"
}
```

#### `POST /api/schedules/:id/enable`

用途：启用计划。

鉴权：需要登录。

成功响应：

- `data.schedule`：启用后的计划对象

#### `POST /api/schedules/:id/disable`

用途：停用计划。

鉴权：需要登录。

成功响应：

- `data.schedule`：停用后的计划对象

说明：

- 停用时服务端会把 `nextRunAt` 清空

#### `POST /api/schedules/:id/run`

用途：立即触发指定计划执行一次。

鉴权：需要登录。

成功响应：

- `data.job`：新建任务对象

说明：

- 即使计划本身是定时任务，这个接口也会立刻创建一个 `source = schedule` 的任务

### 5. 系统状态接口

#### `GET /api/system/status`

用途：返回当前运行配置和默认扫描参数。

鉴权：需要登录。

成功响应结构：

```json
{
  "code": 200,
  "data": {
    "server": {
      "httpPort": "8081",
      "startPath": "/data/pic-lab",
      "startPathBak": "",
      "poolSize": 8,
      "imgCache": false,
      "sqlDebug": false,
      "scanDefaults": {
        "startPath": "/data/pic-lab",
        "startPathBak": "",
        "deleteShow": true,
        "moveFileShow": true,
        "modifyDateShow": false,
        "renameFileShow": true,
        "md5Show": true,
        "deleteAction": false,
        "moveFileAction": false,
        "modifyDateAction": false,
        "renameFileAction": false
      }
    }
  },
  "msg": "ok"
}
```

主要用途：

- 前端初始化默认扫描参数
- 检查当前容器实际加载的运行配置

### 6. 旧扫描接口

这些接口在 `/img` 分组下，使用 Basic Auth。

#### `POST /img/scan`
#### `GET /img/scan`

用途：兼容旧调用方式，当前内部已改为“创建异步任务”。

鉴权：Basic Auth。

请求方式：

- `POST` 有 body 时优先按 JSON 绑定
- 其他情况按 query 参数绑定

可传字段：见上文 `DoScanImgArg`

成功响应：

```json
{
  "code": 202,
  "data": {
    "ret": "ok",
    "jobId": 123,
    "jobUuid": "job-uuid",
    "status": "pending"
  },
  "msg": "扫描任务下发成功"
}
```

失败场景：

- 参数绑定失败 -> `400`
- 创建任务失败 -> `500`

#### `POST /img/delete`
#### `DELETE /img/delete`
#### `GET /img/delete`

用途：兼容旧逻辑，按 `scanUuid` 找到历史删除清单后执行重复文件删除。

鉴权：Basic Auth。

参数：

| 参数 | 来源 | 必填 | 说明 |
| --- | --- | --- | --- |
| `scanUuid` | JSON 或 query | 是 | 扫描 UUID |

`scanUuid` 格式要求：

- 必须匹配：`^\d{4}-\d{2}-\d{2}-\d{2}-\d{2}-\d{2}_[A-Za-z0-9]+$`

成功响应：

```json
{
  "code": 200,
  "data": {
    "ret": "ok"
  },
  "msg": "删除任务执行完成"
}
```

失败场景：

- 缺少 `scanUuid` -> `400`
- `scanUuid` 格式非法 -> `400`
- 路径逃逸校验失败 -> `400`

## 前端路由补充

虽然不属于 API，但当前服务还托管了 SPA 页面：

- `GET /`：返回前端首页
- `GET /assets/*`：前端静态资源
- 非 `/api`、非 `/img` 的其他路径会回退到前端 `index.html`

这也是浏览器直接访问 `/login` 仍能打开页面的原因。

## 重要约束与注意点

1. `/api` 登录态完全依赖内存 Session，重启容器后需要重新登录。
2. `/img/*` 和 `/api/*` 是两套入口，前者偏兼容接口，后者是当前管理台主入口。
3. 任务和动作相关响应中，很多业务信息并非独立字段，而是从数据库 JSON 字段反序列化后返回。
4. `GET /api/jobs/:id/stream` 是长连接 SSE，不适合普通短请求调试方式。
5. 预览接口会严格限制文件路径必须落在扫描根目录或备份目录下，避免任意文件读取。
6. 定时计划的 `scheduleConfig` 实际上是“字符串形式的 JSON”，不是嵌套对象，这点前端或外部调用方容易踩坑。
