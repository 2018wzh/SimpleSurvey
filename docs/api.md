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

## 3. 健康检查

### 3.1 GET `/healthz`

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
      "questionId": "q1",
      "type": "SINGLE_CHOICE",
      "title": "您是否满意？",
      "isRequired": true,
      "options": [
        { "optionId": "o1", "text": "满意" },
        { "optionId": "o2", "text": "不满意" }
      ]
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

### 5.4 问卷统计

- **GET** `/questionnaires/:id/stats`
- **成功响应 (200)**: 返回 `totalResponses` 与 `questionStats`（单/多选计数、数字均值、文本明细）。

### 5.5 问卷答卷明细

- **GET** `/questionnaires/:id/responses`
- **Query 参数**:
  - `page`（默认 `1`）
  - `limit`（默认 `20`）
  - `questionId`（可选，筛选指定题目答案）
- **成功响应 (200)**: 返回 `items`、`total`、`page`、`limit`。

## 6. 填写端（C 端）

### 6.1 获取可填写问卷

- **GET** `/surveys/:id`
- **鉴权**: 可匿名访问；若问卷不允许匿名则需携带 Access Token
- **成功响应 (200)**: 返回问卷完整结构（已脱敏创建者信息）。

### 6.2 提交答卷

- **POST** `/surveys/:id/responses`
- **请求体示例**:

```json
{
  "isAnonymous": true,
  "answers": [
    { "questionId": "q1", "value": "o1" },
    { "questionId": "q2", "value": ["o3", "o4"] },
    { "questionId": "q3", "value": "建议内容" },
    { "questionId": "q4", "value": 30 }
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

## 7. 管理员接口

以下接口均需：

1. 先通过 `/auth/login` 获取 **Access Token**；
2. Token 中角色为 `admin`。

### 7.1 用户列表

- **GET** `/admin/users`
- **Query 参数**:
  - `page`（默认 `1`）
  - `limit`（默认 `20`）
  - `status`（可选：`active`/`disabled`）
  - `role`（可选：`user`/`admin`）
  - `keyword`（可选：用户名模糊搜索）
- **成功响应 (200)**: `data.items` 为用户列表（不包含密码）。

### 7.2 更新用户角色

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

### 7.3 更新用户状态

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

### 7.4 全量问卷列表

- **GET** `/admin/questionnaires`
- **Query 参数**:
  - `page`（默认 `1`）
  - `limit`（默认 `20`）
  - `status`（可选：`draft`/`published`/`closed`）
  - `sortBy`（可选：`updatedAt`）
  - `creatorId`（可选：按创建者筛选）
- **成功响应 (200)**: `data.items` 为问卷完整结构列表。

### 7.5 管理员更新问卷状态

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

## 8. 常见错误码

- `400`：参数错误/校验失败
- `401`：未认证、Token 无效、Token 类型错误、refresh token 无效
- `403`：权限不足（如非管理员访问 `/admin/*`）、账号被禁用、问卷状态不允许提交
- `404`：资源不存在（用户/问卷）
- `409`：注册用户名冲突
- `500`：服务内部错误

## 9. 鉴权与 Token 使用约束

- 业务受保护接口只接受 **Access Token**。
- **Refresh Token** 仅用于 `/auth/refresh`。
- refresh 成功后旧 refresh token 立即失效，请客户端及时替换本地 token 对。