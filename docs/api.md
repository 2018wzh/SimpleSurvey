# RESTful API 设计与文档
## 1. 全局说明
- **Base URL**: `/api/v1`
- **认证方式**: 大部分接口需要携带 JWT Token。请求头格式：`Authorization: Bearer <token>`
- **数据格式**: 请求和响应主体默认采用 `application/json`。
- **分页规范**: 列表请求默认支持 `page` (默认1) 和 `limit` (默认20) 查询参数。列表响应的 `data` 部分应包含 `items` 数组和 `total` 总数，方便后续扩展排序等。
- **通用响应格式**:
```json
{
  "code": 200,
  "message": "success",
  "data": {},
  "meta": {} // 可选扩展字段，如请求ID、服务器时间等
}
```
- **错误响应格式**: 针对 400 等校验错误，建议增加 `details` 字段存放具体字段的错误信息，提高排查效率。
## 2. 身份认证模块 (Identity Module)
系统需要支持用户注册和登录 ，每个用户需要有用户名、密码和注册时间 。
### 2.1 用户注册
- **接口路径**: `POST /auth/register`
- **功能描述**: 用户注册账号 。
- **请求参数 (Body)**:
```json
{
  "username": "zhangsan",
  "password": "secure_password"
}
```
- **响应成功 (201 Created)**:
```json
{
  "code": 201,
  "message": "注册成功",
  "data": {
    "userId": "6511a2b3c4d5e6f7g8h9i0j1"
  }
}
```
### 2.2 用户登录
- **接口路径**: `POST /auth/login`
- **功能描述**: 用户登录账号并获取凭证 。
- **请求参数 (Body)**:
```json
{
  "username": "zhangsan",
  "password": "secure_password"
}
```
- **响应成功 (200 OK)**:
```json
{
  "code": 200,
  "message": "登录成功",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expiresIn": 86400
  }
}
```
## 3. 问卷构建模块 (Form Builder Module)
用户登录后，应只能看到自己创建的问卷 。用户可以使用自己的账号创建问卷 ，添加不同类型的问题 以及设置跳转逻辑 。
### 3.1 创建问卷
- **接口路径**: `POST /questionnaires`
- **权限要求**: 需登录
- **功能描述**: 创建新问卷，包含标题、说明、匿名设置、题目列表和跳转规则 。
- **请求参数 (Body)**: （参考 `model.md` 中的问卷模型）
```json
{
  "title": "2026年产品满意度调查",
  "description": "感谢您参与我们的问卷调查...",
  "settings": {
    "allowAnonymous": true
  },
  "questions": [
    {
      "questionId": "q_1",
      "type": "SINGLE_CHOICE",
      "title": "您最常使用的功能是？",
      "isRequired": true,
      "options": [
        { "optionId": "opt_1", "text": "功能A" }
      ]
    }
  ],
  "logicRules": []
}
```
- **响应成功 (201 Created)**: 返回生成的问卷 ID。
### 3.2 获取我的问卷列表
- **接口路径**: `GET /questionnaires`
- **权限要求**: 需登录
- **功能描述**: 查看自己创建的问卷列表。
- **查询参数 (Query)**: 支持 `page` (默认1)、`limit` (默认20)、`status` (按状态筛选)、`sortBy` (例如: createdAt) 等参数，以应对后续列表显示需求。
- **响应成功 (200 OK)**: 返回包含问卷基础信息（不含详细题目）和分页元数据的对象。
```json
{
  "code": 200,
  "data": {
    "items": [
      {
        "id": "q123",
        "title": "2026年产品满意度调查",
        "status": "published",
        "createdAt": "2026-03-26T10:00:00Z"
      }
    ],
    "total": 120,
    "page": 1,
    "limit": 20
  }
}
```
### 3.3 更新问卷状态 (发布/关闭/截止)
- **接口路径**: `PATCH /questionnaires/:id/status`
- **权限要求**: 需登录且为创建者
- **功能描述**: 发布问卷 、设置问卷截止时间 或关闭问卷 。
- **请求参数 (Body)**:
```json
{
  "status": "published", 
  "deadline": "2026-04-01T23:59:59Z" 
}
```
## 4. 数据收集模块 (Collector Module - C端展示与填写)
每个问卷都应该有一个可以访问的链接，例如 `/survey/xxxxxx` 。填写问卷的人可以通过问卷链接进入问卷 。
### 4.1 获取问卷详情 (用于填写)
- **接口路径**: `GET /surveys/:id`
- **权限要求**: 根据问卷 `allowAnonymous` 设置决定是否必须登录 。
- **功能描述**: 获取完整的问卷结构（题目、选项、跳转规则），供前端按顺序展示和执行跳转 。
- **响应成功 (200 OK)**: 返回去除敏感信息（如创建者信息）后的问卷完整 JSON。
### 4.2 提交答卷
- **接口路径**: `POST /surveys/:id/responses`
- **权限要求**: 根据问卷设置决定，系统需要支持匿名填写 和多人同时填写 。
- **功能描述**: 提交问卷答案。系统在填写时需要检查是否符合要求（必填、数量限制、数值限制等） 。填写完成后可以提交 。每次填写都应该被保存 。
- **请求参数 (Body)**:
```json
{
  "isAnonymous": true,
  "answers": [
    {
      "questionId": "q_1",
      "value": "opt_2"
    },
    {
      "questionId": "q_2",
      "value": ["opt_3", "opt_4"]
    },
    {
      "questionId": "q_4",
      "value": 25
    }
  ],
  "statistics": {
    "completionTime": 300
  }
}
```
- **响应成功 (201 Created)**:
```json
{
  "code": 201,
  "message": "提交成功"
}

```
- **响应失败 (400 Bad Request)**: 填写不符合要求，系统提示错误 。
```json
{
  "code": 400,
  "message": "第2题至少选择1项，最多选择3项"
}
```
## 5. 数据分析模块 (Analytics Module)
创建问卷的人希望可以查看统计结果 。统计结果应该根据已经填写的数据自动生成 。
### 5.1 获取问卷全局统计结果
- **接口路径**: `GET /questionnaires/:id/stats`
- **权限要求**: 需登录且为该问卷的创建者 。
- **功能描述**: 查看整个问卷的统计 。包含总回复数，以及各题目的统计。对于单选题和多选题统计选项选择次数 ，对于数字题计算平均值 。
- **响应成功 (200 OK)**:
```json
{
  "code": 200,
  "data": {
    "totalResponses": 150,
    "questionStats": [
      {
        "questionId": "q_1",
        "type": "SINGLE_CHOICE",
        "totalAnswered": 150,
        "optionCounts": {
          "opt_1": 100,
          "opt_2": 50
        }
      },
      {
        "questionId": "q_4",
        "type": "NUMBER",
        "totalAnswered": 145,
        "averageValue": 28.5
      }
    ]
  }
}
```
### 5.2 查看填空题详细内容 / 原始答卷
- **接口路径**: `GET /questionnaires/:id/responses`
- **权限要求**: 需登录且为该问卷的创建者。
- **功能描述**: 可以查看所有填写内容 ，并支持查看某一道题的统计 。
- **查询参数 (Query)**: `?questionId=q_3&page=1&limit=20` (可选，用于过滤查看特定填空题的回答)。
- **响应成功 (200 OK)**: 返回答卷明细列表。