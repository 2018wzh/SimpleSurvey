<template>
<div class="container">
  <div class="header">
    <h1>我的题库</h1>
    <button class="secondary" @click="$router.push('/')">返回</button>
  </div>

  <div class="card" style="margin-bottom:15px">
    <h3>创建新题库</h3>
    <div class="form-group"><label>题库名称</label><input v-model="newBank.name" /></div>
    <div class="form-group"><label>描述</label><input v-model="newBank.description" /></div>
    <div class="form-group">
      <label>可见性</label>
      <select v-model="newBank.visibility">
        <option value="private">私有</option>
        <option value="team">团队</option>
      </select>
    </div>
    <button @click="createBank">创建</button>
  </div>

  <div v-if="loading" class="card"><p>加载中...</p></div>
  <div v-for="bank in banks" :key="bank.id" class="card" style="margin-bottom:12px">
    <div style="display:flex;justify-content:space-between;align-items:center;cursor:pointer" @click="toggleExpand(bank.id)">
      <div>
        <strong>{{ bank.name }}</strong>
        <span style="font-size:12px;color:#888;margin-left:8px">[{{ bank.visibility==='team'?'团队':'私有' }}]</span>
        <p style="margin:4px 0 0;color:#888;font-size:13px">{{ bank.description || '无描述' }}</p>
      </div>
      <span style="font-size:12px;color:#888">{{ expanded[bank.id] ? '收起' : '展开' }}</span>
    </div>

    <div v-if="expanded[bank.id]" style="margin-top:15px;border-top:1px solid #eee;padding-top:15px">
      <!-- Edit base info -->
      <div style="margin-bottom:10px">
        <input v-model="editForms[bank.id].name" style="width:150px" />
        <input v-model="editForms[bank.id].description" style="width:150px;margin-left:8px" />
        <select v-model="editForms[bank.id].visibility" style="margin-left:8px">
          <option value="private">私有</option>
          <option value="team">团队</option>
        </select>
        <button @click="updateBank(bank.id)" style="margin-left:8px">更新</button>
      </div>

      <!-- Items -->
      <h4>题库题目</h4>
      <div v-if="bank.items && bank.items.length">
        <div v-for="item in bank.items" :key="item.questionId" style="display:flex;justify-content:space-between;align-items:center;padding:6px;border:1px solid #eee;border-radius:4px;margin-bottom:6px">
          <div>
            <span style="font-size:14px">{{ item.questionId }}</span>
            <p v-if="item.pinnedVersionId" style="margin:2px 0 0;font-size:12px;color:#888">固定版本: {{ item.pinnedVersionId }}</p>
          </div>
          <button class="danger" @click="removeItem(bank.id, item.questionId)" style="padding:2px 8px;font-size:12px">移除</button>
        </div>
      </div>
      <div v-else style="color:#888;font-size:13px">暂无题目</div>

      <div style="margin-top:10px">
        <select v-model="addForms[bank.id].questionId" @change="onQuestionChange(bank.id)" style="width:200px">
          <option value="">选择题目</option>
          <option v-for="q in myQuestions" :key="q.id" :value="q.id">{{ q.questionKey }} (v{{ q.currentVersion }})</option>
        </select>
        <select v-model="addForms[bank.id].pinnedVersionId" style="width:160px;margin-left:8px" :disabled="!addForms[bank.id].questionId || !questionVersions[addForms[bank.id].questionId]?.length">
          <option value="">最新版本</option>
          <option v-for="v in (questionVersions[addForms[bank.id].questionId] || [])" :key="v.id" :value="v.id">v{{ v.version }} - {{ v.schema?.title || v.id }}</option>
        </select>
        <button @click="addItem(bank.id)" style="margin-left:8px">添加题目</button>
      </div>

      <!-- Shares -->
      <h4 style="margin-top:15px">共享</h4>
      <div v-if="bank.sharedWith && bank.sharedWith.length">
        <div v-for="s in bank.sharedWith" :key="s.userId" style="display:flex;justify-content:space-between;align-items:center;padding:4px 0">
          <span style="font-size:13px">用户 {{ s.userId }} - 权限: {{ s.permission }}</span>
          <button class="danger" @click="unshare(bank.id, s.userId)" style="padding:2px 8px;font-size:12px">取消共享</button>
        </div>
      </div>
      <div v-else style="color:#888;font-size:13px">未共享给任何人</div>

      <div style="margin-top:10px">
        <select v-model="shareForms[bank.id].targetUserId" style="width:160px">
          <option value="">选择用户</option>
          <option v-for="u in allUsers" :key="u.id" :value="u.id">{{ u.username }}</option>
        </select>
        <select v-model="shareForms[bank.id].permission" style="margin-left:8px">
          <option value="use">使用</option>
          <option value="manage">管理</option>
        </select>
        <button @click="shareBank(bank.id)" style="margin-left:8px">共享</button>
      </div>
    </div>
  </div>
</div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import api from '../api'

const banks = ref([])
const loading = ref(false)
const expanded = reactive({})
const newBank = reactive({ name: '', description: '', visibility: 'private' })
const editForms = reactive({})
const addForms = reactive({})
const shareForms = reactive({})
const myQuestions = ref([])
const questionVersions = reactive({})
const allUsers = ref([])

async function loadData() {
  loading.value = true
  try {
    const [banksRes, questionsRes, usersRes] = await Promise.all([
      api.getQuestionBanks({ limit: 100 }),
      api.getMyQuestions({ limit: 100 }),
      api.getUsers({ limit: 1000 })
    ])
    const items = banksRes.data.data.items || []
    banks.value = items
    myQuestions.value = questionsRes.data.data.items || []
    allUsers.value = usersRes.data.data.items || []
    items.forEach(b => {
      if (!editForms[b.id]) editForms[b.id] = { name: b.name, description: b.description || '', visibility: b.visibility }
      if (!addForms[b.id]) addForms[b.id] = { questionId: '', pinnedVersionId: '' }
      if (!shareForms[b.id]) shareForms[b.id] = { targetUserId: '', permission: 'use' }
    })
  } catch (e) {
    alert('加载题库失败')
  } finally {
    loading.value = false
  }
}

function toggleExpand(id) {
  expanded[id] = !expanded[id]
}

async function createBank() {
  if (!newBank.name.trim()) { alert('请输入题库名称'); return }
  try {
    await api.createQuestionBank({
      name: newBank.name,
      description: newBank.description,
      visibility: newBank.visibility
    })
    alert('创建成功')
    newBank.name = ''
    newBank.description = ''
    newBank.visibility = 'private'
    loadData()
  } catch (e) {
    alert(e.response?.data?.message || '创建失败')
  }
}

async function updateBank(id) {
  try {
    await api.updateQuestionBank(id, editForms[id])
    alert('更新成功')
    loadData()
  } catch (e) {
    alert(e.response?.data?.message || '更新失败')
  }
}

async function addItem(id) {
  const f = addForms[id]
  if (!f.questionId.trim()) { alert('请选择题目'); return }
  try {
    const body = { questionId: f.questionId.trim() }
    if (f.pinnedVersionId.trim()) body.pinnedVersionId = f.pinnedVersionId.trim()
    await api.addQuestionBankItem(id, body)
    alert('添加成功')
    f.questionId = ''
    f.pinnedVersionId = ''
    loadData()
  } catch (e) {
    alert(e.response?.data?.message || '添加失败')
  }
}

async function onQuestionChange(bankId) {
  const qid = addForms[bankId].questionId
  addForms[bankId].pinnedVersionId = ''
  if (!qid) {
    questionVersions[qid] = []
    return
  }
  try {
    const res = await api.getQuestionVersions(qid)
    questionVersions[qid] = res.data.data || []
  } catch (e) {
    questionVersions[qid] = []
  }
}

async function removeItem(bankId, questionId) {
  if (!confirm('确定移除此题目？')) return
  try {
    await api.removeQuestionBankItem(bankId, questionId)
    alert('移除成功')
    loadData()
  } catch (e) {
    alert(e.response?.data?.message || '移除失败')
  }
}

async function shareBank(id) {
  const f = shareForms[id]
  if (!f.targetUserId.trim()) { alert('请选择用户'); return }
  try {
    await api.shareQuestionBank(id, { targetUserId: f.targetUserId.trim(), permission: f.permission })
    alert('共享成功')
    f.targetUserId = ''
    f.permission = 'use'
    loadData()
  } catch (e) {
    alert(e.response?.data?.message || '共享失败')
  }
}

async function unshare(bankId, targetUserId) {
  try {
    await api.unshareQuestionBank(bankId, targetUserId)
    alert('取消共享成功')
    loadData()
  } catch (e) {
    alert(e.response?.data?.message || '取消共享失败')
  }
}

onMounted(loadData)
</script>
