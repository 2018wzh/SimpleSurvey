# SimpleSurvey

一个基于 **Go + Vue 3 + MongoDB + Redis** 的在线问卷系统。

## 快速启动（Docker Compose）

在项目根目录执行：

1. `docker compose up -d --build`
2. 打开前端：`http://localhost:3000`
3. 查看后端健康检查：`http://localhost:8080/health`

停止服务：`docker compose down`

## 文档索引

- 接口文档：`docs/api.md`
- 架构文档：`docs/arch.md`
- 部署文档：`docs/deploy.md`
- 数据模型：`docs/model.md`