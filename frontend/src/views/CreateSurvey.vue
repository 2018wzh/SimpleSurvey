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
  <div class="question-item" v-for="(q, qi) in form.questions" :key="qi">
    <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:10px">
      <strong>题目 {{ qi + 1 }} ({{ q.questionId }})</strong>
      <button class="danger" @click="removeQ(qi)" style="padding:4px 10px">删除</button>
    </div>

    <div class="form-group"><label>题目标题</label><input v-model="q.title" /></div>

    <div class="form-group">
      <label>题型</label>
      <select v-model="q.type" @change="onTypeChange(qi)">
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

    <!-- 选项 (单选/多选) -->
    <div v-if="q.type === 'SINGLE_CHOICE' || q.type === 'MULTIPLE_CHOICE'">
      <div class="option-item" v-for="(opt, oi) in q.options" :key="oi">
        <input v-model="opt.text" :placeholder="'选项' + (oi + 1)" style="flex:1" />
        <button class="danger" @click="q.options.splice(oi, 1)" style="padding:4px 8px">×</button>
      </div>
      <button @click="addOption(qi)" style="padding:4px 12px;margin-top:5px">+ 添加选项</button>
    </div>

    <!-- 多选限制 -->
    <div v-if="q.type === 'MULTIPLE_CHOICE'" style="margin-top:10px">
      <div class="form-group"><label>最少选择</label><input v-model.number="q.validation.minSelect" type="number" min="0" /></div>
      <div class="form-group"><label>最多选择</label><input v-model.number="q.validation.maxSelect" type="number" min="0" /></div>
    </div>

    <!-- 文本限制 -->
    <div v-if="q.type === 'TEXT'" style="margin-top:10px">
      <div class="form-group"><label>最少字数</label><input v-model.number="q.validation.minLength" type="number" min="0" /></div>
      <div class="form-group"><label>最多字数</label><input v-model.number="q.validation.maxLength" type="number" min="0" /></div>
    </div>

    <!-- 数字限制 -->
    <div v-if="q.type === 'NUMBER'" style="margin-top:10px">
      <div class="form-group"><label>最小值</label><input v-model.number="q.validation.minVal" type="number" /></div>
      <div class="form-group"><label>最大值</label><input v-model.number="q.validation.maxVal" type="number" /></div>
      <div class="form-group"><label><input type="checkbox" v-model="q.integerOnly" /> 必须为整数</label></div>
    </div>
  </div>

  <button @click="addQuestion" style="margin-bottom:20px">+ 添加题目</button>

  <h2 style="margin:20px 0 10px">跳转逻辑</h2>
  <div class="question-item" v-for="(rule, ri) in form.logicRules" :key="ri">
    <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:10px">
      <strong>规则 {{ ri + 1 }}</strong>
      <button class="danger" @click="form.logicRules.splice(ri, 1)" style="padding:4px 10px">删除</button>
    </div>

    <div class="form-group">
      <label>源题目ID</label>
      <select v-model="rule.conditionQuestionId">
        <option value="">请选择源题目</option>
        <option
          v-for="q in form.questions"
          :key="q.questionId"
          :value="q.questionId"
        >
          {{ q.questionId }} - {{ q.title || '未命名题目' }}
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

      <select
        v-if="isChoiceQuestion(rule.conditionQuestionId)"
        v-model="rule.conditionValue"
      >
        <option value="">请选择条件值</option>
        <option
          v-for="opt in getQuestionOptions(rule.conditionQuestionId)"
          :key="opt.optionId"
          :value="opt.optionId"
        >
          {{ opt.optionId }} - {{ opt.text || '未命名选项' }}
        </option>
      </select>

      <input
        v-else
        v-model="rule.conditionValue"
        :type="isNumberQuestion(rule.conditionQuestionId) ? 'number' : 'text'"
      />
    </div>

    <div class="form-group">
      <label>跳转到题目ID</label>
      <select v-model="rule.targetQuestionId">
        <option value="">请选择目标题目</option>
        <option
          v-for="q in form.questions"
          :key="q.questionId"
          :value="q.questionId"
        >
          {{ q.questionId }} - {{ q.title || '未命名题目' }}
        </option>
      </select>
    </div>
  </div>

  <button @click="addRule">+ 添加跳转规则</button>

  <div class="error" v-if="error" style="margin-top:15px">{{ error }}</div>
</div>
</template>

<script setup>
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import api from '../api'

const router = useRouter()
const error = ref('')
let qCounter = 0
let oCounter = 0

const form = reactive({
  title: '',
  description: '',
  settings: { allowAnonymous: false, duplicateCheck: 'none', themeColor: '#1677ff' },
  questions: [],
  logicRules: []
})

function nextQId() {
  return 'q' + (++qCounter)
}

function nextOId() {
  return 'o' + (++oCounter)
}

function addQuestion() {
  form.questions.push({
    questionId: nextQId(),
    type: 'SINGLE_CHOICE',
    title: '',
    isRequired: false,
    options: [
      { optionId: nextOId(), text: '' },
      { optionId: nextOId(), text: '' }
    ],
    validation: { minSelect: 0, maxSelect: 0, minLength: 0, maxLength: 0, minVal: null, maxVal: null },
    integerOnly: false
  })
}

function removeQ(i) {
  form.questions.splice(i, 1)
}

function addOption(qi) {
  form.questions[qi].options.push({ optionId: nextOId(), text: '' })
}

function onTypeChange(qi) {
  const q = form.questions[qi]
  if (q.type === 'SINGLE_CHOICE' || q.type === 'MULTIPLE_CHOICE') {
    if (!q.options || q.options.length === 0) {
      q.options = [
        { optionId: nextOId(), text: '' },
        { optionId: nextOId(), text: '' }
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
  form.logicRules.push({
    conditionQuestionId: '',
    operator: 'EQUALS',
    conditionValue: '',
    targetQuestionId: ''
  })
}

async function submit() {
  error.value = ''
  if (!form.title.trim()) {
    error.value = '请输入问卷标题'
    return
  }
  if (form.questions.length === 0) {
    error.value = '请至少添加一道题目'
    return
  }

  const payload = {
    title: form.title,
    description: form.description,
    settings: { ...form.settings },
    questions: form.questions.map(q => {
      const out = {
        questionId: q.questionId,
        type: q.type,
        title: q.title,
        isRequired: q.isRequired
      }
      if (q.type === 'SINGLE_CHOICE' || q.type === 'MULTIPLE_CHOICE') {
        out.options = q.options.filter(o => o.text.trim())
      }
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
      if (Object.keys(v).length > 0) out.validation = v
      return out
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
