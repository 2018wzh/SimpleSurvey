<template>
<div class="container">
  <div class="header">
    <h1>问卷统计</h1>
    <button class="secondary" @click="$router.push('/')">返回</button>
  </div>
  <div v-if="loading" class="card"><p>加载中...</p></div>
  <div v-else-if="errMsg" class="card"><p class="error">{{ errMsg }}</p></div>
  <div v-else>
    <div class="card">
      <h2>{{ survey.title }}</h2>
      <p>总提交数: {{ stats.totalResponses || 0 }}</p>
    </div>

    <div class="card" v-for="qs in stats.questionStats" :key="qs.questionId">
      <h3>{{ getQuestionTitle(qs.questionId) }}</h3>
      <p style="color:#888;font-size:13px">类型: {{ typeLabel(qs.type) }} | 回答人数: {{ qs.totalAnswered || 0 }}</p>

      <!-- 单选/多选 -->
      <div v-if="qs.type==='SINGLE_CHOICE'||qs.type==='MULTIPLE_CHOICE'">
        <table style="width:100%;border-collapse:collapse;margin-top:10px">
          <tr style="border-bottom:1px solid #eee">
            <th style="text-align:left;padding:6px">选项</th>
            <th style="text-align:right;padding:6px">次数</th>
            <th style="text-align:right;padding:6px">占比</th>
          </tr>
          <tr v-for="(count, optId) in (qs.optionCounts || {})" :key="optId" style="border-bottom:1px solid #f5f5f5">
            <td style="padding:6px">{{ getOptionText(qs.questionId, optId) }}</td>
            <td style="text-align:right;padding:6px">{{ count }}</td>
            <td style="text-align:right;padding:6px">{{ qs.totalAnswered ? Math.round(count / qs.totalAnswered * 100) : 0 }}%</td>
          </tr>
        </table>
      </div>

      <!-- 数字填空 -->
      <div v-if="qs.type==='NUMBER'">
        <p v-if="qs.averageValue != null">平均值: {{ qs.averageValue.toFixed(2) }}</p>
        <p style="color:#888;font-size:13px">回答人数: {{ qs.totalAnswered || 0 }}</p>
      </div>

      <!-- 文本填空 -->
      <div v-if="qs.type==='TEXT'">
        <details style="margin-top:8px">
          <summary>查看所有回答 ({{ (qs.textAnswers || []).length }})</summary>
          <ul><li v-for="(r, i) in (qs.textAnswers || [])" :key="i">{{ r }}</li></ul>
        </details>
      </div>
    </div>
  </div>
</div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import api from '../api'

const route = useRoute()
const surveyId = route.params.id
const loading = ref(true)
const errMsg = ref('')
const survey = ref({})
const stats = ref({})

const typeLabel = (t) => ({ SINGLE_CHOICE: '单选题', MULTIPLE_CHOICE: '多选题', TEXT: '文本填空', NUMBER: '数字填空' }[t] || t)

function getQuestionTitle(qid) {
  const q = (survey.value.questions || []).find(q => q.questionId === qid)
  return q ? q.title : qid
}

function getOptionText(qid, optId) {
  const q = (survey.value.questions || []).find(q => q.questionId === qid)
  if (!q || !q.options) return optId
  const opt = q.options.find(o => o.optionId === optId)
  return opt ? opt.text : optId
}

onMounted(async () => {
  try {
    const [sRes, stRes] = await Promise.all([
      api.getQuestionnaireDetail(surveyId),
      api.getStatistics(surveyId)
    ])
    survey.value = sRes.data.data
    stats.value = stRes.data.data
  } catch (e) {
    errMsg.value = e.response?.data?.message || '加载统计失败'
  } finally {
    loading.value = false
  }
})
</script>
