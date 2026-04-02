## Docker Compose 部署说明（与当前仓库实现一致）

### 1. 服务清单

当前 `docker-compose.yml` 启动以下 4 个服务：

1. `mongo`：MongoDB 8，数据持久化到 `./data/mongo`
2. `redis`：Redis 8（AOF 开启），数据持久化到 `./data/redis`
3. `backend`：Go + Gin API 服务，监听 `8080`
4. `frontend`：Vue 3 构建产物 + Nginx 提供静态站点，监听容器内 `80`，映射主机 `3000`

### 2. 访问地址

- 前端入口：`http://localhost:3000`
- 后端健康检查：`http://localhost:8080/health`
- 前端通过 Nginx 反向代理访问后端：`/api/* -> http://backend:8080/api/*`

> 前端代码中的 `baseURL` 为 `/api/v1`，在 Compose 场景下由 `frontend/nginx.conf` 转发到后端服务。

### 3. 一键启动

在仓库根目录执行：

- 启动：`docker compose up -d --build`
- 查看状态：`docker compose ps`
- 查看日志：`docker compose logs -f`
- 停止并移除容器：`docker compose down`

### 4. 环境变量说明（backend）

`backend` 服务由 compose 注入主要配置：

- `SERVER_PORT=8080`
- `MONGO_URI=mongodb://mongo:27017`
- `MONGO_DATABASE=simple_survey`
- `REDIS_ADDR=redis:6379`
- `JWT_SECRET`、Token 过期时间等认证参数

如需限制跨域来源，可额外设置：

- `CORS_ALLOWED_ORIGINS=http://localhost:3000`

前端 API 地址通过构建参数传递（由根目录 `.env` 驱动）：

- `FRONTEND_API_BASE_URL=/api/v1`

`docker-compose.yml` 会将该值注入 `frontend` 构建参数 `VITE_API_BASE_URL`，最终由前端 `import.meta.env.VITE_API_BASE_URL` 生效。

### 5. 一致性检查结论

- 前端 API 前缀：`/api/v1`（代码）✅ 与后端路由分组 `/api/v1`（实现）一致
- 健康检查：`/health`（代码）✅ 与 compose 健康检查配置一致
- 部署拓扑：前端 Nginx + 后端 API + Mongo + Redis ✅ 与当前 compose 一致

