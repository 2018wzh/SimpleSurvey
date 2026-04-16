<template>
<div class="container">
  <div class="header">
    <h1>创建问卷</h1>
    <div>
      <button @click="submit">保存问卷</button>
      <button class="secondary" @click="$router.push('/')" style="margin-left:10px">返回</button>
    </div>
  </div>

  <div class="card">
    <div class="form-group"><label>问卷标题</label><input v-model="form.title" required /></div>
    <div class="form-group"><label>问卷说明</label><textarea v-model="form.description" rows="2"></textarea></div>
    <div class="form-group checkbox-group">
      <label class="checkbox-label">
        <input type="checkbox" v-model="form.settings.allowAnonymous" />
        <span>允许匿名填写</span>
      </label>
    </div>
  </div>

  <h2 style="margin:20px 0 10px">题目列表</h2>
  <div style="margin-bottom:15px">
    <button @click="addLocalQuestion">+ 新建题目</button>
    <button @click="openBankDialog" style="margin-left:10px">从题库选择</button>
    <button @click="openMyQuestionDialog" style="margin-left:10px">从已有题目选择</button>
  </div>

  <div class="question-item" v-for="(q, qi) in form.questions" :key="q.tempId">
    <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:10px">
      <strong>题目 {{ qi + 1 }}</strong>
      <div>
        <button @click="editQuestion(qi)" style="padding:4px 10px;margin-right:5px">编辑</button>
        <button class="danger" @click="removeQ(qi)" style="padding:4px 10px">删除</button>
      </div>
    </div>

    <div v-if="q.editing">
      <div class="form-group"><label>题目标题</label><input v-model="q.title" /></div>
      <div class="form-group">
        <label>题型</label>
        <select v-model="q.type" @change="onTypeChange(q)">
          <option value="SINGLE_CHOICE">单选题</option>
          <option value="MULTIPLE_CHOICE">多选题</option>
          <option value="TEXT">文本填空</option>
          <option value="NUMBER">数字填空</option>
        </select>
      </div>
      <div class="form-group checkbox-row">
        <label class="checkbox-label">
          <input type="checkbox" v-model="q.isRequired" />
          <span>必答题</span>
        </label>
      </div>
      <div v-if="q.type === 'SINGLE_CHOICE' || q.type === 'MULTIPLE_CHOICE'">
        <div class="option-item" v-for="(opt, oi) in q.options" :key="oi">
          <input v-model="opt.text" :placeholder="'选项' + (oi + 1)" style="flex:1" />
          <button class="danger" @click="q.options.splice(oi, 1)" style="padding:4px 8px">×</button>
        </div>
        <button @click="addOption(q)" style="padding:4px 12px;margin-top:5px">+ 添加选项</button>
      </div>
      <div v-if="q.type === 'MULTIPLE_CHOICE'" style="margin-top:10px">
        <div class="form-group"><label>最少选择</label><input v-model.number="q.validation.minSelect" type="number" min="0" /></div>
        <div class="form-group"><label>最多选择</label><input v-model.number="q.validation.maxSelect" type="number" min="0" /></div>
      </div>
      <div v-if="q.type === 'TEXT'" style="margin-top:10px">
        <div class="form-group"><label>最少字数</label><input v-model.number="q.validation.minLength" type="number" min="0" /></div>
        <div class="form-group"><label>最多字数</label><input v-model.number="q.validation.maxLength" type="number" min="0" /></div>
      </div>
      <div v-if="q.type === 'NUMBER'" style="margin-top:10px">
        <div class="form-group"><label>最小值</label><input v-model.number="q.validation.minVal" type="number" /></div>
        <div class="form-group"><label>最大值</label><input v-model.number="q.validation.maxVal" type="number" /></div>
        <div class="form-group"><label><input type="checkbox" v-model="q.integerOnly" /> 必须为整数</label></div>
      </div>
      <div style="margin-top:10px">
        <button @click="q.editing=false">完成编辑</button>
      </div>
    </div>
    <div v-else>
      <p><strong>{{ q.title }}</strong> <span style="color:#888;font-size:12px">({{ typeLabel(q.type) }})</span></p>
      <p v-if="q.isRequired" style="color:red;font-size:12px">* 必答题</p>
      <p v-if="q.source!=='local'" style="color:#888;font-size:12px">来源: {{ q.source==='bank'?'题库':'已有题目' }} | 版本: {{ q.questionVersionId }}</p>
      <p v-if="q.source!=='local'" style="color:#888;font-size:12px">提示：点击“编辑”修改后，保存问卷时会自动为该题创建新版本，不影响旧问卷</p>
    </div>
  </div>

  <div v-if="form.questions.length===0" class="card" style="color:#888">暂无题目，请添加</div>

  <h2 style="margin:20px 0 10px">跳转逻辑</h2>
  <div class="question-item" v-for="(rule, ri) in form.logicRules" :key="ri">
    <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:10px">
      <strong>规则 {{ ri + 1 }}</strong>
      <button class="danger" @click="form.logicRules.splice(ri, 1)" style="padding:4px 10px">删除</button>
    </div>

    <div class="form-group">
      <label>源题目</label>
      <select v-model="rule.conditionQuestionId">
        <option value="">请选择源题目</option>
        <option v-for="q in form.questions" :key="q.questionId" :value="q.questionId">
          {{ q.questionId }} - {{ q.title || '未命名' }}
        </option>
      </select>
    </div>

    <div class="form-group">
      <label>条件类型</label>
      <select v-model="rule.operator">
        <option value="EQUALS">等于</option>
        <option value="CONTAINS">包含</option>
        <option value="GREATER_THAN">大于</option>
        <option value="LESS_THAN">小于</option>
      </select>
    </div>

    <div class="form-group">
      <label>条件值（选项ID或数字）</label>
      <select v-if="isChoiceQuestion(rule.conditionQuestionId)" v-model="rule.conditionValue">
        <option value="">请选择条件值</option>
        <option v-for="opt in getQuestionOptions(rule.conditionQuestionId)" :key="opt.optionId" :value="opt.optionId">
          {{ opt.optionId }} - {{ opt.text || '未命名' }}
        </option>
      </select>
      <input v-else v-model="rule.conditionValue" :type="isNumberQuestion(rule.conditionQuestionId) ? 'number' : 'text'" />
    </div>

    <div class="form-group">
      <label>跳转到题目</label>
      <select v-model="rule.targetQuestionId">
        <option value="">请选择目标题目</option>
        <option v-for="q in form.questions" :key="q.questionId" :value="q.questionId">
          {{ q.questionId }} - {{ q.title || '未命名' }}
        </option>
      </select>
    </div>
  </div>

  <button @click="addRule">+ 添加跳转规则</button>
  <div class="error" v-if="error" style="margin-top:15px">{{ error }}</div>

  <!-- 从题库选择对话框 -->
  <div v-if="bankDialog" class="modal-overlay" @click.self="bankDialog=false">
    <div class="card" style="max-width:600px;max-height:80vh;overflow:auto;margin:50px auto;">
      <h3>从题库选择题目</h3>
      <div v-if="banksLoading">加载中...</div>
      <div v-else>
        <div v-for="bank in banks" :key="bank.id" style="margin-bottom:15px;border:1px solid #eee;padding:10px;border-radius:6px">
          <strong>{{ bank.name }}</strong>
          <div v-if="bank.items && bank.items.length">
            <div v-for="item in bank.items" :key="item.questionId" style="display:flex;justify-content:space-between;align-items:center;margin-top:6px">
              <span style="font-size:14px">{{ item.questionTitle || item.questionId }}</span>
              <button @click="addFromBank(item)" style="padding:2px 8px;font-size:12px">选用</button>
            </div>
          </div>
          <div v-else style="color:#888;font-size:12px">题库为空</div>
        </div>
        <div v-if="banks.length===0" style="color:#888">暂无可用的题库</div>
      </div>
      <button class="secondary" @click="bankDialog=false" style="margin-top:10px">关闭</button>
    </div>
  </div>

  <!-- 从已有题目选择对话框 -->
  <div v-if="myQuestionDialog" class="modal-overlay" @click.self="myQuestionDialog=false">
    <div class="card" style="max-width:600px;max-height:80vh;overflow:auto;margin:50px auto;">
      <h3>从已有题目选择</h3>
      <div v-if="questionsLoading">加载中...</div>
      <div v-else>
        <div v-for="q in myQuestions" :key="q.id" style="display:flex;justify-content:space-between;align-items:center;margin-bottom:8px;padding:8px;border:1px solid #eee;border-radius:6px">
          <div>
            <div style="font-size:14px"><strong>{{ q.questionKey }}</strong></div>
            <div style="color:#888;font-size:12px">当前版本: v{{ q.currentVersion }}</div>
          </div>
          <button @click="addFromMyQuestion(q)" style="padding:2px 8px;font-size:12px">选用</button>
        </div>
        <div v-if="myQuestions.length===0" style="color:#888">暂无已保存的题目</div>
      </div>
      <button class="secondary" @click="myQuestionDialog=false" style="margin-top:10px">关闭</button>
    </div>
  </div>
</div>
</template>

<script setup>
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import api from '../api'

const router = useRouter()
const error = ref('')
let tempCounter = 0
let optCounter = 0

const form = reactive({
  title: '',
  description: '',
  settings: { allowAnonymous: false, duplicateCheck: 'none', themeColor: '#1677ff' },
  questions: [],
  logicRules: []
})

function nextTempId() {
  return 't' + (++tempCounter)
}
function nextOptionId() {
  return 'o' + (++optCounter)
}
function nextQuestionId() {
  // Fallback local id if needed
  return 'lq' + (++tempCounter)
}

function buildValidation(q) {
  const v = {}
  if (q.type === 'MULTIPLE_CHOICE') {
    if (q.validation.minSelect) v.minSelect = q.validation.minSelect
    if (q.validation.maxSelect) v.maxSelect = q.validation.maxSelect
  }
  if (q.type === 'TEXT') {
    if (q.validation.minLength) v.minLength = q.validation.minLength
    if (q.validation.maxLength) v.maxLength = q.validation.maxLength
  }
  if (q.type === 'NUMBER') {
    if (q.validation.minVal !== null && q.validation.minVal !== '') v.minVal = q.validation.minVal
    if (q.validation.maxVal !== null && q.validation.maxVal !== '') v.maxVal = q.validation.maxVal
    if (q.integerOnly) v.numberType = 'integer'
  }
  return v
}

function makeSnapshot(q) {
  const snapshot = {
    type: q.type,
    title: q.title,
    isRequired: q.isRequired,
    meta: q.meta || {}
  }
  if (q.type === 'SINGLE_CHOICE' || q.type === 'MULTIPLE_CHOICE') {
    snapshot.options = (q.options || []).filter(o => o.text.trim()).map(o => ({ optionId: o.optionId, text: o.text, hasOtherInput: false }))
  }
  const v = buildValidation(q)
  if (Object.keys(v).length > 0) snapshot.validation = v
  return snapshot
}

function snapshotsEqual(a, b) {
  return JSON.stringify(a) === JSON.stringify(b)
}

function typeLabel(t) {
  return { SINGLE_CHOICE: '单选题', MULTIPLE_CHOICE: '多选题', TEXT: '文本填空', NUMBER: '数字填空' }[t] || t
}

function createEmptyLocalQuestion() {
  const q = {
    tempId: nextTempId(),
    source: 'local',
    editing: true,
    questionId: nextQuestionId(),
    questionVersionId: '',
    type: 'SINGLE_CHOICE',
    title: '',
    isRequired: false,
    options: [
      { optionId: nextOptionId(), text: '' },
      { optionId: nextOptionId(), text: '' }
    ],
    validation: { minSelect: 0, maxSelect: 0, minLength: 0, maxLength: 0, minVal: null, maxVal: null },
    integerOnly: false,
    meta: {}
  }
  q.originalSnapshot = makeSnapshot(q)
  q.originalQuestionVersionId = ''
  return q
}

function addLocalQuestion() {
  form.questions.push(createEmptyLocalQuestion())
}
function editQuestion(idx) {
  form.questions[idx].editing = true
}
function removeQ(i) {
  form.questions.splice(i, 1)
}
function addOption(q) {
  q.options.push({ optionId: nextOptionId(), text: '' })
}
function onTypeChange(q) {
  if (q.type === 'SINGLE_CHOICE' || q.type === 'MULTIPLE_CHOICE') {
    if (!q.options || q.options.length === 0) {
      q.options = [
        { optionId: nextOptionId(), text: '' },
        { optionId: nextOptionId(), text: '' }
      ]
    }
  }
}

function getQuestionById(questionId) {
  return form.questions.find(q => q.questionId === questionId)
}
function isChoiceQuestion(questionId) {
  const q = getQuestionById(questionId)
  return q && (q.type === 'SINGLE_CHOICE' || q.type === 'MULTIPLE_CHOICE')
}
function isNumberQuestion(questionId) {
  const q = getQuestionById(questionId)
  return q && q.type === 'NUMBER'
}
function getQuestionOptions(questionId) {
  const q = getQuestionById(questionId)
  return q?.options || []
}
function addRule() {
  form.logicRules.push({ conditionQuestionId: '', operator: 'EQUALS', conditionValue: '', targetQuestionId: '' })
}

// ===== Bank selection =====
const bankDialog = ref(false)
const banks = ref([])
const banksLoading = ref(false)

async function openBankDialog() {
  bankDialog.value = true
  banksLoading.value = true
  try {
    const res = await api.getQuestionBanks({ limit: 100 })
    banks.value = res.data.data.items || []
    // For each bank item we don't have title from backend, keep as-is
  } catch (e) {
    alert('加载题库失败')
  } finally {
    banksLoading.value = false
  }
}

async function addFromBank(item) {
  // We only have questionId and pinnedVersionId from bank item.
  // We need to fetch the version schema to display it.
  try {
    const versionRes = await api.getQuestionVersions(item.questionId)
    const versions = versionRes.data.data || []
    const targetVersionId = item.pinnedVersionId || versions[versions.length - 1]?.id
    const version = versions.find(v => v.id === targetVersionId) || versions[versions.length - 1]
    if (!version) {
      alert('无法获取题库题目信息')
      return
    }
    const schema = version.schema
    const q = {
      tempId: nextTempId(),
      source: 'bank',
      editing: false,
      questionId: item.questionId,
      questionVersionId: version.id,
      type: schema.type,
      title: schema.title,
      isRequired: schema.isRequired,
      options: schema.options ? schema.options.map(o => ({ optionId: o.optionId, text: o.text })) : [],
      validation: schema.validation || { minSelect: 0, maxSelect: 0, minLength: 0, maxLength: 0, minVal: null, maxVal: null },
      integerOnly: schema.validation?.numberType === 'integer',
      meta: schema.meta || {}
    }
    q.originalSnapshot = makeSnapshot(q)
    q.originalQuestionVersionId = version.id
    form.questions.push(q)
    bankDialog.value = false
  } catch (e) {
    alert('加载题目详情失败')
  }
}

// ===== My questions selection =====
const myQuestionDialog = ref(false)
const myQuestions = ref([])
const questionsLoading = ref(false)

async function openMyQuestionDialog() {
  myQuestionDialog.value = true
  questionsLoading.value = true
  try {
    const res = await api.getMyQuestions({ limit: 100 })
    myQuestions.value = res.data.data.items || []
  } catch (e) {
    alert('加载题目失败')
  } finally {
    questionsLoading.value = false
  }
}

async function addFromMyQuestion(qEntity) {
  try {
    const versionRes = await api.getQuestionVersions(qEntity.id)
    const versions = versionRes.data.data || []
    const currentVersion = versions.find(v => v.id === qEntity.currentVersionId) || versions[versions.length - 1]
    if (!currentVersion) {
      alert('无法获取题目版本信息')
      return
    }
    const schema = currentVersion.schema
    const q = {
      tempId: nextTempId(),
      source: 'existing',
      editing: false,
      questionId: qEntity.id,
      questionVersionId: currentVersion.id,
      type: schema.type,
      title: schema.title,
      isRequired: schema.isRequired,
      options: schema.options ? schema.options.map(o => ({ optionId: o.optionId, text: o.text })) : [],
      validation: schema.validation || { minSelect: 0, maxSelect: 0, minLength: 0, maxLength: 0, minVal: null, maxVal: null },
      integerOnly: schema.validation?.numberType === 'integer',
      meta: schema.meta || {}
    }
    q.originalSnapshot = makeSnapshot(q)
    q.originalQuestionVersionId = currentVersion.id
    form.questions.push(q)
    myQuestionDialog.value = false
  } catch (e) {
    alert('加载题目详情失败')
  }
}

// ===== Submit =====
async function submit() {
  error.value = ''
  if (!form.title.trim()) { error.value = '请输入问卷标题'; return }
  if (form.questions.length === 0) { error.value = '请至少添加一道题目'; return }

  // Ensure all local questions are saved to backend first
  // For bank/existing questions that have been edited, create a new version automatically
  for (let i = 0; i < form.questions.length; i++) {
    const q = form.questions[i]
    if (q.source === 'local') {
      if (!q.title.trim()) { error.value = `题目 ${i + 1} 标题不能为空`; return }
      if ((q.type === 'SINGLE_CHOICE' || q.type === 'MULTIPLE_CHOICE') && q.options.filter(o => o.text.trim()).length < 2) {
        error.value = `题目 ${i + 1} 至少需要2个选项`; return
      }
      const snapshot = makeSnapshot(q)
      const questionKey = crypto.randomUUID()
      try {
        const res = await api.createQuestion({ questionKey, schema: snapshot, tags: [] })
        const result = res.data.data
        q.questionId = result.id
        q.questionVersionId = result.versionId
      } catch (e) {
        error.value = e.response?.data?.message || `保存题目 ${i + 1} 失败`
        return
      }
    } else if ((q.source === 'bank' || q.source === 'existing') && q.originalSnapshot) {
      const currentSnapshot = makeSnapshot(q)
      if (!snapshotsEqual(currentSnapshot, q.originalSnapshot)) {
        try {
          const res = await api.createQuestionVersion(q.questionId, {
            schema: currentSnapshot,
            baseVersionId: q.originalQuestionVersionId,
            changeType: 'edit',
            note: '在问卷编辑中修改生成新版本'
          })
          q.questionVersionId = res.data.data.versionId
        } catch (e) {
          error.value = e.response?.data?.message || `题目 ${i + 1} 版本更新失败，请确认您有权限修改该题目`
          return
        }
      }
    }
  }

  const payload = {
    title: form.title,
    description: form.description,
    settings: { ...form.settings },
    questions: form.questions.map((q, idx) => {
      const snapshot = makeSnapshot(q)
      return {
        questionId: q.questionId,
        questionVersionId: q.questionVersionId,
        order: idx,
        snapshot: snapshot,
        type: q.type,
        title: q.title,
        isRequired: q.isRequired,
        options: (q.type === 'SINGLE_CHOICE' || q.type === 'MULTIPLE_CHOICE') ? snapshot.options : undefined,
        validation: snapshot.validation,
        meta: q.meta || {}
      }
    }),
    logicRules: form.logicRules
      .filter(r => r.conditionQuestionId && r.targetQuestionId)
      .map(r => ({
        conditionQuestionId: r.conditionQuestionId,
        operator: r.operator,
        conditionValue: r.conditionValue,
        action: 'JUMP_TO',
        actionDetails: { targetQuestionId: r.targetQuestionId }
      }))
  }

  try {
    await api.createQuestionnaire(payload)
    alert('创建成功')
    router.push('/')
  } catch (e) {
    error.value = e.response?.data?.message || '创建失败'
  }
}
</script>

<style scoped>
.modal-overlay {
  position: fixed;
  top: 0; left: 0; right: 0; bottom: 0;
  background: rgba(0,0,0,0.4);
  z-index: 100;
}
</style>
