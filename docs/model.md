# 数据模型
## 用户模型
```json
{
  "_id": ObjectId("..."), // 系统自动生成的唯一ID
  "username": "zhangsan", // 用户名
  "password": "hashed_password_string", // 密码 (argon2 加密后的字符串)
  "createdAt": ISODate("2026-03-26T10:00:00Z"), // 注册时间
  "status": "active", // 账号状态
  "meta_info": { // 可扩展字段，用于存储用户的后续偏好设置、自定义主题等非核心数据
    "theme": "light",
    "language": "zh-CN"
  }
}
```
## 问卷模型
```json
{
  "_id": ObjectId("..."),
  "creatorId": ObjectId("..."), // 关联ID：创建此问卷的用户ID
  "title": "2026年产品满意度调查", // 标题
  "description": "感谢您参与我们的问卷调查...", // 说明
  "settings": {
    "allowAnonymous": true, // 属性：是否允许匿名填写
    "duplicateCheck": "cookie", // IP防重防刷机制 (cookie/ip/account) 等后续高阶属性
    "themeColor": "#3C82F6" // 前端显示主题色扩展
  },
  
  // --- 题目列表 ---
  "questions": [
    {
      "questionId": "q_1", // 题目内部ID（UUID）
      "type": "SINGLE_CHOICE", // 题型：单选（枚举可扩展为 SCALE评分、IMAGE图片等）
      "title": "您最常使用的功能是？",
      "isRequired": true, // 是否必答
      "meta": { "displayMode": "radio_list" }, // 独立存储前端渲染方式、图片URL等自由格式扩展数据
      "options": [
        { "optionId": "opt_1", "text": "功能A", "hasOtherInput": false } // 若含"其它"，hasOtherInput将通知前端提供文本框
      ]
    },
    {
      "questionId": "q_2",
      "type": "MULTIPLE_CHOICE", // 题型：多选
      "title": "您希望增加哪些功能？",
      "isRequired": false,
      "options": [
        { "optionId": "opt_3", "text": "功能C" },
        { "optionId": "opt_4", "text": "功能D" }
      ],
      "validation": {
        "minSelect": 1, // 限制选择数量：最少
        "maxSelect": 3  // 限制选择数量：最多
      }
    },
    {
      "questionId": "q_3",
      "type": "TEXT", // 题型：填空（文本）
      "title": "您的宝贵建议",
      "isRequired": false,
      "validation": {
        "minLength": 10, // 限制：最少字数
        "maxLength": 500 // 限制：最多字数
      }
    },
    {
      "questionId": "q_4",
      "type": "NUMBER", // 题型：填空（数字）
      "title": "您的年龄段预测（请输入真实年龄）",
      "isRequired": true,
      "validation": {
        "numberType": "integer", // 类型：整数/浮点数
        "minVal": 18, // 范围：最小值
        "maxVal": 100 // 范围：最大值
      }
    }
  ],

  // --- 题目跳转规则 ---
  "logicRules": [
    {
      "conditionQuestionId": "q_1", // 触发规则的题目ID
      "operator": "CONTAINS", // 比较操作符，如 EQUALS(单选等值), CONTAINS(多选包含), GREATER_THAN(数字大于), 以支持作业要求的“多选、数字条件跳转”
      "conditionValue": "opt_2", // 触发条件的值（可以为单选ID、多选ID数组、数字定值等）
      "action": "JUMP_TO", // 跳转到指定题目 (未来扩展 action: HIDDEN，SHOW 等等)
      "actionDetails": {
        "targetQuestionId": "q_3" // 跳转目标题目ID
      }
    }
  ],
  
  "createdAt": ISODate("2026-03-26T10:00:00Z"),
  "updatedAt": ISODate("2026-03-26T10:00:00Z"),
  "isDeleted": false // 逻辑删除标识位，方便数据归档恢复
}
```
## 答卷模型
```json
{
  "_id": ObjectId("..."),
  "questionnaireId": ObjectId("..."), // 关联的问卷ID
  "isAnonymous": true, // 标识是否为匿名记录
  "userId": ObjectId("..."), // 填写人的用户ID。如果是匿名填写，此字段为 null 且不存在
  
  // 用户的具体回答
  "answers": [
    {
      "questionId": "q_1",
      "value": "opt_2" // 单选存选项ID
    },
    {
      "questionId": "q_2",
      "value": ["opt_3", "opt_4"] // 多选存选项ID数组
    },
    {
      "questionId": "q_3",
      "value": "产品做得非常棒！" // 文本填空存字符串
    },
    {
      "questionId": "q_4",
      "value": 25 // 数字存数值类型
    }
  ],
  
  "submittedAt": ISODate("2026-03-26T10:15:00Z"), // 提交时间
  "statistics": {
    "completionTime": 300, // 完成问卷的时间（秒）
    "ipAddress": "192.168.1.1", // 用于追踪匿名用户
  }
}
```