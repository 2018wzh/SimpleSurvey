<template>
<div class="container">
  <div class="header">
    <h1>我的题目</h1>
    <div>
      <button @click="openCreateDialog" style="margin-right:10px">+ 创建新题目</button>
      <button class="secondary" @click="$router.push('/')">返回</button>
    </div>
  </div>

  <div v-if="loading" class="card"><p>加载中...</p></div>
  <div v-else-if="questions.length === 0" class="card"><p>暂无保存的题目</p></div>

  <div v-for="q in questions" :key="q.id" class="card" style="margin-bottom:12px">
    <div style="display:flex;justify-content:space-between;align-items:center;cursor:pointer" @click="toggleExpand(q.id)">
      <div>
        <strong>{{ q.questionKey }}</strong>
        <p style="margin:4px 0 0;color:#888;font-size:13px">当前版本: v{{ q.currentVersion }} | 更新: {{ formatDate(q.updatedAt) }}</p>
      </div>
      <span style="font-size:12px;color:#888">{{ expanded[q.id] ? '收起' : '展开' }}</span>
    </div>

    <div v-if="expanded[q.id]" style="margin-top:15px;border-top:1px solid #eee;padding-top:15px">
      <!-- Actions -->
      <div style="margin-bottom:10px">
        <button @click="loadVersions(q.id)" style="margin-right:8px">版本历史</button>
        <button @click="loadUsages(q.id)" style="margin-right:8px">使用情况</button>
        <button @click="loadStats(q.id)" style="margin-right:8px">跨问卷统计</button>
        <button @click="startNewVersion(q.id)">创建新版本</button>
      </div>

      <!-- Versions -->
      <div v-if="versions[q.id]" style="margin-top:10px">
        <h4>版本历史</h4>
        <div v-for="v in versions[q.id]" :key="v.id" style="padding:8px;border:1px solid #eee;border-radius:4px;margin-bottom:6px">
          <div style="display:flex;justify-content:space-between">
            <span><strong>v{{ v.version }}</strong> <span style="color:#888;font-size:12px">({{ v.changeType }})</span></span>
            <button v-if="v.id !== q.currentVersionId" @click="restoreVersion(q.id, v.id)" style="padding:2px 8px;font-size:12px">恢复</button>
          </div>
          <p style="margin:4px 0;font-size:13px">标题: {{ v.schema.title }} | 类型: {{ typeLabel(v.schema.type) }}</p>
          <p v-if="v.note" style="margin:4px 0;font-size:12px;color:#888">备注: {{ v.note }}</p>
        </div>
      </div>

      <!-- New version form -->
      <div v-if="newVersionForm[q.id]" style="margin-top:10px;padding:10px;background:#f9f9f9;border-radius:6px">
        <h4>创建新版本</h4>
        <div class="form-group"><label>标题</label><input v-model="newVersionForm[q.id].title" /></div>
        <div class="form-group">
          <label>题型</label>
          <select v-model="newVersionForm[q.id].type">
            <option value="SINGLE_CHOICE">单选题</option>
            <option value="MULTIPLE_CHOICE">多选题</option>
            <option value="TEXT">文本填空</option>
            <option value="NUMBER">数字填空</option>
          </select>
        </div>
        <div class="form-group checkbox-row">
          <label class="checkbox-label"><input type="checkbox" v-model="newVersionForm[q.id].isRequired" /> 必答题</label>
        </div>
        <div v-if="newVersionForm[q.id].type === 'SINGLE_CHOICE' || newVersionForm[q.id].type === 'MULTIPLE_CHOICE'">
          <div v-for="(opt, oi) in newVersionForm[q.id].options" :key="oi" class="option-item">
            <input v-model="opt.text" placeholder="选项文本" style="flex:1" />
            <button class="danger" @click="newVersionForm[q.id].options.splice(oi,1)" style="padding:4px 8px">×</button>
          </div>
          <button @click="newVersionForm[q.id].options.push({optionId: nextOptId(), text: ''})" style="padding:4px 12px;margin-top:5px">+ 添加选项</button>
        </div>
        <div class="form-group"><label>备注</label><input v-model="newVersionForm[q.id].note" placeholder="可选" /></div>
        <button @click="submitNewVersion(q.id)">提交新版本</button>
        <button class="secondary" @click="newVersionForm[q.id]=null" style="margin-left:8px">取消</button>
      </div>

      <!-- Usages -->
      <div v-if="usages[q.id]" style="margin-top:10px">
        <h4>使用情况</h4>
        <table style="width:100%;border-collapse:collapse">
          <tr style="border-bottom:1px solid #eee">
            <th style="text-align:left;padding:6px">问卷</th>
            <th style="text-align:left;padding:6px">状态</th>
            <th style="text-align:left;padding:6px">版本ID</th>
          </tr>
          <tr v-for="u in usages[q.id]" :key="u.questionnaireId" style="border-bottom:1px solid #f5f5f5">
            <td style="padding:6px">{{ u.questionnaireTitle || u.questionnaireId }}</td>
            <td style="padding:6px">{{ u.status }}</td>
            <td style="padding:6px;font-size:12px">{{ u.questionVersionId }}</td>
          </tr>
        </table>
        <div v-if="usages[q.id].length===0" style="color:#888;font-size:13px">未被任何问卷使用</div>
      </div>

      <!-- Stats -->
      <div v-if="stats[q.id]" style="margin-top:10px">
        <h4>跨问卷统计</h4>
        <p>回答人数: {{ stats[q.id].totalAnswered || 0 }}</p>
        <div v-if="stats[q.id].type==='SINGLE_CHOICE'||stats[q.id].type==='MULTIPLE_CHOICE'">
          <table style="width:100%;border-collapse:collapse">
            <tr style="border-bottom:1px solid #eee"><th style="text-align:left;padding:6px">选项</th><th style="text-align:right;padding:6px">次数</th></tr>
            <tr v-for="(count, optId) in (stats[q.id].optionCounts || {})" :key="optId" style="border-bottom:1px solid #f5f5f5">
              <td style="padding:6px">{{ optId }}</td>
              <td style="text-align:right;padding:6px">{{ count }}</td>
            </tr>
          </table>
        </div>
        <div v-if="stats[q.id].type==='NUMBER' && stats[q.id].averageValue != null">
          <p>平均值: {{ stats[q.id].averageValue.toFixed(2) }}</p>
        </div>
        <div v-if="stats[q.id].type==='TEXT'">
          <details><summary>查看回答 ({{ (stats[q.id].textAnswers || []).length }})</summary>
            <ul><li v-for="(t,i) in stats[q.id].textAnswers" :key="i">{{ t }}</li></ul>
          </details>
        </div>
      </div>
    </div>
  </div>

  <!-- 创建新题目弹窗 -->
  <div v-if="createDialog" class="modal-overlay" @click.self="closeCreateDialog">
    <div class="card" style="max-width:600px;max-height:80vh;overflow:auto;margin:50px auto;">
      <h3>创建新题目</h3>
      <div class="form-group"><label>题目标题</label><input v-model="newQuestion.title" /></div>
      <div class="form-group">
        <label>题型</label>
        <select v-model="newQuestion.type" @change="onNewQuestionTypeChange">
          <option value="SINGLE_CHOICE">单选题</option>
          <option value="MULTIPLE_CHOICE">多选题</option>
          <option value="TEXT">文本填空</option>
          <option value="NUMBER">数字填空</option>
        </select>
      </div>
      <div class="form-group checkbox-row">
        <label class="checkbox-label"><input type="checkbox" v-model="newQuestion.isRequired" /> 必答题</label>
      </div>
      <div v-if="newQuestion.type === 'SINGLE_CHOICE' || newQuestion.type === 'MULTIPLE_CHOICE'">
        <div v-for="(opt, oi) in newQuestion.options" :key="oi" class="option-item">
          <input v-model="opt.text" placeholder="选项文本" style="flex:1" />
          <button class="danger" @click="newQuestion.options.splice(oi,1)" style="padding:4px 8px">×</button>
        </div>
        <button @click="addNewOption" style="padding:4px 12px;margin-top:5px">+ 添加选项</button>
      </div>
      <div v-if="newQuestion.type === 'MULTIPLE_CHOICE'" style="margin-top:10px">
        <div class="form-group"><label>最少选择</label><input v-model.number="newQuestion.validation.minSelect" type="number" min="0" /></div>
        <div class="form-group"><label>最多选择</label><input v-model.number="newQuestion.validation.maxSelect" type="number" min="0" /></div>
      </div>
      <div v-if="newQuestion.type === 'TEXT'" style="margin-top:10px">
        <div class="form-group"><label>最少字数</label><input v-model.number="newQuestion.validation.minLength" type="number" min="0" /></div>
        <div class="form-group"><label>最多字数</label><input v-model.number="newQuestion.validation.maxLength" type="number" min="0" /></div>
      </div>
      <div v-if="newQuestion.type === 'NUMBER'" style="margin-top:10px">
        <div class="form-group"><label>最小值</label><input v-model.number="newQuestion.validation.minVal" type="number" /></div>
        <div class="form-group"><label>最大值</label><input v-model.number="newQuestion.validation.maxVal" type="number" /></div>
        <div class="form-group"><label class="checkbox-label"><input type="checkbox" v-model="newQuestion.integerOnly" /> 必须为整数</label></div>
      </div>
      <div class="form-group"><label>备注</label><input v-model="newQuestion.note" placeholder="可选" /></div>
      <div style="margin-top:15px">
        <button @click="submitNewQuestion">保存</button>
        <button class="secondary" @click="closeCreateDialog" style="margin-left:8px">取消</button>
      </div>
    </div>
  </div>
</div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import api from '../api'

const questions = ref([])
const loading = ref(false)
const expanded = reactive({})
const versions = reactive({})
const usages = reactive({})
const stats = reactive({})
const newVersionForm = reactive({})
let optIdCounter = 0
let createOptIdCounter = 0

const createDialog = ref(false)
const newQuestion = reactive({
  title: '',
  type: 'SINGLE_CHOICE',
  isRequired: false,
  options: [{ optionId: 'co1', text: '' }, { optionId: 'co2', text: '' }],
  validation: { minSelect: 0, maxSelect: 0, minLength: 0, maxLength: 0, minVal: null, maxVal: null },
  integerOnly: false,
  note: ''
})

function nextCreateOptId() {
  return 'co' + (++createOptIdCounter)
}

function openCreateDialog() {
  createDialog.value = true
  newQuestion.title = ''
  newQuestion.type = 'SINGLE_CHOICE'
  newQuestion.isRequired = false
  newQuestion.options = [{ optionId: nextCreateOptId(), text: '' }, { optionId: nextCreateOptId(), text: '' }]
  newQuestion.validation = { minSelect: 0, maxSelect: 0, minLength: 0, maxLength: 0, minVal: null, maxVal: null }
  newQuestion.integerOnly = false
  newQuestion.note = ''
}

function closeCreateDialog() {
  createDialog.value = false
}

function addNewOption() {
  newQuestion.options.push({ optionId: nextCreateOptId(), text: '' })
}

function onNewQuestionTypeChange() {
  if (newQuestion.type === 'SINGLE_CHOICE' || newQuestion.type === 'MULTIPLE_CHOICE') {
    if (!newQuestion.options || newQuestion.options.length === 0) {
      newQuestion.options = [{ optionId: nextCreateOptId(), text: '' }, { optionId: nextCreateOptId(), text: '' }]
    }
  }
}

function buildNewQuestionSchema() {
  const schema = {
    type: newQuestion.type,
    title: newQuestion.title,
    isRequired: newQuestion.isRequired,
    meta: {}
  }
  if (newQuestion.type === 'SINGLE_CHOICE' || newQuestion.type === 'MULTIPLE_CHOICE') {
    schema.options = newQuestion.options.filter(o => o.text.trim()).map(o => ({ optionId: o.optionId, text: o.text }))
    if (schema.options.length < 2) {
      alert('选择题至少需要2个选项')
      return null
    }
  }
  const v = {}
  if (newQuestion.type === 'MULTIPLE_CHOICE') {
    if (newQuestion.validation.minSelect) v.minSelect = newQuestion.validation.minSelect
    if (newQuestion.validation.maxSelect) v.maxSelect = newQuestion.validation.maxSelect
  }
  if (newQuestion.type === 'TEXT') {
    if (newQuestion.validation.minLength) v.minLength = newQuestion.validation.minLength
    if (newQuestion.validation.maxLength) v.maxLength = newQuestion.validation.maxLength
  }
  if (newQuestion.type === 'NUMBER') {
    if (newQuestion.validation.minVal != null && newQuestion.validation.minVal !== '') v.minVal = newQuestion.validation.minVal
    if (newQuestion.validation.maxVal != null && newQuestion.validation.maxVal !== '') v.maxVal = newQuestion.validation.maxVal
    if (newQuestion.integerOnly) v.numberType = 'integer'
  }
  if (Object.keys(v).length) schema.validation = v
  return schema
}

async function submitNewQuestion() {
  const schema = buildNewQuestionSchema()
  if (!schema) return
  try {
    await api.createQuestion({ questionKey: crypto.randomUUID(), schema, tags: [] })
    alert('创建成功')
    closeCreateDialog()
    loadData()
  } catch (e) {
    alert(e.response?.data?.message || '创建失败')
  }
}

function formatDate(d) {
  return d ? new Date(d).toLocaleString('zh-CN') : ''
}
function typeLabel(t) {
  return { SINGLE_CHOICE: '单选题', MULTIPLE_CHOICE: '多选题', TEXT: '文本填空', NUMBER: '数字填空' }[t] || t
}
function nextOptId() {
  return 'vo' + (++optIdCounter)
}

async function loadData() {
  loading.value = true
  try {
    const res = await api.getMyQuestions({ limit: 100 })
    questions.value = res.data.data.items || []
  } catch (e) {
    alert('加载失败')
  } finally {
    loading.value = false
  }
}

function toggleExpand(id) {
  expanded[id] = !expanded[id]
}

async function loadVersions(id) {
  try {
    const res = await api.getQuestionVersions(id)
    versions[id] = res.data.data || []
  } catch (e) {
    alert('加载版本失败')
  }
}

async function restoreVersion(questionId, fromVersionId) {
  if (!confirm('确定恢复到此版本？这将创建一个新版本。')) return
  try {
    await api.restoreQuestionVersion(questionId, { fromVersionId, note: '恢复旧版本' })
    alert('恢复成功')
    loadVersions(questionId)
    loadData()
  } catch (e) {
    alert(e.response?.data?.message || '恢复失败')
  }
}

function startNewVersion(id) {
  // Pre-fill with current version schema if available
  const current = versions[id]?.length ? versions[id][versions[id].length - 1].schema : { type: 'SINGLE_CHOICE', title: '', isRequired: false, options: [{optionId: nextOptId(), text: ''}, {optionId: nextOptId(), text: ''}], validation: {} }
  newVersionForm[id] = {
    type: current.type,
    title: current.title,
    isRequired: current.isRequired,
    options: current.options ? current.options.map(o => ({ optionId: o.optionId, text: o.text })) : [],
    validation: current.validation || {},
    note: ''
  }
}

async function submitNewVersion(questionId) {
  const f = newVersionForm[questionId]
  const schema = {
    type: f.type,
    title: f.title,
    isRequired: f.isRequired,
    meta: {}
  }
  if (f.type === 'SINGLE_CHOICE' || f.type === 'MULTIPLE_CHOICE') {
    schema.options = f.options.filter(o => o.text.trim()).map(o => ({ optionId: o.optionId, text: o.text }))
    if (schema.options.length < 2) { alert('至少需要2个选项'); return }
  }
  const v = {}
  if (f.type === 'MULTIPLE_CHOICE') {
    if (f.validation.minSelect) v.minSelect = f.validation.minSelect
    if (f.validation.maxSelect) v.maxSelect = f.validation.maxSelect
  }
  if (f.type === 'TEXT') {
    if (f.validation.minLength) v.minLength = f.validation.minLength
    if (f.validation.maxLength) v.maxLength = f.validation.maxLength
  }
  if (f.type === 'NUMBER') {
    if (f.validation.minVal != null && f.validation.minVal !== '') v.minVal = f.validation.minVal
    if (f.validation.maxVal != null && f.validation.maxVal !== '') v.maxVal = f.validation.maxVal
    if (f.validation.numberType) v.numberType = f.validation.numberType
  }
  if (Object.keys(v).length) schema.validation = v

  try {
    await api.createQuestionVersion(questionId, { schema, changeType: 'edit', note: f.note })
    alert('创建成功')
    newVersionForm[questionId] = null
    loadVersions(questionId)
    loadData()
  } catch (e) {
    alert(e.response?.data?.message || '创建失败')
  }
}

async function loadUsages(id) {
  try {
    const res = await api.getQuestionUsages(id)
    usages[id] = res.data.data || []
  } catch (e) {
    alert('加载使用情况失败')
  }
}

async function loadStats(id) {
  try {
    const res = await api.getQuestionStats(id)
    stats[id] = res.data.data || {}
  } catch (e) {
    alert('加载统计失败')
  }
}

onMounted(loadData)
</script>
