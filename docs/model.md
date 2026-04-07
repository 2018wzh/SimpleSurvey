# 数据模型

## 用户模型

```json
{
  "_id": ObjectId("..."),
  "username": "zhangsan",
  "password": "hashed_password_string",
  "createdAt": ISODate("2026-03-26T10:00:00Z"),
  "status": "active",
  "meta_info": {
    "theme": "light",
    "language": "zh-CN"
  }
}
```

## 题目主模型

```json
{
  "_id": ObjectId("..."),
  "questionKey": "550e8400-e29b-41d4-a716-446655440000", // 业务稳定标识（UUID），可用于跨系统引用
  "ownerId": ObjectId("..."), // 创建者
  "currentVersion": 3, // 当前最新版本号
  "currentVersionId": ObjectId("..."), // 当前版本文档ID
  "tags": ["人口统计", "基础题"],
  "createdAt": ISODate("2026-03-26T10:00:00Z"),
  "updatedAt": ISODate("2026-04-07T09:00:00Z"),
  "isArchived": false
}
```

## 题目版本模型

```json
{
  "_id": ObjectId("..."),
  "questionId": ObjectId("..."),
  "version": 3,
  "parentVersionId": ObjectId("...") , // 来源版本，可为空（v1）
  "changeType": "edit", // create/edit/restore/fork
  "schema": {
    "type": "NUMBER",
    "title": "你的年龄是？",
    "isRequired": true,
    "options": [],
    "validation": {
      "numberType": "integer",
      "minVal": 1,
      "maxVal": 120
    },
    "meta": {
      "displayMode": "input"
    }
  },
  "createdBy": ObjectId("..."),
  "createdAt": ISODate("2026-04-07T09:00:00Z"),
  "note": "调整年龄范围上限"
}
```

## 题库模型

```json
{
  "_id": ObjectId("..."),
  "name": "基础人口统计题库",
  "ownerId": ObjectId("..."),
  "description": "跨项目复用的基础题",
  "visibility": "team",
  "sharedWith": [
    {
      "userId": ObjectId("..."),
      "permission": "use", // use/manage
      "grantedBy": ObjectId("..."),
      "grantedAt": ISODate("2026-04-07T09:10:00Z"),
      "expiresAt": null
    }
  ],
  "items": [
    {
      "questionId": ObjectId("..."),
      "pinnedVersionId": null, // 可选：固定版本；为空表示默认取最新可用版本
      "addedBy": ObjectId("..."),
      "addedAt": ISODate("2026-04-07T09:21:00Z"),
      "order": 1
    }
  ],
  "createdAt": ISODate("2026-04-07T09:20:00Z"),
  "updatedAt": ISODate("2026-04-07T09:20:00Z")
}
```

## 问卷模型

```json
{
  "_id": ObjectId("..."),
  "creatorId": ObjectId("..."),
  "title": "2026年产品满意度调查",
  "description": "感谢您参与我们的问卷调查...",
  "settings": {
    "allowAnonymous": true,
    "duplicateCheck": "cookie",
    "themeColor": "#3C82F6"
  },
  "questions": [
    {
      "questionId": ObjectId("..."),
      "questionVersionId": ObjectId("..."),
      "order": 1,
      "snapshot": {
        "type": "NUMBER",
        "title": "你的年龄是？"
      }
    }
  ],
  "logicRules": [
    {
      "conditionQuestionRefOrder": 1,
      "operator": "GREATER_THAN",
      "conditionValue": 18,
      "action": "JUMP_TO",
      "actionDetails": {
        "targetQuestionRefOrder": 3
      }
    }
  ],
  "status": "draft", // draft/published/closed
  "publishedAt": null,
  "createdAt": ISODate("2026-03-26T10:00:00Z"),
  "updatedAt": ISODate("2026-04-07T10:00:00Z"),
  "isDeleted": false
}
```

## 答卷模型

```json
{
  "_id": ObjectId("..."),
  "questionnaireId": ObjectId("..."),
  "isAnonymous": true,
  "userId": null,
  "answers": [
    {
      "questionId": ObjectId("..."),
      "questionVersionId": ObjectId("..."),
      "value": 25
    }
  ],
  "submittedAt": ISODate("2026-03-26T10:15:00Z"),
  "statistics": {
    "completionTime": 300,
    "ipAddress": "192.168.1.1"
  }
}
```

## 报表与统计缓存模型 (Analytics Report Cache)

```json
{
  "_id": ObjectId("..."),
  "questionnaireId": ObjectId("..."),
  "reportType": "CROSS_TABULATION", 
  "parametersHash": "a1b2c3d4...", 
  "calculatedAt": ISODate("2026-04-07T10:00:00Z"),
  "expiresAt": ISODate("2026-04-07T11:00:00Z"),
  "resultData": {
    "dimensions": ["q1", "q2"],
    "matrix": [
      { "row": "A", "col": "X", "count": 45, "percentage": 0.3 },
      { "row": "A", "col": "Y", "count": 105, "percentage": 0.7 }
    ],
    "totalSample": 150
  }
}
```

## 索引建议

1. `questions`: `ownerId + updatedAt`、`questionKey`
2. `question_versions`: `questionId + version`、`parentVersionId`
3. `question_banks`: `ownerId + updatedAt`、`sharedWith.userId`、`items.questionId`
4. `questionnaires`: `creatorId + status + updatedAt`
5. `questionnaire_question_usages`: `questionId + questionVersionId`、`questionnaireId`
6. `responses`: `questionnaireId + submittedAt`、`answers.questionId`、`answers.questionVersionId`
7. `analytics_reports`: `questionnaireId + parametersHash`（查询）、`expiresAt`（TTL 删除）

## 关键约束

1. 发布问卷后，`questionRefs[*].questionVersionId` 不可被覆盖更新。
2. 题目编辑必须创建新版本（`question_versions` 新增记录）。
3. 支持从任意历史版本恢复（恢复行为本质上是创建新版本，`changeType=restore`）。
4. 同一题目的多个版本可同时被不同问卷引用。
5. 可查询题目被使用的问卷列表（基于 `questionnaire_question_usages` 或反查问卷引用）。
6. 共享能力仅作用于题库（`question_banks.sharedWith`），不允许直接共享 `questions`。
7. `questionKey` 必须使用 UUID（推荐 UUID v4）。

## 文档关联

- 迁移方案：见 `docs/migration.md`
