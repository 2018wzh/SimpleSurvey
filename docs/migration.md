# 整套系统迁移指南（基于当前设计文档）

> 目标：将现网系统从“问卷内嵌题目”架构平滑迁移到“题目独立 + 版本化 + 题库共享（仅题库）”的新体系，覆盖数据、后端、前端、部署、验证与回滚全链路。

## 0. 仓库内置迁移命令（questionVersionId 回填）

当前后端已提供可执行迁移命令，用于将旧数据中的 `questionnaires.questions[*].questionVersionId` 与 `responses.answers[*].questionVersionId` 回填到新版结构。

- 预演（不落库）：`go run ./cmd/cli migrate --from=1.0 --dry-run`
- 执行（落库）：`go run ./cmd/cli migrate --from=1.0`

说明：

1. 会从 v1.0 问卷内嵌题目生成 `questions` 与 `question_versions` 数据。
2. 会将问卷与答卷中的旧 `questionId` 重新映射到新生成的题目主键，并写入 `questionVersionId`。
3. 会为问卷题目引用补齐 `snapshot`（用于运行期严格校验）。
4. 迁移过程幂等，可重复执行。

## 0.1 服务启动时自动校验与迁移

当前后端在启动初始化阶段会执行数据库字段校验：

1. 若检测到 v1 特征数据（如缺失 `questionVersionId` 或 `snapshot`），会自动触发 v1 迁移。
2. 迁移后会再次执行严格字段校验：
   - `questionnaires.questions[*]` 必须包含 `questionId`、`questionVersionId`、`snapshot`
   - `responses.answers[*]` 必须包含 `questionId`、`questionVersionId`
3. 若校验失败，服务将拒绝启动并返回错误信息。

## 1. 适用范围与目标版本

### 1.1 适用范围

本指南适用于以下文档约束下的系统迁移：

- 数据模型：`docs/model.md`
- 接口定义：`docs/api.md`
- 系统架构：`docs/arch.md`
- 部署拓扑：`docs/deploy.md`

### 1.2 新体系核心变化

1. 题目从问卷中解耦，新增：
   - `questions`
   - `question_versions`
   - `question_banks`（内嵌 `items[]` 与 `sharedWith[]`）
2. 问卷中的题目引用标准化为：`questionId + questionVersionId`。
3. `questionKey` 升级为 UUID（推荐 UUID v4），用于跨系统稳定引用。
4. 共享能力统一到题库：仅允许 `question_banks.sharedWith[]`，不允许直接共享题目。

### 1.3 迁移目标

1. 业务不中断（或控制在可接受窗口内）。
2. 历史答卷可追溯到具体题目版本。
3. 前后端 API 与数据结构保持一致。
4. 支持灰度发布、快速回滚与二次迁移复盘。

## 2. 迁移总览（分阶段）

建议采用 8 个阶段推进：

1. 阶段 A：准备与基线冻结
2. 阶段 B：基础设施与 Schema 扩展
3. 阶段 C：后端兼容改造（双读/双写）
4. 阶段 D：数据回填与一致性修复
5. 阶段 E：前端切换与兼容发布
6. 阶段 F：灰度放量与观测
7. 阶段 G：全量切换与旧结构下线
8. 阶段 H：迁移收尾与长期治理

每个阶段都要求：

- 有进入条件
- 有执行步骤
- 有退出条件
- 有回滚动作

## 3. 阶段 A：准备与基线冻结

### 3.1 进入条件

1. 已确认设计文档版本（model/api/arch/deploy）。
2. 业务方确认迁移窗口与风险级别。
3. 指定迁移负责人（后端、前端、DBA、运维、QA）。

### 3.2 执行步骤

1. 冻结涉及模型与接口的并行需求变更。
2. 备份关键集合：
   - `questionnaires`
   - `responses`
   - （若存在）`question_bank_items`、`question_shares`
3. 产出迁移批次策略（按问卷数量、时间窗口、业务租户等维度）。
4. 定义迁移日志与 checkpoint 规则（支持断点续跑）。

### 3.3 退出条件

1. 备份恢复演练通过。
2. 迁移脚本 dry-run 在测试环境通过。
3. 基线指标已记录（错误率、提交成功率、核心接口延迟）。

### 3.4 回滚动作

若准备阶段失败，停止迁移计划，不进行任何线上结构改动。

## 4. 阶段 B：基础设施与 Schema 扩展

### 4.1 进入条件

阶段 A 完成。

### 4.2 执行步骤

1. 新增集合与索引：
   - `questions`
   - `question_versions`
   - `question_banks`
   - （可选）`questionnaire_question_usages`
2. 扩展旧集合字段（仅新增不删除）：
   - `questionnaires.questions[*].questionVersionId`（过渡期可选）
   - `responses.answers[*].questionVersionId`
3. 校验索引与唯一约束：
   - `questions.questionKey` 唯一（UUID）
   - `question_versions.questionId + version` 唯一

### 4.3 退出条件

1. 所有新集合可读写。
2. 新增索引生效且未产生明显性能退化。

### 4.4 回滚动作

仅回滚新增集合/索引，不影响旧业务路径。

## 5. 阶段 C：后端兼容改造（双读/双写）

### 5.1 进入条件

阶段 B 完成。

### 5.2 执行步骤

1. 写路径改造：
   - 创建/编辑问卷时，同时维护新结构引用（`questionId + questionVersionId`）。
   - 创建题目时强制校验 `questionKey` 为 UUID。
2. 读路径改造：
   - 优先读取新结构。
   - 新结构缺失时回退旧结构（过渡期）。
3. 新接口上线：
   - 题目版本管理：`/questions/:id/versions`、`/questions/:id/restore`
   - 题库管理：`/question-banks/*` 及共享接口
4. 旧接口兼容：
   - 对旧请求保持向后兼容响应（必要时在服务层做适配转换）。

### 5.3 退出条件

1. 后端在测试环境可同时支撑新旧路径。
2. 关键接口冒烟通过（创建问卷、发布、填写、统计、题库共享）。

### 5.4 回滚动作

1. 关闭新读路径开关，回退到旧路径。
2. 保留新写入数据，不删除，避免二次迁移丢失。

## 6. 阶段 D：数据回填与一致性修复

### 6.1 进入条件

阶段 C 完成，且兼容读写已验证。

### 6.2 字段映射规则

| 旧字段 | 新字段 | 规则 |
|---|---|---|
| `questionnaires.questions[].questionId`（历史 string） | `questions.questionKey`（UUID） | 旧值非 UUID 时生成 UUID，并写映射表 |
| `questionnaires.questions[]` 题目结构 | `question_versions.schema` | 初始化为 v1（或按幂等规则复用） |
| 题目顺序 | `questionnaires.questions[].order` | 保持原顺序 |
| `responses.answers[].questionId` | `responses.answers[].questionId`（ObjectId） | 通过映射表归一化 |
| - | `responses.answers[].questionVersionId` | 按问卷已绑定版本回填 |
| `question_bank_items[]`（若有） | `question_banks.items[]` | 归并为题库内嵌数组 |
| `question_shares[]`（若有） | `question_banks.sharedWith[]` | 归并到题库共享关系 |

### 6.3 执行步骤

1. 按批次扫描问卷，创建或复用 `questions` 主记录。
2. 生成 `question_versions`（默认从 v1 起）。
3. 回填问卷题目引用中的 `questionVersionId`。
4. 回填答卷 `questionVersionId`。
5. 迁移题库数据并完成共享关系合并。
6. 对失败批次重试（幂等执行，不重复污染数据）。

### 6.4 退出条件

1. 回填成功率达到目标（建议 100%，至少达到业务约定阈值）。
2. 一致性检查全部通过（见第 10 节）。

### 6.5 回滚动作

1. 停止后续批次。
2. 使用阶段 A 备份按批次回退。
3. 保留映射表与失败日志用于修复后重跑。

## 7. 阶段 E：前端切换与兼容发布

### 7.1 进入条件

后端兼容能力与数据回填已达标。

### 7.2 执行步骤

1. 前端问卷编辑器改为使用新题目/题库接口。
2. 填写页提交答案时携带 `questionVersionId`。
3. 统计页按 `questionId + questionVersionId` 展示。
4. 保留异常降级：当新字段缺失时提示并回退到可读模式。

### 7.3 退出条件

1. 前端构建通过。
2. 端到端关键流程通过（登录、建卷、发布、填写、统计、共享题库）。

### 7.4 回滚动作

回滚前端镜像到上一稳定版本，后端继续保持兼容模式。

## 8. 阶段 F：灰度放量与观测

### 8.1 灰度策略

1. 先按用户组/租户灰度。
2. 再按流量比例逐步放量（如 10% -> 30% -> 50% -> 100%）。

### 8.2 观测指标

1. 接口错误率（4xx/5xx）。
2. 问卷提交成功率。
3. 统计接口耗时。
4. 回填字段缺失率（`questionVersionId` 缺失占比）。

### 8.3 升级/降级判定

1. 指标持续稳定达到阈值：继续放量。
2. 指标异常：立即停止放量并回退读路径开关。

## 9. 阶段 G：全量切换与旧结构下线

### 9.1 进入条件

灰度期稳定，无 P1/P2 级故障。

### 9.2 执行步骤

1. 切换到仅新结构读路径。
2. 停止旧结构写入。
3. 保留旧字段只读观察窗口（建议 1~2 个发布周期）。
4. 观察窗口结束后，清理旧逻辑与历史兼容代码。

### 9.3 退出条件

1. 全量用户在新结构下稳定运行。
2. 旧结构已不再被业务依赖。

### 9.4 回滚动作

在观察窗口内可快速恢复旧读路径；观察窗口后需依赖备份与脚本恢复。

## 10. 迁移验收清单（必须通过）

1. 结构一致性：
   - 已发布问卷题目均绑定 `questionVersionId`。
2. 引用完整性：
   - `questionId` 与 `questionVersionId` 均可命中目标集合。
3. 答卷可追溯：
   - 每条答卷答案能回溯到具体版本。
4. 共享约束正确：
   - 系统仅存在题库共享，不存在题目直接共享。
5. UUID 约束：
   - 新建题目 `questionKey` 全量符合 UUID 规则。
6. 业务闭环可用：
   - 注册/登录 -> 建卷 -> 发布 -> 填写 -> 统计全流程通过。

## 11. 回滚总策略（系统级）

### 11.1 回滚触发

满足任一条件即触发：

1. 提交成功率明显下降并持续。
2. 关键接口持续高错误率。
3. 数据一致性检查失败且短时无法修复。

### 11.2 回滚优先级

1. **优先开关回退**：读路径回退旧结构（最快）。
2. **版本回退**：前后端镜像回退到上一稳定版本。
3. **数据回退**：必要时使用备份恢复关键集合。

### 11.3 回滚后动作

1. 冻结变更。
2. 导出故障窗口日志、迁移批次日志、映射表。
3. 形成复盘并更新下一轮迁移策略。

## 12. 推荐交付物（迁移包）

1. `runbook`：按阶段可执行操作手册（含责任人）。
2. `migration-scripts`：支持 dry-run、幂等、断点续跑。
3. `mapping-export`：旧 ID 与新 `questionId/versionId` 映射导出。
4. `validation-report`：一致性与抽样核对报告。
5. `rollback-playbook`：开关回退、版本回退、数据回退步骤。

## 13. 风险与规避

1. 同名题误合并
   - 规避：以 `questionKey(UUID)` 作为主标识，不按标题合并。
2. 历史答卷无法绑定版本
   - 规避：先迁移问卷映射，再迁移答卷；严格按问卷上下文回填。
3. 灰度期间新旧逻辑混用导致统计偏差
   - 规避：统计统一基于 `questionId + questionVersionId`，并保留对账报表。
4. 迁移中断导致半完成状态
   - 规避：批次化 + checkpoint + 幂等重放。

## 14. 里程碑建议

1. M1：设计冻结与迁移方案评审通过
2. M2：测试环境全链路演练通过
3. M3：生产小流量灰度通过
4. M4：生产全量切换完成
5. M5：观察窗口结束并完成旧结构清理
