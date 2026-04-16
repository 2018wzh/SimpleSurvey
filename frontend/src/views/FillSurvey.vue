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
import { buildSurveyResponsePayload, getNextQuestionIndex, validateQuestionAnswer } from '../utils/surveyRuntime'

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
  return validateQuestionAnswer(q, val)
}

function getNextIndex() {
  return getNextQuestionIndex({
    currentIndex: currentQIndex.value,
    questions: questions.value,
    answers,
    logicRules: survey.value.logicRules || []
  })
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
  const payload = buildSurveyResponsePayload({
    questions: questions.value,
    answers,
    visitedOrder: visitedOrder.value,
    currentIndex: currentQIndex.value,
    isAnonymous: isAnonymous.value,
    startTime
  })

  try {
    await api.submitResponse(surveyId, payload)
    submitted.value = true
  } catch (e) {
    qError.value = e.response?.data?.message || '提交失败'
  }
}
</script>
