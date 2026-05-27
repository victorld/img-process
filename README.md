# img_process

项目简介：个人管理照片和视频的处理工具，主要是解决以下痛点问题
1. 照片实际日期和照片所在的目录的日志不一致
2. 相似或者相同的照片去重
3. 照片库夹杂一些非照片的文件删除
4. 照片库的数据统计分析
5. 使用第三方去重工具性能低下



导入照片我一般用lightroom工具，照片和视频的归档路径如下：

- pic-new
  - 2003
    - 2003-01
      - 2003-01-01 
        - xxx.jpg
        - xxx.jpg
        - xxx.jpg
      - 2003-01-02
        - xxx.jpg
        - xxx.jpg
        - xxx.jpg
      - 2003-01-03
    - 2003-02
      - 2003-02-01
      - 2003-02-02
      - 2003-02-03
  - 2004
    - 2004-01
      - 2004-01-01
      - 2004-01-02
      - 2004-01-03
    
支持的特性：

1. 支持删除无效照片或视频，如：.xxx
1. 支持删除空目录 
1. 支持多种方式计算文件最小日期（文件夹日期、照片EXIF中的拍摄日期、照片文件名日期、文件修改日期中计算最小值）
1. 支持以最小日期修改文件修改日期
1. 支持以最小日期把文件移动到最小日期的目录中
1. 支持查询和实际操作分开运行
1. 支持查重文件分析
1. 支持多协程的文件处理，提高效率
1. 支持文件智能选择，如果在同一目录中出现不同名的相同文件，可以智能删除重复文件
1. 支持直接运行、web api、rpc方式调用
1. 支持主备目录的照片差异对比
1. 支持gps地理坐标缓存，支持照片和视频的拍摄时间缓存
1. 支持 Web 管理台查看扫描历史、实时过程和计划任务


处理性能：  

实际测试40000张照片、视频（495G）整体处理一遍大概需要15秒左右时间

配置说明：

1. 根目录 `config.yaml` 是仓库默认启动配置，Docker 构建和 `docker-compose.yml` 默认都会使用它。
2. macOS 本机常驻服务脚本默认读取根目录 `config.yaml`；如果本机照片目录和容器挂载路径不同，需要先把 `scanArgs.StartPath` 和 `bak.StartPathBak` 调整为本机可访问路径。
3. `basic.SqlDebug` 默认关闭，只有排查数据库问题时再临时开启

## Web 管理台

当前仓库已经包含 Web 管理台能力，后端继续使用 Go + Gin + Gorm + MySQL，前端使用 React + TypeScript + Vite + Ant Design。

Web 管理台支持：

1. 查看扫描历史任务列表
2. 查看单次扫描的实时事件、待执行动作、已执行动作、错误和重复文件
3. 页面中发起扫描任务，默认参数来自当前配置
4. 配置并执行定时扫描计划

主要接口：

1. `POST /api/auth/login`
2. `GET /api/jobs`
3. `POST /api/jobs`
4. `GET /api/jobs/:id`
5. `GET /api/jobs/:id/events`
6. `GET /api/jobs/:id/action-items`
7. `GET /api/jobs/:id/stream`
8. `GET /api/schedules`
9. `POST /api/schedules`

## 运行约定

默认以 macOS 本机常驻 Web 服务作为项目的运行、验收和交付方式，并使用 `monit` 做进程守护，避免 Docker Desktop 对 SMB 挂载目录的二次文件共享开销。扫描任务仍通过 Web 管理台或应用内计划任务执行，不额外使用 cron 直接触发扫描。

每次修改后的标准收口步骤：

```bash
scripts/monit-local-web.sh restart
curl -I http://127.0.0.1:8081/
```

最低要求：

1. 本机 `bin/img-process-web` 已完成构建
2. `http://127.0.0.1:8081/` 可访问
3. `scripts/monit-local-web.sh status` 能看到 `img-process-web` 处于 monitored/running 状态
4. 如果修改了前端，需要确认首页资源哈希或页面行为已更新，而不是继续命中旧构建产物

## Changelist 维护约定

根目录的 `changelist.md` 是当前工作分支相对 `master` 的变更时间线，不是通用产品介绍。只要本分支新增了有效代码、配置、接口、部署或数据库改动，就必须同步更新这份文件。

更新规则：

1. 以 `master` 为基线，按提交时间顺序追加，不重写为按模块或按主题重排。
2. 每个时间线节点固定使用“时间 | commit短哈希 | 提交标题”。
3. 每个时间线节点下固定使用大纲层级挂两部分：
   `#### 功能变化`
   `#### 数据库变化`
4. `#### 数据库变化` 必须优先写字段级变更，推荐固定句式示例：
   `表 xxx：新增字段 a（类型/默认值/可空）；修改字段 b（旧值 -> 新值，是否回填，兼容性影响）；删除字段 c。`
5. 如果只改数据库连接、库初始化或部署方式，没有表结构变化，也要明确写“数据库连接配置变更”或“数据库初始化行为变更”，不要笼统写“数据库有调整”。
6. 如果某次提交没有数据库变化，也必须明确写“本次没有新增表、字段或数据库连接配置变化”。
7. 不再单独维护文末“数据库变化”汇总章节，数据库内容只保留在各自提交节点下。
8. `.playwright-cli`、临时日志、抓图、测试产物等调试文件不进入主时间线，只允许在附注中说明已排除。
9. 文案必须用中文，优先写用户可感知或系统行为层面的事实，不写空泛描述。

## 本地启动场景

适用场景：日常开发、前端/后端功能验收、需要直接访问 macOS 本机照片目录或 SMB 挂载目录。该模式由本机 `monit` 守护 `bin/img-process-web`，避免 Docker Desktop 对外部目录的二次文件共享开销。

详细脚本说明见 [scripts/README.md](scripts/README.md)。

配置文件：

1. 默认读取根目录 `config.yaml`。
2. 本机模式下，`scripts/start-local-web.sh` 会把 `IMG_PROCESS_CONFIG` 固定为根目录 `config.yaml`。
3. 本机专用配置中的 `scanArgs.StartPath` 和 `bak.StartPathBak` 应填写 macOS 可直接访问的绝对路径。

构建并重启本机服务：

```bash
scripts/monit-local-web.sh restart
```

查看服务状态：

```bash
scripts/monit-local-web.sh summary
curl -I http://127.0.0.1:8081/
```

`monit/img-process-web.monitrc` 使用仓库内 `log/` 目录保存守护状态，不依赖系统级 `/opt/homebrew/etc/monitrc`。

前端开发模式仍可使用：

```bash
cd web
npm install
npm run dev
```

前端开发服务器默认代理到 `http://localhost:8081`。

## Docker 启动场景

适用场景：验证容器镜像、部署到 Docker 环境、确认仓库默认配置可以从零启动。该模式使用 `docker-compose.yml` 构建多阶段镜像，并把宿主机照片目录挂载到容器内路径。

项目已提供 `docker-compose.yml` 和多阶段构建镜像：

```bash
docker compose up -d --build app
```

Docker 默认配置：

1. 根目录 `config.yaml` 会复制进镜像，并通过 compose 挂载到 `/app/config.yaml`。
2. 默认数据库连接为 `host.docker.internal:33060`，需要宿主机或其他容器已经对外提供 MySQL。
3. 默认扫描目录为 `/data/pic-new`，对应 `docker-compose.yml` 中的宿主机照片目录挂载。
4. 默认备份目录为 `/data/pic-new-bak`，对应 `docker-compose.yml` 中的备份目录挂载；如果宿主机无法挂载该目录，需要先调整 compose 中的 volume。

查看服务状态并验证访问：

```bash
docker compose ps
curl -I http://127.0.0.1:8081/
```
