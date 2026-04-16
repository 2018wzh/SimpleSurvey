<template>
<div class="container" style="max-width:700px">
  <div v-if="loading" class="card"><p>加载中...</p></div>
  <div v-else-if="errMsg" class="card"><p class="error">{{ errMsg }}</p></div>
  <div v-else-if="submitted" class="card"><h2>提交成功！</h2><p>感谢您的参与。</p><button @click="$router.push('/')">返回首页</button></div>
  <div v-else>
    <div class="card">
      <h1>{{ survey.title }}</h1>
      <p v-if="survey.description">{{ survey.description }}</p>
    </div>

    <!-- Current question -->
    <div class="card" v-if="currentQ">
      <h3>{{ currentQIndex + 1 }}. {{ currentQ.title }}<span v-if="currentQ.isRequired" style="color:red"> *</span></h3>

      <!-- SINGLE_CHOICE -->
      <div v-if="currentQ.type==='SINGLE_CHOICE'">
        <div v-for="opt in currentQ.options" :key="opt.optionId" style="margin:8px 0">
          <label class="choice-label">
            <input type="radio" :value="opt.optionId" v-model="answers[currentQ.questionId]" />
            <span>{{ opt.text }}</span>
          </label>
        </div>
      </div>

      <!-- MULTIPLE_CHOICE -->
      <div v-if="currentQ.type==='MULTIPLE_CHOICE'">
        <div v-for="opt in currentQ.options" :key="opt.optionId" style="margin:8px 0">
          <label class="choice-label">
            <input type="checkbox" :value="opt.optionId" v-model="answers[currentQ.questionId]" />
            <span>{{ opt.text }}</span>
          </label>
        </div>
        <p style="font-size:12px;color:#888" v-if="getValidation(currentQ, 'minSelect') || getValidation(currentQ, 'maxSelect')">
          <span v-if="getValidation(currentQ, 'minSelect')">至少选 {{ getValidation(currentQ, 'minSelect') }} 个</span>
          <span v-if="getValidation(currentQ, 'maxSelect')"> 最多选 {{ getValidation(currentQ, 'maxSelect') }} 个</span>
        </p>
      </div>

      <!-- TEXT -->
      <div v-if="currentQ.type==='TEXT'">
        <textarea v-model="answers[currentQ.questionId]" rows="3" style="width:100%"></textarea>
        <p style="font-size:12px;color:#888" v-if="getValidation(currentQ, 'minLength') || getValidation(currentQ, 'maxLength')">
          <span v-if="getValidation(currentQ, 'minLength')">最少 {{ getValidation(currentQ, 'minLength') }} 字</span>
          <span v-if="getValidation(currentQ, 'maxLength')"> 最多 {{ getValidation(currentQ, 'maxLength') }} 字</span>
        </p>
      </div>

      <!-- NUMBER -->
      <div v-if="currentQ.type==='NUMBER'">
        <input type="number" v-model.number="answers[currentQ.questionId]" style="width:100%" />
        <p style="font-size:12px;color:#888">
          <span v-if="getValidation(currentQ, 'minVal') != null">最小值 {{ getValidation(currentQ, 'minVal') }}</span>
          <span v-if="getValidation(currentQ, 'maxVal') != null"> 最大值 {{ getValidation(currentQ, 'maxVal') }}</span>
          <span v-if="getValidation(currentQ, 'numberType') === 'integer'"> (整数)</span>
        </p>
      </div>

      <p class="error" v-if="qError">{{ qError }}</p>

      <div style="margin-top:15px;display:flex;gap:10px">
        <button class="secondary" v-if="history.length>0" @click="goBack">上一题</button>
        <button @click="goNext">{{ isLast ? '提交' : '下一题' }}</button>
      </div>
    </div>

    <div class="form-group" v-if="survey.settings && survey.settings.allowAnonymous" style="margin-top:10px">
      <label><input type="checkbox" v-model="isAnonymous" /> 匿名提交</label>
    </div>
  </div>
</div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import api from '../api'

const route = useRoute()
const router = useRouter()
const surveyId = route.params.id

const loading = ref(true)
const errMsg = ref('')
const submitted = ref(false)
const survey = ref({})
const answers = reactive({})
const isAnonymous = ref(false)
const qError = ref('')
const startTime = Date.now()

// Navigation state
const currentQIndex = ref(0)
const history = ref([]) // stack of previous indices for back navigation
const visitedOrder = ref([]) // ordered list of visited question indices

const questions = computed(() => survey.value.questions || [])
const currentQ = computed(() => questions.value[currentQIndex.value])
const isLast = computed(() => {
  if (!currentQ.value) return true
  const nextIdx = getNextIndex()
  return nextIdx === null
})

// Helper to safely get validation fields
function getValidation(q, field) {
  if (!q || !q.validation) return undefined
  return q.validation[field]
}

onMounted(async () => {
  try {
    const res = await api.getSurvey(surveyId)
    survey.value = res.data.data
    // Initialize answers
    for (const q of survey.value.questions || []) {
      if (q.type === 'MULTIPLE_CHOICE') answers[q.questionId] = []
      else answers[q.questionId] = q.type === 'NUMBER' ? null : ''
    }
  } catch (e) {
    errMsg.value = e.response?.data?.message || '无法加载问卷'
  } finally {
    loading.value = false
  }
})

function validateCurrent() {
  const q = currentQ.value
  if (!q) return true
  const val = answers[q.questionId]
  const v = q.validation || {}

  if (q.isRequired) {
    if (q.type === 'SINGLE_CHOICE' && !val) return '请选择一个选项'
    if (q.type === 'MULTIPLE_CHOICE' && (!val || val.length === 0)) return '请至少选择一个选项'
    if (q.type === 'TEXT' && (!val || !val.trim())) return '请填写内容'
    if (q.type === 'NUMBER' && (val === null || val === '' || val === undefined)) return '请填写数字'
  }

  if (q.type === 'MULTIPLE_CHOICE' && val && val.length > 0) {
    if (v.minSelect && val.length < v.minSelect) return `至少选择 ${v.minSelect} 个选项`
    if (v.maxSelect && val.length > v.maxSelect) return `最多选择 ${v.maxSelect} 个选项`
  }

  if (q.type === 'TEXT' && val) {
    if (v.minLength && val.length < v.minLength) return `最少输入 ${v.minLength} 个字`
    if (v.maxLength && val.length > v.maxLength) return `最多输入 ${v.maxLength} 个字`
  }

  if (q.type === 'NUMBER' && val !== null && val !== '' && val !== undefined) {
    const num = Number(val)
    if (isNaN(num)) return '请输入有效数字'
    if (v.numberType === 'integer' && !Number.isInteger(num)) return '必须为整数'
    if (v.minVal != null && num < v.minVal) return `不能小于 ${v.minVal}`
    if (v.maxVal != null && num > v.maxVal) return `不能大于 ${v.maxVal}`
  }

  return true
}

function getNextIndex() {
  const q = currentQ.value
  if (!q) return null
  const val = answers[q.questionId]
  const rules = survey.value.logicRules || []

  // Check logic rules
  for (const rule of rules) {
    if (rule.conditionQuestionId !== q.questionId) continue
    if (rule.action !== 'JUMP_TO') continue
    let matched = false
    const rv = rule.conditionValue

    if (rule.operator === 'EQUALS') {
      if (q.type === 'SINGLE_CHOICE') matched = val === rv
      else if (q.type === 'NUMBER') matched = Number(val) === Number(rv)
      else matched = val === rv
    } else if (rule.operator === 'CONTAINS') {
      if (Array.isArray(val)) matched = val.includes(rv)
      else if (typeof val === 'string') matched = val.includes(rv)
    } else if (rule.operator === 'GREATER_THAN') {
      if (q.type === 'NUMBER' && val !== null && val !== '') matched = Number(val) > Number(rv)
    } else if (rule.operator === 'LESS_THAN') {
      if (q.type === 'NUMBER' && val !== null && val !== '') matched = Number(val) < Number(rv)
    }

    if (matched && rule.actionDetails && rule.actionDetails.targetQuestionId) {
      const targetIdx = questions.value.findIndex(qq => qq.questionId === rule.actionDetails.targetQuestionId)
      if (targetIdx >= 0) return targetIdx
    }
  }

  // Default: next sequential question
  const next = currentQIndex.value + 1
  return next < questions.value.length ? next : null
}

function goNext() {
  qError.value = ''
  const v = validateCurrent()
  if (v !== true) { qError.value = v; return }

  const nextIdx = getNextIndex()
  if (nextIdx === null) {
    doSubmit()
  } else {
    history.value.push(currentQIndex.value)
    visitedOrder.value.push(currentQIndex.value)
    currentQIndex.value = nextIdx
  }
}

function goBack() {
  if (history.value.length > 0) {
    currentQIndex.value = history.value.pop()
    qError.value = ''
  }
}

async function doSubmit() {
  // Build answers array only for visited questions + current
  const visited = [...new Set([...visitedOrder.value, currentQIndex.value])]
  const ansArr = []
  for (const idx of visited) {
    const q = questions.value[idx]
    if (!q) continue
    let val = answers[q.questionId]
    if (q.type === 'NUMBER' && val !== null && val !== '' && val !== undefined) val = Number(val)
    if (val === '' || val === null || val === undefined) continue
    if (Array.isArray(val) && val.length === 0) continue
    ansArr.push({ questionId: q.questionId, questionVersionId: q.questionVersionId, value: val })
  }

  const payload = {
    isAnonymous: isAnonymous.value,
    answers: ansArr,
    statistics: { completionTime: Math.round((Date.now() - startTime) / 1000) }
  }

  try {
    await api.submitResponse(surveyId, payload)
    submitted.value = true
  } catch (e) {
    qError.value = e.response?.data?.message || '提交失败'
  }
}
</script>
