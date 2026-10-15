# 接口说明

## 服务地址

本地访问地址：`http://localhost:8080`。

以下表格中的路径均基于这个地址。

## 登录鉴权

标记为“需要登录”的接口，需要携带 HTTP 请求头：

```http
Authorization: Bearer <登录接口返回的 token>
```

其中 `<登录接口返回的 token>` 需要替换为实际 token，去掉尖括号。

服务端会验证 JWT 签名、过期时间、必需字段，并查询 Redis 中的撤销记录。

- token 缺失、无效、过期或已撤销：返回 `401`。
- 无法查询 token 撤销状态：返回 `503`。
- 退出登录撤销当前 token，其他独立签发的 token 不受影响。

## 接口总览

| 功能 | 方法 | 路径 | 需要登录 |
|---|---|---|---|
| HTTP 健康检查 | GET | `/healthz` | 否 |
| 用户注册 | POST | `/api/v1/users` | 否 |
| 用户登录 | POST | `/api/v1/login` | 否 |
| 退出登录 | POST | `/api/v1/logout` | 是 |
| 当前用户 ID | GET | `/api/v1/me` | 是 |
| 发布视频信息 | POST | `/api/v1/videos` | 是 |
| 查询视频列表 | GET | `/api/v1/videos` | 否 |
| 查询视频详情 | GET | `/api/v1/videos/:id` | 否 |
| 点赞视频 | POST | `/api/v1/videos/:id/like` | 是 |
| 取消点赞 | DELETE | `/api/v1/videos/:id/like` | 是 |
| 发布评论 | POST | `/api/v1/videos/:id/comments` | 是 |
| 查询视频评论 | GET | `/api/v1/videos/:id/comments` | 否 |
| 删除自己的评论 | DELETE | `/api/v1/comments/:id` | 是 |
| 关注用户 | POST | `/api/v1/users/:id/follow` | 是 |
| 取消关注 | DELETE | `/api/v1/users/:id/follow` | 是 |
| 查询关注 Feed | GET | `/api/v1/feed/following` | 是 |
| 查询热门视频 | GET | `/api/v1/feed/hot` | 否 |

## 路径中的 ID

`:id` 表示路径参数，调用时替换为实际数字。

| 路径形式 | ID 的含义 |
|---|---|
| `/api/v1/videos/:id` | 视频 ID |
| `/api/v1/videos/:id/like` | 被点赞的视频 ID |
| `/api/v1/videos/:id/comments` | 被评论的视频 ID |
| `/api/v1/comments/:id` | 要删除的评论 ID |
| `/api/v1/users/:id/follow` | 要关注或取消关注的用户 ID |

例如：

- `/api/v1/videos/2/comments`：操作视频 2 的评论。
- `/api/v1/comments/5`：操作评论 5。

点赞人、评论人和关注发起人的用户 ID，由认证中间件从当前 token 中取得。

## Feed 查询参数

### 关注 Feed

接口：`GET /api/v1/feed/following`。

- `page_size`：每页数量，默认 `10`，允许 `1` 到 `50`。
- `cursor`：分页游标，首次请求不传。
- 响应中的 `has_more` 表示是否还有下一页。
- 有下一页时，将响应的 `next_cursor` 作为下一次请求的 `cursor`。

### 热门视频

接口：`GET /api/v1/feed/hot`。

- `limit`：返回数量上限，默认 `10`，允许 `1` 到 `50`。
- `source` 为 `window_hot`：结果来自窗口热榜查询路径。
- `source` 为 `latest`：结果来自最新视频降级路径。
- `degraded` 表示本次是否使用降级路径。
- 实际返回数量可能少于 `limit`。

## PowerShell 调用示例：账号与视频

运行示例时，在同一个 PowerShell 终端中按顺序执行，后面的命令会使用前面保存的变量。

### 1. 准备地址和账号信息

```powershell
$apiBase = "http://localhost:8080"

$authorBody = @{
    username = "demo_author_001"
    password = "videoFeed123"
} | ConvertTo-Json
```

### 2. 注册作者账号

同一个用户名只需注册一次。已经注册过时，直接进入下一步登录。

```powershell
$registeredAuthor = Invoke-RestMethod `
    -Uri "$apiBase/api/v1/users" `
    -Method Post `
    -ContentType "application/json" `
    -Body $authorBody `
    -ErrorAction Stop

$registeredAuthor | ConvertTo-Json -Depth 6
```

成功响应包含 `id`、`username` 和 `created_at`。

### 3. 登录并保存身份信息

```powershell
$authorLogin = Invoke-RestMethod `
    -Uri "$apiBase/api/v1/login" `
    -Method Post `
    -ContentType "application/json" `
    -Body $authorBody `
    -ErrorAction Stop

$authorID = $authorLogin.id

$authorHeaders = @{
    Authorization = "Bearer $($authorLogin.token)"
}
```

`$authorID` 是作者的用户 ID，`$authorHeaders` 保存当前 token 对应的请求头。

查询当前用户：

```powershell
Invoke-RestMethod `
    -Uri "$apiBase/api/v1/me" `
    -Method Get `
    -Headers $authorHeaders `
    -ErrorAction Stop
```

响应中的 `user_id` 应与 `$authorID` 一致。

### 4. 发布视频信息

下面的 `video_url` 是占位地址。实际播放时，需要替换为可访问的视频地址。

```powershell
$videoBody = @{
    title = "接口文档示例视频"
    description = "演示发布视频信息"
    video_url = "https://example.com/demo.mp4"
    cover_url = ""
} | ConvertTo-Json

$videoBytes = [System.Text.Encoding]::UTF8.GetBytes($videoBody)

$publishedVideo = Invoke-RestMethod `
    -Uri "$apiBase/api/v1/videos" `
    -Method Post `
    -ContentType "application/json; charset=utf-8" `
    -Headers $authorHeaders `
    -Body $videoBytes `
    -ErrorAction Stop

$videoID = $publishedVideo.id

$publishedVideo | ConvertTo-Json -Depth 6
```

发布成功后：

- `id` 是新视频的 ID，保存于 `$videoID`。
- `user_id` 是发布者的用户 ID，应与 `$authorID` 一致。
- 每执行一次发布请求，都会创建一条新视频记录。

### 5. 查询刚发布的视频

```powershell
Invoke-RestMethod `
    -Uri "$apiBase/api/v1/videos/$videoID" `
    -Method Get `
    -ErrorAction Stop |
    ConvertTo-Json -Depth 6
```

### 6. 查询视频列表

```powershell
Invoke-RestMethod `
    -Uri "$apiBase/api/v1/videos?page=1&page_size=10" `
    -Method Get `
    -ErrorAction Stop |
    ConvertTo-Json -Depth 6
```

视频列表响应包含：

- `items`：当前页的视频列表。
- `total`：视频总数。
- `page`：当前页码。
- `page_size`：每页数量。

普通视频列表使用页码分页；关注 Feed 使用 `cursor` 游标分页。

## PowerShell 调用示例：关注与互动

本节接着上一节执行，使用同一个 PowerShell 终端中的：

- `$apiBase`：服务地址。
- `$authorID`：作者的用户 ID。
- `$videoID`：刚发布的视频 ID。

### 1. 准备观看者账号

```powershell
$viewerBody = @{
    username = "demo_viewer_001"
    password = "videoFeed123"
} | ConvertTo-Json
```

首次使用时注册一次；已经注册过，直接进行登录。

```powershell
Invoke-RestMethod `
    -Uri "$apiBase/api/v1/users" `
    -Method Post `
    -ContentType "application/json" `
    -Body $viewerBody `
    -ErrorAction Stop |
    ConvertTo-Json -Depth 6
```

### 2. 登录观看者账号

```powershell
$viewerLogin = Invoke-RestMethod `
    -Uri "$apiBase/api/v1/login" `
    -Method Post `
    -ContentType "application/json" `
    -Body $viewerBody `
    -ErrorAction Stop

$viewerID = $viewerLogin.id

$viewerHeaders = @{
    Authorization = "Bearer $($viewerLogin.token)"
}
```

从这里开始，需要登录的互动请求使用 `$viewerHeaders`，代表由观看者发起操作。

### 3. 关注作者

```powershell
Invoke-RestMethod `
    -Uri "$apiBase/api/v1/users/$authorID/follow" `
    -Method Post `
    -Headers $viewerHeaders `
    -ErrorAction Stop
```

成功返回“关注成功”。

URL 中的 `$authorID` 是被关注的人；请求头中的 token 对应关注发起人。

重复关注会返回 `409`。已经关注时，可直接进行后续步骤。

### 4. 点赞视频

```powershell
Invoke-RestMethod `
    -Uri "$apiBase/api/v1/videos/$videoID/like" `
    -Method Post `
    -Headers $viewerHeaders `
    -ErrorAction Stop
```

成功返回“点赞成功”。

重复点赞会返回 `409`。已经点赞时，可直接进行后续步骤。

### 5. 发表评论

```powershell
$commentBody = @{
    content = "这是观看者发表的中文评论"
} | ConvertTo-Json

$commentBytes = [System.Text.Encoding]::UTF8.GetBytes($commentBody)

$createdComment = Invoke-RestMethod `
    -Uri "$apiBase/api/v1/videos/$videoID/comments" `
    -Method Post `
    -ContentType "application/json; charset=utf-8" `
    -Headers $viewerHeaders `
    -Body $commentBytes `
    -ErrorAction Stop

$commentID = $createdComment.id

$createdComment | ConvertTo-Json -Depth 6
```

成功响应中：

- `id` 是评论 ID，保存于 `$commentID`。
- `user_id` 是评论人的 ID，应与 `$viewerID` 一致。
- `video_id` 是被评论的视频 ID，应与 `$videoID` 一致。

每次成功发布评论，都会创建一条新的评论记录。

### 6. 查询视频评论

```powershell
Invoke-RestMethod `
    -Uri "$apiBase/api/v1/videos/$videoID/comments" `
    -Method Get `
    -ErrorAction Stop |
    ConvertTo-Json -Depth 6
```

### 7. 查询观看者的关注 Feed

```powershell
$followingFeed = Invoke-RestMethod `
    -Uri "$apiBase/api/v1/feed/following?page_size=1" `
    -Method Get `
    -Headers $viewerHeaders `
    -ErrorAction Stop

$followingFeed | ConvertTo-Json -Depth 6
```

该 Feed 根据观看者的关注关系，查询被关注用户发布的视频。

如果有下一页，使用返回的游标继续查询：

```powershell
if ($followingFeed.has_more) {
    $cursor = [Uri]::EscapeDataString($followingFeed.next_cursor)

    Invoke-RestMethod `
        -Uri "$apiBase/api/v1/feed/following?page_size=1&cursor=$cursor" `
        -Method Get `
        -Headers $viewerHeaders `
        -ErrorAction Stop |
        ConvertTo-Json -Depth 6
}
```

如果被关注用户只有一条视频，返回 `has_more: false` 是正常结果。

### 8. 查询热门视频

```powershell
Invoke-RestMethod `
    -Uri "$apiBase/api/v1/feed/hot?limit=10" `
    -Method Get `
    -ErrorAction Stop |
    ConvertTo-Json -Depth 6
```

点赞和评论对应的热度通过 Outbox、RabbitMQ 和消费者异步更新。

热门列表还有短期缓存，因此操作成功后，榜单可能稍后才反映变化。示例视频也需要达到对应排名，才会出现在前 10 条中。

## PowerShell 调用示例：取消操作与退出登录

本节接着前面的示例执行，继续使用同一个 PowerShell 终端。

### 1. 删除观看者发表的评论

```powershell
Invoke-RestMethod `
    -Uri "$apiBase/api/v1/comments/$commentID" `
    -Method Delete `
    -Headers $viewerHeaders `
    -ErrorAction Stop
```

成功返回 `204 No Content`，没有响应正文，终端没有显示内容是正常结果。

这里使用评论 ID `$commentID`，并使用评论发表者的 `$viewerHeaders`。

- 评论不存在：返回 `404`。
- 使用其他用户的身份删除：返回 `403`。

### 2. 取消点赞

```powershell
Invoke-RestMethod `
    -Uri "$apiBase/api/v1/videos/$videoID/like" `
    -Method Delete `
    -Headers $viewerHeaders `
    -ErrorAction Stop
```

成功返回 `204 No Content`，没有响应正文。

删除评论、取消点赞对应的热度变化，由后台通过 Outbox 和 RabbitMQ 异步处理。

### 3. 取消关注作者

```powershell
Invoke-RestMethod `
    -Uri "$apiBase/api/v1/users/$authorID/follow" `
    -Method Delete `
    -Headers $viewerHeaders `
    -ErrorAction Stop
```

成功返回 `204 No Content`。

尚未关注该作者时，返回 `404`。

取消关注后，再查询关注 Feed，该作者的视频就不再因这条关注关系被选入结果。

### 4. 观看者退出登录

```powershell
Invoke-RestMethod `
    -Uri "$apiBase/api/v1/logout" `
    -Method Post `
    -Headers $viewerHeaders `
    -ErrorAction Stop
```

成功返回：

```json
{
  "message": "退出登录成功"
}
```

### 5. 继续使用旧 token

保留当前 `$viewerHeaders`，再次查询当前用户：

```powershell
Invoke-RestMethod `
    -Uri "$apiBase/api/v1/me" `
    -Method Get `
    -Headers $viewerHeaders `
    -ErrorAction Stop
```

预期返回 `401`，PowerShell 会显示错误，响应内容为：

```json
{
  "error": "登录凭证已失效，请重新登录"
}
```

这是旧 token 被成功拦截的预期结果。

### 6. 重新登录

```powershell
$viewerLogin = Invoke-RestMethod `
    -Uri "$apiBase/api/v1/login" `
    -Method Post `
    -ContentType "application/json" `
    -Body $viewerBody `
    -ErrorAction Stop

$viewerHeaders = @{
    Authorization = "Bearer $($viewerLogin.token)"
}

Invoke-RestMethod `
    -Uri "$apiBase/api/v1/me" `
    -Method Get `
    -Headers $viewerHeaders `
    -ErrorAction Stop
```

重新登录后，用户 ID 保持不变，新 token 的独立编号会重新生成，可以继续访问需要登录的接口。

## 常见 HTTP 状态码

| 状态码 | 本项目中的典型含义 |
|---|---|
| 200 | 查询、登录或退出登录成功 |
| 201 | 注册、发布、点赞或关注成功 |
| 204 | 删除评论、取消点赞或取消关注成功，无响应正文 |
| 400 | 请求参数不合法 |
| 401 | 登录凭证无效，或账号密码错误 |
| 403 | 无权删除其他用户的评论 |
| 404 | 目标不存在，或尚未关注指定用户 |
| 409 | 重复点赞或重复关注 |
| 500 | 服务端处理失败 |
| 503 | 登录状态验证或 token 撤销操作暂时不可用 |