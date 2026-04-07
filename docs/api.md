# RESTful API 文档（与当前实现同步）

## 1. 全局说明

- **Base URL**: `/api/v1`
- **认证头**: `Authorization: Bearer <access_token>`
- **Content-Type**: `application/json`
- **统一响应结构**:

```json
{
  "code": 200,
  "message": "success",
  "data": {},
  "meta": null,
  "details": null
}
```

> 说明：
>
> - 成功响应默认 `message` 为 `success`（`response.Success`）。
> - `201` 创建类接口会返回业务文案（例如“注册成功”“创建成功”“提交成功”）。
> - 校验失败等错误会在 `details` 中返回具体信息。

## 2. 枚举约定

- **问卷状态**: `draft` | `published` | `closed`
- **用户角色**: `user` | `admin`
- **用户状态**: `active` | `disabled`
- **题型**: `SINGLE_CHOICE` | `MULTIPLE_CHOICE` | `TEXT` | `NUMBER`
- **题库可见性**: `private` | `team`
- **题库共享权限**: `use` | `manage`
- **题目版本变更类型**: `create` | `edit` | `restore` | `fork`

## 3. 健康检查

### 3.1 GET `/health`

- **鉴权**: 无需
- **响应示例**:

```json
{
  "status": "ok"
}
```

## 4. 认证模块

### 4.1 用户注册

- **POST** `/auth/register`
- **请求体**:

```json
{
  "username": "zhangsan",
  "password": "strong-password"
}
```

- **成功响应 (201)**:

```json
{
  "code": 201,
  "message": "注册成功",
  "data": {
    "userId": "507f1f77bcf86cd799439011"
  }
}
```

### 4.2 用户登录（签发 access + refresh）

- **POST** `/auth/login`
- **请求体**:

```json
{
  "username": "zhangsan",
  "password": "strong-password"
}
```

- **成功响应 (200)**:

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "token": "<access_token>",
    "expiresIn": 86400,
    "refreshToken": "<refresh_token>",
    "refreshExpiresIn": 604800
  }
}
```

### 4.3 刷新 Token（refresh token 轮换）

- **POST** `/auth/refresh`
- **请求体**:

```json
{
  "refreshToken": "<refresh_token>"
}
```

- **成功响应 (200)**:

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "token": "<new_access_token>",
    "expiresIn": 86400,
    "refreshToken": "<new_refresh_token>",
    "refreshExpiresIn": 604800
  }
}
```

> 注意：旧 refresh token 会失效（单次使用，轮换策略）。

## 5. 问卷管理（普通用户）

以下接口均需 **Access Token**。

### 5.1 创建问卷

- **POST** `/questionnaires`
- **请求体示例**:

```json
{
  "title": "2026年产品满意度调查",
  "description": "感谢参与",
  "settings": {
    "allowAnonymous": true,
    "duplicateCheck": "none",
    "themeColor": "#1677ff"
  },
  "questions": [
    {
      "questionId": "67f3e5f244f95a7d05b5a111",
      "questionVersionId": "67f3e5f244f95a7d05b5a211",
      "order": 1,
      "snapshot": {
        "type": "SINGLE_CHOICE",
        "title": "您是否满意？"
      }
    }
  ],
  "logicRules": []
}
```

- **成功响应 (201)**:

```json
{
  "code": 201,
  "message": "创建成功",
  "data": {
    "id": "67f3e5f244f95a7d05b5a912"
  }
}
```

### 5.2 查询我的问卷列表

- **GET** `/questionnaires`
- **Query 参数**:
  - `page`（默认 `1`）
  - `limit`（默认 `20`，最大 `100`）
  - `status`（可选）
  - `sortBy`（可选，支持 `updatedAt`；默认按 `createdAt`）
- **成功响应 (200)**:

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "items": [
      {
        "id": "67f3e5f244f95a7d05b5a912",
        "title": "2026年产品满意度调查",
        "status": "draft",
        "createdAt": "2026-03-27T08:00:00Z"
      }
    ],
    "total": 1,
    "page": 1,
    "limit": 20
  }
}
```

### 5.3 更新问卷状态（创建者）

- **PATCH** `/questionnaires/:id/status`
- **请求体**:

```json
{
  "status": "published",
  "deadline": "2026-04-01T23:59:59Z"
}
```

- **成功响应 (200)**:

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "67f3e5f244f95a7d05b5a912",
    "status": "published"
  }
}
```

### 5.4 问卷基础统计分析

- **GET** `/questionnaires/:id/stats`
- **成功响应 (200)**: 返回 `totalResponses` 与 `questionStats`（按 `questionId + questionVersionId` 统计单/多选计数、数字均值、文本明细）。此接口由后端聚合预计算（Redis/MongoDB PipeLine）生成，保障高性能看板。

### 5.5 问卷交叉报表与高级聚合（新增功能）

- **POST** `/questionnaires/:id/reports/crosstab`
- **请求体 (Body)**:
```json
{
  "rowQuestionId": "q1",
  "colQuestionId": "q2",
  "filters": {
    "dateRange": { "start": "2026-04-01", "end": "2026-04-07" },
    "completionStatus": "completed"
  }
}
```
- **成功响应 (200)**: 基于 MongoDB `$facet` / `$group` 聚合管道深度查询生成的透视报表矩阵数据。

### 5.6 问卷答卷明细

- **GET** `/questionnaires/:id/responses`
- **Query 参数**:
  - `page`（默认 `1`）
  - `limit`（默认 `20`）
  - `questionId`（可选，筛选指定题目答案）
  - `questionVersionId`（可选，筛选指定题目版本答案）
- **成功响应 (200)**: 返回 `items`、`total`、`page`、`limit`。

### 5.7 查询某题被哪些问卷使用

- **GET** `/questions/:id/usages`
- **Query 参数**:
  - `questionVersionId`（可选，按版本过滤）
  - `status`（可选：`draft`/`published`/`closed`）
- **成功响应 (200)**: 返回使用该题（或该版本）的问卷列表。

### 5.8 查询单题跨问卷统计

- **GET** `/questions/:id/stats`
- **Query 参数**:
  - `questionVersionId`（可选，不传则聚合所有版本）
  - `from`、`to`（可选，时间区间）
- **成功响应 (200)**: 返回该题在所有问卷中的聚合统计结果。

## 6. 题目与题库管理（普通用户）

以下接口均需 **Access Token**。

### 6.1 创建题目（生成 v1）

- **POST** `/questions`
- **请求体示例**:

```json
{
  "questionKey": "550e8400-e29b-41d4-a716-446655440000",
  "schema": {
    "type": "NUMBER",
    "title": "你的年龄是？",
    "isRequired": true,
    "validation": {
      "numberType": "integer",
      "minVal": 1,
      "maxVal": 120
    }
  },
  "tags": ["人口统计", "基础题"]
}
```

> 约束：`questionKey` 必须为 UUID（推荐 UUID v4）。

- **成功响应 (201)**:

```json
{
  "code": 201,
  "message": "创建成功",
  "data": {
    "id": "67f3e5f244f95a7d05b5a111",
    "version": 1,
    "versionId": "67f3e5f244f95a7d05b5a211"
  }
}
```

### 6.2 创建题目新版本

- **POST** `/questions/:id/versions`
- **请求体示例**:

```json
{
  "baseVersionId": "67f3e5f244f95a7d05b5a211",
  "changeType": "edit",
  "note": "调整年龄范围上限",
  "schema": {
    "type": "NUMBER",
    "title": "你的年龄是？",
    "isRequired": true,
    "validation": {
      "numberType": "integer",
      "minVal": 1,
      "maxVal": 150
    }
  }
}
```

- **成功响应 (201)**: 返回新 `version` 与 `versionId`。

### 6.3 查询题目版本历史

- **GET** `/questions/:id/versions`
- **成功响应 (200)**: 返回题目版本链（含 `parentVersionId`、`changeType`、`note`）。

### 6.4 恢复历史版本（生成新版本）

- **POST** `/questions/:id/restore`
- **请求体**:

```json
{
  "fromVersionId": "67f3e5f244f95a7d05b5a211",
  "note": "恢复到历史版本"
}
```

- **成功响应 (201)**: 返回恢复后新版本信息。

### 6.5 创建题库

- **POST** `/question-banks`
- **请求体示例**:

```json
{
  "name": "基础人口统计题库",
  "description": "跨项目复用的基础题",
  "visibility": "team",
  "items": [
    {
      "questionId": "67f3e5f244f95a7d05b5a111",
      "pinnedVersionId": null,
      "order": 1
    }
  ]
}
```

- **成功响应 (201)**: 返回题库 `id`。

### 6.6 查询我的题库列表

- **GET** `/question-banks`
- **Query 参数**:
  - `page`（默认 `1`）
  - `limit`（默认 `20`）
  - `keyword`（可选）
- **成功响应 (200)**: 返回题库列表（含内嵌 `items` 摘要）。

### 6.7 更新题库信息

- **PATCH** `/question-banks/:id`
- **请求体示例**:

```json
{
  "name": "基础人口统计题库（教学版）",
  "description": "用于教学问卷",
  "visibility": "team"
}
```

- **成功响应 (200)**: 返回更新后的题库基础信息。

### 6.8 向题库加入题目（内嵌 items）

- **POST** `/question-banks/:id/items`
- **请求体示例**:

```json
{
  "questionId": "67f3e5f244f95a7d05b5a111",
  "pinnedVersionId": "67f3e5f244f95a7d05b5a211",
  "order": 2
}
```

- **成功响应 (200)**: 返回题库最新 `items`。

### 6.9 调整题库内题目版本或顺序

- **PATCH** `/question-banks/:id/items/:questionId`
- **请求体示例**:

```json
{
  "pinnedVersionId": null,
  "order": 1
}
```

- **成功响应 (200)**: 返回题库最新 `items`。

### 6.10 从题库移除题目

- **DELETE** `/question-banks/:id/items/:questionId`
- **成功响应 (200)**: 返回删除后的题库 `items`。

### 6.11 共享题库给其他用户

- **POST** `/question-banks/:id/shares`
- **请求体示例**:

```json
{
  "targetUserId": "507f1f77bcf86cd799439012",
  "permission": "use",
  "expiresAt": null
}
```

- **成功响应 (200)**: 返回题库 `sharedWith`。

### 6.12 取消题库共享

- **DELETE** `/question-banks/:id/shares/:targetUserId`
- **成功响应 (200)**: 返回题库 `sharedWith`。

> 约束：系统仅支持共享题库，不支持直接共享题目。

## 7. 填写端（C 端）

### 7.1 获取可填写问卷

- **GET** `/surveys/:id`
- **鉴权**: 可匿名访问；若问卷不允许匿名则需携带 Access Token
- **成功响应 (200)**: 返回问卷完整结构（已脱敏创建者信息，题目以 `questionId + questionVersionId + snapshot` 呈现）。

### 7.2 提交答卷

- **POST** `/surveys/:id/responses`
- **请求体示例**:

```json
{
  "isAnonymous": true,
  "answers": [
    {
      "questionId": "67f3e5f244f95a7d05b5a111",
      "questionVersionId": "67f3e5f244f95a7d05b5a211",
      "value": 30
    }
  ],
  "statistics": {
    "completionTime": 120
  }
}
```

- **成功响应 (201)**:

```json
{
  "code": 201,
  "message": "提交成功"
}
```

## 8. 管理员接口

以下接口均需：

1. 先通过 `/auth/login` 获取 **Access Token**；
2. Token 中角色为 `admin`。

### 8.1 用户列表

- **GET** `/admin/users`
- **Query 参数**:
  - `page`（默认 `1`）
  - `limit`（默认 `20`）
  - `status`（可选：`active`/`disabled`）
  - `role`（可选：`user`/`admin`）
  - `keyword`（可选：用户名模糊搜索）
- **成功响应 (200)**: `data.items` 为用户列表（不包含密码）。

### 8.2 更新用户角色

- **PATCH** `/admin/users/:id/role`
- **请求体**:

```json
{
  "role": "admin"
}
```

- **成功响应 (200)**:

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "507f1f77bcf86cd799439011",
    "role": "admin"
  }
}
```

### 8.3 更新用户状态

- **PATCH** `/admin/users/:id/status`
- **请求体**:

```json
{
  "status": "disabled"
}
```

- **成功响应 (200)**:

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "507f1f77bcf86cd799439011",
    "status": "disabled"
  }
}
```

### 8.4 全量问卷列表

- **GET** `/admin/questionnaires`
- **Query 参数**:
  - `page`（默认 `1`）
  - `limit`（默认 `20`）
  - `status`（可选：`draft`/`published`/`closed`）
  - `sortBy`（可选：`updatedAt`）
  - `creatorId`（可选：按创建者筛选）
- **成功响应 (200)**: `data.items` 为问卷完整结构列表。

### 8.5 管理员更新问卷状态

- **PATCH** `/admin/questionnaires/:id/status`
- **请求体**:

```json
{
  "status": "closed",
  "deadline": "2026-04-30T23:59:59Z"
}
```

- **成功响应 (200)**:

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "67f3e5f244f95a7d05b5a912",
    "status": "closed"
  }
}
```

## 9. 常见错误码

- `400`：参数错误/校验失败
- `401`：未认证、Token 无效、Token 类型错误、refresh token 无效
- `403`：权限不足（如非管理员访问 `/admin/*`）、账号被禁用、问卷状态不允许提交
- `404`：资源不存在（用户/问卷）
- `409`：注册用户名冲突
- `412`：版本冲突（题目版本不匹配、发布后尝试覆盖已绑定版本）
- `500`：服务内部错误

## 10. 鉴权与 Token 使用约束

- 业务受保护接口只接受 **Access Token**。
- **Refresh Token** 仅用于 `/auth/refresh`。
- refresh 成功后旧 refresh token 立即失效，请客户端及时替换本地 token 对。
- 题库共享接口仅允许题库 `owner` 或具备 `manage` 权限的用户调用。
