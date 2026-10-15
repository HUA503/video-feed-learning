# 视频 Feed 流系统

基于 Go、Gin、GORM、MySQL、Redis 和 RabbitMQ 的短视频后端学习项目。

支持用户注册登录、视频信息发布、点赞、评论、关注、关注 Feed 和热门视频列表，并逐步实现消息可靠投递、事件去重、缓存降级及 Docker Compose 部署。

## 浏览器页面

启动后打开 [映流首页](http://localhost:8080/)，即可在浏览器中使用注册登录、最新视频、热门视频、关注 Feed、视频播放、点赞、评论、关注和链接发布功能。

- 页面使用 HTML、CSS、原生 JavaScript，通过同源请求调用 Gin API，不需要安装 Node.js。
- 登录信息保存在当前标签页的 `sessionStorage` 中，刷新后会重新验证登录状态。
- 发布功能接收视频 URL。`example.com/demo.mp4` 等示例地址无法播放，需要填写真实可访问的视频链接。
- 搜索仅筛选当前已经加载的视频；继续加载后可以筛选更多内容。
- 未提供有效封面时，页面使用带有“暂无封面”标记的默认插画。
- 他人的账号目前显示为“创作者 #用户ID”，当前登录账号显示用户名。

前端文件位于 `web/index.html` 和 `web/assets/`。Gin 提供首页与 `/assets` 静态资源入口，Dockerfile 将页面一起复制进运行镜像。

## 技术栈

| 技术 | 用途 |
|---|---|
| Go / Gin | 编写后端服务和 HTTP 接口 |
| GORM / MySQL | 数据访问与业务数据持久化 |
| Redis | 热榜、列表缓存和 token 撤销记录 |
| RabbitMQ | 异步传递热度事件 |
| Docker Compose | 构建、启动和管理各个服务 |

## 已实现功能

### 用户与视频

- 用户注册、密码哈希校验和 JWT 登录鉴权。
- 退出登录，撤销当前 token。
- 视频信息发布、视频详情和视频列表。
- 点赞、取消点赞。
- 发布评论、查询评论和删除自己的评论。
- 关注、取消关注。

### Feed 与热榜

- 根据关注关系查询视频 Feed。
- 基于发布时间和视频 ID 的游标分页。
- 累计热榜与 24 小时滑动窗口热榜。
- 定期清理过期事件的热度贡献。
- 热门列表缓存和 singleflight 请求合并。
- 窗口热榜不可用时，尝试返回最新视频列表。

### 消息与运行管理

- 使用 MySQL 事务同时保存业务记录和 Outbox 事件。
- 后台分批发布待发送事件。
- RabbitMQ 发送确认、消费确认、重试与死信处理。
- 使用事件 ID 和 Redis Lua 脚本处理重复投递。
- RabbitMQ 断线恢复。
- HTTP 服务和后台任务优雅退出。
- Redis AOF 持久化与 Docker 数据卷。
- Docker Compose 部署及容器健康检查。

## 项目结构

```text
.
├── auth/          # JWT 生成与解析
├── cache/         # Redis 连接初始化
├── database/      # MySQL 连接初始化
├── handler/       # 接收 HTTP 请求、读取参数、返回响应
├── middleware/    # 登录鉴权和 token 撤销检查
├── model/         # 数据模型及相关结构体
├── pagination/    # 游标编码与解码
├── queue/         # RabbitMQ 连接、消息发布与消费相关逻辑
├── repository/    # MySQL、Redis 数据访问
├── router/        # HTTP 路由注册
├── service/       # 业务逻辑
├── web/           # 浏览器页面、样式、交互脚本与图标
├── main.go        # 初始化依赖、启动服务和后台任务
├── Dockerfile
├── docker-compose.yaml
├── .dockerignore
└── .env.example
```

## 当前功能范围

视频发布目前保存标题、描述、视频 URL 和封面 URL。

浏览器演示页面已实现。实际视频文件上传和转码尚未实现。

## 本地启动

以下命令使用 Windows PowerShell，在项目根目录执行。

### 1. 准备运行环境

- 安装并启动 Docker Desktop，使用 Linux 容器。
- 确保 Docker Compose 可以使用。
- 如果正在 GoLand 中运行本项目，先停止 Go 服务，释放 8080 端口。

### 2. 准备环境变量

首次运行且项目根目录还没有 `.env` 时，复制模板：

```powershell
Copy-Item -LiteralPath ".env.example" -Destination ".env"
```

编辑 `.env`，将 `JWT_SECRET` 替换为自己生成的随机密钥。

其余连接配置使用模板中的容器地址：

| 环境变量 | 用途 |
|---|---|
| `JWT_SECRET` | JWT 签名与验证密钥 |
| `MYSQL_DSN` | MySQL 连接信息，容器地址为 `mysql:3306` |
| `REDIS_ADDR` | Redis 地址，容器地址为 `redis:6379` |
| `RABBITMQ_URL` | RabbitMQ 连接信息，容器地址为 `rabbitmq:5672` |

Compose 根据 `env_file` 读取 `.env`，把变量传入 Go 容器。

### 3. 构建并启动

```powershell
docker compose up -d --build
```

第一次需要下载基础镜像和 Go 依赖。

查看服务状态：

```powershell
docker compose ps
```

稍等后，api、mysql、redis、rabbitmq 应均显示 `healthy`。

如果遇到 Docker Hub 网络超时，并且本地已有所需镜像，可以使用：

```powershell
docker compose up -d --no-build --pull never
```

此方式使用已有镜像。修改 Go 源码后，需要重新构建镜像才能运行新代码。

### 4. 首次配置 RabbitMQ 死信策略

应用启动时会声明正常队列和死信队列，但当前代码不会自动创建死信转发策略。

新的 RabbitMQ 数据环境需要执行以下初始化操作：

```powershell
$mqAuth = [Convert]::ToBase64String(
    [System.Text.Encoding]::UTF8.GetBytes("guest:guest")
)

$policyBody = @{
    pattern = '^video\.hot\.score$'
    "apply-to" = "queues"
    priority = 10
    definition = @{
        "dead-letter-exchange" = ""
        "dead-letter-routing-key" = "video.hot.score.dead"
    }
} | ConvertTo-Json -Depth 5

Invoke-RestMethod `
    -Uri "http://localhost:18072/api/policies/%2F/hot-score-dlx" `
    -Method Put `
    -Headers @{ Authorization = "Basic $mqAuth" } `
    -ContentType "application/json" `
    -Body $policyBody `
    -ErrorAction Stop
```

查看策略：

```powershell
docker compose exec -T rabbitmq rabbitmqctl list_policies
```

应看到名为 `hot-score-dlx` 的策略。

该策略将正常队列 `video.hot.score` 中被拒绝且不重新入队的消息，转发到 `video.hot.score.dead`。

策略保存在 RabbitMQ 数据卷中。已有该策略的环境，正常重启后继续使用；重新创建全新的数据环境时，需要再次初始化。

### 5. 访问服务

| 服务 | 地址 |
|---|---|
| Go API | `http://localhost:8080` |
| 健康接口 | `http://localhost:8080/healthz` |
| RabbitMQ 管理页面 | `http://localhost:18072` |
| MySQL 本机连接地址 | `127.0.0.1:3307` |
| Redis 本机连接地址 | `127.0.0.1:6379` |

当前本地开发配置的 RabbitMQ 管理账号为 `guest`，密码为 `guest`。

检查 HTTP 服务：

```powershell
Invoke-RestMethod `
    -Uri "http://localhost:8080/healthz" `
    -Method Get `
    -ErrorAction Stop
```

预期返回 `status: ok`。

全新数据环境没有预置的测试账号和视频，需要先注册用户、发布视频，再体验互动和 Feed 功能。

## 常用运行命令

查看容器状态：

```powershell
docker compose ps
```

查看 Go 服务最近的日志：

```powershell
docker compose logs --tail 80 api
```

修改 Go 代码后，重新构建并更新 Go 容器：

```powershell
docker compose up -d --build api
```

停止整套服务：

```powershell
docker compose stop
```

使用已有本地镜像重新启动：

```powershell
docker compose up -d --no-build --pull never
```

## 项目文档

- [接口文档](docs/API.md)
- [架构与关键流程](docs/ARCHITECTURE.md)
