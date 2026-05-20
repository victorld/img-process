# 本地脚本说明

本仓库默认使用 macOS 本地模式运行，由 `monit` 守护 `bin/img-process-web`。Docker 只作为明确需要时的可选方式。

## 常用流程

修改代码后让 `http://127.0.0.1:8081/` 生效：

```bash
scripts/monit-local-web.sh restart
curl -I http://127.0.0.1:8081/
```

`restart` 会先完整构建，再重启 Web 服务；只改 Go 或只改前端时也可以使用同一个命令，避免服务继续使用旧二进制或旧 `web/dist`。

## 脚本职责

| 脚本 | 用途 |
| --- | --- |
| `scripts/build-local.sh` | 完整构建前端产物和本机 Go 二进制。会执行 `npm --prefix web ci`、`npm --prefix web run build` 和 `go build`。 |
| `scripts/monit-local-web.sh` | 管理本仓库自带的 `monit` 运行配置，支持 `start`、`stop`、`restart`、`reload`、`status`、`summary`、`validate`、`unmonitor`；其中 `restart` 会先完整构建本地前端产物和 Go 二进制。 |
| `scripts/start-local-web.sh` | 启动 Web 服务。无参数时前台启动；`daemon` 参数供 `monit` 后台启动，并写入 `log/local-web.pid` 和 `log/local-web.started`。 |
| `scripts/stop-local-web.sh` | 停止本仓库的 `bin/img-process-web` 进程，并清理本地 pid/start 标记。不会按端口杀掉非本项目进程。 |

## Monit 操作

首次启动或重建守护：

```bash
scripts/monit-local-web.sh validate
scripts/monit-local-web.sh start
scripts/monit-local-web.sh status
```

修改 `monit/img-process-web.monitrc` 后，需要重新加载运行时配置：

```bash
scripts/monit-local-web.sh reload
scripts/monit-local-web.sh start
```

停止本机守护和 Web 服务：

```bash
scripts/monit-local-web.sh stop
```

查看简要状态：

```bash
scripts/monit-local-web.sh summary
```

## 运行文件

- `bin/img-process-web`：本机 Web 服务二进制，由构建脚本生成。
- `web/dist/`：前端生产静态资源，由前端构建生成。
- `log/local-web.log`：Web 服务日志。
- `log/local-web.pid`：当前 Web 服务 pid。
- `log/local-web.started`：当前 Web 服务启动时间戳。
- `log/monit/`：运行时 `monit` 配置副本。
- `log/monit.log`、`log/monit.pid`、`log/monit.state`、`log/monit.id`：本仓库 `monit` 守护状态文件。

## 验收检查

```bash
scripts/monit-local-web.sh summary
curl -I http://127.0.0.1:8081/
```

如果修改了前端，还要确认页面行为或 `dist/assets` 资源哈希已经更新。
