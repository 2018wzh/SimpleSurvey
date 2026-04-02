import { createRouter, createWebHistory } from 'vue-router'

const routes = [
  { path: '/login', name: 'Login', component: () => import('../views/Login.vue'), meta: { guest: true } },
  { path: '/register', name: 'Register', component: () => import('../views/Register.vue'), meta: { guest: true } },
  { path: '/', name: 'Home', component: () => import('../views/Home.vue'), meta: { auth: true } },
  { path: '/create', name: 'CreateSurvey', component: () => import('../views/CreateSurvey.vue'), meta: { auth: true } },
  { path: '/survey/:id', name: 'FillSurvey', component: () => import('../views/FillSurvey.vue') },
  { path: '/stats/:id', name: 'Statistics', component: () => import('../views/Statistics.vue'), meta: { auth: true } }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach((to, from, next) => {
  const token = localStorage.getItem('token')
  if (to.meta.auth && !token) {
    next('/login')
  } else if (to.meta.guest && token) {
    next('/')
  } else {
    next()
  }
})

export default router
