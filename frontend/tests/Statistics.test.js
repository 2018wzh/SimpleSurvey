import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import Statistics from '../src/views/Statistics.vue'

const apiMocks = vi.hoisted(() => ({
    getQuestionnaireDetail: vi.fn(),
    getStatistics: vi.fn()
}))

const routeMocks = vi.hoisted(() => ({
    route: { params: { id: 'survey-77' } }
}))

vi.mock('../src/api', () => ({
    default: apiMocks
}))

vi.mock('vue-router', () => ({
    useRoute: () => routeMocks.route
}))

function flushPromises() {
    return new Promise(resolve => setTimeout(resolve, 0))
}

describe('Statistics.vue', () => {
    it('renders questionnaire and question level statistics', async () => {
        apiMocks.getQuestionnaireDetail.mockResolvedValue({
            data: {
                data: {
                    title: '综合统计问卷',
                    questions: [
                        { questionId: 'q1', title: '满意吗', options: [{ optionId: 'a', text: '满意' }, { optionId: 'b', text: '不满意' }] },
                        { questionId: 'q2', title: '年龄', options: [] },
                        { questionId: 'q3', title: '建议', options: [] }
                    ]
                }
            }
        })
        apiMocks.getStatistics.mockResolvedValue({
            data: {
                data: {
                    totalResponses: 6,
                    questionStats: [
                        { questionId: 'q1', type: 'SINGLE_CHOICE', totalAnswered: 6, optionCounts: { a: 4, b: 2 } },
                        { questionId: 'q2', type: 'NUMBER', totalAnswered: 6, averageValue: 23.5 },
                        { questionId: 'q3', type: 'TEXT', textAnswers: ['很好', '继续加油'] }
                    ]
                }
            }
        })

        const wrapper = mount(Statistics, {
            global: {
                stubs: {
                    'router-link': true
                }
            }
        })

        await flushPromises()
        await nextTick()

        expect(apiMocks.getQuestionnaireDetail).toHaveBeenCalledWith('survey-77')
        expect(apiMocks.getStatistics).toHaveBeenCalledWith('survey-77')
        expect(wrapper.text()).toContain('综合统计问卷')
        expect(wrapper.text()).toContain('总提交数: 6')
        expect(wrapper.text()).toContain('满意吗')
        expect(wrapper.text()).toContain('满意')
        expect(wrapper.text()).toContain('4')
        expect(wrapper.text()).toContain('23.50')
        expect(wrapper.text()).toContain('查看所有回答 (2)')
    })
})
