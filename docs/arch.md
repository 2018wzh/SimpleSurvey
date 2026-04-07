# 问卷系统架构设计文档 (arch.md)

## 1. 概述
本项目旨在构建一个高可用、低耦合、模块化的在线问卷系统（类似 Google Forms）。系统需要支持动态表单构建、高并发数据收集以及多维度数据分析报表，满足生产级部署标准。

## 2. 核心技术栈
* **后端:** Golang + Gin 框架
* **前端:** Vue 3 + Pinia + Vue Router + ECharts/Chart.js
* **持久层:** MongoDB
* **缓存与状态管理:** Redis

## 3. 系统整体架构

系统采用**分层与模块化单体 (Modular Monolith)** 结合的架构：

1. **接入层:** Nginx / API Gateway 处理 HTTPS 卸载、跨域配置与基础限流。
2. **应用层 (Golang):** 采用整洁架构 (Clean Architecture)，严格分离业务逻辑与底层实现。
3. **缓存层 (Redis):** 存储高频访问的表单 Schema、JWT 黑名单以及用户状态。
4. **持久层 (MongoDB):** 存储结构化的表单模板和非结构化的用户答卷。

## 4. 后端模块化设计 (Clean Architecture)

后端严格遵循 `Controller -> Service -> Repository` 三层架构，依赖倒置通过接口 (Interfaces) 实现。

### 4.1 核心业务模块
* **Identity Module:** 负责认证、授权、会话控制。采用 Refresh Token 机制实现安全的用户登录状态管理。
* **Form Builder Module:** 负责问卷模板的 CRUD、版本控制与状态流转。
* **Collector Module:** 面向 C 端填写者，负责高并发下的数据校验与落库。应对高并发写入时，可引入消息队列（如 Redis Stream/Kafka）实现异步削峰。
* **Analytics Module (统计与报表聚合):** 负责多维度的问卷统计、报表生成及交叉分析。
  * **实时聚合**: 针对中小型问卷，通过 **MongoDB Aggregation Pipeline** 直接执行聚合（`$match`, `$group`, `$unwind`）。
  * **缓存加速**: 对大流量问卷看板，将聚合结果写入 **Redis 缓存**，设置合理的 TTL 或通过定时任务后台刷新，防止数据库扫表过载。
  * **离线与预计算**: 支持大数据量下复杂交叉分析（Cross-Tabulation）结果的持久化存储，将报表数据落地在专属快照集合（如 `reports_snapshot`）中供快速拉取。

### 4.2 目录结构规范
```text
/cmd/server         # 程序入口
/internal
  /domain           # 核心领域实体与 Repository 接口定义
  /repository       # 数据访问层实现 (MongoDB/Redis)
  /service          # 核心业务逻辑实现
  /delivery/http    # 表现层 (Gin 路由、Controllers、中间件)
/pkg                # 公共工具包 (日志 Zap、配置读取、JWT等)
```

## 5. 当前仓库部署拓扑（实现对齐）

当前仓库默认使用 `docker-compose.yml` 进行本地集成部署，拓扑如下：

1. `frontend`（Nginx 托管 Vue 构建产物）对外暴露 `http://localhost:3000`
2. `frontend` 将 `/api/*` 反向代理到 `backend:8080`
3. `backend` 连接 `mongo` 与 `redis` 完成业务读写与会话/令牌存储

该实现与“接入层 -> 应用层 -> 缓存层 -> 持久层”的分层设计保持一致。
