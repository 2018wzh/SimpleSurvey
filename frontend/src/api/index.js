import axios from 'axios'

const envBaseURL = (import.meta.env.VITE_API_BASE_URL || '').trim()

const api = axios.create({
  baseURL: envBaseURL || '/api/v1',
  headers: { 'Content-Type': 'application/json' }
})

// Request interceptor: attach access token
api.interceptors.request.use(config => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// Response interceptor: unwrap { code, message, data } envelope
api.interceptors.response.use(
  res => res,
  err => {
    // If 401, clear token and redirect to login
    if (err.response && err.response.status === 401) {
      localStorage.removeItem('token')
      localStorage.removeItem('refreshToken')
      if (window.location.pathname !== '/login' && window.location.pathname !== '/register') {
        window.location.href = '/login'
      }
    }
    return Promise.reject(err)
  }
)

export default {
  // ========== Auth ==========
  register(data) {
    return api.post('/auth/register', data)
  },
  login(data) {
    return api.post('/auth/login', data)
  },
  refreshToken(refreshToken) {
    return api.post('/auth/refresh', { refreshToken })
  },

  // ========== Questionnaires (creator side) ==========
  createQuestionnaire(data) {
    return api.post('/questionnaires', data)
  },
  getQuestionnaires(params) {
    return api.get('/questionnaires', { params })
  },
  getQuestionnaireDetail(id) {
    return api.get(`/questionnaires/${id}`)
  },
  updateQuestionnaireStatus(id, data) {
    return api.patch(`/questionnaires/${id}/status`, data)
  },
  getQuestionnaireStats(id) {
    return api.get(`/questionnaires/${id}/stats`)
  },
  // Alias used by Statistics.vue
  getStatistics(id) {
    return api.get(`/questionnaires/${id}/stats`)
  },
  getQuestionnaireResponses(id, params) {
    return api.get(`/questionnaires/${id}/responses`, { params })
  },

  // ========== Surveys (fill side) ==========
  getSurvey(id) {
    return api.get(`/surveys/${id}`)
  },
  submitResponse(id, data) {
    return api.post(`/surveys/${id}/responses`, data)
  }
}
