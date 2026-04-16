import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import Home from '../src/views/Home.vue'

const apiMocks = vi.hoisted(() => ({
    getQuestionnaires: vi.fn(),
    updateQuestionnaireStatus: vi.fn()
}))

const authMocks = vi.hoisted(() => ({
    logout: vi.fn()
}))

const routerPush = vi.hoisted(() => vi.fn())

vi.mock('../src/api', () => ({
    default: apiMocks
}))

vi.mock('../src/stores/auth', () => ({
    useAuthStore: () => authMocks
}))

vi.mock('vue-router', () => ({
    useRouter: () => ({ push: routerPush })
}))

function flushPromises() {
    return new Promise(resolve => setTimeout(resolve, 0))
}

describe('Home.vue', () => {
    beforeEach(() => {
        apiMocks.getQuestionnaires.mockReset()
        apiMocks.updateQuestionnaireStatus.mockReset()
        authMocks.logout.mockReset()
        routerPush.mockReset()
        vi.stubGlobal('confirm', vi.fn(() => true))
        navigator.clipboard.writeText.mockReset()
    })

    it('loads questionnaires and supports publish, copy, and logout actions', async () => {
        apiMocks.getQuestionnaires.mockResolvedValue({
            data: {
                data: {
                    items: [
                        { id: 'q-draft', title: '草稿问卷', status: 'draft', createdAt: '2026-04-15T12:00:00Z' },
                        { id: 'q-live', title: '已发布问卷', status: 'published', createdAt: '2026-04-16T12:00:00Z' }
                    ]
                }
            }
        })
        apiMocks.updateQuestionnaireStatus.mockResolvedValue({ data: { data: null } })

        const wrapper = mount(Home, {
            global: {
                stubs: {
                    'router-link': true
                }
            }
        })

        await flushPromises()
        await nextTick()

        expect(wrapper.text()).toContain('草稿问卷')
        expect(wrapper.text()).toContain('已发布问卷')
        expect(wrapper.findAll('button').some(button => button.text() === '发布')).toBe(true)
        expect(wrapper.findAll('button').some(button => button.text() === '复制链接')).toBe(true)

        const publishButton = wrapper.findAll('button').find(button => button.text() === '发布')
        await publishButton.trigger('click')
        await nextTick()

        const deadlineInput = wrapper.find('input[type="datetime-local"]')
        await deadlineInput.setValue('2026-04-20T10:00')
        const confirmButton = wrapper.findAll('button').find(button => button.text() === '确认发布')
        await confirmButton.trigger('click')
        await flushPromises()
        await nextTick()

        expect(apiMocks.updateQuestionnaireStatus).toHaveBeenCalledWith('q-draft', {
            status: 'published',
            deadline: '2026-04-20T02:00:00.000Z'
        })

        const copyButton = wrapper.findAll('button').find(button => button.text() === '复制链接')
        await copyButton.trigger('click')
        expect(navigator.clipboard.writeText).toHaveBeenCalledWith(expect.stringContaining('/survey/q-live'))
        expect(alert).toHaveBeenCalledWith(expect.stringContaining('链接已复制: '))

        const logoutButton = wrapper.findAll('button').find(button => button.text() === '退出')
        await logoutButton.trigger('click')
        expect(authMocks.logout).toHaveBeenCalled()
        expect(routerPush).toHaveBeenCalledWith('/login')
    })
})
