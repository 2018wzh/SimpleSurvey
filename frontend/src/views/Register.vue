<template>
  <div class="auth-container">
    <h2>注册</h2>
    <form @submit.prevent="handleRegister">
      <div class="form-group">
        <label>用户名</label>
        <input v-model="username" required />
      </div>
      <div class="form-group">
        <label>密码</label>
        <input v-model="password" type="password" required />
      </div>
      <div class="error" v-if="error">{{ error }}</div>
      <button type="submit">注册</button>
      <p style="margin-top: 15px">
        已有账号？<router-link to="/login">登录</router-link>
      </p>
    </form>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import api from '../api'

const router = useRouter()
const username = ref('')
const password = ref('')
const error = ref('')

const handleRegister = async () => {
  try {
    error.value = ''
    await api.register({ username: username.value, password: password.value })
    alert('注册成功，请登录')
    router.push('/login')
  } catch (e) {
    error.value = e.response?.data?.message || '注册失败'
  }
}
</script>
