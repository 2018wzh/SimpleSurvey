import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import Questions from '../src/views/Questions.vue'

const apiMocks = vi.hoisted(() => ({
    getMyQuestions: vi.fn(),
    createQuestion: vi.fn(),
    getQuestionVersions: vi.fn(),
    restoreQuestionVersion: vi.fn(),
    getQuestionUsages: vi.fn(),
    getQuestionStats: vi.fn()
}))

vi.mock('../src/api', () => ({
    default: apiMocks
}))

function flushPromises() {
    return new Promise(resolve => setTimeout(resolve, 0))
}

describe('Questions.vue', () => {
    beforeEach(() => {
        Object.values(apiMocks).forEach(mock => mock.mockReset())
    })

    it('creates a new question and loads versions, usages, and stats for an existing one', async () => {
        apiMocks.getMyQuestions.mockResolvedValue({
            data: {
                data: {
                    items: [
                        { id: 'q-1', questionKey: '年龄题', currentVersion: 2, currentVersionId: 'v-2', updatedAt: '2026-04-16T00:00:00Z' }
                    ]
                }
            }
        })
        apiMocks.createQuestion.mockResolvedValue({ data: { data: { id: 'q-new', versionId: 'v-new' } } })
        apiMocks.getQuestionVersions.mockResolvedValue({
            data: {
                data: [
                    { id: 'v-1', version: 1, changeType: 'init', schema: { type: 'TEXT', title: '年龄题' }, note: '初版' },
                    { id: 'v-2', version: 2, changeType: 'edit', schema: { type: 'TEXT', title: '年龄题（更新）' }, note: '更新版' }
                ]
            }
        })
        apiMocks.getQuestionUsages.mockResolvedValue({
            data: {
                data: [
                    { questionnaireId: 'survey-1', questionnaireTitle: '问卷A', status: 'published', questionVersionId: 'v-2' }
                ]
            }
        })
        apiMocks.getQuestionStats.mockResolvedValue({
            data: {
                data: {
                    totalAnswered: 6,
                    type: 'TEXT',
                    textAnswers: ['很好', '继续']
                }
            }
        })
        apiMocks.restoreQuestionVersion.mockResolvedValue({ data: { data: null } })

        const wrapper = mount(Questions, {
            global: {
                stubs: {
                    'router-link': true
                }
            }
        })

        await flushPromises()
        await nextTick()

        expect(wrapper.text()).toContain('年龄题')
        const expandHeader = wrapper.find('strong')
        expect(expandHeader.exists()).toBe(true)
        await expandHeader.trigger('click')
        await nextTick()

        const versionButton = wrapper.findAll('button').find(button => button.text() === '版本历史')
        await versionButton.trigger('click')
        await flushPromises()
        await nextTick()
        expect(wrapper.text()).toContain('版本历史')
        expect(wrapper.text()).toContain('初版')
        expect(wrapper.text()).toContain('更新版')

        const restoreButton = wrapper.findAll('button').find(button => button.text() === '恢复')
        await restoreButton.trigger('click')
        await flushPromises()
        await nextTick()
        expect(apiMocks.restoreQuestionVersion).toHaveBeenCalledWith('q-1', { fromVersionId: 'v-1', note: '恢复旧版本' })

        const usagesButton = wrapper.findAll('button').find(button => button.text() === '使用情况')
        await usagesButton.trigger('click')
        await flushPromises()
        await nextTick()
        expect(wrapper.text()).toContain('问卷A')
        expect(wrapper.text()).toContain('published')

        const statsButton = wrapper.findAll('button').find(button => button.text() === '跨问卷统计')
        await statsButton.trigger('click')
        await flushPromises()
        await nextTick()
        expect(wrapper.text()).toContain('回答人数: 6')
        expect(wrapper.text()).toContain('很好')
    })
})
