<template>
  <div class="container">
    <div class="header">
      <h1>我的问卷</h1>
      <div>
        <button @click="$router.push('/create')">创建问卷</button>
        <button @click="$router.push('/questions')" style="margin-left: 10px">我的题目</button>
        <button @click="$router.push('/question-banks')" style="margin-left: 10px">我的题库</button>
        <button class="secondary" @click="logout" style="margin-left: 10px">退出</button>
      </div>
    </div>

    <div v-if="questionnaires.length === 0" class="card">
      <p>暂无问卷，点击"创建问卷"开始</p>
    </div>

    <div class="card" v-for="q in questionnaires" :key="q.id">
      <h3>{{ q.title }}</h3>
      <p>状态: {{ statusText(q.status) }} | 创建时间: {{ formatDate(q.createdAt) }}</p>
      <div style="margin-top: 10px">
        <button v-if="q.status === 'draft'" @click="showPublish(q.id)">发布</button>
        <button v-if="q.status === 'published'" class="danger" @click="closeQ(q.id)">关闭</button>
        <button v-if="q.status !== 'draft'" @click="$router.push(`/stats/${q.id}`)" style="margin-left: 10px">统计</button>
        <button v-if="q.status === 'published'" @click="copyLink(q.id)" class="secondary" style="margin-left: 10px">复制链接</button>
      </div>
    </div>

    <!-- 发布对话框 -->
    <div v-if="publishDialog" class="modal-overlay" @click.self="publishDialog = false">
      <div class="card" style="max-width: 400px; margin: 100px auto;">
        <h3>发布问卷</h3>
        <div class="form-group">
          <label>截止时间（可选）</label>
          <input v-model="deadline" type="datetime-local" />
        </div>
        <button @click="doPublish">确认发布</button>
        <button class="secondary" @click="publishDialog = false" style="margin-left: 10px">取消</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import api from '../api'

const router = useRouter()
const auth = useAuthStore()
const questionnaires = ref([])
const publishDialog = ref(false)
const publishId = ref(null)
const deadline = ref('')

const statusText = (s) => ({ draft: '草稿', published: '已发布', closed: '已关闭' }[s] || s)

const formatDate = (d) => d ? new Date(d).toLocaleString('zh-CN') : ''

const loadData = async () => {
  try {
    const res = await api.getQuestionnaires()
    questionnaires.value = res.data.data.items || []
  } catch (e) {
    console.error('加载问卷失败', e)
  }
}

const showPublish = (id) => {
  publishId.value = id
  deadline.value = ''
  publishDialog.value = true
}

const doPublish = async () => {
  try {
    const body = { status: 'published' }
    if (deadline.value) {
      body.deadline = new Date(deadline.value).toISOString()
    }
    await api.updateQuestionnaireStatus(publishId.value, body)
    publishDialog.value = false
    loadData()
  } catch (e) {
    alert(e.response?.data?.message || '发布失败')
  }
}

const closeQ = async (id) => {
  if (!confirm('确定关闭此问卷？')) return
  try {
    await api.updateQuestionnaireStatus(id, { status: 'closed' })
    loadData()
  } catch (e) {
    alert(e.response?.data?.message || '关闭失败')
  }
}

const copyLink = (id) => {
  const link = `${window.location.origin}/survey/${id}`
  navigator.clipboard.writeText(link)
  alert('链接已复制: ' + link)
}

const logout = () => {
  auth.logout()
  router.push('/login')
}

onMounted(loadData)
</script>

<style scoped>
.modal-overlay {
  position: fixed;
  top: 0; left: 0; right: 0; bottom: 0;
  background: rgba(0,0,0,0.4);
  z-index: 100;
}
</style>
