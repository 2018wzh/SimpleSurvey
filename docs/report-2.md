# SimpleSurvey 结构与需求演进报告（大作业一：需求变更）
## 1. 保存常用题目，方便以后重复使用
### 后端变更 (Commit: `a42798b`)
- **数据模型重构**：在 `internal/domain/models.go` 中新增 `QuestionEntity`（题目元数据）和 `QuestionVersion`（题目具体内容版本）模型。`QuestionEntity` 通过 `QuestionKey` (UUID) 唯一标识一道逻辑上的题目。
- **服务层实现**：在 `internal/service/question.go` 中实现了 `Create` 方法。该方法会同时创建一个 `QuestionEntity` 记录和它的第一个 `QuestionVersion`。
- **持久化层**：在 `internal/repository/mongo/question_repository.go` 中实现了对题目及其版本的 CRUD 操作。
### 前端变更 (Commit: `aacba74`)
- **独立管理页**：新增 `frontend/src/views/Questions.vue`。用户可以在该页面独立于问卷创建题目，并为题目分配唯一的 `QuestionKey`。
- **组件复用**：在 `frontend/src/views/CreateSurvey.vue` 中集成了题目选择器，用户可以从已保存的题目库中搜索并一键添加到当前问卷。
## 2. 把题目分享给别人使用
### 后端变更 (Commit: `a42798b`)
- **题库共享模型**：在 `QuestionBank` 模型中引入 `SharedWith` 字段，通过 `QuestionBankShare` 结构体记录共享目标用户 ID 和权限级别（`use` 或 `manage`）。
- **权限校验**：在 `internal/service/question_bank.go` 的 `Get` 和 `AddItem` 等方法中增加鉴权逻辑，确保只有所有者或被授权用户可以访问/操作。
### 前端变更 (Commit: `aacba74`)
- **共享交互实现**：在 `frontend/src/views/QuestionBanks.vue` 中增加了“共享”管理版块。用户可以通过下拉列表选择系统内其他用户，并授予特定权限。
## 3. 修改题目时不要影响已经做好的问卷
### 后端变更 (Commit: `a42798b`)
- **快照机制**：问卷模型 `Questionnaire` 中的 `Questions` 数组不再仅存储引用，而是包含 `Snapshot` (`*QuestionSchema`) 字段。
- **版本锁定**：在问卷创建或保存题目时，后端将题目当时的具体内容直接序列化存储到 `Snapshot` 中。即使原题目后续产生了新版本，已存在问卷中的 `Snapshot` 仍指向旧版本内容。
### 前端变更 (Commit: `aacba74`)
- **静态渲染**：`frontend/src/views/FillSurvey.vue` 调整为优先加载问卷内嵌的题目快照数据，而非实时查询题库的最新状态。
## 4. 记录题目的修改历史
### 后端变更 (Commit: `a42798b`)
- **版本链设计**：`QuestionVersion` 结构体包含 `ParentVersionID`，构建了版本演进的单向链表结构。
- **变更记录**：引入 `ChangeType` 枚举（`create`, `edit`, `restore`, `fork`）和 `Note` 字段，记录每次修改的意图。
- **恢复接口**：实现 `RestoreVersion` 服务方法，通过克隆指定旧版本的内容并创建新版本节点来实现“回滚”。
### 前端变更 (Commit: `aacba74`)
- **历史时间轴**：在 `Questions.vue` 的展开详情中，通过调用 `/questions/:id/versions` 接口展现版本列表。
- **一键恢复**：为每个历史版本提供“恢复”按钮，点击后触发后端恢复逻辑。
## 5. 同一个题目的不同版本可以同时存在
### 后端变更 (Commit: `a42798b`)
- **细粒度引用**：`Questionnaire` 中的题目条目明确记录 `QuestionVersionID`。
- **版本共存逻辑**：数据库中每个版本都是独立的 Document，通过 `QuestionID` 关联。后端支持根据具体的 `VersionID` 检索特定时刻的题目定义。
### 前端变更 (Commit: `aacba74`)
- **显式版本选择**：在 `QuestionBanks.vue` 添加题目到题库时，用户可以选择“最新版本”或“固定特定版本”。
## 6. 查看某个题被哪些问卷使用
### 后端变更 (Commit: `a42798b`)
- **反向关联查询**：在 `QuestionService` 中实现 `GetUsages` 方法。该方法通过聚合查询遍历所有问卷，筛选出包含指定 `QuestionID` 的问卷列表。
### 前端变更 (Commit: `aacba74`)
- **引用明细面板**：在 `Questions.vue` 中新增“使用情况”模态框/折叠层，列出所有关联问卷的名称、状态及所使用的题目版本 ID。
## 7. 建立题目库
### 后端变更 (Commit: `a42798b`)
- **题库实体**：新增 `QuestionBank` 领域模型，包含 `Items` 数组，用于维护题目与版本的有序集合。
- **CRUD 接口**：在 `handlers_question.go` 中提供 `/question-banks` 路由下的完整管理接口。
### 前端变更 (Commit: `aacba74`)
- **题库中心**：新增 `frontend/src/views/QuestionBanks.vue`。用户可在此创建名为“客户满意度”、“基础信息”等不同主题的题库，并将我的题目添加进去。
## 8. 查看单个题目的统计结果（跨问卷）
### 后端变更 (Commit: `a42798b`)
- **全域数据聚合**：在 `QuestionService` 中实现 `GetStats` 方法。
- **核心逻辑**：
    1.  定位使用该题目（或特定版本）的所有 `QuestionnaireID`。
    2.  遍历这些问卷下所有的 `Response`（答卷）。
    3.  根据题目类型（单选、多选、数字、文本）进行分类统计：
        -   选择题：统计各选项 ID 的出现频次。
        -   数字题：计算总平均值。
        -   文本题：收集所有文本回复。
### 前端变更 (Commit: `aacba74`)
- **数据汇总视图**：在 `Questions.vue` 中集成“跨问卷统计”功能。通过表格形式直观展示跨问卷聚合后的样本总量和数据分布图。
