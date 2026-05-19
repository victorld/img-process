# 数据库说明

## 文档范围

本文档基于以下两部分信息整理：

1. 代码中的 Gorm 模型与建表注册逻辑
2. 当前本地运行中的 MySQL `img` 库实际表结构

当前项目启动时会自动注册以下 8 张表：

- `gis_database`
- `img_database`
- `img_record`
- `scan_action_item`
- `scan_event`
- `scan_job`
- `scan_job_log`
- `scan_schedule`

说明：

- 数据库名：`img`
- 当前运行配置连接到 `127.0.0.1:33060`
- 当前库内未定义外键约束，表之间的关联主要靠应用层维护
- 通用主键均为 `id bigint unsigned auto_increment`
- 通用时间字段均为 `created_at`、`updated_at`

## 当前数据概览

基于 2026-05-09 本地运行库统计：

| 表名 | 当前记录数 | 说明 |
| --- | ---: | --- |
| `gis_database` | 0 | 经纬度地址缓存 |
| `img_database` | 0 | 图片拍摄时间和地理信息缓存 |
| `img_record` | 14 | 旧版扫描汇总记录 |
| `scan_action_item` | 468 | 扫描产生的候选动作/已执行动作 |
| `scan_event` | 123 | 任务事件流 |
| `scan_job` | 11 | 扫描任务主表 |
| `scan_job_log` | 584 | 任务日志 |
| `scan_schedule` | 0 | 定时计划 |

## 通用字段

除个别 JSON 文本和业务字段外，所有表都包含以下字段：

| 字段名 | 类型 | 可空 | 说明 |
| --- | --- | --- | --- |
| `id` | `bigint unsigned` | 否 | 主键 |
| `created_at` | `datetime` | 是 | 创建时间 |
| `updated_at` | `datetime` | 是 | 更新时间 |

## 表结构

### 1. `scan_job`

用途：扫描任务主表，是 Web 管理台任务列表、任务详情、计划执行记录的核心表。

索引：

- 主键：`PRIMARY(id)`
- 唯一索引：`idx_scan_job_job_uuid(job_uuid)`
- 普通索引：`idx_scan_job_scan_uuid(scan_uuid)`
- 普通索引：`idx_scan_job_schedule_id(schedule_id)`
- 普通索引：`idx_scan_job_source(source)`
- 普通索引：`idx_scan_job_status(status)`

字段说明：

| 字段名 | 类型 | 可空 | 索引 | 说明 |
| --- | --- | --- | --- | --- |
| `id` | `bigint unsigned` | 否 | PK | 主键 |
| `created_at` | `datetime` | 是 |  | 创建时间 |
| `updated_at` | `datetime` | 是 |  | 更新时间 |
| `job_uuid` | `varchar(64)` | 是 | 唯一 | 任务唯一标识，接口返回的 `jobUuid` |
| `scan_uuid` | `varchar(64)` | 是 | 普通 | 扫描批次 UUID，兼容旧扫描产物目录定位 |
| `source` | `varchar(32)` | 是 | 普通 | 任务来源，当前代码常量有 `manual`、`schedule` |
| `status` | `varchar(32)` | 是 | 普通 | 任务状态：`pending`、`running`、`succeeded`、`failed`、`interrupted`、`skipped` |
| `schedule_id` | `bigint unsigned` | 是 | 普通 | 来源计划 ID，手工任务为空 |
| `queue_at` | `datetime` | 是 |  | 入队时间 |
| `start_at` | `datetime` | 是 |  | 开始执行时间 |
| `end_at` | `datetime` | 是 |  | 执行结束时间 |
| `current_phase` | `varchar(64)` | 是 |  | 当前阶段，例如 `starting`、扫描中、后处理等 |
| `last_heartbeat_at` | `datetime` | 是 |  | 最近心跳时间，用于任务执行存活感知 |
| `processed_count` | `bigint` | 是 |  | 已处理数量 |
| `total_count` | `bigint` | 是 |  | 总处理数量 |
| `has_action` | `tinyint(1)` | 是 |  | 是否包含真实执行动作，0/1 |
| `scan_args` | `longtext` | 是 |  | 扫描参数 JSON |
| `summary_json` | `longtext` | 是 |  | 任务汇总 JSON，前端会从中取目录数等统计 |
| `artifact_path` | `varchar(512)` | 是 |  | 任务产物路径 |
| `error_message` | `text` | 是 |  | 失败错误信息 |

补充：

- 这是新 Web 管理台的数据核心，接口大多围绕本表展开。
- `scan_args` 通常序列化自 `DoScanImgArg`。
- `summary_json` 为动态结构，不适合直接写死 SQL 解析逻辑。

### 2. `scan_action_item`

用途：记录一次扫描过程中识别出的候选动作，以及执行后的结果。前端“待执行动作”“已执行动作”“错误动作”均来自这里。

索引：

- 主键：`PRIMARY(id)`
- 普通索引：`idx_scan_action_item_job_id(job_id)`
- 普通索引：`idx_scan_action_item_action_type(action_type)`
- 普通索引：`idx_scan_action_item_source_path(source_path)`
- 普通索引：`idx_scan_action_item_stage(stage)`
- 普通索引：`idx_scan_action_item_status(status)`
- 普通索引：`idx_scan_action_item_duplicate_group(duplicate_group)`

字段说明：

| 字段名 | 类型 | 可空 | 索引 | 说明 |
| --- | --- | --- | --- | --- |
| `id` | `bigint unsigned` | 否 | PK | 主键 |
| `created_at` | `datetime` | 是 |  | 创建时间 |
| `updated_at` | `datetime` | 是 |  | 更新时间 |
| `job_id` | `bigint unsigned` | 是 | 普通 | 所属任务 ID，对应 `scan_job.id` |
| `action_type` | `varchar(64)` | 是 | 普通 | 动作类型：`delete`、`move`、`rename`、`modify_time`、`delete_empty_dir`、`delete_duplicate` |
| `object_type` | `varchar(32)` | 是 |  | 对象类型：`file` 或 `dir` |
| `source_path` | `varchar(512)` | 是 | 普通 | 源路径 |
| `target_path` | `varchar(1024)` | 是 |  | 目标路径，移动/重命名/重复对保留文件等场景会用到 |
| `reason_code` | `varchar(64)` | 是 |  | 原因编码 |
| `reason_text` | `text` | 是 |  | 原因说明 |
| `stage` | `varchar(32)` | 是 | 普通 | 阶段：`candidate`、`executed` |
| `status` | `varchar(32)` | 是 | 普通 | 状态：`pending`、`succeeded`、`failed`、`skipped` |
| `discovered_at` | `datetime` | 是 |  | 候选动作发现时间 |
| `executed_at` | `datetime` | 是 |  | 执行时间 |
| `error_message` | `text` | 是 |  | 执行失败或跳过原因 |
| `metadata_json` | `longtext` | 是 |  | 动作附加元数据 JSON，包含文件名、预览路径、目标日期、重复对比信息等 |
| `duplicate_group` | `varchar(128)` | 是 | 普通 | 重复文件分组标识 |

`metadata_json` 常见内容：

- 通用字段：`fileName`、`currentPath`、`targetPath`、`targetFileName`
- 时间相关：`dirDate`、`modifyDate`、`fileNameDate`、`shootDate`、`shootDateRaw`、`minDate`、`targetDate`
- 重复项相关：`keepPath`、`keepFileName`
- 手工执行审计：`executedDeleteSide`、`executedDeletePath`、`recommendedDeletePath`、`manualOverride`

### 3. `scan_event`

用途：任务事件流，用于任务详情页的实时过程展示，以及 SSE 推流接口。

索引：

- 主键：`PRIMARY(id)`
- 普通索引：`idx_scan_event_job_id(job_id)`
- 普通索引：`idx_scan_event_event_type(event_type)`
- 普通索引：`idx_scan_event_phase(phase)`

字段说明：

| 字段名 | 类型 | 可空 | 索引 | 说明 |
| --- | --- | --- | --- | --- |
| `id` | `bigint unsigned` | 否 | PK | 主键 |
| `created_at` | `datetime` | 是 |  | 创建时间 |
| `updated_at` | `datetime` | 是 |  | 更新时间 |
| `job_id` | `bigint unsigned` | 是 | 普通 | 所属任务 ID |
| `event_type` | `varchar(32)` | 是 | 普通 | 事件类型：`lifecycle`、`phase`、`progress`、`error`、`artifact` |
| `phase` | `varchar(64)` | 是 | 普通 | 阶段名 |
| `level` | `varchar(16)` | 是 |  | 级别，如 `info`、`warn`、`error` |
| `title` | `varchar(255)` | 是 |  | 标题 |
| `message` | `text` | 是 |  | 详细消息 |
| `related_path` | `varchar(1024)` | 是 |  | 关联文件路径 |
| `payload_json` | `longtext` | 是 |  | 附加结构化数据 |

### 4. `scan_job_log`

用途：记录任务执行日志，和 `scan_event` 相比更偏底层日志明细。

索引：

- 主键：`PRIMARY(id)`
- 普通索引：`idx_scan_job_log_job_id(job_id)`
- 普通索引：`idx_scan_job_log_level(level)`
- 普通索引：`idx_scan_job_log_phase(phase)`

字段说明：

| 字段名 | 类型 | 可空 | 索引 | 说明 |
| --- | --- | --- | --- | --- |
| `id` | `bigint unsigned` | 否 | PK | 主键 |
| `created_at` | `datetime` | 是 |  | 创建时间 |
| `updated_at` | `datetime` | 是 |  | 更新时间 |
| `job_id` | `bigint unsigned` | 是 | 普通 | 所属任务 ID |
| `level` | `varchar(16)` | 是 | 普通 | 日志级别 |
| `phase` | `varchar(64)` | 是 | 普通 | 阶段 |
| `message` | `text` | 是 |  | 日志消息 |
| `payload_json` | `longtext` | 是 |  | 结构化日志负载 |

### 5. `scan_schedule`

用途：定时计划表，用于“每小时/每天/每周/每月/高级 Cron”自动发起扫描任务。

索引：

- 主键：`PRIMARY(id)`
- 普通索引：`idx_scan_schedule_enabled(enabled)`

字段说明：

| 字段名 | 类型 | 可空 | 索引 | 说明 |
| --- | --- | --- | --- | --- |
| `id` | `bigint unsigned` | 否 | PK | 主键 |
| `created_at` | `datetime` | 是 |  | 创建时间 |
| `updated_at` | `datetime` | 是 |  | 更新时间 |
| `name` | `varchar(128)` | 是 |  | 计划名称 |
| `enabled` | `tinyint(1)` | 是 | 普通 | 是否启用，0/1 |
| `timezone` | `varchar(64)` | 是 |  | 时区，默认从服务运行环境读取；为空时使用本地时间 |
| `mode` | `varchar(32)` | 是 |  | 计划模式：`hourly`、`daily`、`weekly`、`monthly`、`custom` |
| `cron_expr` | `varchar(128)` | 是 |  | Cron 表达式 |
| `schedule_config` | `longtext` | 是 |  | 计划配置 JSON，非自定义模式用它生成 Cron |
| `scan_args` | `longtext` | 是 |  | 扫描参数 JSON |
| `last_run_at` | `datetime` | 是 |  | 最近运行时间 |
| `next_run_at` | `datetime` | 是 |  | 下次运行时间 |
| `last_job_id` | `bigint unsigned` | 是 |  | 最近一次触发的任务 ID |
| `last_job_status` | `varchar(32)` | 是 |  | 最近任务状态 |

`schedule_config` JSON 结构：

```json
{
  "hour": 2,
  "minute": 30,
  "weekdays": [1, 5],
  "dayOfMonth": 1
}
```

字段含义：

- `hour`：小时
- `minute`：分钟，每小时模式使用它表示每小时第几分钟执行
- `weekdays`：每周模式使用，0-6 表示周日到周六
- `dayOfMonth`：每月模式使用，默认最小按 1 处理

代码中的 Cron 生成规则：

- `hourly` -> `minute * * * *`
- `daily` -> `minute hour * * *`
- `weekly` -> `minute hour * * weekday_csv`
- `monthly` -> `minute hour dayOfMonth * *`
- `custom` -> 直接使用 `cron_expr`
- 前端会按当前表单值实时展示对应的 5 段 Cron 预览

### 6. `img_record`

用途：旧版扫描汇总表，保留了每次扫描的大量统计结果。新 Web 管理台主要使用 `scan_job` 体系，但历史功能和兼容逻辑仍保留此表。

索引：

- 主键：`PRIMARY(id)`

字段说明：

| 字段名 | 类型 | 可空 | 说明 |
| --- | --- | --- | --- |
| `id` | `bigint unsigned` | 否 | 主键 |
| `created_at` | `datetime` | 是 | 创建时间 |
| `updated_at` | `datetime` | 是 | 更新时间 |
| `scan_args` | `varchar(1000)` | 是 | 扫描参数字符串 |
| `file_total` | `int` | 是 | 文件总数 |
| `file_total_bak` | `int` | 是 | 备份目录文件总数 |
| `dir_total` | `int` | 是 | 目录总数 |
| `dir_total_bak` | `int` | 是 | 备份目录总数 |
| `start_date` | `datetime` | 是 | 开始时间 |
| `use_time` | `int` | 是 | 用时 |
| `base_path` | `varchar(255)` | 是 | 主目录 |
| `base_path_bak` | `varchar(255)` | 是 | 备份目录 |
| `suffix_map` | `varchar(255)` | 是 | 文件后缀统计 |
| `suffix_map_bak` | `varchar(255)` | 是 | 备份目录后缀统计 |
| `year_map` | `varchar(255)` | 是 | 年份统计 |
| `year_map_bak` | `varchar(255)` | 是 | 备份目录年份统计 |
| `bak_new_file_cnt` | `int` | 是 | 备份新增文件数 |
| `bak_delete_file_cnt` | `int` | 是 | 备份删除文件数 |
| `bak_new_file` | `text` | 是 | 备份新增文件明细 |
| `bak_delete_file` | `text` | 是 | 备份删除文件明细 |
| `file_date_cnt` | `int` | 是 | 有时间信息文件数 |
| `delete_file_cnt` | `int` | 是 | 待删除文件数 |
| `modify_date_file_cnt` | `int` | 是 | 待修改文件时间数 |
| `move_file_cnt` | `int` | 是 | 待移动文件数 |
| `rename_file_cnt` | `int` | 是 | 待重命名文件数 |
| `shoot_date_mismatch_file_cnt` | `int` | 是 | 拍摄日期不一致文件数 |
| `shoot_date_null_file_cnt` | `int` | 是 | 拍摄日期为空文件数 |
| `shoot_date_earlier_file_cnt` | `int` | 是 | 拍摄日期更早文件数 |
| `empty_dir_cnt` | `int` | 是 | 空目录数 |
| `dump_file_cnt` | `int` | 是 | 重复文件数 |
| `exif_err_cnt` | `int` | 是 | EXIF 解析错误数 |
| `exif_date_name_set` | `text` | 是 | EXIF 相关异常统计 |
| `is_complete` | `int` | 是 | 是否完整 |
| `remark` | `text` | 是 | 备注 |

### 7. `img_database`

用途：图片拍摄时间和位置信息缓存表。

索引：

- 主键：`PRIMARY(id)`
- 唯一索引：`img_key(img_key)`

字段说明：

| 字段名 | 类型 | 可空 | 索引 | 说明 |
| --- | --- | --- | --- | --- |
| `id` | `bigint unsigned` | 否 | PK | 主键 |
| `created_at` | `datetime` | 是 |  | 创建时间 |
| `updated_at` | `datetime` | 是 |  | 更新时间 |
| `img_key` | `varchar(255)` | 是 | 唯一 | 图片唯一键 |
| `shoot_date` | `varchar(255)` | 是 |  | 拍摄时间 |
| `loc_num` | `varchar(255)` | 是 |  | 经纬度 |
| `loc_addr` | `text` | 是 |  | 地址信息 |
| `loc_street` | `text` | 是 |  | 街道信息 |
| `state` | `int` | 是 |  | 状态，注释定义为 1 表示启用 |
| `remark` | `text` | 是 |  | 备注 |

### 8. `gis_database`

用途：经纬度地址反查缓存表。

索引：

- 主键：`PRIMARY(id)`
- 唯一索引：`loc_num_key(loc_num)`

字段说明：

| 字段名 | 类型 | 可空 | 索引 | 说明 |
| --- | --- | --- | --- | --- |
| `id` | `bigint unsigned` | 否 | PK | 主键 |
| `created_at` | `datetime` | 是 |  | 创建时间 |
| `updated_at` | `datetime` | 是 |  | 更新时间 |
| `loc_num` | `varchar(100)` | 是 | 唯一 | 经纬度键 |
| `loc_addr` | `text` | 是 |  | 地址信息 |
| `loc_street` | `text` | 是 |  | 街道信息 |
| `loc_json` | `text` | 是 |  | 地理服务原始响应 |

## 关联关系

当前库没有外键，应用层默认维护以下逻辑关联：

- `scan_action_item.job_id` -> `scan_job.id`
- `scan_event.job_id` -> `scan_job.id`
- `scan_job_log.job_id` -> `scan_job.id`
- `scan_job.schedule_id` -> `scan_schedule.id`
- `scan_schedule.last_job_id` -> `scan_job.id`

## 业务枚举

### 任务来源 `scan_job.source`

- `manual`：手工触发
- `schedule`：计划触发

### 任务状态 `scan_job.status`

- `pending`：待执行
- `running`：执行中
- `succeeded`：成功
- `failed`：失败
- `interrupted`：中断
- `skipped`：跳过

### 动作类型 `scan_action_item.action_type`

- `delete`：删除文件
- `move`：移动文件
- `rename`：重命名文件
- `modify_time`：修改拍摄时间
- `delete_empty_dir`：删除空目录
- `delete_duplicate`：删除重复项

### 动作对象类型 `scan_action_item.object_type`

- `file`
- `dir`

### 动作阶段 `scan_action_item.stage`

- `candidate`：候选动作
- `executed`：已执行动作

### 动作状态 `scan_action_item.status`

- `pending`
- `succeeded`
- `failed`
- `skipped`

### 事件类型 `scan_event.event_type`

- `lifecycle`
- `phase`
- `progress`
- `error`
- `artifact`

### 计划模式 `scan_schedule.mode`

- `hourly`
- `daily`
- `weekly`
- `monthly`
- `custom`

## 设计观察

1. 新旧两套扫描数据结构并存。`img_record` 更像历史扫描统计表，`scan_job`/`scan_action_item`/`scan_event`/`scan_job_log` 是当前 Web 管理台主链路。
2. 大量扩展信息以 JSON 文本保存在 `scan_job.scan_args`、`scan_job.summary_json`、`scan_action_item.metadata_json`、`scan_schedule.schedule_config` 中，说明系统更偏“弱结构化扩展”。
3. 虽然存在逻辑关联，但数据库未加外键，因此清理脏数据、迁移数据时需要由应用或脚本自己保证一致性。
4. 与扫描执行相关的实时展示更依赖 `scan_event` 和 `scan_job_log`，不是只看 `scan_job.status`。
